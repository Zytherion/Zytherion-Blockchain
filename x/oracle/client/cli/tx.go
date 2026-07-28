package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"zytherion/x/oracle/types"
)

// GetTxCmd returns the root transaction command for the oracle module.
func GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      fmt.Sprintf("%s transaction subcommands", types.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(CmdSubmitPrice())

	return cmd
}

// CmdSubmitPrice returns the cobra command for submitting a price feed entry.
func CmdSubmitPrice() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "submit-price [denom] [price]",
		Short: "Submit a USD price for a whitelisted denom",
		Long: `Submit a USD price feed entry for a whitelisted denom.
The submitter must be a bonded validator.

Example:
  $ zytherion tx oracle submit-price ZYTC 0.045 --from validator1 --chain-id zytherion-1
`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			denom := args[0]
			if denom == "" {
				return fmt.Errorf("denom cannot be empty")
			}

			price, err := sdk.NewDecFromStr(args[1])
			if err != nil {
				return fmt.Errorf("invalid price %q: %w", args[1], err)
			}
			if !price.IsPositive() {
				return fmt.Errorf("price must be positive, got %s", price)
			}

			submitter := clientCtx.GetFromAddress().String()
			msg := types.NewMsgSubmitPrice(submitter, denom, price)

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
