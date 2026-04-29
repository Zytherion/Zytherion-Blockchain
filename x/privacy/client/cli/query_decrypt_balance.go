package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"
)

// CmdDecryptBalance returns a CLI command to decrypt and display a plaintext
// balance by querying the running node's REST endpoint.
//
// Usage:
//
//	zytheriond query privacy decrypt-balance <address> [flags]
//
// The command calls GET /zytherion/privacy/v1/decrypt-balance/{address} on the
// node's REST API (default: http://localhost:1317). The node decrypts the
// on-chain TFHE ciphertext using its in-memory client key and returns JSON.
//
// âš ï¸  PoC / Demo Only â€” the node's TFHE client key is used for decryption.
// In production, only the balance owner should decrypt using an off-chain key.
func CmdDecryptBalance() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "decrypt-balance [address]",
		Short: "[PoC] Decrypt and display the plaintext balance for an address",
		Long: `Decrypt the on-chain TFHE-encrypted balance for the given address.

This command queries the running node's REST API which uses the node's
in-memory TFHE client key to decrypt. For demo/local PoC purposes only.

The node must be running with --api.enable=true (default for ignite chain serve).

Example:
  zytheriond query privacy decrypt-balance zyth1...
  zytheriond query privacy decrypt-balance $(zytheriond keys show alice -a)`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := client.GetClientQueryContext(cmd); err != nil {
				return err
			}

			address := args[0]

			// Determine the node REST API base URL.
			restBase, _ := cmd.Flags().GetString("rest-url")
			if restBase == "" {
				restBase = "http://localhost:1317"
			}

			url := fmt.Sprintf("%s/zytherion/privacy/v1/decrypt-balance/%s", restBase, address)

			resp, err := http.Get(url) //nolint:gosec // PoC: URL is constructed from user arg
			if err != nil {
				return fmt.Errorf("failed to query node REST API at %s: %w\n\nMake sure the node is running (ignite chain serve) and --api.enable=true", url, err)
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to read response: %w", err)
			}

			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("node returned error %d:\n%s", resp.StatusCode, string(body))
			}

			// Pretty-print the JSON.
			var pretty interface{}
			if err := json.Unmarshal(body, &pretty); err != nil {
				// Not JSON â€” print raw.
				cmd.Println(string(body))
				return nil
			}
			bz, _ := json.MarshalIndent(pretty, "", "  ")
			cmd.Println(string(bz))
			return nil
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	cmd.Flags().String("rest-url", "http://localhost:1317", "Node REST API base URL")
	return cmd
}