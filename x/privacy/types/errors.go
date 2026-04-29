package types

// DONTCOVER

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// x/privacy module sentinel errors
var (
	ErrSample = sdkerrors.Register(ModuleName, 1100, "sample error")

	// ErrInvalidAddress is returned when a bech32 address cannot be decoded.
	ErrInvalidAddress = sdkerrors.Register(ModuleName, 1101, "invalid address")

	// ErrNoCiphertextProvided is returned when an encrypted transfer carries
	// no ciphertext bytes for the transferred amount.
	ErrNoCiphertextProvided = sdkerrors.Register(ModuleName, 1102, "no ciphertext provided")

	// ErrInvalidCiphertext is returned when ciphertext bytes cannot be
	// deserialised into a valid BFV rlwe.Ciphertext.
	ErrInvalidCiphertext = sdkerrors.Register(ModuleName, 1103, "invalid ciphertext bytes")

	// ErrHomomorphicAdd is returned when the BFV homomorphic addition fails
	// (e.g. mismatched ciphertext levels/parameters).
	ErrHomomorphicAdd = sdkerrors.Register(ModuleName, 1104, "homomorphic addition failed")

	// ErrHomomorphicSub is returned when the BFV homomorphic subtraction fails
	// (e.g. mismatched ciphertext levels/parameters when deducting from sender).
	ErrHomomorphicSub = sdkerrors.Register(ModuleName, 1105, "homomorphic subtraction failed")

	// ErrNoSenderBalance is returned when the sender has no encrypted balance
	// stored on-chain and therefore cannot transfer encrypted funds.
	ErrNoSenderBalance = sdkerrors.Register(ModuleName, 1106, "sender has no encrypted balance")

	// ErrInvalidDepositAmount is returned when the deposit coin string cannot
	// be parsed or is not a positive amount.
	ErrInvalidDepositAmount = sdkerrors.Register(ModuleName, 1107, "invalid deposit amount")

	// ErrInsufficientBalance is returned when the depositor's bank balance is
	// insufficient to cover the requested deposit amount.
	ErrInsufficientBalance = sdkerrors.Register(ModuleName, 1108, "insufficient balance for deposit")
)
