package keeper

import (
	"encoding/json"
	"fmt"

	"github.com/cometbft/cometbft/libs/log"
	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"zytherion/x/ibc-collateral/types"
)

// Keeper manages all collateral state for the ibccollateral module.
type Keeper struct {
	cdc           codec.BinaryCodec
	storeKey      storetypes.StoreKey
	bankKeeper    types.BankKeeper
	accountKeeper types.AccountKeeper
}

// NewKeeper constructs a new ibccollateral Keeper.
func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,
	bankKeeper types.BankKeeper,
	accountKeeper types.AccountKeeper,
) Keeper {
	return Keeper{
		cdc:           cdc,
		storeKey:      storeKey,
		bankKeeper:    bankKeeper,
		accountKeeper: accountKeeper,
	}
}

// Logger returns a module-tagged logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// StoreKey exposes the store key (used by the AppModule EndBlock / BeginBlock).
func (k Keeper) StoreKey() storetypes.StoreKey {
	return k.storeKey
}

// ─── Collateral Asset management ──────────────────────────────────────────────

// RegisterCollateralAsset persists a CollateralAsset to the KV store.
func (k Keeper) RegisterCollateralAsset(ctx sdk.Context, asset types.CollateralAsset) error {
	if asset.IBCDenom == "" {
		return fmt.Errorf("RegisterCollateralAsset: ibc_denom must not be empty")
	}
	bz, err := json.Marshal(asset)
	if err != nil {
		return fmt.Errorf("RegisterCollateralAsset: marshal error: %w", err)
	}
	store := ctx.KVStore(k.storeKey)
	store.Set([]byte(types.CollateralAssetPrefix+asset.IBCDenom), bz)
	return nil
}

// GetCollateralAsset retrieves a CollateralAsset by its IBC denom.
// Returns ErrDenomNotWhitelisted if no record exists.
func (k Keeper) GetCollateralAsset(ctx sdk.Context, ibcDenom string) (types.CollateralAsset, error) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get([]byte(types.CollateralAssetPrefix + ibcDenom))
	if bz == nil {
		return types.CollateralAsset{}, types.ErrDenomNotWhitelisted.Wrapf("denom: %s", ibcDenom)
	}
	var asset types.CollateralAsset
	if err := json.Unmarshal(bz, &asset); err != nil {
		return types.CollateralAsset{}, fmt.Errorf("GetCollateralAsset: unmarshal error: %w", err)
	}
	return asset, nil
}

// GetAllCollateralAssets returns every registered CollateralAsset in the store.
func (k Keeper) GetAllCollateralAssets(ctx sdk.Context) []types.CollateralAsset {
	store := ctx.KVStore(k.storeKey)
	prefix := []byte(types.CollateralAssetPrefix)
	iterator := sdk.KVStorePrefixIterator(store, prefix)
	defer iterator.Close()

	var assets []types.CollateralAsset
	for ; iterator.Valid(); iterator.Next() {
		var asset types.CollateralAsset
		if err := json.Unmarshal(iterator.Value(), &asset); err != nil {
			continue
		}
		assets = append(assets, asset)
	}
	return assets
}

// ─── Collateral Position management ──────────────────────────────────────────

// positionKey returns the KV key for a collateral position.
func positionKey(owner sdk.AccAddress, ibcDenom string) []byte {
	return []byte(types.CollateralPositionPrefix + owner.String() + "/" + ibcDenom)
}

// GetPosition retrieves an existing CollateralPosition.
// Returns ErrPositionNotFound if no position exists for the owner/denom pair.
func (k Keeper) GetPosition(ctx sdk.Context, owner sdk.AccAddress, ibcDenom string) (types.CollateralPosition, error) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(positionKey(owner, ibcDenom))
	if bz == nil {
		return types.CollateralPosition{}, types.ErrPositionNotFound.Wrapf("owner: %s, denom: %s", owner, ibcDenom)
	}
	var pos types.CollateralPosition
	if err := json.Unmarshal(bz, &pos); err != nil {
		return types.CollateralPosition{}, fmt.Errorf("GetPosition: unmarshal error: %w", err)
	}
	return pos, nil
}

