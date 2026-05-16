// msg_encrypted_transfer.go — Deprecated: use MsgZKTransfer instead.
//
// This file is retained for proto-generated codec compatibility only.
// The MsgEncryptedTransfer type is defined in tx.pb.go (proto-generated).
// No new handlers exist for this message — use MsgZKTransfer for transfers.
package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// Ensure MsgEncryptedTransfer satisfies the sdk.Msg interface (codec compat).
var _ sdk.Msg = &MsgEncryptedTransfer{}

const TypeMsgEncryptedTransfer = "encrypted_transfer"

// NewMsgEncryptedTransfer is deprecated. Use NewMsgZKTransfer instead.
func NewMsgEncryptedTransfer(sender, recipient string, amountCiphertext []byte) *MsgEncryptedTransfer {
	return &MsgEncryptedTransfer{
		Sender:           sender,
		Recipient:        recipient,
		AmountCiphertext: amountCiphertext,
	}
}

// Route implements sdk.Msg.
func (msg *MsgEncryptedTransfer) Route() string { return RouterKey }

// Type implements sdk.Msg.
func (msg *MsgEncryptedTransfer) Type() string { return TypeMsgEncryptedTransfer }

// GetSigners implements sdk.Msg.
func (msg *MsgEncryptedTransfer) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{addr}
}

// GetSignBytes implements sdk.Msg.
func (msg *MsgEncryptedTransfer) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

// ValidateBasic implements sdk.Msg.
// Deprecated: always returns ErrInvalidZKProof to prevent accidental use.
func (msg *MsgEncryptedTransfer) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return sdkerrors.Wrapf(ErrInvalidAddress, "invalid sender address: %s", err)
	}
	if _, err := sdk.AccAddressFromBech32(msg.Recipient); err != nil {
		return sdkerrors.Wrapf(ErrInvalidAddress, "invalid recipient address: %s", err)
	}
	// Reject: this message type is no longer supported. Use MsgZKTransfer.
	return sdkerrors.Wrap(ErrInvalidZKProof,
		"MsgEncryptedTransfer is deprecated — use MsgZKTransfer with a Groth16 proof")
}
