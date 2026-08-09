package cmd

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	dbm "github.com/cometbft/cometbft-db"
	tmcfg "github.com/cometbft/cometbft/config"
	tmcli "github.com/cometbft/cometbft/libs/cli"
	"github.com/cometbft/cometbft/libs/log"
	tmtypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/config"
	"github.com/cosmos/cosmos-sdk/client/debug"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/keys"
	"github.com/cosmos/cosmos-sdk/client/rpc"
	"github.com/cosmos/cosmos-sdk/server"
	serverconfig "github.com/cosmos/cosmos-sdk/server/config"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	"github.com/cosmos/cosmos-sdk/snapshots"
	snapshottypes "github.com/cosmos/cosmos-sdk/snapshots/types"
	"github.com/cosmos/cosmos-sdk/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authcmd "github.com/cosmos/cosmos-sdk/x/auth/client/cli"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/crisis"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutilcli "github.com/cosmos/cosmos-sdk/x/genutil/client/cli"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	"github.com/spf13/cast"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"

	"encoding/json"

	"zytherion/app"
	appparams "zytherion/app/params"
	dilithium5 "zytherion/crypto/dilithium5"
)


// NewRootCmd creates a new root command for a Cosmos SDK application
func NewRootCmd() (*cobra.Command, appparams.EncodingConfig) {
	encodingConfig := app.MakeEncodingConfig()
	initClientCtx := client.Context{}.
		WithCodec(encodingConfig.Marshaler).
		WithInterfaceRegistry(encodingConfig.InterfaceRegistry).
		WithTxConfig(encodingConfig.TxConfig).
		WithLegacyAmino(encodingConfig.Amino).
		WithInput(os.Stdin).
		WithAccountRetriever(types.AccountRetriever{}).
		WithHomeDir(app.DefaultNodeHome).
		WithKeyringOptions(dilithium5KeyringOption).
		WithViper("")

	rootCmd := &cobra.Command{
		Use:   app.Name + "d",
		Short: "Zytherion Blockchain and Cryptocurrency v0.7 (QuantumBFT + ML-KEM)",
		Long: `Zytherion Blockchain and Cryptocurrency v0.7

Founder: Rayhan Aziel Abbrar
Website: https://github.com/Zytherion

Features:
  - Post-Quantum Signatures:    Dilithium5 (ML-DSA Level 5, ~256-bit PQ security)
  - Post-Quantum Key Exchange:  Kyber1024 (ML-KEM-1024, NIST FIPS 203) NEW v0.7
  - Post-Quantum Hashing:       LWR (Ring-LWR / SHAKE-256)
  - Consensus:                  QuantumBFT (Dilithium5 validator signing) + GreenBFT + PoVL VDF
  - Privacy:                    TFHE Homomorphic Encryption (FheUint32, tfhe-rs) — ALWAYS ACTIVE
  - P2P Transport:              Hybrid Kyber1024 + X25519 SecretConnection NEW v0.7
  - Storage:                    Reed-Solomon Erasure Coding (10+6=16 shards)

TFHE is always active — no additional flags required:
  zytheriond start`,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			// set the default command outputs
			cmd.SetOut(cmd.OutOrStdout())
			cmd.SetErr(cmd.ErrOrStderr())
			var err error
			initClientCtx, err = client.ReadPersistentCommandFlags(initClientCtx, cmd.Flags())
			if err != nil {
				return err
			}
			initClientCtx, err = config.ReadFromClientConfig(initClientCtx)
			if err != nil {
				return err
			}
			initClientCtx = initClientCtx.WithKeyringOptions(dilithium5KeyringOption)

			if err := client.SetCmdClientContextHandler(initClientCtx, cmd); err != nil {
				return err
			}

			customAppTemplate, customAppConfig := initAppConfig()
			customTMConfig := initTendermintConfig()
			return server.InterceptConfigsPreRunHandler(
				cmd, customAppTemplate, customAppConfig, customTMConfig,
			)
		},
	}

	initRootCmd(rootCmd, encodingConfig)
	overwriteFlagDefaults(rootCmd, map[string]string{
		flags.FlagChainID:        strings.ReplaceAll(app.Name, "-", ""),
		flags.FlagKeyringBackend: "test",
		"key-type":               "dilithium5",
	})

	return rootCmd, encodingConfig
}

