package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"zytherion/x/oracle/types"
)

// msgServer implements types.MsgServer on top of the keeper.
type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the oracle MsgServer interface.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

// SubmitPrice handles a MsgSubmitPrice transaction.
func (s msgServer) SubmitPrice(goCtx context.Context, req *types.MsgSubmitPrice) (*types.MsgSubmitPriceResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Parse and validate the submitter address.
	submitter, err := sdk.AccAddressFromBech32(req.Submitter)
	if err != nil {
		return nil, fmt.Errorf("invalid submitter address: %w", err)
	}

	// Verify the submitter is a bonded validator.
	if !s.IsValidatorSubmitter(ctx, submitter) {
		return nil, types.ErrUnauthorizedSubmitter
	}

	params := s.GetParams(ctx)

	// Validate that the denom is whitelisted.
	whitelisted := false
	for _, d := range params.WhitelistedDenoms {
		if d == req.Denom {
			whitelisted = true
			break
		}
	}
	if !whitelisted {
		return nil, types.ErrInvalidDenom
	}

	// Check price age: reject if the submission references a height older than MaxPriceAge.
	if ctx.BlockHeight() > params.MaxPriceAge && req.PriceUsd.IsNil() {
		return nil, types.ErrPriceTooOld
	}

	// Build and store the price entry.
	entry := types.PriceEntry{
		Denom:     req.Denom,
		PriceUSD:  req.PriceUsd,
		Submitter: req.Submitter,
		Height:    ctx.BlockHeight(),
		Timestamp: ctx.BlockTime(),
	}
	if err := s.SetPrice(ctx, entry); err != nil {
		return nil, fmt.Errorf("store price: %w", err)
	}

	// Attempt to update the TWAP cache (non-fatal if not enough samples yet).
	twap, err := s.ComputeTWAP(ctx, req.Denom)
	if err == nil {
		s.SetTWAP(ctx, twap)
	}

	// Prune stale price entries.
	s.PruneOldPrices(ctx, ctx.BlockHeight())

	// Emit an event for indexers.
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeSubmitPrice,
		sdk.NewAttribute(types.AttributeKeyDenom, req.Denom),
		sdk.NewAttribute(types.AttributeKeyPrice, req.PriceUsd.String()),
		sdk.NewAttribute(types.AttributeKeySubmitter, req.Submitter),
	))

	s.Logger(ctx).Info("oracle price submitted",
		"denom", req.Denom,
		"price", req.PriceUsd.String(),
		"submitter", req.Submitter,
		"height", ctx.BlockHeight(),
	)

	return &types.MsgSubmitPriceResponse{}, nil
}
