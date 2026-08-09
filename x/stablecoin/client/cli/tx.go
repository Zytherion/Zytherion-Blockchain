package cli

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"

	"zytherion/x/stablecoin/types"
)

// GetTxCmd returns the transaction commands for x/stablecoin.
func GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Stablecoin transaction subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		CmdMintZYTD(),
		CmdBurnZYTD(),
		CmdLiquidate(),
	)

	return cmd
}

// CmdMintZYTD creates the mint-zytd CLI command.
func CmdMintZYTD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mint-zytd",
		Short: "Mint ZYTD by locking IBC collateral",
		Long: `Mint ZYTD stablecoin by locking a whitelisted IBC collateral asset.

Example:
  zytheriond tx stablecoin mint-zytd \
    --collateral-denom uzytc \
    --collateral-amount 2000000000 \
    --zytd-amount 1000000000 \
    --expiration-block-height 100 \
    --from alice --fees 5000zytc --chain-id zytherion -y`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			collateralDenom, _ := cmd.Flags().GetString("collateral-denom")
			collateralAmountStr, _ := cmd.Flags().GetString("collateral-amount")
			zytdAmountStr, _ := cmd.Flags().GetString("zytd-amount")
			expirationBlockHeight, _ := cmd.Flags().GetInt64("expiration-block-height")

			if collateralDenom == "" {
				return fmt.Errorf("--collateral-denom is required")
			}
			if expirationBlockHeight <= 0 {
				return fmt.Errorf("--expiration-block-height must be a positive block number (e.g. current height + 100)")
			}

			collateralAmount, ok := sdk.NewIntFromString(collateralAmountStr)
			if !ok {
				return fmt.Errorf("invalid --collateral-amount: %s", collateralAmountStr)
			}

			zytdAmount, ok := sdk.NewIntFromString(zytdAmountStr)
			if !ok {
				return fmt.Errorf("invalid --zytd-amount: %s", zytdAmountStr)
			}

			msg := types.NewMsgMintZYTD(
				clientCtx.GetFromAddress().String(),
				collateralDenom,
				collateralAmount,
				zytdAmount,
			)
			msg.ExpirationBlockHeight = expirationBlockHeight

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("collateral-denom", "", "IBC denom of collateral asset (e.g. uzytc)")
	cmd.Flags().String("collateral-amount", "0", "Amount of collateral to lock (in base units)")
	cmd.Flags().String("zytd-amount", "0", "Amount of ZYTD to mint (in uzytd)")
	cmd.Flags().Int64("expiration-block-height", 0, "Block height at which this mint tx expires — use current height + 100 (replay protection)")
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}


// CmdBurnZYTD creates the burn-zytd CLI command.
func CmdBurnZYTD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "burn-zytd",
		Short: "Burn ZYTD to reclaim collateral",
		Long: `Burn ZYTD stablecoin to unlock proportional collateral.

Example:
  zytheriond tx stablecoin burn-zytd \
    --collateral-denom ibc/AXLUSDC_CHANNEL_HASH \
    --zytd-amount 900000 \
    --from alice --gas 300000 --chain-id zytherion -y`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			collateralDenom, _ := cmd.Flags().GetString("collateral-denom")
			zytdAmountStr, _ := cmd.Flags().GetString("zytd-amount")

			if collateralDenom == "" {
				return fmt.Errorf("--collateral-denom is required")
			}

			zytdAmount, ok := sdk.NewIntFromString(zytdAmountStr)
			if !ok {
				return fmt.Errorf("invalid --zytd-amount: %s", zytdAmountStr)
			}

			msg := types.NewMsgBurnZYTD(
				clientCtx.GetFromAddress().String(),
				collateralDenom,
				zytdAmount,
			)

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("collateral-denom", "", "IBC denom of collateral asset")
	cmd.Flags().String("zytd-amount", "0", "Amount of ZYTD to burn (in uzytd)")
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

// CmdLiquidate creates the liquidate CLI command.
func CmdLiquidate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "liquidate",
		Short: "Liquidate an undercollateralized position",
		Long: `Liquidate a position that has fallen below its liquidation threshold.

Example:
  zytheriond tx stablecoin liquidate \
    --target <undercollateralized_address> \
    --collateral-denom ibc/AXLUSDC_CHANNEL_HASH \
    --from liquidator --gas 400000 --chain-id zytherion -y`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			target, _ := cmd.Flags().GetString("target")
			collateralDenom, _ := cmd.Flags().GetString("collateral-denom")

			if target == "" {
				return fmt.Errorf("--target is required")
			}
			if collateralDenom == "" {
				return fmt.Errorf("--collateral-denom is required")
			}

			msg := types.NewMsgLiquidate(
				clientCtx.GetFromAddress().String(),
				target,
				collateralDenom,
			)

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("target", "", "Address of the undercollateralized position owner")
	cmd.Flags().String("collateral-denom", "", "IBC denom of collateral asset")
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
