// msg_init_commitment.go — SDK message type for initializing a privacy commitment.
//
// v0.3: ZK proof fields (ZkProof, PublicInputs) removed.
// The commitment is now a plain 32-byte SHA-256 hash of a user secret.
// No trusted setup or ZK circuit is required.
package types

import (
	"encoding/json"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// Ensure MsgInitCommitment satisfies the sdk.Msg interface.
var _ sdk.Msg = &MsgInitCommitment{}

const TypeMsgInitCommitment = "init_commitment"

// NewMsgInitCommitment constructs a MsgInitCommitment.
// commitment must be 32 bytes (SHA-256 of the user's private input).
func NewMsgInitCommitment(creator, amount string, commitment []byte) *MsgInitCommitment {
	return &MsgInitCommitment{
		Creator:    creator,
		Amount:     amount,
		Commitment: commitment,
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
// Uses standard JSON encoding since MsgInitCommitment is not a proto-registered type.
func (msg *MsgInitCommitment) GetSignBytes() []byte {
	bz, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}
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
	return nil
}
