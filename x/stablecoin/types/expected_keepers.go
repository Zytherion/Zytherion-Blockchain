package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	ibccollateraltypes "zytherion/x/ibc-collateral/types"
	oracletypes "zytherion/x/oracle/types"
)

// OracleKeeper defines the expected oracle keeper interface used by x/stablecoin.
type OracleKeeper interface {
	GetTWAP(ctx sdk.Context, denom string) (oracletypes.TWAPData, error)
	GetLatestPrice(ctx sdk.Context, denom string) (oracletypes.PriceEntry, error)
}

// IBCCollateralKeeper defines the expected ibc-collateral keeper interface used by x/stablecoin.
type IBCCollateralKeeper interface {
	GetCollateralAsset(ctx sdk.Context, ibcDenom string) (ibccollateraltypes.CollateralAsset, error)
	GetPosition(ctx sdk.Context, owner sdk.AccAddress, ibcDenom string) (ibccollateraltypes.CollateralPosition, error)
	SetPosition(ctx sdk.Context, pos ibccollateraltypes.CollateralPosition)
	LockCollateral(ctx sdk.Context, owner sdk.AccAddress, ibcDenom string, amount sdk.Int) error
	UnlockCollateral(ctx sdk.Context, owner sdk.AccAddress, ibcDenom string, amount sdk.Int) error
	UpdateMintedZYTD(ctx sdk.Context, owner sdk.AccAddress, ibcDenom string, delta sdk.Int) error
}

// BankKeeper defines the expected bank keeper interface used by x/stablecoin.
type BankKeeper interface {
	SendCoins(ctx sdk.Context, fromAddr sdk.AccAddress, toAddr sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromAccountToModule(ctx sdk.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx sdk.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	MintCoins(ctx sdk.Context, moduleName string, amt sdk.Coins) error
	BurnCoins(ctx sdk.Context, moduleName string, amt sdk.Coins) error
	SpendableCoins(ctx sdk.Context, addr sdk.AccAddress) sdk.Coins
	GetBalance(ctx sdk.Context, addr sdk.AccAddress, denom string) sdk.Coin
}
