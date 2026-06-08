// msg_server_init_commitment.go — InitCommitment handler.
//
// Handles MsgInitCommitment: the user's first step to enter the privacy system.
//
// # v0.3 changes
//
// ZK proof verification has been REMOVED. The commitment is now validated
// only as a 32-byte value (the SHA-256 hash of any private input).
// No ZK circuit or Groth16 proof is required or accepted.
//
// Flow:
//  1. Parse depositor address and coin amount.
//  2. Escrow plaintext coins: user bank account → privacy module account.
//  3. Validate commitment bytes structurally (must be 32 bytes, non-zero).
//  4. Store commitment on-chain (32 bytes).
//  5. Emit deposit event (denom visible, amount hidden in commitment).
package keeper

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"zytherion/x/privacy/types"
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

	// ── 3. Escrow plaintext coins → module account ──────────────────────────
	if err := ms.bankKeeper.SendCoinsFromAccountToModule(
		ctx,
		creatorAddr,
		types.ModuleName,
		sdk.NewCoins(coin),
	); err != nil {
		return nil, fmt.Errorf("%w: bank transfer failed — %s",
			types.ErrInsufficientBalance, err)
	}

	// ── 4. Validate commitment bytes (stateless) ───────────────────────────────
	// In v0.3: commitment is a raw 32-byte value (typically SHA-256 of a secret).
	// ZK proof verification is removed; the commitment is validated structurally only.
	if err := validateCommitmentBytes(msg.Commitment); err != nil {
		return nil, fmt.Errorf("%w: %s", types.ErrInvalidCommitment, err)
	}

	// ── 5. Store commitment ───────────────────────────────────────────────────
	if err := ms.SetCommitment(ctx, creatorAddr, msg.Commitment); err != nil {
		return nil, fmt.Errorf("failed to store commitment: %w", err)
	}

	// ── 6. Emit event ─────────────────────────────────────────────────────────
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
		"commitment_prefix", fmt.Sprintf("%x", msg.Commitment[:4]),
	)

	return &types.MsgInitCommitmentResponse{}, nil
}

// validateCommitmentBytes checks that a commitment is structurally valid.
//
// Rules:
//   - Must be exactly 32 bytes (SHA-256 size).
//   - Must not be the all-zero value (which would indicate a default/unset value).
func validateCommitmentBytes(commitment []byte) error {
	if len(commitment) != sha256.Size {
		return fmt.Errorf("commitment must be %d bytes, got %d", sha256.Size, len(commitment))
	}
	// Check for all-zero commitment (invalid — indicates an uninitialized value)
	allZero := true
	for _, b := range commitment {
		if b != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		return errors.New("commitment must not be the zero value")
	}
	return nil
}
