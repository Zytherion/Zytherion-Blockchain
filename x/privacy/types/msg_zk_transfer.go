// msg_zk_transfer.go — SDK message type for ZK-proven private transfers.
//
// Replaces MsgEncryptedTransfer. Instead of carrying an FHE ciphertext blob,
// this message carries:
//   - The sender's NEW commitment (after deducting the transferred amount)
//   - The recipient's NEW commitment (after adding the transferred amount)
//   - A Groth16 ZK proof proving the above transition is valid
//   - Public inputs for the verifier
//   - A nullifier to prevent double-spending
package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// Ensure MsgZKTransfer satisfies the sdk.Msg interface.
var _ sdk.Msg = &MsgZKTransfer{}

const TypeMsgZKTransfer = "zk_transfer"



// NewMsgZKTransfer constructs a MsgZKTransfer.
func NewMsgZKTransfer(
	sender, recipient string,
	senderNewCommitment, recipientNewCommitment []byte,
	nullifier, zkProof, publicInputs []byte,
) *MsgZKTransfer {
	return &MsgZKTransfer{
		Sender:                 sender,
		Recipient:              recipient,
		SenderNewCommitment:    senderNewCommitment,
		RecipientNewCommitment: recipientNewCommitment,
		Nullifier:              nullifier,
		ZkProof:                zkProof,
		PublicInputs:           publicInputs,
	}
}

// Route implements sdk.Msg.
func (msg *MsgZKTransfer) Route() string { return RouterKey }

// Type implements sdk.Msg.
func (msg *MsgZKTransfer) Type() string { return TypeMsgZKTransfer }

// GetSigners implements sdk.Msg — the sender must sign.
func (msg *MsgZKTransfer) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{addr}
}

// GetSignBytes implements sdk.Msg.
func (msg *MsgZKTransfer) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

// ValidateBasic implements sdk.Msg — stateless validation only.
func (msg *MsgZKTransfer) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return sdkerrors.Wrapf(ErrInvalidAddress, "invalid sender: %s", err)
	}
	if _, err := sdk.AccAddressFromBech32(msg.Recipient); err != nil {
		return sdkerrors.Wrapf(ErrInvalidAddress, "invalid recipient: %s", err)
	}
	if len(msg.SenderNewCommitment) != 32 {
		return sdkerrors.Wrapf(ErrInvalidCommitment, "sender new commitment must be 32 bytes, got %d", len(msg.SenderNewCommitment))
	}
	if len(msg.RecipientNewCommitment) != 32 {
		return sdkerrors.Wrapf(ErrInvalidCommitment, "recipient new commitment must be 32 bytes, got %d", len(msg.RecipientNewCommitment))
	}
	if len(msg.Nullifier) != 32 {
		return sdkerrors.Wrapf(ErrInvalidCommitment, "nullifier must be 32 bytes, got %d", len(msg.Nullifier))
	}
	if len(msg.ZkProof) == 0 {
		return ErrInvalidZKProof
	}
	if len(msg.PublicInputs) != 64 {
		return sdkerrors.Wrapf(ErrInvalidZKProof, "public inputs must be 64 bytes, got %d", len(msg.PublicInputs))
	}
	return nil
}
