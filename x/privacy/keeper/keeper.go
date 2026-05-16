package keeper

import (
	"fmt"

	"github.com/cometbft/cometbft/libs/log"
	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	"zytherion/x/privacy/types"
	"zytherion/x/privacy/zk"
)

type (
	Keeper struct {
		cdc        codec.BinaryCodec
		storeKey   storetypes.StoreKey
		memKey     storetypes.StoreKey
		paramstore paramtypes.Subspace
		bankKeeper types.BankKeeper
		// zkVK holds the serialized Groth16 verifying key loaded at startup.
		// It is reused for every VerifyTransferProof call — no re-parsing per tx.
		zkVK []byte
	}
)

func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey,
	memKey storetypes.StoreKey,
	ps paramtypes.Subspace,
	bankKeeper types.BankKeeper,
	zkVK []byte,
) *Keeper {
	if !ps.HasKeyTable() {
		ps = ps.WithKeyTable(types.ParamKeyTable())
	}
	return &Keeper{
		cdc:        cdc,
		storeKey:   storeKey,
		memKey:     memKey,
		paramstore: ps,
		bankKeeper: bankKeeper,
		zkVK:       zkVK,
	}
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

func (k Keeper) StoreKey() storetypes.StoreKey {
	return k.storeKey
}

// ── ZK Verifying Key ──────────────────────────────────────────────────────────

// ZKVerifyingKey returns the raw verifying key bytes.
func (k Keeper) ZKVerifyingKey() []byte {
	return k.zkVK
}

// VerifyTransferProof verifies a Groth16 proof against the chain's verifying key.
// This is the ONLY on-chain cryptographic call — fully deterministic.
func (k Keeper) VerifyTransferProof(proofBytes, publicInputs []byte) error {
	if len(k.zkVK) == 0 {
		return fmt.Errorf("ZK verifying key not initialised in keeper")
	}
	return zk.VerifyTransferProof(proofBytes, publicInputs, k.zkVK)
}

// ── Commitment store ──────────────────────────────────────────────────────────

// SetCommitment stores a 32-byte ZK commitment for an account.
// Overwrites any existing commitment (used when updating after a transfer).
func (k Keeper) SetCommitment(ctx sdk.Context, addr sdk.AccAddress, commitment []byte) error {
	if err := zk.ValidateCommitmentBytes(commitment); err != nil {
		return fmt.Errorf("SetCommitment: %w", err)
	}
	store := ctx.KVStore(k.storeKey)
	store.Set(types.CommitmentKey(addr), commitment)
	k.Logger(ctx).Debug("commitment updated",
		"address", addr.String(),
		"commitment_prefix", fmt.Sprintf("%x", commitment[:4]),
	)
	return nil
}

// GetCommitment retrieves the 32-byte commitment for an account.
// Returns (nil, false) if no commitment is registered.
func (k Keeper) GetCommitment(ctx sdk.Context, addr sdk.AccAddress) ([]byte, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.CommitmentKey(addr))
	if bz == nil {
		return nil, false
	}
	return bz, true
}

// HasCommitment reports whether an account has a registered commitment.
func (k Keeper) HasCommitment(ctx sdk.Context, addr sdk.AccAddress) bool {
	return ctx.KVStore(k.storeKey).Has(types.CommitmentKey(addr))
}

// ── Nullifier store ───────────────────────────────────────────────────────────

// SetNullifier marks a nullifier as spent on-chain.
// Used to prevent double-spending of the same commitment.
func (k Keeper) SetNullifier(ctx sdk.Context, nullifier []byte) {
	store := ctx.KVStore(k.storeKey)
	store.Set(types.NullifierKey(nullifier), []byte{1})
}

// HasNullifier returns true if the nullifier has already been spent.
func (k Keeper) HasNullifier(ctx sdk.Context, nullifier []byte) bool {
	return ctx.KVStore(k.storeKey).Has(types.NullifierKey(nullifier))
}