// initTendermintConfig overrides CometBFT default timeouts for Zytherion.
//
// All timeout values are set to 1s (from the original 5s default) to maximise
// block throughput. GreenBFT's idle-window FHE scheduler is compact enough to
// run within a 1s block interval, so there is no latency penalty.
func initTendermintConfig() *tmcfg.Config {
	cfg := tmcfg.DefaultConfig()

	// ── Consensus timeouts (all 1 second) ─────────────────────────────────
	// TimeoutPropose: how long a validator waits for a block proposal before
	//   moving to the next round. Lower = faster round recovery.
	cfg.Consensus.TimeoutPropose = 1 * time.Second
	// TimeoutProposeDelta: added per round to TimeoutPropose (backoff).
	cfg.Consensus.TimeoutProposeDelta = 500 * time.Millisecond
	// TimeoutPrevote: wait for +2/3 prevotes before moving to precommit.
	cfg.Consensus.TimeoutPrevote = 1 * time.Second
	cfg.Consensus.TimeoutPrevoteDelta = 500 * time.Millisecond
	// TimeoutPrecommit: wait for +2/3 precommits before committing block.
	cfg.Consensus.TimeoutPrecommit = 1 * time.Second
	cfg.Consensus.TimeoutPrecommitDelta = 500 * time.Millisecond
	// TimeoutCommit: delay after committing a block before starting next round.
	//   This is the primary lever for block time. 1s → ~1 block/second.
	cfg.Consensus.TimeoutCommit = 1 * time.Second

	return cfg
}

func initRootCmd(
	rootCmd *cobra.Command,
	encodingConfig appparams.EncodingConfig,
) {
	// Set config
	initSDKConfig()

	gentxModule := app.ModuleBasics[genutiltypes.ModuleName].(genutil.AppModuleBasic)
	rootCmd.AddCommand(
		safeInitCmd(),
		genutilcli.CollectGenTxsCmd(banktypes.GenesisBalancesIterator{}, app.DefaultNodeHome, gentxModule.GenTxValidator),
		genutilcli.MigrateGenesisCmd(),
		genutilcli.GenTxCmd(
			app.ModuleBasics,
			encodingConfig.TxConfig,
			banktypes.GenesisBalancesIterator{},
			app.DefaultNodeHome,
		),
		genutilcli.ValidateGenesisCmd(app.ModuleBasics),
		AddGenesisAccountCmd(app.DefaultNodeHome),
		tmcli.NewCompletionCmd(rootCmd, true),
		debug.Cmd(),
		config.Cmd(),
		// this line is used by starport scaffolding # root/commands
	)

	a := appCreator{
		encodingConfig,
	}

	// add server commands
	server.AddCommands(
		rootCmd,
		app.DefaultNodeHome,
		a.newApp,
		a.appExport,
		addModuleInitFlags,
	)

	// Customize the version command to print Zytherion's specific info
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "version" {
			originalRunE := cmd.RunE
			cmd.RunE = func(subCmd *cobra.Command, args []string) error {
				long, _ := subCmd.Flags().GetBool("long")
				output, _ := subCmd.Flags().GetString("output")
				if output == "json" {
					if originalRunE != nil {
						return originalRunE(subCmd, args)
					}
				}

				subCmd.Println("Zytherion Blockchain and Cryptocurrency")
				subCmd.Println("Version: v0.7.0 (QuantumBFT + ML-KEM)")
				subCmd.Println("Founder: Rayhan Aziel Abbrar")
				subCmd.Println("Signature:   CRYSTALS-Dilithium5 (ML-DSA Level 5)")
				subCmd.Println("Key Exchange: CRYSTALS-Kyber1024 (ML-KEM-1024) [NEW v0.7]")
				subCmd.Println("Hashing:     LWR (Ring-LWR / SHAKE-256)")
				subCmd.Println("Consensus:   QuantumBFT (Dilithium5) + GreenBFT + PoVL VDF")
				subCmd.Println("TFHE:        tfhe-rs (Zama) via CGo (Always ON)")
				subCmd.Println("P2P:         Kyber1024 + X25519 Hybrid KEM [NEW v0.7]")
				subCmd.Println("ZK-SNARK:    REMOVED (v0.3)")

				if long && originalRunE != nil {
					subCmd.Println()
					subCmd.Println("--- Cosmos SDK Build Info ---")
					return originalRunE(subCmd, args)
				}
				return nil
			}
		}
	}

	// add keybase, auxiliary RPC, query, and tx child commands
	// Dilithium5 + Kyber1024 are registered as supported algorithms via keyring options.
	keysCmd := keys.Commands(app.DefaultNodeHome)
	keysCmd.AddCommand(TFHEKeysCmd())
	keysCmd.AddCommand(KyberCmd())

	rootCmd.AddCommand(
		rpc.StatusCommand(),
		queryCommand(),
		txCommand(),
		keysCmd,
	)
}

