package types

import "encoding/json"

// GenesisState defines the oracle module genesis state.
type GenesisState struct {
	Params OracleParams `json:"params"`
}

// DefaultGenesis returns the default genesis state for the oracle module.
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params: DefaultOracleParams(),
	}
}

// Validate performs basic genesis state validation.
func (gs GenesisState) Validate() error {
	// Validate whitelisted denoms are non-empty
	if len(gs.Params.WhitelistedDenoms) == 0 {
		return ErrInvalidDenom.Wrapf("genesis must have at least one whitelisted denom")
	}
	return nil
}

// MarshalJSON implements json.Marshaler for GenesisState.
func (gs GenesisState) MarshalJSON() ([]byte, error) {
	type Alias GenesisState
	return json.Marshal(Alias(gs))
}

// UnmarshalJSON implements json.Unmarshaler for GenesisState.
func (gs *GenesisState) UnmarshalJSON(data []byte) error {
	type Alias GenesisState
	aux := &Alias{}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}
	*gs = GenesisState(*aux)
	return nil
}
