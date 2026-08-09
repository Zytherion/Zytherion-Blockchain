package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

var (
	Amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
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

// RegisterInterfaces registers the stablecoin module message types with the interface registry.
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations(
		(*sdk.Msg)(nil),
		&MsgMintZYTD{},
		&MsgBurnZYTD{},
		&MsgLiquidate{},
		&MsgRegisterZYTDKey{},
	)
	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}
