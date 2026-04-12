package cli

import (
	"encoding/json"
	"fmt"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"

	"zytherion/x/privacy/keeper"
	"zytherion/x/privacy/types"
)

// CmdCarbonSaved returns a CLI command that queries the estimated CO₂ savings
// per Zytherion transaction compared to Bitcoin Proof-of-Work.
//
// Usage:
//
//	zytheriond query privacy carbon-saved [flags]
//
// The command reads the current block height from the chain and uses the
// built-in CCAF/IEA carbon model to compute and display the savings report.
// No KVStore data is required — the calculation is deterministic.
func CmdCarbonSaved() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "carbon-saved",
		Short: "Query estimated CO₂ saved per transaction vs Proof-of-Work",
		Long: `Display the Green Badge: estimated carbon savings per Zytherion transaction
compared to a Bitcoin PoW chain, based on CCAF 2024 energy data and IEA carbon intensity.

The model uses:
  - Bitcoin PoW:   ~900 kWh per transaction (CCAF 2024)
  - Zytherion BFT: ~0.00017 kWh per transaction (Cosmos SDK benchmark)
  - Carbon factor:  475 gCO₂/kWh (IEA 2024 world average)

Resulting in approximately 427 kg CO₂ saved per transaction.

Example:
  zytheriond query privacy carbon-saved
  zytheriond query privacy carbon-saved --output json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			// Fetch latest block height via an ABCI info call.
			// This gives us the chain's current height for cumulative stats.
			var blockHeight int64
			status, err := clientCtx.Client.Status(cmd.Context())
			if err == nil && status != nil {
				blockHeight = status.SyncInfo.LatestBlockHeight
			}
			// If we can't get status, fall back to 0 (still shows per-tx stats).

			// Use the keeper calculation directly (pure-Go, no gRPC needed).
			// We construct a minimal context with just the block height — no
			// KVStore access is needed for the carbon query.
			report := computeCarbonReport(blockHeight)

			outputFormat, _ := cmd.Flags().GetString(flags.FlagOutput)
			if outputFormat == "json" {
				bz, err := json.MarshalIndent(report, "", "  ")
				if err != nil {
					return fmt.Errorf("failed to marshal carbon report: %w", err)
				}
				cmd.Println(string(bz))
				return nil
			}

			// Default human-readable output.
			printCarbonReport(cmd, report)
			return nil
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// computeCarbonReport runs the same model as keeper.CarbonSavedPerTx but
// without requiring a full sdk.Context — it uses the block height directly.
func computeCarbonReport(blockHeight int64) keeper.CarbonReport {
	const (
		powEnergy      = 900.0
		bftEnergy      = 0.00017
		carbonIntensity = 475.0
	)
	savedEnergy := powEnergy - bftEnergy
	savedGrams := savedEnergy * carbonIntensity
	savedKg := savedGrams / 1000.0

	r := keeper.CarbonReport{
		PoWChainName:           "Bitcoin (PoW)",
		PoWEnergyPerTxKWh:      powEnergy,
		BFTEnergyPerTxKWh:      bftEnergy,
		CarbonIntensityGPerKWh: carbonIntensity,
		SavedEnergyPerTxKWh:    savedEnergy,
		SavedGramsCO2PerTx:     savedGrams,
		SavedKgCO2PerTx:        savedKg,
		Note: "Model: CCAF 2024 Bitcoin energy est. vs Cosmos SDK BFT benchmark. " +
			"Carbon intensity: IEA 2024 world average (475 gCO₂/kWh). " +
			"Module: " + types.ModuleName,
	}
	if blockHeight > 0 {
		r.EstimatedTotalTxs = blockHeight
		r.CumulativeSavedKgCO2 = float64(blockHeight) * savedKg
	}
	return r
}

// printCarbonReport writes a human-readable report to the cobra command output.
func printCarbonReport(cmd *cobra.Command, r keeper.CarbonReport) {
	cmd.Println("┌─────────────────────────────────────────────────────┐")
	cmd.Println("│           🌿  Zytherion Green Badge Report           │")
	cmd.Println("├─────────────────────────────────────────────────────┤")
	cmd.Printf("│  Comparison chain  : %-30s │\n", r.PoWChainName)
	cmd.Printf("│  PoW energy/tx     : %-.6f kWh              │\n", r.PoWEnergyPerTxKWh)
	cmd.Printf("│  BFT energy/tx     : %-.8f kWh          │\n", r.BFTEnergyPerTxKWh)
	cmd.Printf("│  Carbon intensity  : %.1f gCO₂/kWh              │\n", r.CarbonIntensityGPerKWh)
	cmd.Println("├─────────────────────────────────────────────────────┤")
	cmd.Printf("│  💚 Saved per tx   : %.2f kg CO₂              │\n", r.SavedKgCO2PerTx)
	cmd.Printf("│  💚 Saved per tx   : %.2f g CO₂              │\n", r.SavedGramsCO2PerTx)
	if r.EstimatedTotalTxs > 0 {
		cmd.Println("├─────────────────────────────────────────────────────┤")
		cmd.Printf("│  Est. total txs    : %-10d                   │\n", r.EstimatedTotalTxs)
		cmd.Printf("│  🌍 Total saved    : %.2f tonne CO₂      │\n", r.CumulativeSavedKgCO2/1000.0)
	}
	cmd.Println("└─────────────────────────────────────────────────────┘")
	cmd.Printf("\nNote: %s\n", r.Note)
}
