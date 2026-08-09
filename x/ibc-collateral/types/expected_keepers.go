package types

import sdk "github.com/cosmos/cosmos-sdk/types"

// ─── Query request / response types ──────────────────────────────────────────

// QueryPositionRequest is the request type for the QueryPosition RPC.
type QueryPositionRequest struct {
	Owner    string `json:"owner"`
	IBCDenom string `json:"ibc_denom"`
}

// QueryPositionResponse is the response type for the QueryPosition RPC.
type QueryPositionResponse struct {
	Position CollateralPosition `json:"position"`
}

// QueryAssetsRequest is the request type for the QueryAssets RPC.
type QueryAssetsRequest struct{}

// QueryAssetsResponse is the response type for the QueryAssets RPC.
type QueryAssetsResponse struct {
	Assets []CollateralAsset `json:"assets"`
}

// QueryTotalLockedRequest is the request type for the QueryTotalLocked RPC.
type QueryTotalLockedRequest struct {
	IBCDenom string `json:"ibc_denom"`
}

// QueryTotalLockedResponse is the response type for the QueryTotalLocked RPC.
type QueryTotalLockedResponse struct {
	IBCDenom    string  `json:"ibc_denom"`
	TotalLocked sdk.Int `json:"total_locked"`
}

// ─── AccountKeeper interface ──────────────────────────────────────────────────

// AccountKeeper defines the expected account keeper methods required by this module.
type AccountKeeper interface {
	GetModuleAddress(name string) sdk.AccAddress
}

// ─── BankKeeper interface ─────────────────────────────────────────────────────

// BankKeeper defines the expected bank keeper methods required by this module.
type BankKeeper interface {
	// SendCoins transfers coins between accounts.
	SendCoins(ctx sdk.Context, fromAddr sdk.AccAddress, toAddr sdk.AccAddress, amt sdk.Coins) error

	// SendCoinsFromAccountToModule transfers coins from a user account to a named module account.
	SendCoinsFromAccountToModule(ctx sdk.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error

	// SendCoinsFromModuleToAccount transfers coins from a named module account to a user account.
	SendCoinsFromModuleToAccount(ctx sdk.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error

	// MintCoins creates new coins and adds them to the module account balance.
	MintCoins(ctx sdk.Context, moduleName string, amounts sdk.Coins) error

	// BurnCoins destroys coins from a module account balance.
	BurnCoins(ctx sdk.Context, moduleName string, amounts sdk.Coins) error

	// SpendableCoins returns the spendable coin balance of an account.
	SpendableCoins(ctx sdk.Context, addr sdk.AccAddress) sdk.Coins
}
