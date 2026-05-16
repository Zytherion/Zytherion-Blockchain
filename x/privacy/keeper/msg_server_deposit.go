// msg_server_deposit.go — Deprecated Deposit handler shim.
//
// MsgDeposit is retained for proto-generated codec compatibility.
// All new deposits MUST use MsgInitCommitment instead.
//
// This handler always returns an error directing the caller to use the ZK flow.
package keeper

import (
	"context"
	"fmt"

	"zytherion/x/privacy/types"
)

// Deposit handles MsgDeposit — DEPRECATED.
// Returns ErrInvalidZKProof to prevent accidental use of the old FHE flow.
func (ms msgServer) Deposit(
	_ context.Context,
	_ *types.MsgDeposit,
) (*types.MsgDepositResponse, error) {
	return nil, fmt.Errorf("%w: MsgDeposit is deprecated — use MsgInitCommitment with a Groth16 proof",
		types.ErrInvalidZKProof)
}