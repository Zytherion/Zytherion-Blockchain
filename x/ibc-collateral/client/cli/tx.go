package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"zytherion/x/ibc-collateral/types"
)

// GetTxCmd returns the root transaction command for the ibccollateral module.
func GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      fmt.Sprintf("%s transactions subcommands", types.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(CmdLockCollateral())
	cmd.AddCommand(CmdUnlockCollateral())

	return cmd
}

// ─── CmdLockCollateral ─────────────────────────────────────────────────────────

// CmdLockCollateral returns the CLI command to lock IBC collateral into the vault.
func CmdLockCollateral() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lock-collateral [ibc-denom] [amount]",
		Short: "Lock IBC collateral into the ibccollateral vault",
		Long: `Lock a specified amount of an IBC-bridged asset into the collateral vault.

The IBC denom must appear in the module whitelist (see: query assets).
The amount is the integer coin amount (no denom suffix — the ibc-denom argument is used).

Example:
  $ zytherion tx ibccollateral lock-collateral ibc/ATOM 1000000`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			ibcDenom := args[0]

			rawAmount, ok := sdk.NewIntFromString(args[1])
			if !ok {
				return fmt.Errorf("invalid amount: %q is not a valid integer", args[1])
			}
			if !rawAmount.IsPositive() {
				return fmt.Errorf("amount must be positive, got %s", rawAmount)
			}

			owner := clientCtx.GetFromAddress()
			msg := types.NewMsgLockCollateral(owner.String(), ibcDenom, rawAmount)

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			// Encode message to JSON bytes for the tx builder.
			msgBz, err := json.Marshal(msg)
			if err != nil {
				return fmt.Errorf("failed to marshal message: %w", err)
			}

			// Use the tx factory for broadcast.
			txf, err := tx.NewFactoryCLI(clientCtx, cmd.Flags())
			if err != nil {
				return err
			}
			_ = txf
			_ = msgBz

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// ─── CmdUnlockCollateral ───────────────────────────────────────────────────────

// CmdUnlockCollateral returns the CLI command to unlock IBC collateral from the vault.
func CmdUnlockCollateral() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unlock-collateral [ibc-denom] [amount]",
		Short: "Unlock IBC collateral from the ibccollateral vault",
		Long: `Unlock a specified amount of previously locked IBC collateral.

The unlock will be rejected if you have any outstanding ZYTD debt against
the specified collateral position.

Example:
  $ zytherion tx ibccollateral unlock-collateral ibc/ATOM 500000`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			ibcDenom := args[0]

			rawAmount, ok := sdk.NewIntFromString(args[1])
			if !ok {
				return fmt.Errorf("invalid amount: %q is not a valid integer", args[1])
			}
			if !rawAmount.IsPositive() {
				return fmt.Errorf("amount must be positive, got %s", rawAmount)
			}

			owner := clientCtx.GetFromAddress()
			msg := types.NewMsgUnlockCollateral(owner.String(), ibcDenom, rawAmount)

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
