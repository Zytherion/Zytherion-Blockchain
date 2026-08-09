package ibccollateral

import (
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"zytherion/x/ibc-collateral/keeper"
	"zytherion/x/ibc-collateral/types"
)

// DefaultGenesis returns the default genesis state for the ibccollateral module.
// It registers all hardcoded default collateral assets (ZYTC, axlUSDC, mUSDT, ATOM, wBTC, wETH).
func DefaultGenesis() *types.GenesisState {
	return types.DefaultGenesis()
}

// ValidateGenesis validates the genesis state and returns an error on any failure.
func ValidateGenesis(data types.GenesisState) error {
	return data.Validate()
}

// InitGenesis initialises the ibccollateral module from a genesis state.
// It registers every whitelisted CollateralAsset and restores any existing positions.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	// Register all whitelisted collateral assets.
	for _, asset := range genState.CollateralAssets {
		if err := k.RegisterCollateralAsset(ctx, asset); err != nil {
			panic(fmt.Sprintf("InitGenesis: failed to register collateral asset %q: %v", asset.IBCDenom, err))
		}
	}

	// Restore any positions exported from a previous chain state.
	for _, pos := range genState.Positions {
		// Ensure Amount and MintedZYTD are initialised (JSON zero-value can be nil).
		if pos.Amount.IsNil() {
			pos.Amount = sdk.ZeroInt()
		}
		if pos.MintedZYTD.IsNil() {
			pos.MintedZYTD = sdk.ZeroInt()
		}
		k.SetPosition(ctx, pos)

		// Recalculate total locked counters from the restored positions.
		existing := k.GetTotalLocked(ctx, pos.IBCDenom)
		k.SetTotalLockedPublic(ctx, pos.IBCDenom, existing.Add(pos.Amount))
	}

	ctx.Logger().Info("ibccollateral: genesis initialised",
		"assets", len(genState.CollateralAssets),
		"positions", len(genState.Positions),
	)
}

// ExportGenesis returns the current module state as a GenesisState for export.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	return &types.GenesisState{
		CollateralAssets: k.GetAllCollateralAssets(ctx),
		Positions:        k.GetAllPositions(ctx),
	}
}

// MustMarshalJSON marshals a GenesisState to JSON bytes, panicking on error.
func MustMarshalJSON(gs *types.GenesisState) []byte {
	bz, err := json.Marshal(gs)
	if err != nil {
		panic(fmt.Sprintf("MustMarshalJSON: %v", err))
	}
	return bz
}

// MustUnmarshalJSON unmarshals JSON bytes into a GenesisState, panicking on error.
func MustUnmarshalJSON(bz []byte) *types.GenesisState {
	var gs types.GenesisState
	if err := json.Unmarshal(bz, &gs); err != nil {
		panic(fmt.Sprintf("MustUnmarshalJSON: %v", err))
	}
	return &gs
}
