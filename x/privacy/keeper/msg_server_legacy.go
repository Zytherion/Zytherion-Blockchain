// msg_server_legacy.go — Disabled handlers for legacy proto-generated messages.
//
// MsgServer requires EncryptedTransfer and Deposit handlers since they are
// registered in the protobuf service descriptor (tx.pb.go).
// Both always return clear errors directing users to the v0.3 API.
package keeper

import (
	"context"
	"errors"

	"zytherion/x/privacy/types"
)

// EncryptedTransfer handles MsgEncryptedTransfer — disabled in v0.3.
func (ms msgServer) EncryptedTransfer(
	_ context.Context,
	_ *types.MsgEncryptedTransfer,
) (*types.MsgEncryptedTransferResponse, error) {
	return nil, errors.New(
		"encrypted_transfer is disabled in Zytherion v0.3 — " +
			"use: zytheriond tx privacy tfhe-submit --ciphertext <file> --from <key>",
	)
}

// Deposit handles MsgDeposit — disabled in v0.3.
func (ms msgServer) Deposit(
	_ context.Context,
	_ *types.MsgDeposit,
) (*types.MsgDepositResponse, error) {
	return nil, errors.New(
		"deposit is disabled in Zytherion v0.3 — " +
			"use: zytheriond tx privacy init-commitment <amount> --commitment <hex32> --from <key>",
	)
}
