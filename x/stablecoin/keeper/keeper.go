package keeper

import (
	"encoding/json"
	"fmt"

	"github.com/cometbft/cometbft/libs/log"
	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"zytherion/x/stablecoin/types"
)

// Keeper is the stablecoin module keeper.
type Keeper struct {
	cdc                 codec.BinaryCodec
	storeKey            storetypes.StoreKey
	bankKeeper          types.BankKeeper
	oracleKeeper        types.OracleKeeper
	ibcCollateralKeeper types.IBCCollateralKeeper
}

// NewKeeper creates a new stablecoin Keeper.
func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,
	bankKeeper types.BankKeeper,
	oracleKeeper types.OracleKeeper,
	ibcCollateralKeeper types.IBCCollateralKeeper,
) Keeper {
	return Keeper{
		cdc:                 cdc,
		storeKey:            storeKey,
		bankKeeper:          bankKeeper,
		oracleKeeper:        oracleKeeper,
		ibcCollateralKeeper: ibcCollateralKeeper,
	}
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// GetStore returns the module's KV store.
func (k Keeper) GetStore(ctx sdk.Context) sdk.KVStore {
	return ctx.KVStore(k.storeKey)
}

// ─── Params ──────────────────────────────────────────────────────────────────

// GetParams returns the stablecoin module params from the KV store.
func (k Keeper) GetParams(ctx sdk.Context) types.StablecoinParams {
	store := k.GetStore(ctx)
	bz := store.Get([]byte(types.ParamsKey))
	if bz == nil {
		return types.DefaultStablecoinParams()
	}
	var params types.StablecoinParams
	if err := json.Unmarshal(bz, &params); err != nil {
		return types.DefaultStablecoinParams()
	}
	return params
}

// SetParams stores the stablecoin module params.
func (k Keeper) SetParams(ctx sdk.Context, params types.StablecoinParams) {
	bz, err := json.Marshal(params)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal stablecoin params: %v", err))
	}
	k.GetStore(ctx).Set([]byte(types.ParamsKey), bz)
}

// ─── MintRecord ──────────────────────────────────────────────────────────────

// mintRecordKey returns the KV key for a MintRecord.
func mintRecordKey(owner sdk.AccAddress, ibcDenom string) []byte {
	return []byte(fmt.Sprintf("%s%s/%s", types.MintRecordPrefix, owner.String(), ibcDenom))
}

// GetMintRecord retrieves the MintRecord for owner+ibcDenom.
func (k Keeper) GetMintRecord(ctx sdk.Context, owner sdk.AccAddress, ibcDenom string) (types.MintRecord, error) {
	store := k.GetStore(ctx)
	bz := store.Get(mintRecordKey(owner, ibcDenom))
	if bz == nil {
		return types.MintRecord{}, types.ErrMintRecordNotFound
	}
	var record types.MintRecord
	if err := json.Unmarshal(bz, &record); err != nil {
		return types.MintRecord{}, fmt.Errorf("unmarshal mint record: %w", err)
	}
	return record, nil
}

// SetMintRecord stores a MintRecord.
func (k Keeper) SetMintRecord(ctx sdk.Context, record types.MintRecord) {
	owner, _ := sdk.AccAddressFromBech32(record.Owner)
	bz, err := json.Marshal(record)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal mint record: %v", err))
	}
	k.GetStore(ctx).Set(mintRecordKey(owner, record.IBCDenom), bz)
}

// DeleteMintRecord removes a MintRecord.
func (k Keeper) DeleteMintRecord(ctx sdk.Context, owner sdk.AccAddress, ibcDenom string) {
	k.GetStore(ctx).Delete(mintRecordKey(owner, ibcDenom))
}

// ─── TotalSupply ─────────────────────────────────────────────────────────────

// GetTotalSupply returns the total ZYTD minted.
func (k Keeper) GetTotalSupply(ctx sdk.Context) sdk.Int {
	store := k.GetStore(ctx)
	bz := store.Get([]byte(types.TotalSupplyKey))
	if bz == nil {
		return sdk.ZeroInt()
	}
	var supply sdk.Int
	if err := supply.Unmarshal(bz); err != nil {
		return sdk.ZeroInt()
	}
	return supply
}

// SetTotalSupply stores the total ZYTD supply.
func (k Keeper) SetTotalSupply(ctx sdk.Context, supply sdk.Int) {
	bz, err := supply.Marshal()
	if err != nil {
		panic(fmt.Sprintf("failed to marshal total supply: %v", err))
	}
	k.GetStore(ctx).Set([]byte(types.TotalSupplyKey), bz)
}

