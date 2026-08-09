package cli

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"zytherion/x/oracle/types"
)

// GetQueryCmd returns the root query command for the oracle module.
func GetQueryCmd(queryRoute string) *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      fmt.Sprintf("Querying commands for the %s module", types.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		CmdQueryPrice(),
		CmdQueryTWAP(),
		CmdQueryAllPrices(),
	)

	return cmd
}

// CmdQueryPrice returns the cobra command to query the latest price for a denom.
func CmdQueryPrice() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "price [denom]",
		Short: "Query the latest oracle price for a denom",
		Long: `Query the most recent price entry submitted by any bonded validator for the given denom.

Example:
  $ zytherion query oracle price ZYTC
`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			denom := args[0]
			if denom == "" {
				return fmt.Errorf("denom cannot be empty")
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.QueryPrice(cmd.Context(), &types.QueryPriceRequest{Denom: denom})
			if err != nil {
				return err
			}

			bz, err := json.MarshalIndent(res, "", "  ")
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), string(bz))
			return nil
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// CmdQueryTWAP returns the cobra command to query the TWAP for a denom.
func CmdQueryTWAP() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "twap [denom]",
		Short: "Query the TWAP (time-weighted average price) for a denom",
		Long: `Query the latest computed TWAP for the given denom.
The TWAP is the median of all price submissions within the configured window.

Example:
  $ zytherion query oracle twap ZYTC
`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			denom := args[0]
			if denom == "" {
				return fmt.Errorf("denom cannot be empty")
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.QueryTWAP(cmd.Context(), &types.QueryTWAPRequest{Denom: denom})
			if err != nil {
				return err
			}

			bz, err := json.MarshalIndent(res, "", "  ")
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), string(bz))
			return nil
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// CmdQueryAllPrices returns the cobra command to query all price entries for a denom.
func CmdQueryAllPrices() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "prices [denom] [from-height]",
		Short: "Query all price submissions for a denom from a given block height",
		Long: `Query all price feed submissions for the given denom at or after the specified block height.

Example:
  $ zytherion query oracle prices ZYTC 1000
`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			denom := args[0]
			if denom == "" {
				return fmt.Errorf("denom cannot be empty")
			}

			var fromHeight int64
			if len(args) == 2 {
				fromHeight, err = strconv.ParseInt(args[1], 10, 64)
				if err != nil {
					return fmt.Errorf("invalid from-height %q: %w", args[1], err)
				}
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.QueryAllPrices(cmd.Context(), &types.QueryAllPricesRequest{
				Denom:      denom,
				FromHeight: fromHeight,
			})
			if err != nil {
				return err
			}

			bz, err := json.MarshalIndent(res.Prices, "", "  ")
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), string(bz))
			return nil
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
