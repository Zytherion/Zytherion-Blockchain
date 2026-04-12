// Package greenbft implements Green BFT efficiency features for the Zytherion
// consensus layer. These features reduce validator energy consumption by
// dynamically adjusting block timing and applying performance-to-power metrics.
package greenbft

import (
	"encoding/binary"
	"sync/atomic"
	"time"

	"github.com/cometbft/cometbft/libs/log"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	// DefaultCommitTimeout is the target timeout when the chain is active.
	DefaultCommitTimeout = 1 * time.Second

	// IdleCommitTimeout is the extended timeout applied when the mempool is
	// consistently empty, saving validator CPU cycles during quiet periods.
	IdleCommitTimeout = 5 * time.Second

	// LowTxThreshold is the rolling-average transaction count below which the
	// chain is considered "idle" and IdleCommitTimeout is recommended.
	LowTxThreshold = 5

	// rollingWindow is the number of recent blocks used for the rolling average.
	rollingWindow = 10

	// adaptiveTimeoutKey is the KVStore key used to persist the suggested timeout.
	adaptiveTimeoutKey = "greenbft/timeout_ns"
)

// AdaptiveTimeoutManager tracks block loads over a rolling window and suggests
// a commit timeout that saves CPU when the mempool is consistently empty.
//
// ── How block timing works in CometBFT v0.37 ────────────────────────────────
// timeout_commit is a NODE-LOCAL configuration parameter (config.toml). It
// cannot be changed via ResponseEndBlock.ConsensusParamUpdates because those
// only cover Block / Evidence / Validator / Version params.
//
// What we DO instead:
//   1. Persist the recommended timeout in KVStore key "greenbft/timeout_ns".
//   2. Return it in an ABCI Event ("green_bft.recommended_timeout_ms") so that
//      any monitoring process or sidecar can read it from block results and
//      apply it to config.toml via SIGHUP / zero-downtime config reload.
//   3. Use Block.MaxGas in ConsensusParamUpdates as a machine-readable signal
//      of the current load state that CometBFT DOES propagate to all nodes.
//
// This approach is fully compatible with CometBFT v0.37 / Cosmos SDK v0.47.
type AdaptiveTimeoutManager struct {
	storeKey storetypes.StoreKey
	logger   log.Logger

	// txTotal is an atomic counter incremented by the PQC ante decorator for
	// every transaction that passes ante checks within the current block.
	// It is read and reset atomically at the start of each EndBlock.
	txTotal int64

	// circular buffer of recent per-block tx counts
	txCounts [rollingWindow]int64
	head     int
	filled   bool
}

// NewAdaptiveTimeoutManager constructs a manager backed by the given KVStore.
// storeKey is typically the privacy module store (shared read/write access).
func NewAdaptiveTimeoutManager(storeKey storetypes.StoreKey, logger log.Logger) *AdaptiveTimeoutManager {
	return &AdaptiveTimeoutManager{
		storeKey: storeKey,
		logger:   logger,
	}
}

// AddTx atomically increments the within-block transaction counter.
// Called by the PQC ante decorator for every accepted transaction.
func (m *AdaptiveTimeoutManager) AddTx() {
	atomic.AddInt64(&m.txTotal, 1)
}

// RecordBlockLoad snapshots and resets the atomic tx counter, updates the
// rolling window, and persists the recommended timeout to the KVStore.
//
// Must be called at the beginning of EndBlocker (before mm.EndBlock) so that
// the KVStore write is included in the block's AppHash via the next Commit.
//
// Returns the current tx count that was flushed.
func (m *AdaptiveTimeoutManager) RecordBlockLoad(ctx sdk.Context, txCount int) {
	m.txCounts[m.head] = int64(txCount)
	m.head = (m.head + 1) % rollingWindow
	if m.head == 0 {
		m.filled = true
	}

	suggested := m.SuggestedCommitTimeout()

	// Persist the suggested timeout (nanoseconds, big-endian) for off-chain
	// monitoring without requiring a separate RPC.
	store := ctx.KVStore(m.storeKey)
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], uint64(suggested))
	store.Set([]byte(adaptiveTimeoutKey), buf[:])

	m.logger.Info("greenbft adaptive timeout",
		"block_height", ctx.BlockHeight(),
		"tx_count", txCount,
		"rolling_avg", m.rollingAvg(),
		"suggested_timeout_ms", suggested.Milliseconds(),
	)
}

// DrainTxCount atomically reads and resets the within-block transaction counter.
// Call this at the top of EndBlocker to get an accurate count for the block
// that just finished without racing with concurrent ante checks.
func (m *AdaptiveTimeoutManager) DrainTxCount() int {
	return int(atomic.SwapInt64(&m.txTotal, 0))
}

// SuggestedCommitTimeout returns the recommended timeout_commit duration based
// on recent block load. When the rolling average is below LowTxThreshold the
// chain is considered idle and the extended idle timeout is returned.
func (m *AdaptiveTimeoutManager) SuggestedCommitTimeout() time.Duration {
	avg := m.rollingAvg()
	if avg < LowTxThreshold {
		return IdleCommitTimeout
	}
	return DefaultCommitTimeout
}

// GetRecommendedTimeout is the public entry point for fetching the current
// recommended timeout — identical to SuggestedCommitTimeout but follows the
// naming convention expected by callers in app.go.
func (m *AdaptiveTimeoutManager) GetRecommendedTimeout() time.Duration {
	return m.SuggestedCommitTimeout()
}

// GetPersistedTimeout reads the last persisted suggested timeout from the store.
// Returns DefaultCommitTimeout if no value has been stored yet.
func (m *AdaptiveTimeoutManager) GetPersistedTimeout(ctx sdk.Context) time.Duration {
	store := ctx.KVStore(m.storeKey)
	bz := store.Get([]byte(adaptiveTimeoutKey))
	if len(bz) < 8 {
		return DefaultCommitTimeout
	}
	ns := binary.BigEndian.Uint64(bz)
	return time.Duration(ns)
}

// rollingAvg returns the average tx count over the filled portion of the buffer.
func (m *AdaptiveTimeoutManager) rollingAvg() float64 {
	size := rollingWindow
	if !m.filled {
		size = m.head
	}
	if size == 0 {
		return 0
	}
	var sum int64
	for i := 0; i < size; i++ {
		sum += m.txCounts[i]
	}
	return float64(sum) / float64(size)
}
