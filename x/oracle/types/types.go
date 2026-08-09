package types

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// PriceEntry represents a single price submission from a validator.
type PriceEntry struct {
	Denom     string    `json:"denom"`
	PriceUSD  sdk.Dec   `json:"price_usd"`
	Submitter string    `json:"submitter"` // bech32 string
	Height    int64     `json:"height"`
	Timestamp time.Time `json:"timestamp"`
}

// TWAPData holds a computed time-weighted average price for a denom.
type TWAPData struct {
	Denom       string  `json:"denom"`
	TWAP        sdk.Dec `json:"twap"`
	WindowStart int64   `json:"window_start"`
	WindowEnd   int64   `json:"window_end"`
	NumSamples  int     `json:"num_samples"`
}

// OracleParams are the governance parameters for the oracle module.
type OracleParams struct {
	TwapWindowBlocks  int64    `json:"twap_window_blocks"`
	MinSubmissions    int      `json:"min_submissions"`
	MaxPriceAge       int64    `json:"max_price_age"`
	WhitelistedDenoms []string `json:"whitelisted_denoms"`
}

// DefaultOracleParams returns the default oracle module parameters.
func DefaultOracleParams() OracleParams {
	return OracleParams{
		TwapWindowBlocks:  30,
		MinSubmissions:    2,
		MaxPriceAge:       5,
		WhitelistedDenoms: []string{"ZYTC", "axlUSDC", "mUSDT", "ATOM", "wBTC", "wETH"},
	}
}