// SetPosition persists a CollateralPosition to the KV store.
func (k Keeper) SetPosition(ctx sdk.Context, pos types.CollateralPosition) {
	store := ctx.KVStore(k.storeKey)
	owner, err := sdk.AccAddressFromBech32(pos.Owner)
	if err != nil {
		k.Logger(ctx).Error("SetPosition: invalid bech32 address", "owner", pos.Owner, "error", err)
		return
	}
	bz, err := json.Marshal(pos)
	if err != nil {
		k.Logger(ctx).Error("SetPosition: marshal failed", "error", err)
		return
	}
	store.Set(positionKey(owner, pos.IBCDenom), bz)
}

// deletePosition removes a CollateralPosition from the KV store.
func (k Keeper) deletePosition(ctx sdk.Context, owner sdk.AccAddress, ibcDenom string) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(positionKey(owner, ibcDenom))
}

// GetAllPositions iterates the entire position prefix and returns all stored positions.
func (k Keeper) GetAllPositions(ctx sdk.Context) []types.CollateralPosition {
	store := ctx.KVStore(k.storeKey)
	prefix := []byte(types.CollateralPositionPrefix)
	iterator := sdk.KVStorePrefixIterator(store, prefix)
	defer iterator.Close()

	var positions []types.CollateralPosition
	for ; iterator.Valid(); iterator.Next() {
		var pos types.CollateralPosition
		if err := json.Unmarshal(iterator.Value(), &pos); err != nil {
			continue
		}
		positions = append(positions, pos)
	}
	return positions
}

// ─── Total Locked counter ─────────────────────────────────────────────────────

// GetTotalLocked returns the total amount of a given IBC denom locked in the vault.
// Returns sdk.ZeroInt() if no coins of that denom have been locked yet.
func (k Keeper) GetTotalLocked(ctx sdk.Context, ibcDenom string) sdk.Int {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get([]byte(types.TotalLockedPrefix + ibcDenom))
	if bz == nil {
		return sdk.ZeroInt()
	}
	var amount sdk.Int
	if err := json.Unmarshal(bz, &amount); err != nil {
		return sdk.ZeroInt()
	}
	return amount
}

// SetTotalLockedPublic is the exported version of setTotalLocked, required
// by genesis initialisation to restore total-locked counters on restart.
func (k Keeper) SetTotalLockedPublic(ctx sdk.Context, ibcDenom string, amount sdk.Int) {
	k.setTotalLocked(ctx, ibcDenom, amount)
}

// setTotalLocked persists the total locked amount for an IBC denom.
func (k Keeper) setTotalLocked(ctx sdk.Context, ibcDenom string, amount sdk.Int) {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(amount)
	if err != nil {
		k.Logger(ctx).Error("setTotalLocked: marshal failed", "denom", ibcDenom, "error", err)
		return
	}
	store.Set([]byte(types.TotalLockedPrefix+ibcDenom), bz)
}

// ─── Core collateral operations ───────────────────────────────────────────────

