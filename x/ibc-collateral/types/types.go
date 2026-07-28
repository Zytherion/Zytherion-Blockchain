package types

import sdk "github.com/cosmos/cosmos-sdk/types"

// CollateralAsset defines an IBC-bridged asset that can be used as collateral.
type CollateralAsset struct {
	IBCDenom             string  `json:"ibc_denom"`
	BaseDenom            string  `json:"base_denom"`
	MinRatio             sdk.Dec `json:"min_ratio"`
	LiquidationThreshold sdk.Dec `json:"liquidation_threshold"`
	IsActive             bool    `json:"is_active"`
}

// CollateralPosition represents an individual collateral lock by a user.
type CollateralPosition struct {
	Owner      string  `json:"owner"` // bech32
	IBCDenom   string  `json:"ibc_denom"`
	Amount     sdk.Int `json:"amount"`
	LockedAt   int64   `json:"locked_at"`
	MintedZYTD sdk.Int `json:"minted_zytd"`
}

// DefaultCollateralAssets returns the hardcoded whitelist of accepted IBC collateral assets.
func DefaultCollateralAssets() []CollateralAsset {
	return []CollateralAsset{
		{
			IBCDenom:             "uzytc",
			BaseDenom:            "ZYTC",
			MinRatio:             sdk.NewDecWithPrec(200, 2),
			LiquidationThreshold: sdk.NewDecWithPrec(150, 2),
			IsActive:             true,
		},
		{
			IBCDenom:             "ibc/axlUSDC",
			BaseDenom:            "axlUSDC",
			MinRatio:             sdk.NewDecWithPrec(110, 2),
			LiquidationThreshold: sdk.NewDecWithPrec(105, 2),
			IsActive:             true,
		},
		{
			IBCDenom:             "ibc/mUSDT",
			BaseDenom:            "mUSDT",
			MinRatio:             sdk.NewDecWithPrec(110, 2),
			LiquidationThreshold: sdk.NewDecWithPrec(105, 2),
			IsActive:             true,
		},
		{
			IBCDenom:             "ibc/ATOM",
			BaseDenom:            "ATOM",
			MinRatio:             sdk.NewDecWithPrec(160, 2),
			LiquidationThreshold: sdk.NewDecWithPrec(130, 2),
			IsActive:             true,
		},
		{
			IBCDenom:             "ibc/wBTC",
			BaseDenom:            "wBTC",
			MinRatio:             sdk.NewDecWithPrec(150, 2),
			LiquidationThreshold: sdk.NewDecWithPrec(125, 2),
			IsActive:             true,
		},
		{
			IBCDenom:             "ibc/wETH",
			BaseDenom:            "wETH",
			MinRatio:             sdk.NewDecWithPrec(150, 2),
			LiquidationThreshold: sdk.NewDecWithPrec(125, 2),
			IsActive:             true,
		},
	}
}
