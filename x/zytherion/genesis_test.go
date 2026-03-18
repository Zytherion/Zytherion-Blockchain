package zytherion_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	keepertest "zytherion/testutil/keeper"
	"zytherion/testutil/nullify"
	"zytherion/x/zytherion"
	"zytherion/x/zytherion/types"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),

		// this line is used by starport scaffolding # genesis/test/state
	}

	k, ctx := keepertest.ZytherionKeeper(t)
	zytherion.InitGenesis(ctx, *k, genesisState)
	got := zytherion.ExportGenesis(ctx, *k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	// this line is used by starport scaffolding # genesis/test/assert
}
