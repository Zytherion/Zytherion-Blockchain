package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"zytherion/x/ibc-collateral/types"
)

// GetQueryCmd returns the root query command for the ibccollateral module.
func GetQueryCmd(queryRoute string) *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      fmt.Sprintf("Querying commands for the %s module", types.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(CmdQueryPosition())
	cmd.AddCommand(CmdQueryAssets())
	cmd.AddCommand(CmdQueryTotalLocked())

	return cmd
}

// ─── CmdQueryPosition ─────────────────────────────────────────────────────────

// CmdQueryPosition returns the CLI command to query a collateral position.
func CmdQueryPosition() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "position [owner] [ibc-denom]",
		Short: "Query a collateral position for an owner and IBC denom",
		Long: `Query the collateral position (locked amount, ZYTD debt, lock time) for a given
owner address and IBC denom pair.

Example:
  $ zytherion query ibccollateral position cosmos1abc... ibc/ATOM`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			owner := args[0]
			ibcDenom := args[1]

			// Build the query request.
			req := &types.QueryPositionRequest{
				Owner:    owner,
				IBCDenom: ibcDenom,
			}

			reqBz, err := json.Marshal(req)
			if err != nil {
				return fmt.Errorf("failed to marshal query request: %w", err)
			}

			// Query the chain via ABCI.
			queryRoute := fmt.Sprintf("custom/%s/position", types.StoreKey)
			res, _, err := clientCtx.QueryWithData(queryRoute, reqBz)
			if err != nil {
				return fmt.Errorf("query failed: %w", err)
			}

			return clientCtx.PrintBytes(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// ─── CmdQueryAssets ───────────────────────────────────────────────────────────

// CmdQueryAssets returns the CLI command to list all registered collateral assets.
func CmdQueryAssets() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "assets",
		Short: "List all registered collateral assets",
		Long: `List every IBC-bridged asset registered in the collateral whitelist,
including their minimum collateral ratio and liquidation threshold.

Example:
  $ zytherion query ibccollateral assets`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryRoute := fmt.Sprintf("custom/%s/assets", types.StoreKey)
			res, _, err := clientCtx.QueryWithData(queryRoute, nil)
			if err != nil {
				return fmt.Errorf("query failed: %w", err)
			}

			return clientCtx.PrintBytes(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// ─── CmdQueryTotalLocked ──────────────────────────────────────────────────────

// CmdQueryTotalLocked returns the CLI command to query the total locked amount for a denom.
func CmdQueryTotalLocked() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "total-locked [ibc-denom]",
		Short: "Query the total amount locked for an IBC denom",
		Long: `Query the aggregate locked balance of a specific IBC-bridged collateral
asset across all positions in the vault.

Example:
  $ zytherion query ibccollateral total-locked ibc/ATOM`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			ibcDenom := args[0]

			req := &types.QueryTotalLockedRequest{IBCDenom: ibcDenom}
			reqBz, err := json.Marshal(req)
			if err != nil {
				return fmt.Errorf("failed to marshal query request: %w", err)
			}

			queryRoute := fmt.Sprintf("custom/%s/total-locked", types.StoreKey)
			res, _, err := clientCtx.QueryWithData(queryRoute, reqBz)
			if err != nil {
				return fmt.Errorf("query failed: %w", err)
			}

			return clientCtx.PrintBytes(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
