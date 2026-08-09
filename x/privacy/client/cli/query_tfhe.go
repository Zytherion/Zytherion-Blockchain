package cli

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"

	"zytherion/x/privacy/types"
)

// CmdTFHEResult queries the TFHE ciphertext reconstruction result for a commitment.
func CmdTFHEResult() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tfhe-result",
		Short: "Query and reconstruct TFHE ciphertext result from erasure-coded shards",
		RunE: func(cmd *cobra.Command, args []string) error {
			commitment, err := cmd.Flags().GetString("commitment")
			if err != nil || commitment == "" {
				return fmt.Errorf("--commitment hex string flag is required")
			}

			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			commitBz, err := hex.DecodeString(commitment)
			if err == nil && len(commitBz) == 32 {
				queryClient := types.NewQueryClient(clientCtx)
				res, err := queryClient.TFHEResult(cmd.Context(), &types.QueryTFHEResultRequest{
					CommitmentHash:    commitBz,
					CommitmentHashHex: commitment,
				})
				if err == nil {
					bz, err := json.MarshalIndent(res, "", "  ")
					if err == nil {
						fmt.Fprintln(cmd.OutOrStdout(), string(bz))
						return nil
					}
				}
			}

			// Fallback to REST HTTP endpoint
			nodeURI := "http://localhost:1317"
			url := fmt.Sprintf("%s/zytherion/privacy/v1/tfhe/result/%s", nodeURI, commitment)

			resp, err := http.Get(url)
			if err != nil {
				return fmt.Errorf("failed to query TFHE result: %w", err)
			}
			defer resp.Body.Close()

			body, _ := io.ReadAll(resp.Body)
			var result map[string]interface{}
			if err := json.Unmarshal(body, &result); err != nil {
				cmd.Println(string(body))
				return nil
			}

			out, _ := json.MarshalIndent(result, "", "  ")
			cmd.Println(string(out))
			return nil
		},
	}

	cmd.Flags().String("commitment", "", "32-byte commitment hash (64 hex characters)")
	_ = cmd.MarkFlagRequired("commitment")
	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

// CmdTFHEStatus queries the real-time status of the TFHE subsystem.
func CmdTFHEStatus() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tfhe-status",
		Short: "Query the real-time operational status of the TFHE subsystem",
		RunE: func(cmd *cobra.Command, args []string) error {
			nodeURI := "http://localhost:1317"
			url := fmt.Sprintf("%s/zytherion/privacy/v1/tfhe/status", nodeURI)

			resp, err := http.Get(url)
			if err != nil {
				return fmt.Errorf("failed to query TFHE status: %w", err)
			}
			defer resp.Body.Close()

			body, _ := io.ReadAll(resp.Body)
			var result map[string]interface{}
			if err := json.Unmarshal(body, &result); err != nil {
				cmd.Println(string(body))
				return nil
			}

			out, _ := json.MarshalIndent(result, "", "  ")
			cmd.Println(string(out))
			return nil
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
