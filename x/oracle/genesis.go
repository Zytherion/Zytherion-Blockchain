package oracle

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"zytherion/x/oracle/keeper"
	"zytherion/x/oracle/types"
)

// InitGenesis initializes the oracle module's state from a provided genesis state.
// It stores the oracle parameters.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	k.SetParams(ctx, genState.Params)
}

// ExportGenesis returns the oracle module's exported genesis state.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()
	genesis.Params = k.GetParams(ctx)
	return genesis
}
