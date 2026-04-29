package privacy

import (
	"context"
	"encoding/json"
	"fmt"
	// this line is used by starport scaffolding # 1

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/cobra"

	abci "github.com/cometbft/cometbft/abci/types"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"zytherion/x/privacy/client/cli"
	"zytherion/x/privacy/keeper"
	"zytherion/x/privacy/pqc"
	"zytherion/x/privacy/types"
)

var (
	_ module.AppModule      = AppModule{}
	_ module.AppModuleBasic = AppModuleBasic{}
)

// ----------------------------------------------------------------------------
// AppModuleBasic
// ----------------------------------------------------------------------------

// AppModuleBasic implements the AppModuleBasic interface that defines the independent methods a Cosmos SDK module needs to implement.
type AppModuleBasic struct {
	cdc codec.BinaryCodec
}

func NewAppModuleBasic(cdc codec.BinaryCodec) AppModuleBasic {
	return AppModuleBasic{cdc: cdc}
}

// Name returns the name of the module as a string
func (AppModuleBasic) Name() string {
	return types.ModuleName
}

// RegisterLegacyAminoCodec registers the amino codec for the module, which is used to marshal and unmarshal structs to/from []byte in order to persist them in the module's KVStore
func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	types.RegisterCodec(cdc)
}

// RegisterInterfaces registers a module's interface types and their concrete implementations as proto.Message
func (a AppModuleBasic) RegisterInterfaces(reg cdctypes.InterfaceRegistry) {
	types.RegisterInterfaces(reg)
}

// DefaultGenesis returns a default GenesisState for the module, marshalled to json.RawMessage. The default GenesisState need to be defined by the module developer and is primarily used for testing
func (AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	return cdc.MustMarshalJSON(types.DefaultGenesis())
}

// ValidateGenesis used to validate the GenesisState, given in its json.RawMessage form
func (AppModuleBasic) ValidateGenesis(cdc codec.JSONCodec, config client.TxEncodingConfig, bz json.RawMessage) error {
	var genState types.GenesisState
	if err := cdc.UnmarshalJSON(bz, &genState); err != nil {
		return fmt.Errorf("failed to unmarshal %s genesis state: %w", types.ModuleName, err)
	}
	return genState.Validate()
}

// RegisterGRPCGatewayRoutes registers the gRPC Gateway routes for the module
func (AppModuleBasic) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *runtime.ServeMux) {
	types.RegisterQueryHandlerClient(context.Background(), mux, types.NewQueryClient(clientCtx))
}

// GetTxCmd returns the root Tx command for the module. The subcommands of this root command are used by end-users to generate new transactions containing messages defined in the module
func (a AppModuleBasic) GetTxCmd() *cobra.Command {
	return cli.GetTxCmd()
}

// GetQueryCmd returns the root query command for the module. The subcommands of this root command are used by end-users to generate new queries to the subset of the state defined by the module
func (AppModuleBasic) GetQueryCmd() *cobra.Command {
	return cli.GetQueryCmd(types.StoreKey)
}

// ----------------------------------------------------------------------------
// AppModule
// ----------------------------------------------------------------------------

// AppModule implements the AppModule interface that defines the inter-dependent methods that modules need to implement
type AppModule struct {
	AppModuleBasic

	keeper        keeper.Keeper
	accountKeeper types.AccountKeeper
	bankKeeper    types.BankKeeper
}

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

// RegisterServices registers a gRPC query service to respond to the module-specific gRPC queries
func (am AppModule) RegisterServices(cfg module.Configurator) {
	types.RegisterMsgServer(cfg.MsgServer(), keeper.NewMsgServerImpl(am.keeper))
	types.RegisterQueryServer(cfg.QueryServer(), am.keeper)
}

// RegisterInvariants registers the invariants of the module. If an invariant deviates from its predicted value, the InvariantRegistry triggers appropriate logic (most often the chain will be halted)
func (am AppModule) RegisterInvariants(_ sdk.InvariantRegistry) {}

