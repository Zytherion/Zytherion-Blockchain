package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"zytherion/x/privacy/types"
)

func (ms msgServer) EncryptedTransfer(
	goCtx context.Context,
	msg *types.MsgEncryptedTransfer,
) (*types.MsgEncryptedTransferResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid sender â€” %s", types.ErrInvalidAddress, err)
	}
	recipientAddr, err := sdk.AccAddressFromBech32(msg.Recipient)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid recipient â€” %s", types.ErrInvalidAddress, err)
	}

	if !ms.HasEncryptedBalance(ctx, senderAddr) {
		return nil, fmt.Errorf("%w: %s", types.ErrNoSenderBalance, msg.Sender)
	}

	if err := ms.HomomorphicSub(ctx, senderAddr, msg.AmountCiphertext); err != nil {
		return nil, fmt.Errorf("%w: %s", types.ErrHomomorphicSub, err)
	}

	if err := ms.HomomorphicAdd(ctx, recipientAddr, msg.AmountCiphertext); err != nil {
		return nil, fmt.Errorf("%w: %s", types.ErrHomomorphicAdd, err)
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeEncryptedTransfer,
			sdk.NewAttribute(types.AttributeKeySender, msg.Sender),
			sdk.NewAttribute(types.AttributeKeyRecipient, msg.Recipient),
		),
	)

	ms.Logger(ctx).Info("encrypted transfer processed",
		"sender", msg.Sender,
		"recipient", msg.Recipient,
	)

	return &types.MsgEncryptedTransferResponse{}, nil
}