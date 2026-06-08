// tx_deposit.go — InitCommitment CLI command (v0.3: ZK proof removed).
//
// Usage:
//
//	zytheriond tx privacy init-commitment <amount> --commitment <hex32> --from <key>
//
// The commitment is a 32-byte SHA-256 hash of the user's secret value.
// No ZK proof is required in v0.3 — the commitment is stored as-is.
package cli

import (
	"encoding/hex"
	"fmt"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"

	"zytherion/x/privacy/types"
)

const flagCommitmentHex = "commitment"

// CmdInitCommitment returns a CLI command to initialize a privacy commitment.
//
// Deposits plaintext coins into the module escrow and registers a 32-byte
// commitment on-chain. In v0.3, no ZK proof is required — the commitment is
// a SHA-256 hash of any secret value known only to the user.
func CmdInitCommitment() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init-commitment [amount] --commitment <hex32>",
		Short: "Deposit tokens and register a privacy commitment (v0.3: no ZK proof required)",
		Long: `Deposit tokens into the privacy module and register a commitment.

v0.3 change: ZK-SNARK proof is NO LONGER required. The commitment is a
32-byte SHA-256 hash of a secret value you choose.

Step 1: Generate your commitment off-chain:
  # Using bash:
  echo -n "my-secret-blinding-factor" | sha256sum

  # Using Go:
  go run -e 'package main; import ("crypto/sha256"; "fmt"); func main() { h := sha256.Sum256([]byte("secret")); fmt.Printf("%x\n", h[:]) }'

Step 2: Submit the commitment on-chain:
  zytheriond tx privacy init-commitment 1000000uzyt \
    --commitment <64-hex-chars> \
    --from alice \
    --chain-id zytherion

Keep your secret value — it is your key to prove ownership of the commitment.

For TFHE (FHE) privacy operations, use:
  zytheriond tx privacy tfhe-submit --ciphertext <file> --from alice`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			amount := args[0]

			// ── 1. Parse commitment hex ────────────────────────────────────────
			commitmentHex, _ := cmd.Flags().GetString(flagCommitmentHex)
			if commitmentHex == "" {
				return fmt.Errorf("--commitment flag is required (64 hex chars = 32 bytes SHA-256 hash)")
			}
			commitment, err := hex.DecodeString(commitmentHex)
			if err != nil {
				return fmt.Errorf("invalid --commitment hex: %w", err)
			}
			if len(commitment) != 32 {
				return fmt.Errorf("--commitment must decode to exactly 32 bytes, got %d", len(commitment))
			}

			// ── 2. Build and broadcast message ─────────────────────────────────
			creator := clientCtx.GetFromAddress().String()

			msg := types.NewMsgInitCommitment(creator, amount, commitment)
			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			fmt.Printf("Submitting InitCommitment: amount=%s, commitment=%s...\n",
				amount, commitmentHex[:8]+"...")
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(flagCommitmentHex, "", "32-byte commitment as 64 hex chars (SHA-256 of your secret)")
	_ = cmd.MarkFlagRequired(flagCommitmentHex)
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

// CmdDeposit is a deprecated alias directing users to init-commitment.
func CmdDeposit() *cobra.Command {
	return &cobra.Command{
		Use:        "deposit",
		Deprecated: "Use 'init-commitment <amount> --commitment <hex>' instead.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("deposit is deprecated. Use: zytheriond tx privacy init-commitment <amount> --commitment <hex32> --from <key>")
		},
	}
}
