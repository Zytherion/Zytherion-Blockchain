// green_slashing.go — Performance-to-Power (P2P) energy efficiency metric.
//
// This file adds a latency tracking system to the privacy keeper that forms
// the basis for the "Energy-Efficient Slashing" Green BFT feature.
//
// Design:
//   - Each validator's average block-signing latency is maintained as a
//     running exponential moving average (EMA) stored in the privacy KVStore.
//   - The P2P score (0.0–1.0) is derived from this latency: low latency → high
//     score → full rewards; high latency → score approaches MinScoreMultiplier.
//   - Applying the penalty via ApplyP2PPenalty keeps the x/slashing module
//     completely unmodified — rewards are adjusted at the app/distribution layer.

package keeper

import (
	"encoding/binary"
	"fmt"
	"math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	// validatorLatencyKeyPrefix is the KVStore key prefix for per-validator
	// latency data. Full key: prefix + consensus address bytes.
	validatorLatencyKeyPrefix = "greenbft/latency/"

	// maxLatencyMs is the latency (ms) at which a validator's P2P score
	// reaches its minimum. Validators slower than this cap receive MinScoreMultiplier.
	maxLatencyMs = 2000 // 2 seconds

	// emaAlpha is the smoothing factor for the exponential moving average.
	// A value of 0.2 gives significant weight to recent samples while smoothing
	// out transient network spikes.
	emaAlpha = 0.2

	// MinScoreMultiplier is the floor of the reward multiplier applied to a
	// validator with maximum latency. A value of 0.85 means validators can
	// lose at most 15% of their rewards — punitive enough to incentivise
	// optimisation without destabilising the validator set.
	MinScoreMultiplier = 0.85
)

// validatorLatencyKey returns the full KVStore key for a validator's EMA latency.
func validatorLatencyKey(consAddr sdk.ConsAddress) []byte {
	prefix := []byte(validatorLatencyKeyPrefix)
	key := make([]byte, len(prefix)+len(consAddr))
	copy(key, prefix)
	copy(key[len(prefix):], consAddr)
	return key
}

// getStoredLatencyMs retrieves the stored EMA latency for a validator in ms.
// Returns 0 if not yet stored (new or never recorded).
func (k Keeper) getStoredLatencyMs(ctx sdk.Context, consAddr sdk.ConsAddress) float64 {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(validatorLatencyKey(consAddr))
	if len(bz) < 8 {
		return 0
	}
	bits := binary.BigEndian.Uint64(bz)
	return math.Float64frombits(bits)
}

// setStoredLatencyMs writes the EMA latency for a validator.
func (k Keeper) setStoredLatencyMs(ctx sdk.Context, consAddr sdk.ConsAddress, latencyMs float64) {
	store := ctx.KVStore(k.storeKey)
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], math.Float64bits(latencyMs))
	store.Set(validatorLatencyKey(consAddr), buf[:])
}

// RecordValidatorLatency records an observed block-signing latency for a
// validator using an exponential moving average so that transient network
// spikes don't permanently penalise a well-behaved validator.
//
// latencyMs should be the duration in milliseconds between the block proposal
// time and when the validator's PreCommit vote was observed.
func (k Keeper) RecordValidatorLatency(ctx sdk.Context, consAddr sdk.ConsAddress, latencyMs int64) {
	current := k.getStoredLatencyMs(ctx, consAddr)
	var updated float64
	if current == 0 {
		// First observation — seed the EMA with the raw value.
		updated = float64(latencyMs)
	} else {
		// EMA: updated = α * new + (1-α) * old
		updated = emaAlpha*float64(latencyMs) + (1-emaAlpha)*current
	}
	k.setStoredLatencyMs(ctx, consAddr, updated)

	k.Logger(ctx).Debug("greenbft latency recorded",
		"validator", consAddr.String(),
		"latency_ms", latencyMs,
		"ema_ms", updated,
	)
}

// GetValidatorP2PScore returns a normalised Performance-to-Power score in
// [0.0, 1.0] for the given validator.
//
//	score = 1 – clamp(avgLatencyMs / maxLatencyMs, 0, 1)
//
// A score of 1.0 means the validator has zero observed latency (optimal).
// A score approaching 0.0 means the validator is consistently at or above
// maxLatencyMs (wasted energy, network drag).
//
// Validators with no recorded latency receive a score of 1.0 (benefit of doubt).
func (k Keeper) GetValidatorP2PScore(ctx sdk.Context, consAddr sdk.ConsAddress) float64 {
	avg := k.getStoredLatencyMs(ctx, consAddr)
	if avg <= 0 {
		return 1.0
	}
	ratio := avg / float64(maxLatencyMs)
	if ratio > 1.0 {
		ratio = 1.0
	}
	return 1.0 - ratio
}

// ApplyP2PPenalty scales baseReward by the validator's P2P score, capped
// at MinScoreMultiplier so the penalty is at most (1 - MinScoreMultiplier) = 15%.
//
// Example:
//
//	score 1.0 → multiplier 1.0  → full reward
//	score 0.5 → multiplier 0.925 → slight reduction
//	score 0.0 → multiplier 0.85  → maximum 15% reduction
//
// This is additive to existing slashing — a double-signing validator still
// faces full slashing on top of any P2P penalty.
func (k Keeper) ApplyP2PPenalty(ctx sdk.Context, consAddr sdk.ConsAddress, baseReward sdk.Dec) sdk.Dec {
	score := k.GetValidatorP2PScore(ctx, consAddr)

	// Map score [0,1] → multiplier [MinScoreMultiplier, 1.0]
	// multiplier = MinScoreMultiplier + score * (1.0 - MinScoreMultiplier)
	multiplier := MinScoreMultiplier + score*(1.0-MinScoreMultiplier)

	mulDec := sdk.MustNewDecFromStr(formatFloat(multiplier))
	result := baseReward.Mul(mulDec)

	k.Logger(ctx).Debug("greenbft P2P penalty applied",
		"validator", consAddr.String(),
		"p2p_score", score,
		"multiplier", multiplier,
		"base_reward", baseReward.String(),
		"adjusted_reward", result.String(),
	)

	return result
}

// formatFloat converts a float64 to a string with enough precision for sdk.Dec.
func formatFloat(f float64) string {
	// sdk.Dec uses 18 decimal places; %.18f is precise enough here.
	return fmt.Sprintf("%.18f", f)
}
