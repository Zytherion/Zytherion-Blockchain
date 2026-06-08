// msg_legacy_stubs.go — Minimal sdk.Msg wrappers for protobuf-generated types.
//
// MsgEncryptedTransfer and MsgDeposit are defined in tx.pb.go (generated).
// This file provides only the sdk.Msg interface methods required by the SDK.
//
// Both message types are no longer accepted by any handler in v0.3.
// Their handlers always return an error directing users to the new TFHE flow.
package types

import (
	"errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// ── MsgEncryptedTransfer (proto-generated in tx.pb.go) ───────────────────────

var _ sdk.Msg = &MsgEncryptedTransfer{}

const TypeMsgEncryptedTransfer = "encrypted_transfer"

func (msg *MsgEncryptedTransfer) Route() string { return RouterKey }
func (msg *MsgEncryptedTransfer) Type() string  { return TypeMsgEncryptedTransfer }

func (msg *MsgEncryptedTransfer) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{addr}
}

func (msg *MsgEncryptedTransfer) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

// ValidateBasic always returns an error — use tfhe-submit instead.
func (msg *MsgEncryptedTransfer) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return sdkerrors.Wrapf(ErrInvalidAddress, "invalid sender: %s", err)
	}
	return errors.New("encrypted_transfer removed in v0.3 — use: zytheriond tx privacy tfhe-submit --ciphertext <file>")
}

// ── MsgDeposit (proto-generated in tx.pb.go) ─────────────────────────────────

var _ sdk.Msg = &MsgDeposit{}

func (msg *MsgDeposit) Route() string { return RouterKey }
func (msg *MsgDeposit) Type() string  { return "deposit" }

func (msg *MsgDeposit) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{addr}
}

func (msg *MsgDeposit) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

// ValidateBasic always returns an error — use init-commitment instead.
func (msg *MsgDeposit) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return sdkerrors.Wrapf(ErrInvalidAddress, "invalid creator: %s", err)
	}
	return errors.New("deposit removed in v0.3 — use: zytheriond tx privacy init-commitment <amount> --commitment <hex>")
}
