package keeper_test

import (
	"testing"
	"time"

	tmdb "github.com/cometbft/cometbft-db"
	"github.com/cometbft/cometbft/libs/log"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/store"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"zytherion/x/oracle/keeper"
	"zytherion/x/oracle/types"
)

func TestOracleKeeperPriceSubmission(t *testing.T) {
	storeKey := sdk.NewKVStoreKey(types.StoreKey)
	db := tmdb.NewMemDB()
	stateStore := store.NewCommitMultiStore(db)
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	k := keeper.NewKeeper(cdc, storeKey, nil)
	ctx := sdk.NewContext(stateStore, tmproto.Header{
		Height: 10,
		Time:   time.Now(),
	}, false, log.NewNopLogger())

	// Test SetParams / GetParams
	params := types.DefaultOracleParams()
	k.SetParams(ctx, params)
	require.Equal(t, params, k.GetParams(ctx))

	// Test Price Submission
	entry := types.PriceEntry{
		Denom:     "ZYTC",
		PriceUSD:  sdk.MustNewDecFromStr("0.85"),
		Submitter: "zyth1abc",
		Height:    10,
		Timestamp: ctx.BlockTime(),
	}

	err := k.SetPrice(ctx, entry)
	require.NoError(t, err)

	// Get latest price
	latest, err := k.GetLatestPrice(ctx, "ZYTC")
	require.NoError(t, err)
	require.Equal(t, entry.Denom, latest.Denom)
	require.Equal(t, entry.PriceUSD, latest.PriceUSD)

	// Get price history
	history := k.GetPriceHistory(ctx, "ZYTC", 5)
	require.Len(t, history, 1)
	require.Equal(t, entry.PriceUSD, history[0].PriceUSD)
}

func TestOracleKeeperTWAP(t *testing.T) {
	storeKey := sdk.NewKVStoreKey(types.StoreKey)
	db := tmdb.NewMemDB()
	stateStore := store.NewCommitMultiStore(db)
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	k := keeper.NewKeeper(cdc, storeKey, nil)
	ctx := sdk.NewContext(stateStore, tmproto.Header{
		Height: 10,
		Time:   time.Now(),
	}, false, log.NewNopLogger())

	params := types.DefaultOracleParams()
	params.MinSubmissions = 2
	k.SetParams(ctx, params)

	// Submit prices for denom ZYTC from multiple submitters at different heights (since priceKey is by denom/height)
	err := k.SetPrice(ctx, types.PriceEntry{
		Denom:     "ZYTC",
		PriceUSD:  sdk.MustNewDecFromStr("1.00"),
		Submitter: "zyth1abc",
		Height:    9,
		Timestamp: ctx.BlockTime(),
	})
	require.NoError(t, err)

	err = k.SetPrice(ctx, types.PriceEntry{
		Denom:     "ZYTC",
		PriceUSD:  sdk.MustNewDecFromStr("2.00"),
		Submitter: "zyth1def",
		Height:    10,
		Timestamp: ctx.BlockTime(),
	})
	require.NoError(t, err)

	// Median should be computed. Since we have 1.00 and 2.00, median is average: 1.50
	twap, err := k.ComputeTWAP(ctx, "ZYTC")
	require.NoError(t, err)
	require.Equal(t, sdk.MustNewDecFromStr("1.50"), twap.TWAP)
}
