package privacy_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	keepertest "zytherion/testutil/keeper"
	"zytherion/testutil/nullify"
	"zytherion/x/privacy"
	"zytherion/x/privacy/types"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),

		// this line is used by starport scaffolding # genesis/test/state
	}

	k, ctx := keepertest.PrivacyKeeper(t)
	privacy.InitGenesis(ctx, *k, genesisState)
	got := privacy.ExportGenesis(ctx, *k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	// this line is used by starport scaffolding # genesis/test/assert
}