// ─── Collateral Ratio ────────────────────────────────────────────────────────

// GetCollateralRatio returns the current collateral ratio for an owner+ibcDenom position.
func (k Keeper) GetCollateralRatio(ctx sdk.Context, owner sdk.AccAddress, ibcDenom string) (sdk.Dec, error) {
	record, err := k.GetMintRecord(ctx, owner, ibcDenom)
	if err != nil {
		return sdk.ZeroDec(), err
	}
	if record.Minted.IsZero() {
		return sdk.NewDec(999), nil // effectively infinite ratio
	}

	position, err := k.ibcCollateralKeeper.GetPosition(ctx, owner, ibcDenom)
	if err != nil {
		return sdk.ZeroDec(), fmt.Errorf("get position: %w", err)
	}

	twap, err := k.oracleKeeper.GetTWAP(ctx, ibcDenom)
	if err != nil {
		return sdk.ZeroDec(), types.ErrOraclePriceUnavailable
	}

	collateralUSD := sdk.NewDecFromInt(position.Amount).Mul(twap.TWAP)
	mintedDec := sdk.NewDecFromInt(record.Minted)
	return collateralUSD.Quo(mintedDec), nil
}

// GetMaxMintable returns the maximum ZYTD mintable given collateral amount and asset.
func (k Keeper) GetMaxMintable(ctx sdk.Context, ibcDenom string, collateralAmount sdk.Int) (sdk.Int, error) {
	asset, err := k.ibcCollateralKeeper.GetCollateralAsset(ctx, ibcDenom)
	if err != nil {
		return sdk.ZeroInt(), fmt.Errorf("get collateral asset: %w", err)
	}

	twap, err := k.oracleKeeper.GetTWAP(ctx, ibcDenom)
	if err != nil {
		return sdk.ZeroInt(), types.ErrOraclePriceUnavailable
	}

	collateralUSD := sdk.NewDecFromInt(collateralAmount).Mul(twap.TWAP)
	maxMintable := collateralUSD.Quo(asset.MinRatio)
	return maxMintable.TruncateInt(), nil
}

// ─── Bank helpers ────────────────────────────────────────────────────────────

// BankKeeper exposes the bank keeper to message servers.
func (k Keeper) BankKeeper() types.BankKeeper {
	return k.bankKeeper
}

// OracleKeeper exposes the oracle keeper to message servers.
func (k Keeper) OracleKeeper() types.OracleKeeper {
	return k.oracleKeeper
}

// IBCCollateralKeeper exposes the ibc-collateral keeper to message servers.
func (k Keeper) IBCCollateralKeeper() types.IBCCollateralKeeper {
	return k.ibcCollateralKeeper
}

// ─── Dilithium5 PQC Key Registry ─────────────────────────────────────────────

// pqcKeyRegistryKey returns the KV store key for a user's registered Dilithium5 public key.
func pqcKeyRegistryKey(owner sdk.AccAddress) []byte {
	return []byte(fmt.Sprintf("%s%s", types.ZYTDKeyRegistryPrefix, owner.String()))
}

// SetZYTDPQCKey stores a user's Dilithium5 public key in the registry.
// The pubkey must be exactly 2592 bytes (Dilithium5 / ML-DSA-87 public key size).
// Calling this again overwrites the existing key for the same address.
func (k Keeper) SetZYTDPQCKey(ctx sdk.Context, owner sdk.AccAddress, pubkey []byte) {
	k.GetStore(ctx).Set(pqcKeyRegistryKey(owner), pubkey)
}

// GetZYTDPQCKey retrieves the registered Dilithium5 public key for an address.
// Returns ErrPQCKeyNotRegistered if no key has been registered.
func (k Keeper) GetZYTDPQCKey(ctx sdk.Context, owner sdk.AccAddress) ([]byte, error) {
	bz := k.GetStore(ctx).Get(pqcKeyRegistryKey(owner))
	if bz == nil {
		return nil, types.ErrPQCKeyNotRegistered
	}
	return bz, nil
}

// HasZYTDPQCKey returns true if the address has a registered Dilithium5 key.
func (k Keeper) HasZYTDPQCKey(ctx sdk.Context, owner sdk.AccAddress) bool {
	return k.GetStore(ctx).Has(pqcKeyRegistryKey(owner))
}

