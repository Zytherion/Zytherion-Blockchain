package cli

import (
	"fmt"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"

	"zytherion/x/privacy/fhe"
	"zytherion/x/privacy/types"
)

// CmdEncryptedTransfer returns a CLI command to submit an encrypted transfer.
//
// Usage:
//
//	zytheriond tx privacy encrypted-transfer <recipient> <amount> --from <key> [flags]
//
// The amount is a raw uint64 plaintext (e.g. token micro-units). The CLI
// encrypts it on-the-fly using a fresh TFHE context before broadcasting.
func CmdEncryptedTransfer() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "encrypted-transfer [recipient] [amount]",
		Short: "Encrypt an amount on-the-fly and transfer it to a recipient",
		Long: `Submit an encrypted transfer message.

The amount is provided as a plain integer (uint64). The CLI creates a local
TFHE context, encrypts the value immediately, and embeds the resulting
ciphertext in the transaction — no pre-computed ciphertext file required.

Example:
  zytheriond tx privacy encrypted-transfer cosmos1xyz... 1000000 --from alice`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			recipient := args[0]

			// ── 1. Parse plaintext amount ─────────────────────────────────────
			amount, err := strconv.ParseUint(args[1], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid amount %q: must be a non-negative integer (uint64): %w", args[1], err)
			}

			// ── 2. Create a local FHE context and encrypt on-the-fly ──────────
			// A fresh context is created here (same TFHE parameters as the keeper)
			// so that the ciphertext is compatible with on-chain homomorphic ops.
			fmt.Println("Initialising TFHE context and encrypting amount (this may take a moment)...")
			fheCtx, err := fhe.NewContext()
			if err != nil {
				return fmt.Errorf("failed to create FHE context: %w", err)
			}

			// Encrypt returns compressed bytes
			ciphertextBytes, err := fheCtx.Encrypt(amount)
			if err != nil {
				return fmt.Errorf("failed to encrypt amount: %w", err)
			}

			fmt.Printf("Amount %d encrypted (%d bytes). Broadcasting transaction...\n", amount, len(ciphertextBytes))

			// ── 3. Build and broadcast the message ────────────────────────────
			sender := clientCtx.GetFromAddress().String()

			msg := types.NewMsgEncryptedTransfer(sender, recipient, ciphertextBytes)
			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