// queryCommand returns the sub-command to send queries to the app
func queryCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        "query",
		Aliases:                    []string{"q"},
		Short:                      "Querying subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		authcmd.GetAccountCmd(),
		rpc.ValidatorCommand(),
		rpc.BlockCommand(),
		authcmd.QueryTxsByEventsCmd(),
		authcmd.QueryTxCmd(),
	)

	app.ModuleBasics.AddQueryCommands(cmd)
	cmd.PersistentFlags().String(flags.FlagChainID, "", "The network chain ID")

	return cmd
}

// txCommand returns the sub-command to send transactions to the app
func txCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        "tx",
		Short:                      "Transactions subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		authcmd.GetSignCommand(),
		authcmd.GetSignBatchCommand(),
		authcmd.GetMultiSignCommand(),
		authcmd.GetValidateSignaturesCommand(),
		flags.LineBreak,
		authcmd.GetBroadcastCommand(),
		authcmd.GetEncodeCommand(),
		authcmd.GetDecodeCommand(),
	)

	app.ModuleBasics.AddTxCommands(cmd)
	cmd.PersistentFlags().String(flags.FlagChainID, "", "The network chain ID")

	return cmd
}

func addModuleInitFlags(startCmd *cobra.Command) {
	crisis.AddModuleInitFlags(startCmd)

	// this line is used by starport scaffolding # root/arguments
}

func overwriteFlagDefaults(c *cobra.Command, defaults map[string]string) {
	set := func(s *pflag.FlagSet, key, val string) {
		if f := s.Lookup(key); f != nil {
			f.DefValue = val
			f.Value.Set(val)
		}
	}
	for key, val := range defaults {
		set(c.Flags(), key, val)
		set(c.PersistentFlags(), key, val)
	}
	for _, c := range c.Commands() {
		overwriteFlagDefaults(c, defaults)
	}
}

type appCreator struct {
	encodingConfig appparams.EncodingConfig
}

// newApp creates a new Cosmos SDK app
func (a appCreator) newApp(
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	appOpts servertypes.AppOptions,
) servertypes.Application {
	var cache sdk.MultiStorePersistentCache

	if cast.ToBool(appOpts.Get(server.FlagInterBlockCache)) {
		cache = store.NewCommitKVStoreCacheManager()
	}

	skipUpgradeHeights := make(map[int64]bool)
	for _, h := range cast.ToIntSlice(appOpts.Get(server.FlagUnsafeSkipUpgrades)) {
		skipUpgradeHeights[int64(h)] = true
	}

	pruningOpts, err := server.GetPruningOptionsFromFlags(appOpts)
	if err != nil {
		panic(err)
	}

	homeDir := cast.ToString(appOpts.Get(flags.FlagHome))
	chainID := cast.ToString(appOpts.Get(flags.FlagChainID))
	if chainID == "" {
		// fallback to genesis chain-id
		appGenesis, err := tmtypes.GenesisDocFromFile(filepath.Join(homeDir, "config", "genesis.json"))
		if err != nil {
			panic(err)
		}

		chainID = appGenesis.ChainID
	}

	snapshotDir := filepath.Join(cast.ToString(appOpts.Get(flags.FlagHome)), "data", "snapshots")
	snapshotDB, err := dbm.NewDB("metadata", dbm.GoLevelDBBackend, snapshotDir)
	if err != nil {
		panic(err)
	}
	snapshotStore, err := snapshots.NewStore(snapshotDB, snapshotDir)
	if err != nil {
		panic(err)
	}

	snapshotOptions := snapshottypes.NewSnapshotOptions(
		cast.ToUint64(appOpts.Get(server.FlagStateSyncSnapshotInterval)),
		cast.ToUint32(appOpts.Get(server.FlagStateSyncSnapshotKeepRecent)),
	)

	return app.New(
		logger,
		db,
		traceStore,
		true,
		skipUpgradeHeights,
		cast.ToString(appOpts.Get(flags.FlagHome)),
		cast.ToUint(appOpts.Get(server.FlagInvCheckPeriod)),
		a.encodingConfig,
		appOpts,
		baseapp.SetPruning(pruningOpts),
		baseapp.SetMinGasPrices(cast.ToString(appOpts.Get(server.FlagMinGasPrices))),
		baseapp.SetHaltHeight(cast.ToUint64(appOpts.Get(server.FlagHaltHeight))),
		baseapp.SetHaltTime(cast.ToUint64(appOpts.Get(server.FlagHaltTime))),
		baseapp.SetMinRetainBlocks(cast.ToUint64(appOpts.Get(server.FlagMinRetainBlocks))),
		baseapp.SetInterBlockCache(cache),
		baseapp.SetTrace(cast.ToBool(appOpts.Get(server.FlagTrace))),
		baseapp.SetIndexEvents(cast.ToStringSlice(appOpts.Get(server.FlagIndexEvents))),
		baseapp.SetSnapshot(snapshotStore, snapshotOptions),
		baseapp.SetIAVLCacheSize(cast.ToInt(appOpts.Get(server.FlagIAVLCacheSize))),
		baseapp.SetIAVLDisableFastNode(cast.ToBool(appOpts.Get(server.FlagDisableIAVLFastNode))),
		baseapp.SetChainID(chainID),
	)
}

