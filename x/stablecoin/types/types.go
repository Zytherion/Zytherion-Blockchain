package types

import sdk "github.com/cosmos/cosmos-sdk/types"

// StablecoinParams holds module-level parameters.
type StablecoinParams struct {
	ZYTDDenom           string  `json:"zytd_denom"`
	StabilityFeePerYear sdk.Dec `json:"stability_fee_per_year"`
	LiquidationPenalty  sdk.Dec `json:"liquidation_penalty"`
	LiquidatorReward    sdk.Dec `json:"liquidator_reward"`
	ProtocolFeeReceiver string  `json:"protocol_fee_receiver"`
}

// DefaultStablecoinParams returns sensible defaults.
func DefaultStablecoinParams() StablecoinParams {
	return StablecoinParams{
		ZYTDDenom:           ZYTDDenom,
		StabilityFeePerYear: sdk.NewDecWithPrec(5, 3),  // 0.5%
		LiquidationPenalty:  sdk.NewDecWithPrec(10, 2), // 10%
		LiquidatorReward:    sdk.NewDecWithPrec(8, 2),  // 8%
		ProtocolFeeReceiver: ModuleAccountName,
	}
}

// MintRecord tracks how much ZYTD was minted against a collateral position.
type MintRecord struct {
	Owner         string  `json:"owner"`          // bech32
	IBCDenom      string  `json:"ibc_denom"`      // which collateral was used
	Minted        sdk.Int `json:"minted"`         // ZYTD amount minted (uzytd)
	CollateralUSD sdk.Dec `json:"collateral_usd"` // USD value at mint time
	MintedAt      int64   `json:"minted_at"`      // block height
}
