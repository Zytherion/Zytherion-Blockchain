package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

// TestP2PScore_NewValidator verifies that a validator with no recorded latency
// receives a perfect score of 1.0 (benefit of the doubt).
func TestP2PScore_NewValidator(t *testing.T) {
	k, ctx := newTestKeeper(t)
	consAddr := sdk.ConsAddress([]byte("validator_addr_00001"))

	score := k.GetValidatorP2PScore(ctx, consAddr)
	require.InDelta(t, 1.0, score, 1e-9, "new validator must have score 1.0")
}

// TestP2PScore_Calculation verifies the score formula at specific latencies.
func TestP2PScore_Calculation(t *testing.T) {
	k, ctx := newTestKeeper(t)
	consAddr := sdk.ConsAddress([]byte("validator_addr_00002"))

	// Record a mid-range latency of 1000ms (half of maxLatencyMs=2000ms).
	// After one observation the EMA equals the raw value.
	k.RecordValidatorLatency(ctx, consAddr, 1000)
	score := k.GetValidatorP2PScore(ctx, consAddr)
	// score = 1 - 1000/2000 = 0.5
	require.InDelta(t, 0.5, score, 0.01, "1000ms latency → score ~0.5")
}

// TestP2PScore_MaxLatency verifies that a very high latency caps the score
// at 0.0 (ratio clamped to 1.0).
func TestP2PScore_MaxLatency(t *testing.T) {
	k, ctx := newTestKeeper(t)
	consAddr := sdk.ConsAddress([]byte("validator_addr_00003"))

	// Record latency well above 2000ms threshold.
	k.RecordValidatorLatency(ctx, consAddr, 9999)
	score := k.GetValidatorP2PScore(ctx, consAddr)
	require.InDelta(t, 0.0, score, 0.01, "extreme latency → score ~0.0")
}

// TestApplyP2PPenalty_FullReward verifies that a perfect-score validator
// receives 100% of the base reward.
func TestApplyP2PPenalty_FullReward(t *testing.T) {
	k, ctx := newTestKeeper(t)
	consAddr := sdk.ConsAddress([]byte("validator_addr_00004"))

	// No latency recorded → score=1.0 → multiplier=1.0.
	base := sdk.MustNewDecFromStr("1000.000000")
	result := k.ApplyP2PPenalty(ctx, consAddr, base)
	require.True(t, result.Equal(base),
		"zero latency must give full reward, got %s", result)
}

// TestApplyP2PPenalty_MaxPenalty verifies that the reward penalty is capped
// at 15% (MinScoreMultiplier = 0.85) even for the worst validators.
func TestApplyP2PPenalty_MaxPenalty(t *testing.T) {
	k, ctx := newTestKeeper(t)
	consAddr := sdk.ConsAddress([]byte("validator_addr_00005"))

	// Record very high latency to drive score → 0.
	k.RecordValidatorLatency(ctx, consAddr, 9999)

	base := sdk.MustNewDecFromStr("1000.000000")
	result := k.ApplyP2PPenalty(ctx, consAddr, base)

	// Result must not be less than 85% of base.
	minExpected := sdk.MustNewDecFromStr("850.000000")
	require.True(t, result.GTE(minExpected),
		"penalty must not exceed 15%%, result=%s min=%s", result, minExpected)
	require.True(t, result.LT(base),
		"high-latency validator must receive less than full reward")
}

// TestRecordValidatorLatency_EMA verifies that the running average smooths
// out transient spikes (first spike + low value → avg < spike).
func TestRecordValidatorLatency_EMA(t *testing.T) {
	k, ctx := newTestKeeper(t)
	consAddr := sdk.ConsAddress([]byte("validator_addr_00006"))

	// First observation seeds the EMA.
	k.RecordValidatorLatency(ctx, consAddr, 100)
	score1 := k.GetValidatorP2PScore(ctx, consAddr)

	// Second observation: very high latency spike.
	k.RecordValidatorLatency(ctx, consAddr, 9999)
	score2 := k.GetValidatorP2PScore(ctx, consAddr)

	// Score after spike must be lower, but not zero (EMA smoothing).
	require.Less(t, score2, score1, "spike should lower score")
	require.Greater(t, score2, 0.0, "EMA must prevent score from hitting 0 on single spike")
}
