package cli

import (
	"os"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"

	"zytherion/x/privacy/types"
)

// CmdEncryptedTransfer returns a CLI command to submit an encrypted transfer.
//
// Usage:
//
//	zytheriond tx privacy encrypted-transfer <recipient> <ciphertext-file> [flags]
//
// The ciphertext file must contain the raw binary output of
// rlwe.Ciphertext.MarshalBinary (produced off-chain by the FHE client).
func CmdEncryptedTransfer() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "encrypted-transfer [recipient] [ciphertext-file]",
		Short: "Transfer a homomorphically encrypted amount to a recipient",
		Long: `Submit an encrypted transfer message.

The ciphertext file must contain the raw binary BFV ciphertext for the amount
(produced off-chain via the FHE client using fhe.Context.Encrypt followed by
rlwe.Ciphertext.MarshalBinary).

Example:
  zytheriond tx privacy encrypted-transfer cosmos1xyz... /path/to/amount.ct --from alice`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			recipient := args[0]
			ciphertextPath := args[1]

			ciphertextBytes, err := os.ReadFile(ciphertextPath)
			if err != nil {
				return err
			}

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
