package types

import errorsmod "cosmossdk.io/errors"

var (
	// ErrInvalidDenom is returned when the submitted denom is not in the whitelist.
	ErrInvalidDenom = errorsmod.Register(ModuleName, 1300, "denom not whitelisted")

	// ErrPriceTooOld is returned when a price submission is beyond MaxPriceAge blocks.
	ErrPriceTooOld = errorsmod.Register(ModuleName, 1301, "price submission too old")

	// ErrInsufficientSubmissions is returned when there are not enough price submissions.
	ErrInsufficientSubmissions = errorsmod.Register(ModuleName, 1302, "not enough price submissions for valid TWAP")

	// ErrUnauthorizedSubmitter is returned when the submitter is not an active bonded validator.
	ErrUnauthorizedSubmitter = errorsmod.Register(ModuleName, 1303, "submitter is not an active validator")
)