// InitGenesis performs the module's genesis initialization. It returns no validator updates.
func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, gs json.RawMessage) []abci.ValidatorUpdate {
	var genState types.GenesisState
	// Initialize global index to index in genesis state
	cdc.MustUnmarshalJSON(gs, &genState)

	InitGenesis(ctx, am.keeper, genState)

	return []abci.ValidatorUpdate{}
}

// ExportGenesis returns the module's exported genesis state as raw JSON bytes.
func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	genState := ExportGenesis(ctx, am.keeper)
	return cdc.MustMarshalJSON(genState)
}

// ConsensusVersion is a sequence number for state-breaking change of the module. It should be incremented on each consensus-breaking change introduced by the module. To avoid wrong/empty versions, the initial version should be set to 1
func (AppModule) ConsensusVersion() uint64 { return 1 }

// BeginBlock contains the logic that is automatically triggered at the beginning of each block
func (am AppModule) BeginBlock(_ sdk.Context, _ abci.RequestBeginBlock) {}

// EndBlock is triggered at the end of every block.
// It computes the LWE-SHA3 Hybrid block hash for this block and stores it in
// the privacy module KVStore.  Because the write happens before Commit(), the
// stored hash is included in the multistore root — and therefore in the
// canonical Cosmos AppHash of the NEXT block.
//
// Hash algorithm: LWE-SHA3-Hybrid (Ring-LWE, n=256, q=3329)
// Output: 96-byte fixed-size hash (32-byte seed || 64-byte LWE b-vector)
// Fallback: SHA3-256 (32-byte) if LWE encounters an unexpected error.
//
// Hash input (canonical):
//
//	H_n = LWEHash(CanonicalBlockData_n, H_{n-1})
//
// This fulfils the PQC-secured blockchain structure: each block carries a
// quantum-resistant commitment that chains back to genesis via PrevHash.
func (am AppModule) EndBlock(ctx sdk.Context, req abci.RequestEndBlock) []abci.ValidatorUpdate {
	store := ctx.KVStore(am.keeper.StoreKey())

	// ── 1. Retrieve previous PQC hash ────────────────────────────────────────
	// Zero slice on genesis (height 1 has no prior PQC hash).
	prevPQCHash := store.Get([]byte(types.LatestPQCHashKey))

	// ── 2. Collect raw transaction bytes from the block context ──────────────
	blockHeader := ctx.BlockHeader()
	txData := blockHeader.DataHash // 32-byte Merkle root of the block's txs
	appHashPrev := blockHeader.AppHash

	// Assemble the BlockHashInput for canonical serialisation.
	input := pqc.BlockHashInput{
		Height:       ctx.BlockHeight(),
		PrevHash:     prevPQCHash,
		Transactions: [][]byte{txData},
		AppHash:      appHashPrev,
	}

	// ── 3. Compute LWE-SHA3 Hybrid hash for this block ───────────────────────
	// GenerateLWEBlockHashWithFallback attempts the Ring-LWE construction first
	// and transparently falls back to SHA3-256 on any error.
	hashBytes := pqc.GenerateLWEBlockHashWithFallback(input)

	// Determine which algorithm produced the hash (for logging).
	hashAlgo := pqc.HashAlgorithm // "LWE-SHA3-Hybrid"
	if len(hashBytes) == pqc.HashSize {
		hashAlgo = "SHA3-256-Fallback"
	}

	// ── 4. Persist (latest only) ──────────────────────────────────────────────
	store.Set([]byte(types.LatestPQCHashKey), hashBytes)

	ctx.Logger().Info("PQC block hash computed",
		"height", ctx.BlockHeight(),
		"algorithm", hashAlgo,
		"hash_size_bytes", len(hashBytes),
		"pqc_hash_hex", fmt.Sprintf("%x", hashBytes[:16])+"…",
	)

	return []abci.ValidatorUpdate{}
}
