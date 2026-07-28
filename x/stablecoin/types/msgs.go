package types

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ─── MsgMintZYTD ─────────────────────────────────────────────────────────────

const (
	TypeMsgMintZYTD    = "mint_zytd"
	TypeMsgBurnZYTD    = "burn_zytd"
	TypeMsgLiquidate   = "liquidate"
)

// MsgMintZYTD mints ZYTD stablecoin against locked collateral.
// The sender MUST sign ExpirationBlockHeight alongside the message body.
// ExpirationBlockHeight must satisfy: current <= expiry <= current+5.
type MsgMintZYTD struct {
	Sender               string  `json:"sender"`
	IBCDenom             string  `json:"ibc_denom"`
	CollateralAmount     sdk.Int `json:"collateral_amount"`
	RequestedZYTD        sdk.Int `json:"requested_zytd"`
	// PQC fields — Dilithium5 (ML-DSA Level 5) quantum-safe authentication.
	ExpirationBlockHeight int64  `json:"expiration_block_height"` // forward-only replay guard
	Dilithium5PubKey     []byte  `json:"dilithium5_pub_key"`      // 2592 bytes
	Dilithium5Sig        []byte  `json:"dilithium5_sig"`          // 4595 bytes
}

var _ sdk.Msg = &MsgMintZYTD{}

func NewMsgMintZYTD(sender, ibcDenom string, collateralAmount, requestedZYTD sdk.Int) *MsgMintZYTD {
	return &MsgMintZYTD{Sender: sender, IBCDenom: ibcDenom, CollateralAmount: collateralAmount, RequestedZYTD: requestedZYTD}
}

// GetExpirationBlockHeight satisfies ante.ExpirationBlockHeighter.
func (msg *MsgMintZYTD) GetExpirationBlockHeight() int64 { return msg.ExpirationBlockHeight }

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
	if msg.IBCDenom == "" {
		return fmt.Errorf("ibc_denom cannot be empty")
	}
	if msg.CollateralAmount.IsNil() || !msg.CollateralAmount.IsPositive() {
		return fmt.Errorf("collateral_amount must be positive")
	}
	if msg.RequestedZYTD.IsNil() || !msg.RequestedZYTD.IsPositive() {
		return ErrZeroMintAmount
	}
	if msg.ExpirationBlockHeight <= 0 {
		return fmt.Errorf("expiration_block_height must be positive")
	}
	return nil
}

func (msg *MsgMintZYTD) ProtoMessage()  {}
func (msg *MsgMintZYTD) Reset()         {}
func (msg *MsgMintZYTD) String() string { return fmt.Sprintf("%+v", *msg) }

// MsgMintZYTDResponse is the response type for MsgMintZYTD.
type MsgMintZYTDResponse struct {
	MintedAmount sdk.Int `json:"minted_amount"`
}

func (m *MsgMintZYTDResponse) ProtoMessage()  {}
func (m *MsgMintZYTDResponse) Reset()         {}
func (m *MsgMintZYTDResponse) String() string { return "" }

// ─── MsgBurnZYTD ─────────────────────────────────────────────────────────────

// MsgBurnZYTD burns ZYTD and unlocks proportional collateral.
// Burn is permitted in ALL migration phases (Phase 1, 2, 3).
type MsgBurnZYTD struct {
	Sender               string  `json:"sender"`
	IBCDenom             string  `json:"ibc_denom"`
	ZYTDAmount           sdk.Int `json:"zytd_amount"`
	// PQC fields — required in Phase 3 (hard enforcement).
	ExpirationBlockHeight int64  `json:"expiration_block_height"`
	Dilithium5PubKey     []byte  `json:"dilithium5_pub_key"`
	Dilithium5Sig        []byte  `json:"dilithium5_sig"`
}

var _ sdk.Msg = &MsgBurnZYTD{}

func NewMsgBurnZYTD(sender, ibcDenom string, zytdAmount sdk.Int) *MsgBurnZYTD {
	return &MsgBurnZYTD{Sender: sender, IBCDenom: ibcDenom, ZYTDAmount: zytdAmount}
}

// GetExpirationBlockHeight satisfies ante.ExpirationBlockHeighter.
func (msg *MsgBurnZYTD) GetExpirationBlockHeight() int64 { return msg.ExpirationBlockHeight }

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
	if msg.IBCDenom == "" {
		return fmt.Errorf("ibc_denom cannot be empty")
	}
	if msg.ZYTDAmount.IsNil() || !msg.ZYTDAmount.IsPositive() {
		return fmt.Errorf("zytd_amount must be positive")
	}
	if msg.ExpirationBlockHeight <= 0 {
		return fmt.Errorf("expiration_block_height must be positive")
	}
	return nil
}

func (msg *MsgBurnZYTD) ProtoMessage()  {}
func (msg *MsgBurnZYTD) Reset()         {}
func (msg *MsgBurnZYTD) String() string { return fmt.Sprintf("%+v", *msg) }

// MsgBurnZYTDResponse is the response type for MsgBurnZYTD.
type MsgBurnZYTDResponse struct {
	UnlockedAmount sdk.Int `json:"unlocked_amount"`
}

