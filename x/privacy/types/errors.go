// errors.go — Sentinel errors for the x/privacy module.
// DONTCOVER

package types

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// x/privacy module sentinel errors
var (
	ErrSample = sdkerrors.Register(ModuleName, 1100, "sample error")

	// ErrInvalidAddress is returned when a bech32 address cannot be decoded.
	ErrInvalidAddress = sdkerrors.Register(ModuleName, 1101, "invalid address")

	// ErrInvalidZKProof is returned when a ZK proof cannot be deserialized
	// or the proof field is empty in a ZKTransfer/InitCommitment message.
	ErrInvalidZKProof = sdkerrors.Register(ModuleName, 1102, "invalid or missing ZK proof")

	// ErrInvalidCommitment is returned when a commitment fails structural
	// validation (wrong size, out of range, or zero value).
	ErrInvalidCommitment = sdkerrors.Register(ModuleName, 1103, "invalid commitment bytes")

	// ErrProofVerificationFailed is returned when the Groth16 verifier rejects
	// the submitted proof — either the proof is malformed or the witness is wrong.
	ErrProofVerificationFailed = sdkerrors.Register(ModuleName, 1104, "ZK proof verification failed")

	// ErrNoCommitment is returned when an account has no commitment stored
	// on-chain (analogous to the old ErrNoSenderBalance).
	ErrNoCommitment = sdkerrors.Register(ModuleName, 1105, "account has no commitment registered")

	// ErrNullifierAlreadySpent is returned when a nullifier has already been
	// recorded on-chain, indicating a double-spend attempt.
	ErrNullifierAlreadySpent = sdkerrors.Register(ModuleName, 1106, "nullifier already spent — double-spend detected")

	// ErrInvalidDepositAmount is returned when the deposit coin string cannot
	// be parsed or is not a positive amount.
	ErrInvalidDepositAmount = sdkerrors.Register(ModuleName, 1107, "invalid deposit amount")

	// ErrInsufficientBalance is returned when the depositor's bank balance is
	// insufficient to cover the requested deposit amount.
	ErrInsufficientBalance = sdkerrors.Register(ModuleName, 1108, "insufficient balance for deposit")
)
