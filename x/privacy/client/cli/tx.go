package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"zytherion/x/privacy/types"
)

var (
	DefaultRelativePacketTimeoutTimestamp = uint64((time.Duration(10) * time.Minute).Nanoseconds())
)

const (
	flagPacketTimeoutTimestamp = "packet-timeout-timestamp"
	listSeparator              = ","
)

// GetTxCmd returns the transaction commands for the privacy module (v0.3).
func GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      fmt.Sprintf("%s transactions subcommands", types.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(CmdTFHESubmit())       // Submit TFHE ciphertext
	cmd.AddCommand(CmdInitCommitment())   // Register 32-byte commitment
	cmd.AddCommand(CmdDeposit())          // Deprecated alias → init-commitment
	cmd.AddCommand(CmdZKTransfer())       // Removed in v0.3 (returns error)
	cmd.AddCommand(CmdEncryptedTransfer()) // Removed in v0.3 (returns error)
	// this line is used by starport scaffolding # 1

	return cmd
}
