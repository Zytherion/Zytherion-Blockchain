package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
)

var (
	Amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewAminoCodec(Amino)
)

func init() {
	RegisterCodec(Amino)
}

func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgMintZYTD{}, "stablecoin/MintZYTD", nil)
	cdc.RegisterConcrete(&MsgBurnZYTD{}, "stablecoin/BurnZYTD", nil)
	cdc.RegisterConcrete(&MsgLiquidate{}, "stablecoin/Liquidate", nil)
	cdc.RegisterConcrete(&MsgRegisterZYTDKey{}, "stablecoin/RegisterZYTDKey", nil)
}

// RegisterInterfaces is a no-op for stablecoin — messages are Amino-routed only.
// RegisterImplementations requires v2 proto-generated types; our manual structs
// do NOT have a protoreflect descriptor, so calling it would panic at startup.
func RegisterInterfaces(_ codectypes.InterfaceRegistry) {}

// msgServerKey is used for server registration (compatible with amino-based msg routing).
type msgServerRegistrar interface {
	RegisterMsgServer(srv MsgServer)
}

// RegisterMsgServer is a no-op stub for amino-based message routing compatibility.
// In a protobuf-first setup, this would register via the module configurator.
func RegisterMsgServer(_ interface{}, _ MsgServer) {}
