package types

import "fmt"

// GenesisState defines the genesis state for the ibccollateral module.
type GenesisState struct {
	// CollateralAssets is the whitelisted set of IBC collateral assets.
	CollateralAssets []CollateralAsset `json:"collateral_assets"`
	// Positions holds all active collateral positions at export time.
	Positions []CollateralPosition `json:"positions"`
}

// DefaultGenesis returns the default genesis state populated with the
// default collateral asset whitelist and no open positions.
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		CollateralAssets: DefaultCollateralAssets(),
		Positions:        []CollateralPosition{},
	}
}

// Validate performs basic stateless validation of the genesis state.
func (gs GenesisState) Validate() error {
	seen := make(map[string]struct{})
	for _, asset := range gs.CollateralAssets {
		if asset.IBCDenom == "" {
			return fmt.Errorf("collateral asset has empty ibc_denom")
		}
		if _, dup := seen[asset.IBCDenom]; dup {
			return fmt.Errorf("duplicate collateral asset ibc_denom: %s", asset.IBCDenom)
		}
		seen[asset.IBCDenom] = struct{}{}
		if asset.MinRatio.IsNil() || asset.MinRatio.IsNegative() {
			return fmt.Errorf("invalid min_ratio for asset %s", asset.IBCDenom)
		}
		if asset.LiquidationThreshold.IsNil() || asset.LiquidationThreshold.IsNegative() {
			return fmt.Errorf("invalid liquidation_threshold for asset %s", asset.IBCDenom)
		}
	}
	for _, pos := range gs.Positions {
		if pos.Owner == "" {
			return fmt.Errorf("collateral position has empty owner")
		}
		if pos.IBCDenom == "" {
			return fmt.Errorf("collateral position for owner %s has empty ibc_denom", pos.Owner)
		}
		if pos.Amount.IsNegative() {
			return fmt.Errorf("collateral position for owner %s/%s has negative amount", pos.Owner, pos.IBCDenom)
		}
		if pos.MintedZYTD.IsNegative() {
			return fmt.Errorf("collateral position for owner %s/%s has negative minted_zytd", pos.Owner, pos.IBCDenom)
		}
	}
	return nil
}

// Reset implements proto.Message.
func (gs *GenesisState) Reset() { *gs = GenesisState{} }

// String implements proto.Message.
func (gs *GenesisState) String() string {
	return fmt.Sprintf("GenesisState{Assets: %d, Positions: %d}", len(gs.CollateralAssets), len(gs.Positions))
}

// ProtoMessage implements proto.Message.
func (gs *GenesisState) ProtoMessage() {}
