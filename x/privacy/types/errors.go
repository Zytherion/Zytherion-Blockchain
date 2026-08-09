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

	// ErrInvalidCommitment is returned when a commitment fails structural
	// validation (wrong size, out of range, or zero value).
	ErrInvalidCommitment = sdkerrors.Register(ModuleName, 1103, "invalid commitment bytes")

	// ErrNoCommitment is returned when an account has no commitment stored
	// on-chain.
	ErrNoCommitment = sdkerrors.Register(ModuleName, 1105, "account has no commitment registered")

	// ErrInvalidDepositAmount is returned when the deposit coin string cannot
	// be parsed or is not a positive amount.
	ErrInvalidDepositAmount = sdkerrors.Register(ModuleName, 1107, "invalid deposit amount")

	// ErrInsufficientBalance is returned when the depositor's bank balance is
	// insufficient to cover the requested deposit amount.
	ErrInsufficientBalance = sdkerrors.Register(ModuleName, 1108, "insufficient balance for deposit")

	// ── TFHE errors ──────────────────────────────────────────────────────────
	// Note: ErrTFHEDisabled has been removed in v0.5.3 — TFHE is always active.

	// ErrInvalidCiphertext is returned when a submitted TFHE ciphertext is
	// malformed, the wrong size, or fails commitment validation.
	ErrInvalidCiphertext = sdkerrors.Register(ModuleName, 1201, "invalid TFHE ciphertext")

	// ErrShardOperationFailed is returned when erasure coding, shard
	// distribution, or shard reconstruction fails.
	ErrShardOperationFailed = sdkerrors.Register(ModuleName, 1202, "TFHE shard operation failed")

	// ErrShardReconstructionFailed is returned specifically when insufficient
	// shards are available to reconstruct the original ciphertext.
	ErrShardReconstructionFailed = sdkerrors.Register(ModuleName, 1203, "TFHE ciphertext reconstruction failed: insufficient shards")

	// ErrTFHEKeyNotFound is returned when no TFHE key pair is found for the
	// requested operation.
	ErrTFHEKeyNotFound = sdkerrors.Register(ModuleName, 1204, "TFHE key not found")

	// ErrTFHEQuotaExceeded is returned when an account tries to submit a new
	// TFHE ciphertext while they already have an active commitment stored.
	// Each account may hold at most one active TFHE commitment at a time.
	ErrTFHEQuotaExceeded = sdkerrors.Register(ModuleName, 1205, "TFHE ciphertext quota exceeded: revoke existing commitment first")

	// ErrShardAuthFailed is returned when a shard's Dilithium5 signature or
	// Merkle proof fails validation.
	ErrShardAuthFailed = sdkerrors.Register(ModuleName, 1206, "shard authentication failed: invalid signature or Merkle proof")
)
