// msg_init_commitment.go — SDK message type for initializing a privacy commitment.
//
// Replaces the FHE-dependent parts of MsgDeposit.
// The user escrows plaintext coins into the module account and registers
// a ZK-verified commitment on-chain. No plaintext is ever stored.
//
// Flow:
//  1. Bank sends coins: user → privacy module escrow
//  2. Chain verifies ZK proof of commitment validity
//  3. Commitment is stored on-chain (32 bytes)
package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// Ensure MsgInitCommitment satisfies the sdk.Msg interface.
var _ sdk.Msg = &MsgInitCommitment{}

const TypeMsgInitCommitment = "init_commitment"



// NewMsgInitCommitment constructs a MsgInitCommitment.
func NewMsgInitCommitment(creator, amount string, commitment, zkProof, publicInputs []byte) *MsgInitCommitment {
	return &MsgInitCommitment{
		Creator:      creator,
		Amount:       amount,
		Commitment:   commitment,
		ZkProof:      zkProof,
		PublicInputs: publicInputs,
	}
}

// Route implements sdk.Msg.
func (msg *MsgInitCommitment) Route() string { return RouterKey }

// Type implements sdk.Msg.
func (msg *MsgInitCommitment) Type() string { return TypeMsgInitCommitment }

// GetSigners implements sdk.Msg — the creator must sign.
func (msg *MsgInitCommitment) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{addr}
}

// GetSignBytes implements sdk.Msg.
func (msg *MsgInitCommitment) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

// ValidateBasic implements sdk.Msg — stateless validation only.
func (msg *MsgInitCommitment) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return sdkerrors.Wrapf(ErrInvalidAddress, "invalid creator address: %s", err)
	}
	coin, err := sdk.ParseCoinNormalized(msg.Amount)
	if err != nil {
		return sdkerrors.Wrapf(ErrInvalidDepositAmount, "amount %q is not a valid coin: %s", msg.Amount, err)
	}
	if !coin.IsPositive() {
		return sdkerrors.Wrapf(ErrInvalidDepositAmount, "deposit amount must be positive, got %s", msg.Amount)
	}
	if len(msg.Commitment) != 32 {
		return sdkerrors.Wrapf(ErrInvalidCommitment, "commitment must be 32 bytes, got %d", len(msg.Commitment))
	}
	if len(msg.ZkProof) == 0 {
		return ErrInvalidZKProof
	}
	if len(msg.PublicInputs) != 64 {
		return sdkerrors.Wrapf(ErrInvalidZKProof, "public inputs must be 64 bytes, got %d", len(msg.PublicInputs))
	}
	return nil
}
