// msgs.go — sdk.Msg interface implementations for x/oracle.
//
// Struct definitions (MsgSubmitPrice, MsgSubmitPriceResponse)
// are auto-generated from proto/zytherion/oracle/tx.proto in tx.pb.go.
package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const TypeMsgSubmitPrice = "submit_price"

var _ sdk.Msg = &MsgSubmitPrice{}

// NewMsgSubmitPrice constructs a new MsgSubmitPrice.
func NewMsgSubmitPrice(submitter, denom string, price sdk.Dec) *MsgSubmitPrice {
	return &MsgSubmitPrice{Submitter: submitter, Denom: denom, PriceUsd: price}
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
	if msg.PriceUsd.IsNil() || !msg.PriceUsd.IsPositive() {
		return fmt.Errorf("price must be positive")
	}
	return nil
}
