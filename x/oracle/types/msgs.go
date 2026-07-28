package types

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const TypeMsgSubmitPrice = "submit_price"

var _ sdk.Msg = &MsgSubmitPrice{}

// MsgSubmitPrice is the transaction message for submitting a price feed entry.
type MsgSubmitPrice struct {
	Submitter string  `json:"submitter"`
	Denom     string  `json:"denom"`
	PriceUSD  sdk.Dec `json:"price_usd"`
}

// NewMsgSubmitPrice constructs a new MsgSubmitPrice.
func NewMsgSubmitPrice(submitter, denom string, price sdk.Dec) *MsgSubmitPrice {
	return &MsgSubmitPrice{Submitter: submitter, Denom: denom, PriceUSD: price}
}

func (msg *MsgSubmitPrice) Route() string { return RouterKey }
func (msg *MsgSubmitPrice) Type() string  { return TypeMsgSubmitPrice }

func (msg *MsgSubmitPrice) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Submitter)
	return []sdk.AccAddress{addr}
}

func (msg *MsgSubmitPrice) GetSignBytes() []byte {
	bz, _ := Amino.MarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgSubmitPrice) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Submitter)
	if err != nil {
		return fmt.Errorf("invalid submitter address: %w", err)
	}
	if msg.Denom == "" {
		return fmt.Errorf("denom cannot be empty")
	}
	if msg.PriceUSD.IsNil() || !msg.PriceUSD.IsPositive() {
		return fmt.Errorf("price must be positive")
	}
	return nil
}

// ProtoMessage implements proto.Message interface.
func (msg *MsgSubmitPrice) ProtoMessage() {}

// Reset implements proto.Message interface.
func (msg *MsgSubmitPrice) Reset() {}

// String implements proto.Message interface.
func (msg *MsgSubmitPrice) String() string { return fmt.Sprintf("%+v", *msg) }

// MsgSubmitPriceResponse is the response type for MsgSubmitPrice.
type MsgSubmitPriceResponse struct{}

func (m *MsgSubmitPriceResponse) ProtoMessage()  {}
func (m *MsgSubmitPriceResponse) Reset()         {}
func (m *MsgSubmitPriceResponse) String() string { return "" }

// MsgServer defines the oracle Msg service interface.
type MsgServer interface {
	SubmitPrice(context.Context, *MsgSubmitPrice) (*MsgSubmitPriceResponse, error)
}
