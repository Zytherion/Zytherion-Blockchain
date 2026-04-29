//go:build !notfhe
// +build !notfhe

package keeper_test

import (
	"testing"

	dbm "github.com/cometbft/cometbft-db"
	"github.com/cometbft/cometbft/libs/log"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/store"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/stretchr/testify/require"

	"zytherion/x/privacy/fhe"
	keeperpkg "zytherion/x/privacy/keeper"
	"zytherion/x/privacy/types"
)

func newTestKeeper(t *testing.T) (keeperpkg.Keeper, sdk.Context) {
	t.Helper()

	storeKey := sdk.NewKVStoreKey(types.StoreKey)
	memKey := storetypes.NewMemoryStoreKey(types.MemStoreKey)
	paramsKey := sdk.NewKVStoreKey(paramstypes.StoreKey)
	paramsTKey := sdk.NewTransientStoreKey(paramstypes.TStoreKey)

	db := dbm.NewMemDB()
	ms := store.NewCommitMultiStore(db)
	ms.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	ms.MountStoreWithDB(memKey, storetypes.StoreTypeMemory, nil)
	ms.MountStoreWithDB(paramsKey, storetypes.StoreTypeIAVL, db)
	ms.MountStoreWithDB(paramsTKey, storetypes.StoreTypeTransient, db)
	require.NoError(t, ms.LoadLatestVersion())

	ctx := sdk.NewContext(ms, tmproto.Header{}, false, log.NewNopLogger())

	ir := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(ir)
	amino := codec.NewLegacyAmino()

	pk := paramskeeper.NewKeeper(cdc, amino, paramsKey, paramsTKey)
	subspace := pk.Subspace(types.ModuleName)

	fheCtx, err := fhe.NewContext()
	require.NoError(t, err, "fhe.NewContext must not fail in test setup")

	k := *keeperpkg.NewKeeper(cdc, storeKey, memKey, subspace, nil, fheCtx)
	return k, ctx
}

// TestSetGetEncryptedBalance verifies that a ciphertext stored via
// SetEncryptedBalance can be retrieved byte-for-byte via GetEncryptedBalance.
func TestSetGetEncryptedBalance(t *testing.T) {
	k, ctx := newTestKeeper(t)
	addr := sdk.AccAddress([]byte("recipient_addr_001__"))

	fheCtx := k.FHEContext()

	// Encrypt returns compressed bytes directly.
	ctBytes, err := fheCtx.Encrypt(500_000)
	require.NoError(t, err)

	require.False(t, k.HasEncryptedBalance(ctx, addr))
	k.SetEncryptedBalance(ctx, addr, ctBytes)
	require.True(t, k.HasEncryptedBalance(ctx, addr))

	got, found := k.GetEncryptedBalance(ctx, addr)
	require.True(t, found)
	require.Equal(t, ctBytes, got, "retrieved bytes must match stored bytes exactly")
}

// TestHomomorphicBalanceUpdate simulates two encrypted deposits and verifies
// HomomorphicAdd correctly accumulates: Enc(300) + Enc(700) â†’ 1000.
func TestHomomorphicBalanceUpdate(t *testing.T) {
	k, ctx := newTestKeeper(t)

	fheCtx := k.FHEContext()

	recipientAddr := sdk.AccAddress([]byte("recipient_addr_002__"))

	// First deposit: Enc(300)
	ct300, err := fheCtx.Encrypt(300)
	require.NoError(t, err)
	k.SetEncryptedBalance(ctx, recipientAddr, ct300)

	// Second deposit via HomomorphicAdd: adds Enc(700)
	ct700, err := fheCtx.Encrypt(700)
	require.NoError(t, err)
	require.NoError(t, k.HomomorphicAdd(ctx, recipientAddr, ct700))

	// Verify by decrypting (off-chain only)
	result, err := k.DecryptBalance(ctx, recipientAddr)
	require.NoError(t, err)
	require.EqualValues(t, uint64(1000), result, "Enc(300) + Enc(700) must decrypt to 1000")
}

// TestGetEncryptedBalance_NotFound verifies that an unknown address returns
// (nil, false) without panicking.
func TestGetEncryptedBalance_NotFound(t *testing.T) {
	k, ctx := newTestKeeper(t)
	bz, found := k.GetEncryptedBalance(ctx, sdk.AccAddress([]byte("unknown____________")))
	require.False(t, found)
	require.Nil(t, bz)
}

// TestEncryptedBalanceKeyIsolation confirms that setting balance for addr A
// does not affect addr B.
func TestEncryptedBalanceKeyIsolation(t *testing.T) {
	k, ctx := newTestKeeper(t)

	fheCtx := k.FHEContext()

	addrA := sdk.AccAddress([]byte("isolation_addr_A____"))
	addrB := sdk.AccAddress([]byte("isolation_addr_B____"))

	ctA, err := fheCtx.Encrypt(100)
	require.NoError(t, err)
	k.SetEncryptedBalance(ctx, addrA, ctA)

	require.False(t, k.HasEncryptedBalance(ctx, addrB))
	bz, found := k.GetEncryptedBalance(ctx, addrB)
	require.False(t, found)
	require.Nil(t, bz)
}