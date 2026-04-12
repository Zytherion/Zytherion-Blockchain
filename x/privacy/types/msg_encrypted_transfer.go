// Package types defines the message types for the privacy module's
// encrypted transfer transaction.
package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// Ensure MsgEncryptedTransfer satisfies the sdk.Msg interface.
var _ sdk.Msg = &MsgEncryptedTransfer{}

const TypeMsgEncryptedTransfer = "encrypted_transfer"

// MsgEncryptedTransfer is the message a user submits to transfer a
// homomorphically encrypted amount to a recipient.
//
// The Amount field carries a BFV ciphertext (serialised via
// rlwe.Ciphertext.MarshalBinary) that encodes the token amount under the
// chain's public FHE key.  The validator never decrypts the amount; instead it
// fetches the recipient's current encrypted balance and performs a homomorphic
// addition (AddCiphertexts), storing the result back on-chain.
//
// The Sender pays gas as a plaintext account; the privacy of the amount is
// preserved because the ciphertext is opaque to on-chain observers.
type MsgEncryptedTransfer struct {
	// Sender is the bech32 address of the originator (pays gas).
	Sender string `json:"sender"`

	// Recipient is the bech32 address whose encrypted balance is updated.
	Recipient string `json:"recipient"`

	// AmountCiphertext is the binary-encoded BFV ciphertext for the amount
	// to add to the recipient's current encrypted balance.
	// Produced by fhe.Context.Encrypt followed by rlwe.Ciphertext.MarshalBinary.
	AmountCiphertext []byte `json:"amount_ciphertext"`
}

// NewMsgEncryptedTransfer constructs and returns a MsgEncryptedTransfer.
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

// GetSigners implements sdk.Msg — the sender must sign.
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

// ValidateBasic implements sdk.Msg — performs stateless validation.
func (msg *MsgEncryptedTransfer) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return sdkerrors.Wrapf(ErrInvalidAddress, "invalid sender address: %s", err)
	}
	if _, err := sdk.AccAddressFromBech32(msg.Recipient); err != nil {
		return sdkerrors.Wrapf(ErrInvalidAddress, "invalid recipient address: %s", err)
	}
	if len(msg.AmountCiphertext) == 0 {
		return ErrNoCiphertextProvided
	}
	return nil
}

// ProtoMessage is a no-op stub so that MsgEncryptedTransfer satisfies the
// proto.Message interface used by codec.MarshalJSON.
func (msg *MsgEncryptedTransfer) ProtoMessage() {}

// Reset is a no-op stub required by proto.Message.
func (msg *MsgEncryptedTransfer) Reset() {}

// String returns a human-readable summary for logging.
func (msg *MsgEncryptedTransfer) String() string {
	return "MsgEncryptedTransfer{sender=" + msg.Sender + " recipient=" + msg.Recipient + "}"
}
