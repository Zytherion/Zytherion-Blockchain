package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"zytherion/x/privacy/types"
)

// Deposit handles MsgDeposit â€” converting plaintext bank tokens into an
// encrypted balance stored on-chain in the privacy module.
//
// Flow:
//  1. Parse and validate msg.Amount into a sdk.Coin.
//  2. Move the coins from the depositor's account to the privacy module
//     escrow account via bankKeeper.SendCoinsFromAccountToModule.
//  3. Encrypt the plaintext coin amount using the keeper's shared TFHE context.
//  4. Homomorphically ADD the new deposit ciphertext to any existing balance
//     (auto-initialising to Enc(0) if no prior balance exists).
//  5. Persist the updated ciphertext via SetEncryptedBalance.
//
// All on-chain state references only compressed TFHE ciphertexts.
// No plaintext amounts are stored after step 3.
func (ms msgServer) Deposit(
	goCtx context.Context,
	msg *types.MsgDeposit,
) (*types.MsgDepositResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// 1. Parse depositor address
	creatorAddr, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid creator address â€” %s",
			types.ErrInvalidAddress, err)
	}

	// 2. Parse coin
	coin, err := sdk.ParseCoinNormalized(msg.Amount)
	if err != nil {
		return nil, fmt.Errorf("%w: %q is not a valid coin â€” %s",
			types.ErrInvalidDepositAmount, msg.Amount, err)
	}
	if !coin.IsPositive() {
		return nil, fmt.Errorf("%w: deposit amount must be positive, got %s",
			types.ErrInvalidDepositAmount, msg.Amount)
	}

	// 3. Escrow the plaintext coins into the privacy module account
	if err := ms.bankKeeper.SendCoinsFromAccountToModule(
		ctx,
		creatorAddr,
		types.ModuleName,
		sdk.NewCoins(coin),
	); err != nil {
		return nil, fmt.Errorf("%w: bank transfer failed â€” %s",
			types.ErrInsufficientBalance, err)
	}

	// 4. Encrypt the plaintext amount (returns compressed TFHE ciphertext)
	depositAmount := uint64(coin.Amount.Int64())
	depositBytes, err := ms.EncryptAmount(depositAmount)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt deposit amount: %w", err)
	}

	// 5. Homomorphic add to existing balance (auto-init to Enc(0) if absent)
	if err := ms.HomomorphicAdd(ctx, creatorAddr, depositBytes); err != nil {
		return nil, fmt.Errorf("%w: %s", types.ErrHomomorphicAdd, err)
	}

	// 6. Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeDeposit,
			sdk.NewAttribute(types.AttributeKeyCreator, msg.Creator),
			sdk.NewAttribute(types.AttributeKeyDepositDenom, coin.Denom),
		),
	)

	ms.Logger(ctx).Info("privacy deposit processed",
		"creator", msg.Creator,
		"coin", coin.String(),
	)

	return &types.MsgDepositResponse{}, nil
}