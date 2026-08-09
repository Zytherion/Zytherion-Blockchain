package types

const (
	ModuleName = "oracle"
	StoreKey   = "oracle"
	RouterKey  = "oracle"

	// Key prefixes for KVStore
	PriceKeyPrefix  = "price/"
	TWAPKeyPrefix   = "twap/"
	OracleParamsKey = "oracle_params"

	// Event types
	EventTypeSubmitPrice  = "submit_price"
	AttributeKeyDenom     = "denom"
	AttributeKeyPrice     = "price"
	AttributeKeySubmitter = "submitter"
)