// appExport creates a new simapp (optionally at a given height)
func (a appCreator) appExport(
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	height int64,
	forZeroHeight bool,
	jailAllowedAddrs []string,
	appOpts servertypes.AppOptions,
	modulesToExport []string,
) (servertypes.ExportedApp, error) {
	homePath, ok := appOpts.Get(flags.FlagHome).(string)
	if !ok || homePath == "" {
		return servertypes.ExportedApp{}, errors.New("application home not set")
	}

	app := app.New(
		logger,
		db,
		traceStore,
		height == -1, // -1: no height provided
		map[int64]bool{},
		homePath,
		uint(1),
		a.encodingConfig,
		appOpts,
	)

	if height != -1 {
		if err := app.LoadHeight(height); err != nil {
			return servertypes.ExportedApp{}, err
		}
	}

	return app.ExportAppStateAndValidators(forZeroHeight, jailAllowedAddrs, modulesToExport)
}

// initAppConfig helps to override default appConfig template and configs.
// return "", nil if no custom configuration is required for the application.
func initAppConfig() (string, interface{}) {
	// The following code snippet is just for reference.

	type CustomAppConfig struct {
		serverconfig.Config
	}

	// Optionally allow the chain developer to overwrite the SDK's default
	// server config.
	srvCfg := serverconfig.DefaultConfig()
	// The SDK's default minimum gas price is set to "" (empty value) inside
	// app.toml. If left empty by validators, the node will halt on startup.
	// However, the chain developer can set a default app.toml value for their
	// validators here.
	//
	// In summary:
	// - if you leave srvCfg.MinGasPrices = "", all validators MUST tweak their
	//   own app.toml config,
	// - if you set srvCfg.MinGasPrices non-empty, validators CAN tweak their
	//   own app.toml to override, or use this default value.
	//
	// In simapp, we set the min gas prices to 0.
	srvCfg.MinGasPrices = "0zytc"

	customAppConfig := CustomAppConfig{
		Config: *srvCfg,
	}
	customAppTemplate := serverconfig.DefaultConfigTemplate

	return customAppTemplate, customAppConfig
}

func safeInitCmd() *cobra.Command {
	cmd := genutilcli.InitCmd(app.ModuleBasics, app.DefaultNodeHome)
	originalRunE := cmd.RunE
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		home, _ := cmd.Flags().GetString(flags.FlagHome)
		if home == "" {
			home = app.DefaultNodeHome
		}
		// Clear leftover key/genesis files and database from previous runs to guarantee clean init
		_ = os.Remove(filepath.Join(home, "config", "priv_validator_key.json"))
		_ = os.Remove(filepath.Join(home, "config", "genesis.json"))
		_ = os.RemoveAll(filepath.Join(home, "data"))
		return originalRunE(cmd, args)
	}
	return cmd
}


func displayInitOutput(cmd *cobra.Command, toPrint interface{}) error {
	out, err := json.MarshalIndent(toPrint, "", " ")
	if err != nil {
		return err
	}
	cmd.Println(string(out))
	return nil
}







func dilithium5KeyringOption(options *keyring.Options) {
	options.SupportedAlgos = append(options.SupportedAlgos, dilithium5.Dilithium5Algo)
	options.SupportedAlgosLedger = append(options.SupportedAlgosLedger, dilithium5.Dilithium5Algo)
}
