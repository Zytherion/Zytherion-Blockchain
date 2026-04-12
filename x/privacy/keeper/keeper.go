package keeper

import (
	"fmt"

	"github.com/cometbft/cometbft/libs/log"
	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	"zytherion/x/privacy/types"
)

type (
	Keeper struct {
		cdc        codec.BinaryCodec
		storeKey   storetypes.StoreKey
		memKey     storetypes.StoreKey
		paramstore paramtypes.Subspace

		bankKeeper types.BankKeeper
	}
)

func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey,
	memKey storetypes.StoreKey,
	ps paramtypes.Subspace,
	bankKeeper types.BankKeeper,
) *Keeper {
	// set KeyTable if it has not already been set
	if !ps.HasKeyTable() {
		ps = ps.WithKeyTable(types.ParamKeyTable())
	}

	return &Keeper{
		cdc:        cdc,
		storeKey:   storeKey,
		memKey:     memKey,
		paramstore: ps,
		bankKeeper: bankKeeper,
	}
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// StoreKey returns the primary KVStore key for the privacy module.
// External callers (e.g. the app-level PQC hash integration) use this to
// obtain direct store access for reading and writing PQC block hash entries.
func (k Keeper) StoreKey() storetypes.StoreKey {
	return k.storeKey
}

// ── Encrypted Balance Storage ────────────────────────────────────────────────
//
// Encrypted balances are stored verbatim as BFV ciphertext bytes under the key
//   types.EncryptedBalanceKey(addr)
//
// The keeper never decrypts these bytes. The only operations performed on the
// ciphertext are:
//   (a) Store (SetEncryptedBalance)
//   (b) Retrieve (GetEncryptedBalance)
//   (c) Homomorphic addition in the msg server (via fhe.AddCiphertexts)
//
// This preserves the privacy invariant: on-chain state contains only
// ciphertexts; validators and chain observers cannot learn the plaintext amount.

// SetEncryptedBalance stores the binary-encoded BFV ciphertext for addr's
// encrypted balance.  Any previous value is overwritten.
// ciphertextBytes must be the output of rlwe.Ciphertext.MarshalBinary.
func (k Keeper) SetEncryptedBalance(ctx sdk.Context, addr sdk.AccAddress, ciphertextBytes []byte) {
	store := ctx.KVStore(k.storeKey)
	key := types.EncryptedBalanceKey(addr)
	store.Set(key, ciphertextBytes)

	k.Logger(ctx).Debug("encrypted balance updated",
		"address", addr.String(),
		"ciphertext_len", len(ciphertextBytes),
	)
}

// GetEncryptedBalance retrieves the binary-encoded BFV ciphertext for addr's
// encrypted balance.  Returns (nil, false) if no balance has been set yet
// (the zero state — a ciphertext encoding 0 should be set on first deposit).
func (k Keeper) GetEncryptedBalance(ctx sdk.Context, addr sdk.AccAddress) ([]byte, bool) {
	store := ctx.KVStore(k.storeKey)
	key := types.EncryptedBalanceKey(addr)
	bz := store.Get(key)
	if bz == nil {
		return nil, false
	}
	return bz, true
}

// HasEncryptedBalance reports whether addr has an encrypted balance stored.
func (k Keeper) HasEncryptedBalance(ctx sdk.Context, addr sdk.AccAddress) bool {
	store := ctx.KVStore(k.storeKey)
	return store.Has(types.EncryptedBalanceKey(addr))
}
