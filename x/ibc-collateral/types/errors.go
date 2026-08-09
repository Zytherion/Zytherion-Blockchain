package types

import errorsmod "cosmossdk.io/errors"

var (
	// ErrDenomNotWhitelisted is returned when a denom is not in the collateral whitelist.
	ErrDenomNotWhitelisted = errorsmod.Register(ModuleName, 1400, "IBC denom not whitelisted as collateral")

	// ErrAssetNotActive is returned when the collateral asset is administratively disabled.
	ErrAssetNotActive = errorsmod.Register(ModuleName, 1401, "collateral asset is not active")

	// ErrInsufficientCollateral is returned when the collateral amount is below the minimum ratio for the requested ZYTD.
	ErrInsufficientCollateral = errorsmod.Register(ModuleName, 1402, "collateral amount below minimum for requested ZYTD")

	// ErrPositionNotFound is returned when no collateral position exists for the given owner/denom pair.
	ErrPositionNotFound = errorsmod.Register(ModuleName, 1403, "collateral position not found")

	// ErrUnlockDenied is returned when an unlock is attempted while there is outstanding ZYTD debt.
	ErrUnlockDenied = errorsmod.Register(ModuleName, 1404, "cannot unlock: outstanding ZYTD debt")

	// ErrInsufficientLockedAmount is returned when the unlock amount exceeds the locked balance.
	ErrInsufficientLockedAmount = errorsmod.Register(ModuleName, 1405, "unlock amount exceeds locked balance")

	// ErrInvalidAddress is returned when a bech32 address is malformed.
	ErrInvalidAddress = errorsmod.Register(ModuleName, 1406, "invalid address")

	// ErrInvalidAmount is returned when the coin amount is invalid (zero or negative).
	ErrInvalidAmount = errorsmod.Register(ModuleName, 1407, "invalid amount: must be positive")
)
