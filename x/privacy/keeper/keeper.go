package keeper

import (
	"fmt"

	"github.com/cometbft/cometbft/libs/log"
	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	"zytherion/x/privacy/fhe"
	"zytherion/x/privacy/types"
)

type (
	Keeper struct {
		cdc        codec.BinaryCodec
		storeKey   storetypes.StoreKey
		memKey     storetypes.StoreKey
		paramstore paramtypes.Subspace
		bankKeeper types.BankKeeper
		fheCtx     *fhe.Context
	}
)

func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey,
	memKey storetypes.StoreKey,
	ps paramtypes.Subspace,
	bankKeeper types.BankKeeper,
	fheCtx *fhe.Context,
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
		fheCtx:     fheCtx,
	}
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

func (k Keeper) StoreKey() storetypes.StoreKey {
	return k.storeKey
}

func (k Keeper) FHEContext() *fhe.Context {
	return k.fheCtx
}

// EncryptAmount encrypts a plaintext uint64 using the TFHE context.
// Returns TFHE-rs compressed ciphertext bytes (NOT yet ZSTD compressed â€”
// the caller decides when to store; only SetEncryptedBalance applies ZSTD).
func (k Keeper) EncryptAmount(amount uint64) ([]byte, error) {
	if k.fheCtx == nil {
		return nil, fmt.Errorf("fhe context not initialised in keeper")
	}
	return k.fheCtx.Encrypt(amount)
}

// HomomorphicAdd adds ctB to addr's stored balance homomorphically.
// Auto-initialises addr to Enc(0) if no balance exists yet.
// Internally: load+ZSTD-decompress â†’ AddCiphertexts â†’ ZSTD-compress+store.
func (k Keeper) HomomorphicAdd(ctx sdk.Context, addr sdk.AccAddress, ctB []byte) error {
	if k.fheCtx == nil {
		return fmt.Errorf("fhe context not initialised in keeper")
	}
	ctA, found := k.GetEncryptedBalance(ctx, addr)
	if !found {
		zero, err := k.EncryptAmount(0)
		if err != nil {
			return fmt.Errorf("HomomorphicAdd: init zero balance: %w", err)
		}
		ctA = zero
	}
	result, err := k.fheCtx.AddCiphertexts(ctA, ctB)
	if err != nil {
		return fmt.Errorf("HomomorphicAdd: %w", err)
	}
	k.SetEncryptedBalance(ctx, addr, result)
	return nil
}

// HomomorphicSub subtracts ctB from addr's stored balance homomorphically.
// Returns error if addr has no balance.
// Internally: load+ZSTD-decompress â†’ SubCiphertexts â†’ ZSTD-compress+store.
func (k Keeper) HomomorphicSub(ctx sdk.Context, addr sdk.AccAddress, ctB []byte) error {
	if k.fheCtx == nil {
		return fmt.Errorf("fhe context not initialised in keeper")
	}
	ctA, found := k.GetEncryptedBalance(ctx, addr)
	if !found {
		return fmt.Errorf("HomomorphicSub: no balance for %s", addr.String())
	}
	result, err := k.fheCtx.SubCiphertexts(ctA, ctB)
	if err != nil {
		return fmt.Errorf("HomomorphicSub: %w", err)
	}
	k.SetEncryptedBalance(ctx, addr, result)
	return nil
}

// DecryptBalance decrypts the stored balance for addr.
// Internally: load+ZSTD-decompress â†’ Decrypt.
func (k Keeper) DecryptBalance(ctx sdk.Context, addr sdk.AccAddress) (uint64, error) {
	if k.fheCtx == nil {
		return 0, fmt.Errorf("fhe context not initialised in keeper")
	}
	bz, found := k.GetEncryptedBalance(ctx, addr)
	if !found {
		return 0, fmt.Errorf("DecryptBalance: no balance for %s", addr.String())
	}
	return k.fheCtx.Decrypt(bz)
}

// SetEncryptedBalance compresses ciphertextBytes with ZSTD then stores it.
//
// Storage format: ZSTD( TFHE-rs-compressed-ciphertext )
// This double-compression reduces on-chain size from ~21KB to ~5KB,
// cutting gas costs for WritePerByte by ~75%.
func (k Keeper) SetEncryptedBalance(ctx sdk.Context, addr sdk.AccAddress, ciphertextBytes []byte) {
	zstdBytes, err := zstdCompressForStore(ciphertextBytes)
	if err != nil {
		// Non-fatal: fall back to storing uncompressed with a warning.
		// This should never happen in practice.
		k.Logger(ctx).Error("zstd compress failed â€” storing uncompressed",
			"address", addr.String(),
			"error", err,
		)
		zstdBytes = ciphertextBytes
	}

	store := ctx.KVStore(k.storeKey)
	store.Set(types.EncryptedBalanceKey(addr), zstdBytes)

	k.Logger(ctx).Debug("encrypted balance updated",
		"address", addr.String(),
		"tfhe_size", len(ciphertextBytes),
		"stored_size", len(zstdBytes),
		"compression_ratio", fmt.Sprintf("%.1f%%", 100.0*float64(len(zstdBytes))/float64(len(ciphertextBytes))),
	)
}

// GetEncryptedBalance retrieves and ZSTD-decompresses the stored ciphertext.
// Returns TFHE-rs compressed bytes ready for homomorphic operations.
//
// All callers (HomomorphicAdd, HomomorphicSub, DecryptBalance) receive
// TFHE-rs bytes and are unaware of the ZSTD storage layer.
func (k Keeper) GetEncryptedBalance(ctx sdk.Context, addr sdk.AccAddress) ([]byte, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.EncryptedBalanceKey(addr))
	if bz == nil {
		return nil, false
	}
	// Attempt ZSTD decompression. If it fails (e.g. legacy uncompressed data),
	// return the raw bytes so old balances aren't bricked.
	decompressed, err := zstdDecompressFromStore(bz)
	if err != nil {
		k.Logger(ctx).Error("zstd decompress failed â€” returning raw bytes (legacy?)",
			"address", addr.String(),
			"error", err,
		)
		return bz, true
	}
	return decompressed, true
}

// HasEncryptedBalance reports whether addr has an encrypted balance stored.
func (k Keeper) HasEncryptedBalance(ctx sdk.Context, addr sdk.AccAddress) bool {
	return ctx.KVStore(k.storeKey).Has(types.EncryptedBalanceKey(addr))
}