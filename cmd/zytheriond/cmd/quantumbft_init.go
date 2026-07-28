package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"zytherion/app"
	"zytherion/quantumbft"
)

// QuantumBFTCmd returns the root `quantumbft` subcommand group.
// Usage: zytheriond quantumbft <subcommand>
func QuantumBFTCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "quantumbft",
		Short: "QuantumBFT — post-quantum consensus key management (v0.6)",
		Long: `QuantumBFT replaces CometBFT's Ed25519 consensus signing with
Dilithium5 (ML-DSA-87, NIST Category 5, ~256-bit post-quantum security).

QuantumBFT manages the validator consensus key used to sign:
  - Block proposals
  - Pre-votes  (consensus phase 1)
  - Pre-commits (consensus phase 2)

Key sizes (Dilithium5 vs Ed25519):
  Public key:  2592 bytes  (vs Ed25519: 32 bytes)
  Private key: 4864 bytes  (vs Ed25519: 64 bytes)
  Signature:   4595 bytes  (vs Ed25519: 64 bytes)

Algorithm: Dilithium5 = ML-DSA-87 (FIPS 204), NIST Category 5`,
	}

	cmd.AddCommand(
		quantumBFTInitCmd(),
		quantumBFTShowCmd(),
		quantumBFTValidateCmd(),
	)

	return cmd
}

// quantumBFTInitCmd implements `zytheriond quantumbft init`.
func quantumBFTInitCmd() *cobra.Command {
	var (
		flagKeyFile   string
		flagOverwrite bool
	)

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Generate a new QuantumBFT (Dilithium5) validator consensus key",
		Long: `Generate a new Dilithium5 validator key for QuantumBFT consensus signing.

The key is written to:
  <home>/config/quantum_validator_key.json   (private — chmod 600)
  <home>/data/quantum_validator_state.json   (double-sign protection)

WARNING: If a key file already exists, use --overwrite to replace it.
         Replacing an active validator key will cause it to miss blocks
         until the new key is registered in the validator set.`,

		RunE: func(cmd *cobra.Command, args []string) error {
			home, _ := cmd.Flags().GetString("home")
			if home == "" {
				home = app.DefaultNodeHome
			}

			keyPath := flagKeyFile
			if keyPath == "" {
				keyPath = filepath.Join(home, "config", quantumbft.DefaultKeyFileName)
			}

			// Handle existing key file.
			if quantumbft.IsQuantumKeyFile(keyPath) && !flagOverwrite {
				return fmt.Errorf(
					"quantum validator key already exists at %s\n"+
						"  Use --overwrite to regenerate (WARNING: this will break consensus\n"+
						"  until the new pubkey is registered in the validator set)", keyPath)
			}

			cmd.Printf("Generating QuantumBFT (Dilithium5) validator key...\n")
			cmd.Printf("  Algorithm:   Dilithium5 (ML-DSA-87, NIST Category 5)\n")
			cmd.Printf("  Key file:    %s\n\n", keyPath)

			// Remove existing non-quantum or overwritten key file.
			_ = os.Remove(keyPath)

			pv, err := quantumbft.GenerateValidatorKey(keyPath)
			if err != nil {
				return fmt.Errorf("key generation failed: %w", err)
			}

			// Also update genesis.json if present
			genesisPath := filepath.Join(home, "config", "genesis.json")
			if pubKey, err := pv.GetPubKey(); err == nil {
				_ = app.PatchGenesisWithQuantumPubKey(genesisPath, pubKey, nil)
			}

			info := quantumbft.GetKeyInfo(pv)
			pkB64 := info.PubKeyB64
			if len(pkB64) > 32 {
				pkB64 = pkB64[:32] + "..."
			}

			cmd.Printf("✅ QuantumBFT key generated successfully!\n\n")
			cmd.Printf("  Address:     %s\n", info.Address)
			cmd.Printf("  PubKey:      %s\n", pkB64)
			cmd.Printf("  PubKey size: %d bytes  (Ed25519 was 32 bytes)\n", info.PubKeyLen)
			cmd.Printf("  Sig size:    %d bytes  (Ed25519 was 64 bytes)\n", info.SigLen)
			cmd.Printf("  Algorithm:   %s\n\n", info.Algorithm)
			cmd.Printf("Next steps:\n")
			cmd.Printf("  zytheriond quantumbft validate       # verify key integrity\n")
			cmd.Printf("  zytheriond start                     # node auto-uses quantum key\n")

			return nil
		},
	}

	cmd.Flags().StringVar(&flagKeyFile, "key-file", "",
		"Path to write the key (default: <home>/config/quantum_validator_key.json)")
	cmd.Flags().BoolVar(&flagOverwrite, "overwrite", false,
		"Overwrite existing key file (WARNING: active validators will miss blocks)")

	return cmd
}

// quantumBFTShowCmd implements `zytheriond quantumbft show`.
func quantumBFTShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Display QuantumBFT validator key information",
		RunE: func(cmd *cobra.Command, args []string) error {
			home, _ := cmd.Flags().GetString("home")
			if home == "" {
				home = app.DefaultNodeHome
			}

			keyPath := filepath.Join(home, "config", quantumbft.DefaultKeyFileName)
			pv, err := quantumbft.LoadValidatorKey(keyPath)
			if err != nil {
				return fmt.Errorf("%w\n\nRun 'zytheriond quantumbft init' to generate a key", err)
			}

			info := quantumbft.GetKeyInfo(pv)
			cmd.Printf("QuantumBFT Validator Key\n")
			cmd.Printf("  Address:     %s\n", info.Address)
			cmd.Printf("  PubKey(b64): %s\n", info.PubKeyB64)
			cmd.Printf("  PubKey size: %d bytes\n", info.PubKeyLen)
			cmd.Printf("  Algorithm:   %s\n", info.Algorithm)
			cmd.Printf("  Key file:    %s\n", pv.KeyFilePath())
			cmd.Printf("  State file:  %s\n", pv.StateFilePath())
			return nil
		},
	}
}

// quantumBFTValidateCmd implements `zytheriond quantumbft validate`.
func quantumBFTValidateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate the QuantumBFT key file (checks key integrity)",
		RunE: func(cmd *cobra.Command, args []string) error {
			home, _ := cmd.Flags().GetString("home")
			if home == "" {
				home = app.DefaultNodeHome
			}

			keyPath := filepath.Join(home, "config", quantumbft.DefaultKeyFileName)
			cmd.Printf("Validating QuantumBFT key: %s\n", keyPath)
			if err := quantumbft.ValidateKeyFile(keyPath); err != nil {
				return fmt.Errorf("❌ Key validation FAILED: %w", err)
			}
			cmd.Printf("✅ Key valid — Dilithium5 pubkey matches private key derivation\n")
			return nil
		},
	}
}
