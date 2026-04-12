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
)
