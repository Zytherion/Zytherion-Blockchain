package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"zytherion/x/oracle/types"
)

var _ types.QueryServer = Keeper{}

// QueryPrice returns the latest price entry for a given denom.
func (k Keeper) QueryPrice(goCtx context.Context, req *types.QueryPriceRequest) (*types.QueryPriceResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	if req.Denom == "" {
		return nil, fmt.Errorf("denom cannot be empty")
	}

	entry, err := k.GetLatestPrice(ctx, req.Denom)
	if err != nil {
		return nil, fmt.Errorf("oracle: %w", err)
	}

	return &types.QueryPriceResponse{
		Denom:     entry.Denom,
		PriceUsd:  entry.PriceUSD,
		Submitter: entry.Submitter,
		Height:    entry.Height,
	}, nil
}

// QueryTWAP returns the latest cached TWAP for a given denom.
func (k Keeper) QueryTWAP(goCtx context.Context, req *types.QueryTWAPRequest) (*types.QueryTWAPResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	if req.Denom == "" {
		return nil, fmt.Errorf("denom cannot be empty")
	}

	twap, err := k.GetTWAP(ctx, req.Denom)
	if err != nil {
		// If no cached TWAP, try computing on the fly.
		twap, err = k.ComputeTWAP(ctx, req.Denom)
		if err != nil {
			return nil, fmt.Errorf("oracle: %w", err)
		}
	}

	return &types.QueryTWAPResponse{
		Denom:       twap.Denom,
		Twap:        twap.TWAP,
		WindowStart: twap.WindowStart,
		WindowEnd:   twap.WindowEnd,
		NumSamples:  int32(twap.NumSamples),
	}, nil
}

// QueryAllPrices returns all stored price entries for a denom from a given height.
func (k Keeper) QueryAllPrices(goCtx context.Context, req *types.QueryAllPricesRequest) (*types.QueryAllPricesResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	if req.Denom == "" {
		return nil, fmt.Errorf("denom cannot be empty")
	}

	entries := k.GetPriceHistory(ctx, req.Denom, req.FromHeight)
	var respPrices []*types.QueryPriceResponse
	for _, entry := range entries {
		respPrices = append(respPrices, &types.QueryPriceResponse{
			Denom:     entry.Denom,
			PriceUsd:  entry.PriceUSD,
			Submitter: entry.Submitter,
			Height:    entry.Height,
		})
	}

	return &types.QueryAllPricesResponse{Prices: respPrices}, nil
}
