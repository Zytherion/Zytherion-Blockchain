package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"zytherion/x/ibc-collateral/types"
)

// QueryServer wraps Keeper and provides query handler methods.
// Because this module does not use proto-generated gRPC stubs, queries are
// dispatched through the legacy ABCI querier (see module.go LegacyQuerierHandler).
type QueryServer struct {
	Keeper
}

// NewQueryServer creates a QueryServer wrapping the given keeper.
func NewQueryServer(k Keeper) *QueryServer {
	return &QueryServer{Keeper: k}
}

// ─── QueryPosition ─────────────────────────────────────────────────────────────

// QueryPosition returns the collateral position for an owner/denom pair.
func (q *QueryServer) QueryPosition(ctx sdk.Context, req *types.QueryPositionRequest) (*types.QueryPositionResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("QueryPosition: nil request")
	}
	ownerAddr, err := sdk.AccAddressFromBech32(req.Owner)
	if err != nil {
		return nil, types.ErrInvalidAddress.Wrapf("owner: %s", err)
	}
	pos, err := q.Keeper.GetPosition(ctx, ownerAddr, req.IBCDenom)
	if err != nil {
		return nil, err
	}
	return &types.QueryPositionResponse{Position: pos}, nil
}

// ─── QueryAssets ────────────────────────────────────────────────────────────────

// QueryAssets returns all registered collateral assets.
func (q *QueryServer) QueryAssets(ctx sdk.Context, _ *types.QueryAssetsRequest) (*types.QueryAssetsResponse, error) {
	assets := q.Keeper.GetAllCollateralAssets(ctx)
	return &types.QueryAssetsResponse{Assets: assets}, nil
}

// ─── QueryTotalLocked ─────────────────────────────────────────────────────────

// QueryTotalLocked returns the total amount locked for a given IBC denom.
func (q *QueryServer) QueryTotalLocked(ctx sdk.Context, req *types.QueryTotalLockedRequest) (*types.QueryTotalLockedResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("QueryTotalLocked: nil request")
	}
	if req.IBCDenom == "" {
		return nil, fmt.Errorf("QueryTotalLocked: ibc_denom must not be empty")
	}
	total := q.Keeper.GetTotalLocked(ctx, req.IBCDenom)
	return &types.QueryTotalLockedResponse{
		IBCDenom:    req.IBCDenom,
		TotalLocked: total,
	}, nil
}
