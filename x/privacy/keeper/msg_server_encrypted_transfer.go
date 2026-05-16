// msg_server_encrypted_transfer.go — Deprecated EncryptedTransfer handler shim.
//
// MsgEncryptedTransfer is retained for proto/codec compatibility.
// This handler always returns an error — use MsgZKTransfer instead.
package keeper

import (
	"context"
	"fmt"

	"zytherion/x/privacy/types"
)

// EncryptedTransfer handles MsgEncryptedTransfer — DEPRECATED.
// Returns ErrInvalidZKProof to prevent use of the old FHE-based flow.
func (ms msgServer) EncryptedTransfer(
	_ context.Context,
	_ *types.MsgEncryptedTransfer,
) (*types.MsgEncryptedTransferResponse, error) {
	return nil, fmt.Errorf("%w: MsgEncryptedTransfer is deprecated — use MsgZKTransfer with a Groth16 proof",
		types.ErrInvalidZKProof)
}
