package types

import errorsmod "cosmossdk.io/errors"

// Error codes 1500 series for x/stablecoin.
var (
	ErrCollateralRatioTooLow   = errorsmod.Register(ModuleName, 1500, "collateral ratio below minimum for this asset")
	ErrZeroMintAmount          = errorsmod.Register(ModuleName, 1501, "ZYTD mint amount must be greater than zero")
	ErrInsufficientZYTDBalance = errorsmod.Register(ModuleName, 1502, "insufficient ZYTD balance to burn")
	ErrPositionHealthy         = errorsmod.Register(ModuleName, 1503, "position is healthy, cannot liquidate")
	ErrOraclePriceUnavailable  = errorsmod.Register(ModuleName, 1504, "oracle TWAP price not available for collateral denom")
	ErrExceedsCollateralValue  = errorsmod.Register(ModuleName, 1505, "requested ZYTD exceeds allowed mint amount for collateral")
	ErrMintRecordNotFound      = errorsmod.Register(ModuleName, 1506, "mint record not found for owner and collateral denom")
	ErrExceedsMintedAmount     = errorsmod.Register(ModuleName, 1507, "burn amount exceeds minted ZYTD for this position")

	// PQC key registry errors (1508–1512)
	ErrPQCKeyNotRegistered = errorsmod.Register(ModuleName, 1508, "no Dilithium5 key registered for this address — call MsgRegisterZYTDKey first")
	ErrInvalidPQCSig       = errorsmod.Register(ModuleName, 1509, "invalid Dilithium5 signature on ZYTD message")
	ErrPQCKeyMismatch      = errorsmod.Register(ModuleName, 1510, "provided Dilithium5 public key does not match the registered key for this address")
)
