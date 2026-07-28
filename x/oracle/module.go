package oracle

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

	"zytherion/x/oracle/client/cli"
	"zytherion/x/oracle/keeper"
	"zytherion/x/oracle/types"
)

var (
	_ module.AppModule      = AppModule{}
	_ module.AppModuleBasic = AppModuleBasic{}
)

// ─────────────────────────────────────────────────────────────────────────────
// AppModuleBasic
// ─────────────────────────────────────────────────────────────────────────────

// AppModuleBasic implements the AppModuleBasic interface for the oracle module.
type AppModuleBasic struct {
	cdc codec.BinaryCodec
}

// NewAppModuleBasic creates a new AppModuleBasic.
func NewAppModuleBasic(cdc codec.BinaryCodec) AppModuleBasic {
	return AppModuleBasic{cdc: cdc}
}

// Name returns the oracle module name.
func (AppModuleBasic) Name() string {
	return types.ModuleName
}

// RegisterLegacyAminoCodec registers the legacy amino codec.
func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	types.RegisterCodec(cdc)
}

// RegisterInterfaces registers the module's interface types.
func (a AppModuleBasic) RegisterInterfaces(reg cdctypes.InterfaceRegistry) {
	types.RegisterInterfaces(reg)
}

// DefaultGenesis returns the default genesis state marshalled to JSON.
func (AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	gs := types.DefaultGenesis()
	bz, err := json.Marshal(gs)
	if err != nil {
		panic(fmt.Sprintf("oracle: failed to marshal default genesis: %v", err))
	}
	return bz
}

// ValidateGenesis validates the genesis state.
func (AppModuleBasic) ValidateGenesis(cdc codec.JSONCodec, config client.TxEncodingConfig, bz json.RawMessage) error {
	var genState types.GenesisState
	if err := json.Unmarshal(bz, &genState); err != nil {
		return fmt.Errorf("failed to unmarshal %s genesis state: %w", types.ModuleName, err)
	}
	return genState.Validate()
}

// RegisterGRPCGatewayRoutes registers the gRPC Gateway routes.
// The oracle module does not expose gRPC-gateway HTTP routes at this time.
func (AppModuleBasic) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *runtime.ServeMux) {
	// No gRPC-gateway routes for oracle in this version.
	_ = context.Background()
}

// GetTxCmd returns the root Tx command for the oracle module.
func (a AppModuleBasic) GetTxCmd() *cobra.Command {
	return cli.GetTxCmd()
}

// GetQueryCmd returns the root query command for the oracle module.
func (AppModuleBasic) GetQueryCmd() *cobra.Command {
	return cli.GetQueryCmd(types.StoreKey)
}

// ─────────────────────────────────────────────────────────────────────────────
// AppModule
// ─────────────────────────────────────────────────────────────────────────────

// AppModule implements the AppModule interface for the oracle module.
type AppModule struct {
	AppModuleBasic

	keeper keeper.Keeper
}

// NewAppModule creates a new oracle AppModule.
func NewAppModule(
	cdc codec.Codec,
	keeper keeper.Keeper,
) AppModule {
	return AppModule{
		AppModuleBasic: NewAppModuleBasic(cdc),
		keeper:         keeper,
	}
}

// Name returns the oracle module name.
func (am AppModule) Name() string {
	return types.ModuleName
}

// RegisterServices registers module gRPC services with the module configurator.
// Since we use amino-based message routing (no generated .pb.go), we register
// a lightweight handler adapter instead of full gRPC service descriptors.
func (am AppModule) RegisterServices(cfg module.Configurator) {
	types.RegisterMsgServer(cfg.MsgServer(), keeper.NewMsgServerImpl(am.keeper))
	types.RegisterQueryServer(cfg.QueryServer(), am.keeper)
}

// RegisterInvariants registers module invariants.
func (am AppModule) RegisterInvariants(_ sdk.InvariantRegistry) {}

// InitGenesis performs the oracle module's genesis initialization.
func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, gs json.RawMessage) []abci.ValidatorUpdate {
	var genState types.GenesisState
	if err := json.Unmarshal(gs, &genState); err != nil {
		panic(fmt.Sprintf("oracle: failed to unmarshal genesis state: %v", err))
	}
	InitGenesis(ctx, am.keeper, genState)
	return []abci.ValidatorUpdate{}
}

// ExportGenesis returns the module's exported genesis state as raw JSON bytes.
func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	genState := ExportGenesis(ctx, am.keeper)
	bz, err := json.Marshal(genState)
	if err != nil {
		panic(fmt.Sprintf("oracle: failed to marshal genesis state: %v", err))
	}
	return bz
}

// ConsensusVersion returns the module's consensus version.
func (AppModule) ConsensusVersion() uint64 { return 1 }

// BeginBlock is called at the beginning of every block.
// The oracle module refreshes TWAPs for all whitelisted denoms at each block start.
func (am AppModule) BeginBlock(ctx sdk.Context, _ abci.RequestBeginBlock) {
	params := am.keeper.GetParams(ctx)
	for _, denom := range params.WhitelistedDenoms {
		twap, err := am.keeper.ComputeTWAP(ctx, denom)
		if err != nil {
			// Not enough submissions yet — this is normal early in chain life.
			continue
		}
		am.keeper.SetTWAP(ctx, twap)
	}
}

// EndBlock is called at the end of every block.
// The oracle module prunes stale price entries at each block end.
func (am AppModule) EndBlock(ctx sdk.Context, _ abci.RequestEndBlock) []abci.ValidatorUpdate {
	am.keeper.PruneOldPrices(ctx, ctx.BlockHeight())
	return []abci.ValidatorUpdate{}
}
