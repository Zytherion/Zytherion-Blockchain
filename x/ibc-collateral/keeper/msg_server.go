package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"zytherion/x/ibc-collateral/types"
)

// msgServer is the concrete implementation of types.MsgServer.
type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

// Ensure compile-time interface compliance.
var _ types.MsgServer = msgServer{}

// ─── LockCollateral ───────────────────────────────────────────────────────────

// LockCollateral handles a MsgLockCollateral message.
// It validates the asset, transfers tokens to the vault, and emits an event.
func (s msgServer) LockCollateral(ctx sdk.Context, msg *types.MsgLockCollateral) (*types.MsgLockCollateralResponse, error) {
	ownerAddr, err := sdk.AccAddressFromBech32(msg.Owner)
	if err != nil {
		return nil, types.ErrInvalidAddress.Wrapf("owner: %s", err)
	}

	if err := s.Keeper.LockCollateral(ctx, ownerAddr, msg.IBCDenom, msg.Amount); err != nil {
		return nil, err
	}

	// Retrieve updated position to include the post-lock amount in the response.
	pos, err := s.Keeper.GetPosition(ctx, ownerAddr, msg.IBCDenom)
	if err != nil {
		return nil, fmt.Errorf("LockCollateral: failed to fetch updated position: %w", err)
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeLockCollateral,
			sdk.NewAttribute(types.AttributeKeyOwner, msg.Owner),
			sdk.NewAttribute(types.AttributeKeyIBCDenom, msg.IBCDenom),
			sdk.NewAttribute(types.AttributeKeyAmount, msg.Amount.String()),
		),
	)

	return &types.MsgLockCollateralResponse{
		Owner:    msg.Owner,
		IBCDenom: msg.IBCDenom,
		Locked:   pos.Amount,
	}, nil
}

// ─── UnlockCollateral ─────────────────────────────────────────────────────────

// UnlockCollateral handles a MsgUnlockCollateral message.
// It validates outstanding debt, releases tokens from the vault, and emits an event.
func (s msgServer) UnlockCollateral(ctx sdk.Context, msg *types.MsgUnlockCollateral) (*types.MsgUnlockCollateralResponse, error) {
	ownerAddr, err := sdk.AccAddressFromBech32(msg.Owner)
	if err != nil {
		return nil, types.ErrInvalidAddress.Wrapf("owner: %s", err)
	}

	// Snapshot the position before unlocking to capture the remaining amount later.
	oldPos, err := s.Keeper.GetPosition(ctx, ownerAddr, msg.IBCDenom)
	if err != nil {
		return nil, err
	}

	if err := s.Keeper.UnlockCollateral(ctx, ownerAddr, msg.IBCDenom, msg.Amount); err != nil {
		return nil, err
	}

	remaining := oldPos.Amount.Sub(msg.Amount)
	if remaining.IsNegative() {
		remaining = sdk.ZeroInt()
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeUnlockCollateral,
			sdk.NewAttribute(types.AttributeKeyOwner, msg.Owner),
			sdk.NewAttribute(types.AttributeKeyIBCDenom, msg.IBCDenom),
			sdk.NewAttribute(types.AttributeKeyAmount, msg.Amount.String()),
		),
	)

	return &types.MsgUnlockCollateralResponse{
		Owner:     msg.Owner,
		IBCDenom:  msg.IBCDenom,
		Unlocked:  msg.Amount,
		Remaining: remaining,
	}, nil
}
