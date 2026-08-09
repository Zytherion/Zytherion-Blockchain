// msgs.go — sdk.Msg interface implementations for x/ibc-collateral.
//
// Struct definitions (MsgLockCollateral, MsgUnlockCollateral, etc.)
// are auto-generated from proto/zytherion/ibccollateral/tx.proto in tx.pb.go.
package types

import (
	"encoding/json"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	TypeMsgLockCollateral   = "lock_collateral"
	TypeMsgUnlockCollateral = "unlock_collateral"
)

var (
	_ sdk.Msg = &MsgLockCollateral{}
	_ sdk.Msg = &MsgUnlockCollateral{}
)

// ─── MsgLockCollateral ────────────────────────────────────────────────────────

// NewMsgLockCollateral constructs a MsgLockCollateral message.
func NewMsgLockCollateral(owner string, ibcDenom string, amount sdk.Int) *MsgLockCollateral {
	return &MsgLockCollateral{
		Owner:    owner,
		IbcDenom: ibcDenom,
		Amount:   amount,
	}
}

func (msg *MsgLockCollateral) Route() string { return RouterKey }
func (msg *MsgLockCollateral) Type() string  { return TypeMsgLockCollateral }

func (msg *MsgLockCollateral) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(msg.Owner)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{addr}
}

func (msg *MsgLockCollateral) GetSignBytes() []byte {
	bz, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return sdk.MustSortJSON(bz)
}

func (msg *MsgLockCollateral) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Owner); err != nil {
		return ErrInvalidAddress.Wrapf("invalid owner address: %s", err)
	}
	if msg.IbcDenom == "" {
		return ErrDenomNotWhitelisted.Wrap("ibc_denom must not be empty")
	}
	if !msg.Amount.IsPositive() {
		return ErrInvalidAmount.Wrap("amount must be positive")
	}
	return nil
}

// ─── MsgUnlockCollateral ──────────────────────────────────────────────────────

// NewMsgUnlockCollateral constructs a MsgUnlockCollateral message.
func NewMsgUnlockCollateral(owner string, ibcDenom string, amount sdk.Int) *MsgUnlockCollateral {
	return &MsgUnlockCollateral{
		Owner:    owner,
		IbcDenom: ibcDenom,
		Amount:   amount,
	}
}

func (msg *MsgUnlockCollateral) Route() string { return RouterKey }
func (msg *MsgUnlockCollateral) Type() string  { return TypeMsgUnlockCollateral }

func (msg *MsgUnlockCollateral) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(msg.Owner)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{addr}
}

func (msg *MsgUnlockCollateral) GetSignBytes() []byte {
	bz, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return sdk.MustSortJSON(bz)
}

func (msg *MsgUnlockCollateral) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Owner); err != nil {
		return ErrInvalidAddress.Wrapf("invalid owner address: %s", err)
	}
	if msg.IbcDenom == "" {
		return ErrDenomNotWhitelisted.Wrap("ibc_denom must not be empty")
	}
	if !msg.Amount.IsPositive() {
		return ErrInvalidAmount.Wrap("amount must be positive")
	}
	return nil
}
