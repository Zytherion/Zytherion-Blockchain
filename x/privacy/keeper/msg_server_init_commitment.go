// msg_server_init_commitment.go — InitCommitment handler.
//
// Handles MsgInitCommitment: the user's first step to enter the privacy system.
//
// Flow:
//  1. Parse depositor address and coin amount.
//  2. Escrow plaintext coins: user bank account → privacy module account.
//  3. Validate commitment bytes structurally.
//  4. Verify Groth16 proof that the commitment correctly encodes the deposit amount.
//  5. Store commitment on-chain (32 bytes, replaces old 5 KB FHE ciphertext).
//  6. Emit deposit event (denom visible, amount hidden in commitment).
//
// After this call, the user's plaintext balance lives in the module escrow
// and their privacy balance is represented by the stored commitment.
// No plaintext is stored on-chain after step 2.
package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"zytherion/x/privacy/types"
	"zytherion/x/privacy/zk"
)

func (ms msgServer) InitCommitment(
	goCtx context.Context,
	msg *types.MsgInitCommitment,
) (*types.MsgInitCommitmentResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// ── 1. Parse depositor address ────────────────────────────────────────────
	creatorAddr, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid creator address — %s", types.ErrInvalidAddress, err)
	}

	// ── 2. Parse and validate coin ────────────────────────────────────────────
	coin, err := sdk.ParseCoinNormalized(msg.Amount)
	if err != nil {
		return nil, fmt.Errorf("%w: %q is not a valid coin — %s",
			types.ErrInvalidDepositAmount, msg.Amount, err)
	}
	if !coin.IsPositive() {
		return nil, fmt.Errorf("%w: deposit amount must be positive, got %s",
			types.ErrInvalidDepositAmount, msg.Amount)
	}

	// ── 3. Escrow plaintext coins → module account ─────────────────────────
	if err := ms.bankKeeper.SendCoinsFromAccountToModule(
		ctx,
		creatorAddr,
		types.ModuleName,
		sdk.NewCoins(coin),
	); err != nil {
		return nil, fmt.Errorf("%w: bank transfer failed — %s",
			types.ErrInsufficientBalance, err)
	}

	// ── 4. Validate commitment bytes (stateless) ──────────────────────────────
	if err := zk.ValidateCommitmentBytes(msg.Commitment); err != nil {
		return nil, fmt.Errorf("%w: %s", types.ErrInvalidCommitment, err)
	}

	// ── 5. Verify ZK proof ────────────────────────────────────────────────────
	// The proof certifies: "I know (amount, blinding) such that
	//   Commitment = MiMC(amount, blinding)  and  amount = deposited_amount"
	if err := ms.VerifyTransferProof(msg.ZkProof, msg.PublicInputs); err != nil {
		return nil, fmt.Errorf("%w: %s", types.ErrProofVerificationFailed, err)
	}

	// ── 6. Store commitment ───────────────────────────────────────────────────
	if err := ms.SetCommitment(ctx, creatorAddr, msg.Commitment); err != nil {
		return nil, fmt.Errorf("failed to store commitment: %w", err)
	}

	// ── 7. Emit event ─────────────────────────────────────────────────────────
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeInitCommitment,
			sdk.NewAttribute(types.AttributeKeyCreator, msg.Creator),
			sdk.NewAttribute(types.AttributeKeyDepositDenom, coin.Denom),
			// Amount is intentionally omitted — it is hidden in the commitment.
		),
	)

	ms.Logger(ctx).Info("commitment initialised",
		"creator", msg.Creator,
		"denom", coin.Denom,
	)

	return &types.MsgInitCommitmentResponse{}, nil
}
