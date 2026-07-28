package stablecoin

import (
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"zytherion/x/stablecoin/keeper"
	"zytherion/x/stablecoin/types"
)

// GenesisState defines the genesis state of x/stablecoin.
type GenesisState struct {
	Params      types.StablecoinParams `json:"params"`
	MintRecords []types.MintRecord     `json:"mint_records"`
	TotalSupply string                 `json:"total_supply"`
}

// DefaultGenesis returns the default genesis state.
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:      types.DefaultStablecoinParams(),
		MintRecords: []types.MintRecord{},
		TotalSupply: "0",
	}
}

// ValidateGenesis validates the genesis state.
func ValidateGenesis(gs GenesisState) error {
	if gs.Params.ZYTDDenom == "" {
		return fmt.Errorf("stablecoin genesis: ZYTDDenom cannot be empty")
	}
	return nil
}

// InitGenesis initializes the stablecoin module from genesis state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, gs GenesisState) {
	k.SetParams(ctx, gs.Params)

	supply := sdk.ZeroInt()
	if gs.TotalSupply != "" {
		var ok bool
		supply, ok = sdk.NewIntFromString(gs.TotalSupply)
		if !ok {
			supply = sdk.ZeroInt()
		}
	}
	k.SetTotalSupply(ctx, supply)

	for _, record := range gs.MintRecords {
		k.SetMintRecord(ctx, record)
	}
}

// ExportGenesis exports the stablecoin module's genesis state.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *GenesisState {
	params := k.GetParams(ctx)
	supply := k.GetTotalSupply(ctx)

	// Collect all mint records via KV iteration
	store := k.GetStore(ctx)
	iterator := sdk.KVStorePrefixIterator(store, []byte(types.MintRecordPrefix))
	defer iterator.Close()

	var records []types.MintRecord
	for ; iterator.Valid(); iterator.Next() {
		var record types.MintRecord
		if err := json.Unmarshal(iterator.Value(), &record); err == nil {
			records = append(records, record)
		}
	}

	return &GenesisState{
		Params:      params,
		MintRecords: records,
		TotalSupply: supply.String(),
	}
}

// ProtoMessage implements proto.Message for GenesisState.
func (g *GenesisState) ProtoMessage()  {}
func (g *GenesisState) Reset()         {}
func (g *GenesisState) String() string { return fmt.Sprintf("%+v", *g) }
