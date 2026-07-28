// tx_encrypted_transfer.go — TFHE Submit CLI command (v0.3).
//
// In v0.3, ZK-proven transfers (zk-transfer) have been REPLACED by
// TFHE ciphertext submission (tfhe-submit). This file provides:
//
//   - CmdTFHESubmit: submit a TFHE ciphertext (~21 KB) to the network
//   - CmdZKTransfer: deprecated stub that returns a clear error message
//   - CmdEncryptedTransfer: deprecated stub (was pre-v0.2 name)
//
// Usage:
//
//	zytheriond tx privacy tfhe-submit --ciphertext <file.bin> --from <key>
package cli

import (
	"fmt"
	"os"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"

	"zytherion/x/privacy/types"
)

const flagProofFile = "proof"
const flagCiphertextFile = "ciphertext"

// CmdTFHESubmit returns a CLI command to submit a TFHE ciphertext to the network.
//
// The ciphertext must be a serialised FheUint32 (~16-21 KB) produced by
// the TFHE Go engine (EncryptUint32) or an external tfhe-rs client.
//
// The network will:
//  1. Validate the ciphertext size (1 KB – 32 KB).
//  2. Compute SHA-256 commitment hash.
//  3. Erasure-code into 16 Reed-Solomon shards.
//  4. Distribute shards to peer nodes (ReplicationFactor=4).
//  5. Store shard metadata on-chain.
func CmdTFHESubmit() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tfhe-submit --ciphertext <file.bin>",
		Short: "Submit a TFHE ciphertext to the network for distributed storage (v0.4)",
		Long: `Submit a TFHE FheUint32 ciphertext to the Zytherion network.

TFHE is always active — no --enable-tfhe flag required.

Steps:

  1. Encrypt a value using the TFHE Go engine:
     (See x/privacy/tfhe/engine.go for the Go API)

  2. Save the ciphertext bytes to a file:
     e.g., ciphertext.bin (~16-21 KB)

  3. Submit:
     zytheriond tx privacy tfhe-submit \
       --ciphertext ciphertext.bin \
       --from alice \
       --chain-id zytherion

The commitment hash (SHA-256 of ciphertext) will be returned in the response.
Use it with:
  zytheriond query privacy tfhe-result --commitment <hex32>`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			// ── 1. Load ciphertext file ────────────────────────────────────────
			ctFile, _ := cmd.Flags().GetString(flagCiphertextFile)
			if ctFile == "" {
				return fmt.Errorf("--ciphertext flag is required (path to FheUint32 ciphertext .bin file)")
			}

			ciphertext, err := os.ReadFile(ctFile)
			if err != nil {
				return fmt.Errorf("failed to read ciphertext file %q: %w", ctFile, err)
			}

			if len(ciphertext) < 1024 {
				return fmt.Errorf("ciphertext too small (%d bytes); minimum 1 KB", len(ciphertext))
			}
			if len(ciphertext) > 32*1024 {
				return fmt.Errorf("ciphertext too large (%d bytes); maximum 32 KB", len(ciphertext))
			}

			// ── 3. Build and broadcast message ────────────────────────────────
			sender := clientCtx.GetFromAddress().String()
			msg := types.NewMsgTFHESubmit(sender, ciphertext)
			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			fmt.Printf("Submitting TFHE ciphertext: size=%d bytes...\n", len(ciphertext))
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(flagCiphertextFile, "", "Path to TFHE FheUint32 ciphertext binary file (~16-21 KB)")
	_ = cmd.MarkFlagRequired(flagCiphertextFile)
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

// CmdZKTransfer is a deprecated stub. ZK transfers were removed in v0.3.
// The new privacy mechanism is TFHE via tx privacy tfhe-submit.
func CmdZKTransfer() *cobra.Command {
	return &cobra.Command{
		Use:        "zk-transfer",
		Deprecated: "ZK transfers removed in v0.3. Use 'tfhe-submit --ciphertext <file>' instead.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf(
				"zk-transfer was removed in Zytherion v0.3 (ZK-SNARK subsystem deleted).\n" +
					"Use TFHE instead:\n" +
					"  zytheriond tx privacy tfhe-submit --ciphertext <ciphertext.bin> --from <key>")
		},
	}
}

// CmdEncryptedTransfer is a deprecated stub (pre-v0.2 name).
func CmdEncryptedTransfer() *cobra.Command {
	return &cobra.Command{
		Use:        "encrypted-transfer",
		Deprecated: "Removed in v0.3. Use 'tfhe-submit --ciphertext <file>' instead.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf(
				"encrypted-transfer was removed in Zytherion v0.3.\n" +
					"Use: zytheriond tx privacy tfhe-submit --ciphertext <file.bin> --from <key>")
		},
	}
}
