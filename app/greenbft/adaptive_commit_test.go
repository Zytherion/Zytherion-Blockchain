package greenbft_test

import (
	"testing"

	dbm "github.com/cometbft/cometbft-db"
	"github.com/cometbft/cometbft/libs/log"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/store"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"zytherion/app/greenbft"
)

// newTestContext creates a minimal in-memory context for greenbft tests.
func newTestContext(t *testing.T) (sdk.Context, storetypes.StoreKey) {
	t.Helper()
	storeKey := sdk.NewKVStoreKey("test_greenbft")
	db := dbm.NewMemDB()
	ms := store.NewCommitMultiStore(db)
	ms.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, ms.LoadLatestVersion())
	ctx := sdk.NewContext(ms, tmproto.Header{Height: 100}, false, log.NewNopLogger())
	return ctx, storeKey
}

// TestAdaptiveTimeout_Idle verifies that when recent blocks have few txs, the
// manager recommends IdleCommitTimeout to save validator CPU.
func TestAdaptiveTimeout_Idle(t *testing.T) {
	ctx, key := newTestContext(t)
	mgr := greenbft.NewAdaptiveTimeoutManager(key, log.NewNopLogger())

	// Feed 10 blocks with 0 transactions each.
	for i := 0; i < 10; i++ {
		mgr.RecordBlockLoad(ctx, 0)
	}
	require.Equal(t, greenbft.IdleCommitTimeout, mgr.SuggestedCommitTimeout(),
		"empty mempool should suggest IdleCommitTimeout")
}

// TestAdaptiveTimeout_Busy verifies that when blocks carry many txs, the
// manager recommends DefaultCommitTimeout for normal operation.
func TestAdaptiveTimeout_Busy(t *testing.T) {
	ctx, key := newTestContext(t)
	mgr := greenbft.NewAdaptiveTimeoutManager(key, log.NewNopLogger())

	// Feed 10 blocks with 100 transactions each (well above threshold).
	for i := 0; i < 10; i++ {
		mgr.RecordBlockLoad(ctx, 100)
	}
	require.Equal(t, greenbft.DefaultCommitTimeout, mgr.SuggestedCommitTimeout(),
		"busy chain should suggest DefaultCommitTimeout")
}

// TestAdaptiveTimeout_Transition verifies that a switch from busy → idle
// is correctly reflected after the rolling window fills with idle data.
func TestAdaptiveTimeout_Transition(t *testing.T) {
	ctx, key := newTestContext(t)
	mgr := greenbft.NewAdaptiveTimeoutManager(key, log.NewNopLogger())

	// 10 busy blocks.
	for i := 0; i < 10; i++ {
		mgr.RecordBlockLoad(ctx, 50)
	}
	require.Equal(t, greenbft.DefaultCommitTimeout, mgr.SuggestedCommitTimeout())

	// Now go idle for 10 more blocks (window fully replaced).
	for i := 0; i < 10; i++ {
		mgr.RecordBlockLoad(ctx, 0)
	}
	require.Equal(t, greenbft.IdleCommitTimeout, mgr.SuggestedCommitTimeout(),
		"after full idle window, should suggest IdleCommitTimeout")
}

// TestAdaptiveTimeout_Persistance verifies that RecordBlockLoad persists the
// suggested timeout to the KVStore and GetPersistedTimeout reads it back.
func TestAdaptiveTimeout_Persistance(t *testing.T) {
	ctx, key := newTestContext(t)
	mgr := greenbft.NewAdaptiveTimeoutManager(key, log.NewNopLogger())

	mgr.RecordBlockLoad(ctx, 0)
	persisted := mgr.GetPersistedTimeout(ctx)
	// With a single idle block the rolling avg is 0 → idleTimeout expected.
	require.Equal(t, greenbft.IdleCommitTimeout, persisted)
}

// TestAddTxAndDrainCount verifies that AddTx increments the atomic counter
// and DrainTxCount returns the snapshot then resets to zero.
func TestAddTxAndDrainCount(t *testing.T) {
	_, key := newTestContext(t)
	mgr := greenbft.NewAdaptiveTimeoutManager(key, log.NewNopLogger())

	require.Equal(t, 0, mgr.DrainTxCount(), "initial count must be 0")

	mgr.AddTx()
	mgr.AddTx()
	mgr.AddTx()

	count := mgr.DrainTxCount()
	require.Equal(t, 3, count, "expected 3 after 3 AddTx calls")
	require.Equal(t, 0, mgr.DrainTxCount(), "counter must reset to 0 after drain")
}

// TestGetRecommendedTimeout verifies it matches SuggestedCommitTimeout.
func TestGetRecommendedTimeout(t *testing.T) {
	ctx, key := newTestContext(t)
	mgr := greenbft.NewAdaptiveTimeoutManager(key, log.NewNopLogger())

	for i := 0; i < 10; i++ {
		mgr.RecordBlockLoad(ctx, 0)
	}
	require.Equal(t, mgr.SuggestedCommitTimeout(), mgr.GetRecommendedTimeout(),
		"GetRecommendedTimeout must equal SuggestedCommitTimeout")
}

