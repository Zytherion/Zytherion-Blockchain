package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"zytherion/x/stablecoin/types"
)

// QueryMintRecord returns the mint record for a given owner and ibc denom.
func (k Keeper) QueryMintRecord(goCtx context.Context, owner, ibcDenom string) (types.MintRecord, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	addr, err := sdk.AccAddressFromBech32(owner)
	if err != nil {
		return types.MintRecord{}, fmt.Errorf("invalid owner address: %w", err)
	}
	return k.GetMintRecord(ctx, addr, ibcDenom)
}

// QueryCollateralRatio returns the live collateral ratio for a position.
func (k Keeper) QueryCollateralRatio(goCtx context.Context, owner, ibcDenom string) (sdk.Dec, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	addr, err := sdk.AccAddressFromBech32(owner)
	if err != nil {
		return sdk.ZeroDec(), fmt.Errorf("invalid owner address: %w", err)
	}
	return k.GetCollateralRatio(ctx, addr, ibcDenom)
}

// QueryTotalSupply returns the total ZYTD in circulation.
func (k Keeper) QueryTotalSupply(goCtx context.Context) (sdk.Int, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	return k.GetTotalSupply(ctx), nil
}

// QueryMaxMintable returns the max ZYTD mintable given collateral denom and amount.
func (k Keeper) QueryMaxMintable(goCtx context.Context, ibcDenom string, collateralAmount sdk.Int) (sdk.Int, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	return k.GetMaxMintable(ctx, ibcDenom, collateralAmount)
}
