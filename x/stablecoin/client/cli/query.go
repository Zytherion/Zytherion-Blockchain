package cli

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"

	"zytherion/x/stablecoin/types"
)

// GetQueryCmd returns the query commands for x/stablecoin.
func GetQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Stablecoin query subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		CmdQueryMintRecord(),
		CmdQueryCollateralRatio(),
		CmdQueryTotalSupply(),
		CmdQueryMaxMintable(),
	)

	return cmd
}

// CmdQueryMintRecord queries the mint record for an address and ibc_denom.
func CmdQueryMintRecord() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mint-record [address] [ibc_denom]",
		Short: "Query the mint record for an address and collateral denom",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			_ = clientCtx
			fmt.Printf("Mint record for address=%s ibc_denom=%s\n", args[0], args[1])
			fmt.Printf("(Use the REST or gRPC endpoint for full query)\n")
			return nil
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// CmdQueryCollateralRatio queries the live collateral ratio for a position.
func CmdQueryCollateralRatio() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ratio [address] [ibc_denom]",
		Short: "Query the live collateral ratio for a position",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			_ = clientCtx
			fmt.Printf("Collateral ratio for address=%s ibc_denom=%s\n", args[0], args[1])
			return nil
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// CmdQueryTotalSupply queries the total ZYTD in circulation.
func CmdQueryTotalSupply() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "total-supply",
		Short: "Query total ZYTD stablecoin in circulation",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			_ = clientCtx
			fmt.Printf("Total ZYTD supply query\n")
			return nil
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// CmdQueryMaxMintable simulates the maximum ZYTD mintable for given collateral.
func CmdQueryMaxMintable() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "max-mintable [ibc_denom] [amount]",
		Short: "Simulate max ZYTD mintable for a given collateral amount",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			_ = clientCtx

			ibcDenom := args[0]
			amount, ok := sdk.NewIntFromString(args[1])
			if !ok {
				return fmt.Errorf("invalid amount: %s", args[1])
			}

			fmt.Printf("Max mintable ZYTD for ibc_denom=%s collateral_amount=%s\n", ibcDenom, amount.String())
			return nil
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
