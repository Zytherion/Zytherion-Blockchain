package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	testkeeper "zytherion/testutil/keeper"
	"zytherion/x/zytherion/types"
)

func TestGetParams(t *testing.T) {
	k, ctx := testkeeper.ZytherionKeeper(t)
	params := types.DefaultParams()

	k.SetParams(ctx, params)

	require.EqualValues(t, params, k.GetParams(ctx))
}
