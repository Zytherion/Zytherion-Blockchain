package types

const (
	ModuleName        = "ibccollateral"
	StoreKey          = "collateral"   // must NOT be a prefix of "ibc" — Cosmos SDK assertNoPrefix
	RouterKey         = "collateral"
	ModuleAccountName = "ibc_collateral_vault"

	// KV store key prefixes
	CollateralAssetPrefix    = "asset/"
	CollateralPositionPrefix = "position/"
	TotalLockedPrefix        = "locked/"

	// Event types
	EventTypeCollateralReceived = "collateral_received"
	EventTypeLockCollateral     = "lock_collateral"
	EventTypeUnlockCollateral   = "unlock_collateral"

	// Event attribute keys
	AttributeKeyOwner    = "owner"
	AttributeKeyIBCDenom = "ibc_denom"
	AttributeKeyAmount   = "amount"
)
