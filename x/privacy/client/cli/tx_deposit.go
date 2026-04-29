package cli

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"

	"zytherion/x/privacy/types"
)

// CmdDeposit returns a CLI command to deposit plaintext bank tokens into the
// privacy module, receiving an equivalent encrypted balance.
//
// Usage:
//
//	zytheriond tx privacy deposit <amount> [flags]
//
// The <amount> argument is a standard Cosmos coin string, e.g. "1000uzytc".
// The coins are moved from the signer's bank account to the privacy module
// escrow account; the signer's encrypted balance is updated on-chain.
//
// Example:
//
//	zytheriond tx privacy deposit 500uzytc --from alice --chain-id zytherion
func CmdDeposit() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deposit [amount]",
		Short: "Deposit plaintext tokens into the privacy module as an encrypted balance",
		Long: `Deposit plaintext bank tokens into the privacy module.

The tokens are escrowed into the privacy module account, and your on-chain
encrypted balance is updated by homomorphically adding the deposit amount
to your existing ciphertext (or creating a new one if you have no balance yet).

The amount argument must be a valid Cosmos coin string, e.g. "1000uzytc".

Example:
  zytheriond tx privacy deposit 1000uzytc --from alice --chain-id zytheriondevnet`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			amount := args[0]
			creator := clientCtx.GetFromAddress().String()

			msg := types.NewMsgDeposit(creator, amount)
			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
