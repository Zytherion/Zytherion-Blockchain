package types

const (
	ModuleName        = "stablecoin"
	StoreKey          = "stablecoin"
	RouterKey         = "stablecoin"
	ModuleAccountName = "stablecoin_mint"
	ZYTDDenom         = "uzytd"

	MintRecordPrefix      = "mint/"   // mint/<owner_bech32>/<ibcDenom> -> MintRecord JSON
	TotalSupplyKey        = "supply"  // -> sdk.Int JSON
	ParamsKey             = "sc_params"
	ZYTDKeyRegistryPrefix = "pqckey/" // pqckey/<owner_bech32> -> Dilithium5 pubkey bytes (2592 B)

	// Events
	EventTypeMintZYTD    = "mint_zytd"
	EventTypeBurnZYTD    = "burn_zytd"
	EventTypeLiquidation = "liquidation"

	AttributeKeyOwner           = "owner"
	AttributeKeyIBCDenom        = "ibc_denom"
	AttributeKeyCollateralAmount = "collateral_amount"
	AttributeKeyZYTDAmount      = "zytd_amount"
	AttributeKeyCollateralRatio = "collateral_ratio"
	AttributeKeyLiquidator      = "liquidator"
	AttributeKeyTarget          = "target"
)
