package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tuneinsight/lattigo/v4/bfv"
	"github.com/tuneinsight/lattigo/v4/rlwe"

	"zytherion/x/privacy/fhe"
	"zytherion/x/privacy/types"
)

// EncryptedTransfer handles MsgEncryptedTransfer.
//
// The transfer flow — entirely on encrypted data — is:
//
//  1. Decode AmountCiphertext → incoming rlwe.Ciphertext (the amount to add).
//  2. Fetch recipient's current encrypted balance from the KVStore.
//     If the recipient has no balance yet, initialise it to Enc(0).
//  3. Homomorphically add the two ciphertexts:
//     newBalance = AddCiphertexts(currentBalance, amount)
//     This is h_{new} = h_{current} + h_{amount} in ℤ_q — NO DECRYPTION.
//  4. Serialise newBalance and store it back under the recipient's key.
//
// At no point during this function is any plaintext amount revealed.
// The validator learns only: a transfer occurred from sender to recipient.
func (ms msgServer) EncryptedTransfer(
	goCtx context.Context,
	msg *types.MsgEncryptedTransfer,
) (*types.MsgEncryptedTransferResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// ── 1. Validate and decode recipient address ─────────────────────────────
	recipientAddr, err := sdk.AccAddressFromBech32(msg.Recipient)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", types.ErrInvalidAddress, err)
	}

	// ── 2. Obtain shared BFV parameters ─────────────────────────────────────
	// We use the same parameter set that was used to produce the ciphertext.
	// NewContext() is cheap after first call (A·s is cached); the returned
	// FHE context is used only for its evaluator — the secret key is NOT used.
	fheCtx, err := fhe.NewContext()
	if err != nil {
		return nil, fmt.Errorf("failed to initialise FHE context: %w", err)
	}

	bfvParams := fheCtx.Params()

	// ── 3. Deserialise the incoming amount ciphertext ────────────────────────
	amountCt := rlwe.NewCiphertext(bfvParams.Parameters, 1, bfvParams.MaxLevel())
	if err := amountCt.UnmarshalBinary(msg.AmountCiphertext); err != nil {
		return nil, fmt.Errorf("%w: %s", types.ErrInvalidCiphertext, err)
	}

	// ── 4. Fetch (or initialise) the recipient's current encrypted balance ───
	var currentCt *rlwe.Ciphertext

	if existingBytes, found := ms.GetEncryptedBalance(ctx, recipientAddr); found {
		// Deserialise the stored ciphertext.
		currentCt = rlwe.NewCiphertext(bfvParams.Parameters, 1, bfvParams.MaxLevel())
		if err := currentCt.UnmarshalBinary(existingBytes); err != nil {
			// Corrupt store entry — log and reset to zero rather than panicking.
			ms.Logger(ctx).Error("corrupt encrypted balance — resetting to zero",
				"recipient", msg.Recipient, "error", err)
			currentCt = nil
		}
	}

	if currentCt == nil {
		// No prior balance (or corrupt state): initialise to Enc(0).
		// We encrypt the zero plaintext using the public key so no secret key
		// is ever used server-side during normal operation.
		zeroSlots := make([]uint64, bfvParams.N())
		zeroPt := bfv.NewPlaintext(bfvParams, bfvParams.MaxLevel())
		bfv.NewEncoder(bfvParams).Encode(zeroSlots, zeroPt)
		currentCt = bfv.NewEncryptor(bfvParams, fheCtx.PublicKey()).EncryptNew(zeroPt)
	}

	// ── 5. Homomorphic addition: newBalance = currentBalance ⊕ amountCt ─────
	// This is the key privacy-preserving step: the validator computes
	//   c_new = c_current + c_amount   (in ciphertext space, mod q)
	// which is equivalent to:
	//   Dec(c_new) = Dec(c_current) + Dec(c_amount)   (mod t)
	// WITHOUT ever learning the plaintext values.
	newBalanceCt, err := fheCtx.AddCiphertexts(currentCt, amountCt)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", types.ErrHomomorphicAdd, err)
	}

	// ── 6. Serialise and persist the updated balance ─────────────────────────
	newBalanceBytes, err := newBalanceCt.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("failed to serialise new encrypted balance: %w", err)
	}

	ms.SetEncryptedBalance(ctx, recipientAddr, newBalanceBytes)

	// ── 7. Emit an event (plaintext metadata only — no amount revealed) ──────
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.TypeMsgEncryptedTransfer,
			sdk.NewAttribute("sender", msg.Sender),
			sdk.NewAttribute("recipient", msg.Recipient),
			sdk.NewAttribute("ciphertext_size_bytes", fmt.Sprintf("%d", len(newBalanceBytes))),
		),
	)

	ms.Logger(ctx).Info("encrypted transfer processed",
		"sender", msg.Sender,
		"recipient", msg.Recipient,
		"new_ciphertext_len", len(newBalanceBytes),
	)

	return &types.MsgEncryptedTransferResponse{}, nil
}