func (m *MsgBurnZYTDResponse) ProtoMessage()  {}
func (m *MsgBurnZYTDResponse) Reset()         {}
func (m *MsgBurnZYTDResponse) String() string { return "" }

// ─── MsgLiquidate ────────────────────────────────────────────────────────────

// MsgLiquidate liquidates an undercollateralized position.
// Liquidations remain allowed in all migration phases.
type MsgLiquidate struct {
	Liquidator           string `json:"liquidator"`
	TargetOwner          string `json:"target_owner"`
	IBCDenom             string `json:"ibc_denom"`
	// PQC fields — required in Phase 3 (hard enforcement).
	ExpirationBlockHeight int64 `json:"expiration_block_height"`
	Dilithium5PubKey     []byte `json:"dilithium5_pub_key"`
	Dilithium5Sig        []byte `json:"dilithium5_sig"`
}

var _ sdk.Msg = &MsgLiquidate{}

func NewMsgLiquidate(liquidator, targetOwner, ibcDenom string) *MsgLiquidate {
	return &MsgLiquidate{Liquidator: liquidator, TargetOwner: targetOwner, IBCDenom: ibcDenom}
}

// GetExpirationBlockHeight satisfies ante.ExpirationBlockHeighter.
func (msg *MsgLiquidate) GetExpirationBlockHeight() int64 { return msg.ExpirationBlockHeight }

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
	if msg.IBCDenom == "" {
		return fmt.Errorf("ibc_denom cannot be empty")
	}
	if msg.ExpirationBlockHeight <= 0 {
		return fmt.Errorf("expiration_block_height must be positive")
	}
	return nil
}

func (msg *MsgLiquidate) ProtoMessage()  {}
func (msg *MsgLiquidate) Reset()         {}
func (msg *MsgLiquidate) String() string { return fmt.Sprintf("%+v", *msg) }

// MsgLiquidateResponse is the response type for MsgLiquidate.
type MsgLiquidateResponse struct {
	SeizedAmount sdk.Int `json:"seized_amount"`
}

func (m *MsgLiquidateResponse) ProtoMessage()  {}
func (m *MsgLiquidateResponse) Reset()         {}
func (m *MsgLiquidateResponse) String() string { return "" }

// ─── MsgServer interface ──────────────────────────────────────────────────────

// ─── MsgRegisterZYTDKey ───────────────────────────────────────────────────────

// MsgRegisterZYTDKey registers a Dilithium5 public key for the sender address.
// After registration, the sender's ZYTD transactions must include a valid
// Dilithium5 signature over the message body.
// This message is always allowed — in all migration phases.
type MsgRegisterZYTDKey struct {
	Sender               string `json:"sender"`
	Dilithium5PubKey     []byte `json:"dilithium5_pub_key"` // must be exactly 2592 bytes
	ExpirationBlockHeight int64 `json:"expiration_block_height"`
}

var _ sdk.Msg = &MsgRegisterZYTDKey{}

func NewMsgRegisterZYTDKey(sender string, pubkey []byte) *MsgRegisterZYTDKey {
	return &MsgRegisterZYTDKey{Sender: sender, Dilithium5PubKey: pubkey}
}

// GetExpirationBlockHeight satisfies ante.ExpirationBlockHeighter.
func (msg *MsgRegisterZYTDKey) GetExpirationBlockHeight() int64 { return msg.ExpirationBlockHeight }

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
	// Dilithium5 public key is exactly 2592 bytes (mode5.PublicKeySize).
	const dilithium5PubKeySize = 2592
	if len(msg.Dilithium5PubKey) != dilithium5PubKeySize {
		return fmt.Errorf("dilithium5_pub_key must be %d bytes, got %d", dilithium5PubKeySize, len(msg.Dilithium5PubKey))
	}
	if msg.ExpirationBlockHeight <= 0 {
		return fmt.Errorf("expiration_block_height must be positive")
	}
	return nil
}

func (msg *MsgRegisterZYTDKey) ProtoMessage()  {}
func (msg *MsgRegisterZYTDKey) Reset()         {}
func (msg *MsgRegisterZYTDKey) String() string { return fmt.Sprintf("%+v", *msg) }

// MsgRegisterZYTDKeyResponse is the response for MsgRegisterZYTDKey.
type MsgRegisterZYTDKeyResponse struct {
	RegisteredPubKeyHash string `json:"registered_pub_key_hash"` // hex SHA-256 of pubkey
}

func (m *MsgRegisterZYTDKeyResponse) ProtoMessage()  {}
func (m *MsgRegisterZYTDKeyResponse) Reset()         {}
func (m *MsgRegisterZYTDKeyResponse) String() string { return "" }

// ─── MsgServer interface ──────────────────────────────────────────────────────

// MsgServer is the stablecoin message handler interface.
type MsgServer interface {
	MintZYTD(context.Context, *MsgMintZYTD) (*MsgMintZYTDResponse, error)
	BurnZYTD(context.Context, *MsgBurnZYTD) (*MsgBurnZYTDResponse, error)
	Liquidate(context.Context, *MsgLiquidate) (*MsgLiquidateResponse, error)
	RegisterZYTDKey(context.Context, *MsgRegisterZYTDKey) (*MsgRegisterZYTDKeyResponse, error)
}
