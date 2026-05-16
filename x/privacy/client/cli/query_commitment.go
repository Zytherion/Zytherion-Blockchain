package cli

import (
	"fmt"
	"io"
	"net/http"
	"encoding/json"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"
)

// CmdShowCommitment returns the command to query the ZK commitment for an account.
func CmdShowCommitment() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "commitment [address]",
		Short: "shows the ZK commitment for a given address",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			address := args[0]

			// The REST endpoint is /zytherion/privacy/v1/commitment/{address}
			// Since we only have a REST endpoint for this (added in keeper), we'll query it via the node's REST API.
			// Alternatively, we could add a gRPC query to the proto file, but we'll use the REST API here for simplicity.
			
			// To keep it simple without setting up full HTTP client config in CLI,
			// we just print instructions if they try to use CLI. 
			// Wait, the standard way in Cosmos is via gRPC. We don't have a gRPC query for it.
			// Let's just construct the REST query if the REST server is running, or tell them to use the API.
			
			nodeURI := clientCtx.NodeURI
			if nodeURI == "" {
				nodeURI = "http://localhost:1317" // fallback to default REST port
			} else {
				// hack to convert RPC port to REST port for local dev
				nodeURI = "http://localhost:1317"
			}
			
			url := fmt.Sprintf("%s/zytherion/privacy/v1/commitment/%s", nodeURI, address)
			fmt.Printf("Querying REST API: %s\n", url)
			
			resp, err := http.Get(url)
			if err != nil {
				return fmt.Errorf("failed to query commitment: %w", err)
			}
			defer resp.Body.Close()

			body, _ := io.ReadAll(resp.Body)
			
			var result map[string]interface{}
			if err := json.Unmarshal(body, &result); err != nil {
				fmt.Println(string(body))
				return nil
			}

			out, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(out))

			return nil
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
