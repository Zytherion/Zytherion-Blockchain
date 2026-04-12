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
	"github.com/tuneinsight/lattigo/v4/rlwe"

	"zytherion/x/privacy/fhe"
	keeperpkg "zytherion/x/privacy/keeper"
	"zytherion/x/privacy/types"
)

// newTestKeeper builds a minimal in-memory keeper suitable for unit tests
// without depending on any testutil packages that reference missing modules.
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

	k := *keeperpkg.NewKeeper(cdc, storeKey, memKey, subspace, nil)
	return k, ctx
}

// TestSetGetEncryptedBalance verifies that a ciphertext stored via
// SetEncryptedBalance can be retrieved byte-for-byte via GetEncryptedBalance.
func TestSetGetEncryptedBalance(t *testing.T) {
	k, ctx := newTestKeeper(t)
	addr := sdk.AccAddress([]byte("recipient_addr_001__"))

	fheCtx, err := fhe.NewContext()
	require.NoError(t, err)

	ct, err := fheCtx.Encrypt(500_000)
	require.NoError(t, err)
	ctBytes, err := ct.MarshalBinary()
	require.NoError(t, err)

	// Nothing stored yet.
	require.False(t, k.HasEncryptedBalance(ctx, addr))

	k.SetEncryptedBalance(ctx, addr, ctBytes)
	require.True(t, k.HasEncryptedBalance(ctx, addr))

	got, found := k.GetEncryptedBalance(ctx, addr)
	require.True(t, found)
	require.Equal(t, ctBytes, got, "retrieved bytes must match stored bytes exactly")
}

// TestHomomorphicBalanceUpdate is the golden-path integration test.
// It simulates two encrypted deposits and verifies Enc(300) + Enc(700) → 1000.
func TestHomomorphicBalanceUpdate(t *testing.T) {
	k, ctx := newTestKeeper(t)

	fheCtx, err := fhe.NewContext()
	require.NoError(t, err)

	recipientAddr := sdk.AccAddress([]byte("recipient_addr_002__"))
	bfvParams := fheCtx.Params()

	// ── First deposit: Enc(300) ───────────────────────────────────────────────
	ct300, err := fheCtx.Encrypt(300)
	require.NoError(t, err)
	ct300Bytes, err := ct300.MarshalBinary()
	require.NoError(t, err)
	k.SetEncryptedBalance(ctx, recipientAddr, ct300Bytes)

	// ── Second deposit: Enc(700) — homomorphic add ────────────────────────────
	ct700, err := fheCtx.Encrypt(700)
	require.NoError(t, err)

	existingBytes, found := k.GetEncryptedBalance(ctx, recipientAddr)
	require.True(t, found)

	// Deserialise stored ciphertext.
	currentCt := rlwe.NewCiphertext(bfvParams.Parameters, 1, bfvParams.MaxLevel())
	require.NoError(t, currentCt.UnmarshalBinary(existingBytes))

	// Homomorphic addition — no decryption.
	newBalanceCt, err := fheCtx.AddCiphertexts(currentCt, ct700)
	require.NoError(t, err)

	newBalanceBytes, err := newBalanceCt.MarshalBinary()
	require.NoError(t, err)
	k.SetEncryptedBalance(ctx, recipientAddr, newBalanceBytes)

	// ── Verify by decrypting (off-chain only) ─────────────────────────────────
	finalBytes, found := k.GetEncryptedBalance(ctx, recipientAddr)
	require.True(t, found)

	finalCt := rlwe.NewCiphertext(bfvParams.Parameters, 1, bfvParams.MaxLevel())
	require.NoError(t, finalCt.UnmarshalBinary(finalBytes))

	result, err := fheCtx.Decrypt(finalCt)
	require.NoError(t, err)
	require.EqualValues(t, uint64(1000), result,
		"Enc(300) ⊕ Enc(700) must decrypt to 1000")
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
// does not affect addr B (keys are properly namespaced).
func TestEncryptedBalanceKeyIsolation(t *testing.T) {
	k, ctx := newTestKeeper(t)

	fheCtx, err := fhe.NewContext()
	require.NoError(t, err)

	addrA := sdk.AccAddress([]byte("isolation_addr_A____"))
	addrB := sdk.AccAddress([]byte("isolation_addr_B____"))

	ctA, err := fheCtx.Encrypt(100)
	require.NoError(t, err)
	ctABytes, _ := ctA.MarshalBinary()
	k.SetEncryptedBalance(ctx, addrA, ctABytes)

	// B should still be absent.
	require.False(t, k.HasEncryptedBalance(ctx, addrB))
	bz, found := k.GetEncryptedBalance(ctx, addrB)
	require.False(t, found)
	require.Nil(t, bz)
}
