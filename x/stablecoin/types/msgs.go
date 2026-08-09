// msgs.go — sdk.Msg interface implementations for x/stablecoin.
//
// Struct definitions (MsgMintZYTD, MsgBurnZYTD, MsgLiquidate, MsgRegisterZYTDKey)
// are auto-generated from proto/zytherion/stablecoin/tx.proto in tx.pb.go.
package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	TypeMsgMintZYTD    = "mint_zytd"
	TypeMsgBurnZYTD    = "burn_zytd"
	TypeMsgLiquidate   = "liquidate"
)

// ─── MsgMintZYTD ─────────────────────────────────────────────────────────────

var _ sdk.Msg = &MsgMintZYTD{}

func NewMsgMintZYTD(sender, ibcDenom string, collateralAmount, requestedZYTD sdk.Int) *MsgMintZYTD {
	return &MsgMintZYTD{
		Sender:           sender,
		IbcDenom:         ibcDenom,
		CollateralAmount: collateralAmount,
		RequestedZytd:    requestedZYTD,
	}
}

func (msg *MsgMintZYTD) Route() string { return RouterKey }
func (msg *MsgMintZYTD) Type() string  { return TypeMsgMintZYTD }

func (msg *MsgMintZYTD) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Sender)
	return []sdk.AccAddress{addr}
}

func (msg *MsgMintZYTD) GetSignBytes() []byte {
	bz, _ := ModuleCdc.MarshalJSON(msg)
	return bz
}

func (msg *MsgMintZYTD) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return fmt.Errorf("invalid sender address: %w", err)
	}
	if msg.IbcDenom == "" {
		return fmt.Errorf("ibc_denom cannot be empty")
	}
	if msg.CollateralAmount.IsNil() || !msg.CollateralAmount.IsPositive() {
		return fmt.Errorf("collateral_amount must be positive")
	}
	if msg.RequestedZytd.IsNil() || !msg.RequestedZytd.IsPositive() {
		return ErrZeroMintAmount
	}
	if msg.ExpirationBlockHeight <= 0 {
		return fmt.Errorf("expiration_block_height must be positive")
	}
	return nil
}

// ─── MsgBurnZYTD ─────────────────────────────────────────────────────────────

var _ sdk.Msg = &MsgBurnZYTD{}

func NewMsgBurnZYTD(sender, ibcDenom string, zytdAmount sdk.Int) *MsgBurnZYTD {
	return &MsgBurnZYTD{
		Sender:     sender,
		IbcDenom:   ibcDenom,
		ZytdAmount: zytdAmount,
	}
}

func (msg *MsgBurnZYTD) Route() string { return RouterKey }
func (msg *MsgBurnZYTD) Type() string  { return TypeMsgBurnZYTD }

func (msg *MsgBurnZYTD) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Sender)
	return []sdk.AccAddress{addr}
}

func (msg *MsgBurnZYTD) GetSignBytes() []byte {
	bz, _ := ModuleCdc.MarshalJSON(msg)
	return bz
}

func (msg *MsgBurnZYTD) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return fmt.Errorf("invalid sender address: %w", err)
	}
	if msg.IbcDenom == "" {
		return fmt.Errorf("ibc_denom cannot be empty")
	}
	if msg.ZytdAmount.IsNil() || !msg.ZytdAmount.IsPositive() {
		return fmt.Errorf("zytd_amount must be positive")
	}
	if msg.ExpirationBlockHeight <= 0 {
		return fmt.Errorf("expiration_block_height must be positive")
	}
	return nil
}

// ─── MsgLiquidate ────────────────────────────────────────────────────────────

var _ sdk.Msg = &MsgLiquidate{}

func NewMsgLiquidate(liquidator, targetOwner, ibcDenom string) *MsgLiquidate {
	return &MsgLiquidate{
		Liquidator:  liquidator,
		TargetOwner: targetOwner,
		IbcDenom:    ibcDenom,
	}
}

func (msg *MsgLiquidate) Route() string { return RouterKey }
func (msg *MsgLiquidate) Type() string  { return TypeMsgLiquidate }

func (msg *MsgLiquidate) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Liquidator)
	return []sdk.AccAddress{addr}
}

func (msg *MsgLiquidate) GetSignBytes() []byte {
	bz, _ := ModuleCdc.MarshalJSON(msg)
	return bz
}

func (msg *MsgLiquidate) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Liquidator)
	if err != nil {
		return fmt.Errorf("invalid liquidator address: %w", err)
	}
	_, err = sdk.AccAddressFromBech32(msg.TargetOwner)
	if err != nil {
		return fmt.Errorf("invalid target_owner address: %w", err)
	}
	if msg.IbcDenom == "" {
		return fmt.Errorf("ibc_denom cannot be empty")
	}
	if msg.ExpirationBlockHeight <= 0 {
		return fmt.Errorf("expiration_block_height must be positive")
	}
	return nil
}

// ─── MsgRegisterZYTDKey ───────────────────────────────────────────────────────

var _ sdk.Msg = &MsgRegisterZYTDKey{}

func NewMsgRegisterZYTDKey(sender string, pubkey []byte) *MsgRegisterZYTDKey {
	return &MsgRegisterZYTDKey{
		Sender:           sender,
		Dilithium5PubKey: pubkey,
	}
}

func (msg *MsgRegisterZYTDKey) Route() string { return RouterKey }
func (msg *MsgRegisterZYTDKey) Type() string  { return "register_zytd_key" }

func (msg *MsgRegisterZYTDKey) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Sender)
	return []sdk.AccAddress{addr}
}

func (msg *MsgRegisterZYTDKey) GetSignBytes() []byte {
	bz, _ := ModuleCdc.MarshalJSON(msg)
	return bz
}

func (msg *MsgRegisterZYTDKey) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return fmt.Errorf("invalid sender address: %w", err)
	}
	const dilithium5PubKeySize = 2592
	if len(msg.Dilithium5PubKey) != dilithium5PubKeySize {
		return fmt.Errorf("dilithium5_pub_key must be %d bytes, got %d", dilithium5PubKeySize, len(msg.Dilithium5PubKey))
	}
	if msg.ExpirationBlockHeight <= 0 {
		return fmt.Errorf("expiration_block_height must be positive")
	}
	return nil
}
