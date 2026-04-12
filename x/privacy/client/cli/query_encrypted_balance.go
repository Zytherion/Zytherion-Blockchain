package cli

import (
	"encoding/hex"
	"fmt"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"

	"zytherion/x/privacy/types"
)

// CmdShowEncryptedBalance returns a CLI command to query an account's encrypted
// balance from the privacy module KVStore.
//
// Usage:
//
//	zytheriond query privacy show-encrypted-balance <address> [flags]
//
// The command returns the raw ciphertext bytes as a hex string.  Decryption
// must be performed off-chain with the secret FHE key.
func CmdShowEncryptedBalance() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show-encrypted-balance [address]",
		Short: "Query the encrypted balance ciphertext for an address",
		Long: `Query the raw BFV ciphertext stored as the encrypted balance for an address.

The result is printed as a hex-encoded string. To decrypt it, use the FHE
client off-chain with the secret key.

Example:
  zytheriond query privacy show-encrypted-balance cosmos1xyz...`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			addr, err := sdk.AccAddressFromBech32(args[0])
			if err != nil {
				return fmt.Errorf("invalid address: %w", err)
			}

			// Build the raw KVStore key: EncryptedBalanceKeyPrefix | addr_bytes.
			storeKey := types.EncryptedBalanceKey(addr)

			// QueryStore fetches a raw KVStore value via ABCI "/store/<name>/key".
			// No gRPC proto definition is required for this direct store read.
			resBytes, _, err := clientCtx.QueryStore(storeKey, types.StoreKey)
			if err != nil {
				return err
			}

			if len(resBytes) == 0 {
				return fmt.Errorf("no encrypted balance found for %s", args[0])
			}

			cmd.Printf("address: %s\nciphertext_hex: %s\nciphertext_len: %d bytes\n",
				args[0],
				hex.EncodeToString(resBytes),
				len(resBytes),
			)
			return nil
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
