package ibccollateral

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/cobra"

	abci "github.com/cometbft/cometbft/abci/types"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"zytherion/x/ibc-collateral/client/cli"
	"zytherion/x/ibc-collateral/keeper"
	"zytherion/x/ibc-collateral/types"
)

// Compile-time interface checks.
var (
	_ module.AppModule      = AppModule{}
	_ module.AppModuleBasic = AppModuleBasic{}
)

// ─── AppModuleBasic ────────────────────────────────────────────────────────────

// AppModuleBasic defines the independent module methods (no keeper dependency).
type AppModuleBasic struct {
	cdc codec.BinaryCodec
}

// NewAppModuleBasic creates a new AppModuleBasic.
func NewAppModuleBasic(cdc codec.BinaryCodec) AppModuleBasic {
	return AppModuleBasic{cdc: cdc}
}

// Name returns the module name.
func (AppModuleBasic) Name() string { return types.ModuleName }

// RegisterLegacyAminoCodec registers the module's types on the legacy Amino codec.
func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	types.RegisterCodec(cdc)
}

// RegisterInterfaces registers the module's interface types.
func (AppModuleBasic) RegisterInterfaces(reg cdctypes.InterfaceRegistry) {
	types.RegisterInterfaces(reg)
}

// DefaultGenesis returns the module's default genesis state as raw JSON.
func (AppModuleBasic) DefaultGenesis(_ codec.JSONCodec) json.RawMessage {
	gs := types.DefaultGenesis()
	bz, err := json.Marshal(gs)
	if err != nil {
		panic(fmt.Sprintf("ibccollateral: DefaultGenesis marshal error: %v", err))
	}
	return bz
}

// ValidateGenesis validates the module genesis state encoded as raw JSON.
func (AppModuleBasic) ValidateGenesis(_ codec.JSONCodec, _ client.TxEncodingConfig, bz json.RawMessage) error {
	var gs types.GenesisState
	if err := json.Unmarshal(bz, &gs); err != nil {
		return fmt.Errorf("failed to unmarshal %s genesis state: %w", types.ModuleName, err)
	}
	return gs.Validate()
}

// RegisterGRPCGatewayRoutes is a no-op for this module (no proto-generated gateway stubs).
func (AppModuleBasic) RegisterGRPCGatewayRoutes(_ client.Context, _ *runtime.ServeMux) {}

// GetTxCmd returns the root Tx command for the module.
func (AppModuleBasic) GetTxCmd() *cobra.Command {
	return cli.GetTxCmd()
}

// GetQueryCmd returns the root Query command for the module.
func (AppModuleBasic) GetQueryCmd() *cobra.Command {
	return cli.GetQueryCmd(types.StoreKey)
}

// ─── AppModule ─────────────────────────────────────────────────────────────────

// AppModule implements the full AppModule interface.
type AppModule struct {
	AppModuleBasic

	keeper        keeper.Keeper
	accountKeeper types.AccountKeeper
	bankKeeper    types.BankKeeper
}

// NewAppModule creates a new AppModule.
func NewAppModule(
	cdc codec.Codec,
	keeper keeper.Keeper,
	accountKeeper types.AccountKeeper,
	bankKeeper types.BankKeeper,
) AppModule {
	return AppModule{
		AppModuleBasic: NewAppModuleBasic(cdc),
		keeper:         keeper,
		accountKeeper:  accountKeeper,
		bankKeeper:     bankKeeper,
	}
}

// RegisterServices registers the module's services.
// This module uses legacy ABCI querier routing (no proto-generated gRPC stubs).
func (am AppModule) RegisterServices(_ module.Configurator) {}

// RegisterInvariants is a no-op (no invariants defined).
func (am AppModule) RegisterInvariants(_ sdk.InvariantRegistry) {}

// InitGenesis performs module genesis initialisation.
func (am AppModule) InitGenesis(ctx sdk.Context, _ codec.JSONCodec, gs json.RawMessage) []abci.ValidatorUpdate {
	var genState types.GenesisState
	if err := json.Unmarshal(gs, &genState); err != nil {
		panic(fmt.Sprintf("ibccollateral: InitGenesis unmarshal error: %v", err))
	}
	InitGenesis(ctx, am.keeper, genState)
	return []abci.ValidatorUpdate{}
}

// ExportGenesis returns the current module state as raw JSON.
func (am AppModule) ExportGenesis(ctx sdk.Context, _ codec.JSONCodec) json.RawMessage {
	genState := ExportGenesis(ctx, am.keeper)
	bz, err := json.Marshal(genState)
	if err != nil {
		panic(fmt.Sprintf("ibccollateral: ExportGenesis marshal error: %v", err))
	}
	return bz
}

// ConsensusVersion returns the module's consensus version.
func (AppModule) ConsensusVersion() uint64 { return 1 }

// BeginBlock is a no-op for this module.
func (am AppModule) BeginBlock(_ sdk.Context, _ abci.RequestBeginBlock) {}

// EndBlock is a no-op for this module.
func (am AppModule) EndBlock(_ sdk.Context, _ abci.RequestEndBlock) []abci.ValidatorUpdate {
	return []abci.ValidatorUpdate{}
}

// ─── Suppress unused import ────────────────────────────────────────────────────

// ensure "context" is consumed (required for grpc-gateway import compatibility).
var _ = context.Background
