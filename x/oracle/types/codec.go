package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	// Amino is the legacy amino codec for the oracle module.
	Amino = codec.NewLegacyAmino()
)

func init() {
	RegisterCodec(Amino)
	sdk.RegisterLegacyAminoCodec(Amino)
}

// RegisterCodec registers oracle message types with the legacy amino codec.
func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgSubmitPrice{}, "oracle/SubmitPrice", nil)
}

// RegisterInterfaces is a no-op for oracle — messages are Amino-routed only.
// RegisterImplementations requires v2 proto-generated types; our manual structs
// do NOT have a protoreflect descriptor, so calling it would panic at startup.
func RegisterInterfaces(_ cdctypes.InterfaceRegistry) {}
