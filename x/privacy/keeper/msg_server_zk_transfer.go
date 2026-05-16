// msg_server_zk_transfer.go — ZK-proven private transfer handler.
//
// This replaces msg_server_encrypted_transfer.go.
//
// On-chain flow:
//  1. Decode and validate message fields.
//  2. Verify sender has a registered commitment.
//  3. Verify the Groth16 proof (deterministic, constant-time).
//  4. Check nullifier has not been spent (double-spend prevention).
//  5. Update sender's commitment to senderNewCommitment.
//  6. Set/accumulate recipient's commitment to recipientNewCommitment.
//  7. Mark nullifier as spent.
//  8. Emit event (no amounts revealed).
//
// Privacy guarantee: No plaintext amounts appear anywhere in this function.
// Validators learn only: "A transferred to B, nullifier N is now spent."
package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"zytherion/x/privacy/types"
	"zytherion/x/privacy/zk"
)

func (ms msgServer) ZKTransfer(
	goCtx context.Context,
	msg *types.MsgZKTransfer,
) (*types.MsgZKTransferResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// ── 1. Parse addresses ────────────────────────────────────────────────────
	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid sender — %s", types.ErrInvalidAddress, err)
	}
	recipientAddr, err := sdk.AccAddressFromBech32(msg.Recipient)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid recipient — %s", types.ErrInvalidAddress, err)
	}

	// ── 2. Validate commitment bytes (stateless) ──────────────────────────────
	if err := zk.ValidateCommitmentBytes(msg.SenderNewCommitment); err != nil {
		return nil, fmt.Errorf("%w: sender commitment — %s", types.ErrInvalidCommitment, err)
	}
	if err := zk.ValidateCommitmentBytes(msg.RecipientNewCommitment); err != nil {
		return nil, fmt.Errorf("%w: recipient commitment — %s", types.ErrInvalidCommitment, err)
	}
	if len(msg.Nullifier) != 32 {
		return nil, fmt.Errorf("%w: nullifier must be 32 bytes", types.ErrInvalidCommitment)
	}

	// ── 3. Sender must have a registered commitment ───────────────────────────
	if !ms.HasCommitment(ctx, senderAddr) {
		return nil, fmt.Errorf("%w: %s", types.ErrNoCommitment, msg.Sender)
	}

	// ── 4. Verify Groth16 proof ───────────────────────────────────────────────
	// VerifyTransferProof is deterministic and constant-time.
	// It returns error on ANY invalid proof — no partial acceptance.
	if err := ms.VerifyTransferProof(msg.ZkProof, msg.PublicInputs); err != nil {
		return nil, fmt.Errorf("%w: %s", types.ErrProofVerificationFailed, err)
	}

	// ── 5. Double-spend check ─────────────────────────────────────────────────
	if ms.HasNullifier(ctx, msg.Nullifier) {
		return nil, fmt.Errorf("%w", types.ErrNullifierAlreadySpent)
	}

	// ── 6. Update sender's commitment ────────────────────────────────────────
	if err := ms.SetCommitment(ctx, senderAddr, msg.SenderNewCommitment); err != nil {
		return nil, fmt.Errorf("failed to update sender commitment: %w", err)
	}

	// ── 7. Set recipient's commitment ─────────────────────────────────────────
	// If recipient has no commitment yet, this initialises it.
	// If they already have one, we overwrite with the new commitment.
	// (In a production UTXO model, you'd accumulate; here we store the latest.)
	if err := ms.SetCommitment(ctx, recipientAddr, msg.RecipientNewCommitment); err != nil {
		return nil, fmt.Errorf("failed to set recipient commitment: %w", err)
	}

	// ── 8. Mark nullifier as spent ────────────────────────────────────────────
	ms.SetNullifier(ctx, msg.Nullifier)

	// ── 9. Emit event (no amounts — privacy preserved) ────────────────────────
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeZKTransfer,
			sdk.NewAttribute(types.AttributeKeySender, msg.Sender),
			sdk.NewAttribute(types.AttributeKeyRecipient, msg.Recipient),
		),
	)

	ms.Logger(ctx).Info("ZK transfer processed",
		"sender", msg.Sender,
		"recipient", msg.Recipient,
	)

	return &types.MsgZKTransferResponse{}, nil
}