// LockCollateral transfers `amount` of `ibcDenom` from `owner` into the vault
// module account and records / updates the owner's CollateralPosition.
func (k Keeper) LockCollateral(
	ctx sdk.Context,
	owner sdk.AccAddress,
	ibcDenom string,
	amount sdk.Int,
) error {
	// 1. Validate the asset is whitelisted and active.
	asset, err := k.GetCollateralAsset(ctx, ibcDenom)
	if err != nil {
		return err
	}
	if !asset.IsActive {
		return types.ErrAssetNotActive.Wrapf("denom: %s", ibcDenom)
	}

	// 2. Transfer tokens from owner to vault module account.
	coin := sdk.NewCoin(ibcDenom, amount)
	if err := k.bankKeeper.SendCoinsFromAccountToModule(
		ctx,
		owner,
		types.ModuleAccountName,
		sdk.NewCoins(coin),
	); err != nil {
		return fmt.Errorf("LockCollateral: bank transfer failed: %w", err)
	}

	// 3. Load or create the position.
	pos, err := k.GetPosition(ctx, owner, ibcDenom)
	if err != nil {
		// Fresh position — initialise with zero minted.
		pos = types.CollateralPosition{
			Owner:      owner.String(),
			IBCDenom:   ibcDenom,
			Amount:     sdk.ZeroInt(),
			LockedAt:   ctx.BlockTime().Unix(),
			MintedZYTD: sdk.ZeroInt(),
		}
	}

	// 4. Add to the locked amount.
	pos.Amount = pos.Amount.Add(amount)
	k.SetPosition(ctx, pos)

	// 5. Update the total locked counter.
	k.setTotalLocked(ctx, ibcDenom, k.GetTotalLocked(ctx, ibcDenom).Add(amount))

	k.Logger(ctx).Info("collateral locked",
		"owner", owner.String(),
		"denom", ibcDenom,
		"amount", amount.String(),
	)
	return nil
}

// UnlockCollateral releases `amount` of `ibcDenom` from the vault back to `owner`.
// The position must have MintedZYTD == 0 before any funds are released.
func (k Keeper) UnlockCollateral(
	ctx sdk.Context,
	owner sdk.AccAddress,
	ibcDenom string,
	amount sdk.Int,
) error {
	// 1. Load the existing position.
	pos, err := k.GetPosition(ctx, owner, ibcDenom)
	if err != nil {
		return err
	}

	// 2. Ensure there is no outstanding ZYTD debt.
	if pos.MintedZYTD.IsPositive() {
		return types.ErrUnlockDenied.Wrapf(
			"owner %s has %s ZYTD outstanding against %s collateral",
			owner, pos.MintedZYTD, ibcDenom,
		)
	}

	// 3. Ensure the requested unlock does not exceed the locked amount.
	if amount.GT(pos.Amount) {
		return types.ErrInsufficientLockedAmount.Wrapf(
			"requested %s but only %s is locked",
			amount, pos.Amount,
		)
	}

	// 4. Return tokens to the owner.
	coin := sdk.NewCoin(ibcDenom, amount)
	if err := k.bankKeeper.SendCoinsFromModuleToAccount(
		ctx,
		types.ModuleAccountName,
		owner,
		sdk.NewCoins(coin),
	); err != nil {
		return fmt.Errorf("UnlockCollateral: bank transfer failed: %w", err)
	}

	// 5. Update the position.
	pos.Amount = pos.Amount.Sub(amount)
	if pos.Amount.IsZero() {
		k.deletePosition(ctx, owner, ibcDenom)
	} else {
		k.SetPosition(ctx, pos)
	}

	// 6. Update the total locked counter.
	newTotal := k.GetTotalLocked(ctx, ibcDenom).Sub(amount)
	if newTotal.IsNegative() {
		newTotal = sdk.ZeroInt()
	}
	k.setTotalLocked(ctx, ibcDenom, newTotal)

	k.Logger(ctx).Info("collateral unlocked",
		"owner", owner.String(),
		"denom", ibcDenom,
		"amount", amount.String(),
	)
	return nil
}

// UpdateMintedZYTD adjusts the MintedZYTD field on an existing position by delta.
// delta may be positive (mint) or negative (burn/repay).
func (k Keeper) UpdateMintedZYTD(
	ctx sdk.Context,
	owner sdk.AccAddress,
	ibcDenom string,
	delta sdk.Int,
) error {
	pos, err := k.GetPosition(ctx, owner, ibcDenom)
	if err != nil {
		return err
	}
	newMinted := pos.MintedZYTD.Add(delta)
	if newMinted.IsNegative() {
		return fmt.Errorf("UpdateMintedZYTD: resulting minted_zytd would be negative (%s)", newMinted)
	}
	pos.MintedZYTD = newMinted
	k.SetPosition(ctx, pos)
	return nil
}
