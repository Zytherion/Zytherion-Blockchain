package types

import (
	"encoding/json"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ─── Message types ────────────────────────────────────────────────────────────

const (
	TypeMsgLockCollateral   = "lock_collateral"
	TypeMsgUnlockCollateral = "unlock_collateral"
)

// Ensure interface compliance at compile time.
var (
	_ sdk.Msg = &MsgLockCollateral{}
	_ sdk.Msg = &MsgUnlockCollateral{}
)

// ─── MsgLockCollateral ────────────────────────────────────────────────────────

// MsgLockCollateral locks IBC collateral into the vault and optionally mints ZYTD.
type MsgLockCollateral struct {
	Owner    string  `json:"owner"`
	IBCDenom string  `json:"ibc_denom"`
	Amount   sdk.Int `json:"amount"`
}

// NewMsgLockCollateral constructs a MsgLockCollateral message.
func NewMsgLockCollateral(owner string, ibcDenom string, amount sdk.Int) *MsgLockCollateral {
	return &MsgLockCollateral{
		Owner:    owner,
		IBCDenom: ibcDenom,
		Amount:   amount,
	}
}

// Route implements sdk.Msg.
func (msg *MsgLockCollateral) Route() string { return RouterKey }

// Type implements sdk.Msg.
func (msg *MsgLockCollateral) Type() string { return TypeMsgLockCollateral }

// Reset implements proto.Message.
func (msg *MsgLockCollateral) Reset() { *msg = MsgLockCollateral{} }

// String implements proto.Message.
func (msg *MsgLockCollateral) String() string {
	bz, _ := json.Marshal(msg)
	return string(bz)
}

// ProtoMessage implements proto.Message.
func (msg *MsgLockCollateral) ProtoMessage() {}

// GetSigners implements sdk.Msg.
func (msg *MsgLockCollateral) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(msg.Owner)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{addr}
}

// GetSignBytes implements sdk.Msg.
func (msg *MsgLockCollateral) GetSignBytes() []byte {
	bz, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return sdk.MustSortJSON(bz)
}

// ValidateBasic implements sdk.Msg — stateless validation only.
func (msg *MsgLockCollateral) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Owner); err != nil {
		return ErrInvalidAddress.Wrapf("invalid owner address: %s", err)
	}
	if msg.IBCDenom == "" {
		return ErrDenomNotWhitelisted.Wrap("ibc_denom must not be empty")
	}
	if !msg.Amount.IsPositive() {
		return ErrInvalidAmount.Wrap("amount must be positive")
	}
	return nil
}

// ─── MsgLockCollateralResponse ────────────────────────────────────────────────

// MsgLockCollateralResponse is the response type for MsgLockCollateral.
type MsgLockCollateralResponse struct {
	Owner    string  `json:"owner"`
	IBCDenom string  `json:"ibc_denom"`
	Locked   sdk.Int `json:"locked"`
}

// Reset implements proto.Message.
func (msg *MsgLockCollateralResponse) Reset() { *msg = MsgLockCollateralResponse{} }

// String implements proto.Message.
func (msg *MsgLockCollateralResponse) String() string {
	bz, _ := json.Marshal(msg)
	return string(bz)
}

// ProtoMessage implements proto.Message.
func (msg *MsgLockCollateralResponse) ProtoMessage() {}

// ─── MsgUnlockCollateral ──────────────────────────────────────────────────────

// MsgUnlockCollateral releases previously locked collateral back to the owner.
type MsgUnlockCollateral struct {
	Owner    string  `json:"owner"`
	IBCDenom string  `json:"ibc_denom"`
	Amount   sdk.Int `json:"amount"`
}

// NewMsgUnlockCollateral constructs a MsgUnlockCollateral message.
func NewMsgUnlockCollateral(owner string, ibcDenom string, amount sdk.Int) *MsgUnlockCollateral {
	return &MsgUnlockCollateral{
		Owner:    owner,
		IBCDenom: ibcDenom,
		Amount:   amount,
	}
}

// Route implements sdk.Msg.
func (msg *MsgUnlockCollateral) Route() string { return RouterKey }

// Type implements sdk.Msg.
func (msg *MsgUnlockCollateral) Type() string { return TypeMsgUnlockCollateral }

// Reset implements proto.Message.
func (msg *MsgUnlockCollateral) Reset() { *msg = MsgUnlockCollateral{} }

// String implements proto.Message.
func (msg *MsgUnlockCollateral) String() string {
	bz, _ := json.Marshal(msg)
	return string(bz)
}

// ProtoMessage implements proto.Message.
func (msg *MsgUnlockCollateral) ProtoMessage() {}

// GetSigners implements sdk.Msg.
func (msg *MsgUnlockCollateral) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(msg.Owner)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{addr}
}

// GetSignBytes implements sdk.Msg.
func (msg *MsgUnlockCollateral) GetSignBytes() []byte {
	bz, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return sdk.MustSortJSON(bz)
}

// ValidateBasic implements sdk.Msg — stateless validation only.
func (msg *MsgUnlockCollateral) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Owner); err != nil {
		return ErrInvalidAddress.Wrapf("invalid owner address: %s", err)
	}
	if msg.IBCDenom == "" {
		return ErrDenomNotWhitelisted.Wrap("ibc_denom must not be empty")
	}
	if !msg.Amount.IsPositive() {
		return ErrInvalidAmount.Wrap("amount must be positive")
	}
	return nil
}

// ─── MsgUnlockCollateralResponse ─────────────────────────────────────────────

// MsgUnlockCollateralResponse is the response type for MsgUnlockCollateral.
type MsgUnlockCollateralResponse struct {
	Owner     string  `json:"owner"`
	IBCDenom  string  `json:"ibc_denom"`
	Unlocked  sdk.Int `json:"unlocked"`
	Remaining sdk.Int `json:"remaining"`
}

// Reset implements proto.Message.
func (msg *MsgUnlockCollateralResponse) Reset() { *msg = MsgUnlockCollateralResponse{} }

// String implements proto.Message.
func (msg *MsgUnlockCollateralResponse) String() string {
	bz, _ := json.Marshal(msg)
	return string(bz)
}

// ProtoMessage implements proto.Message.
func (msg *MsgUnlockCollateralResponse) ProtoMessage() {}

// ─── MsgServer interface ──────────────────────────────────────────────────────

// MsgServer defines the gRPC-style message server interface for the ibccollateral module.
type MsgServer interface {
	LockCollateral(ctx sdk.Context, msg *MsgLockCollateral) (*MsgLockCollateralResponse, error)
	UnlockCollateral(ctx sdk.Context, msg *MsgUnlockCollateral) (*MsgUnlockCollateralResponse, error)
}
