package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

// RegisterCodec registers the module's concrete message types on the legacy
// Amino codec so they can be used in transactions signed with Amino.
func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgLockCollateral{}, "ibccollateral/LockCollateral", nil)
	cdc.RegisterConcrete(&MsgUnlockCollateral{}, "ibccollateral/UnlockCollateral", nil)
}

// RegisterInterfaces registers ibc-collateral message types with the interface registry.
func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations(
		(*sdk.Msg)(nil),
		&MsgLockCollateral{},
		&MsgUnlockCollateral{},
	)
	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

var (
	// Amino is the legacy amino codec instance shared across the module.
	Amino = codec.NewLegacyAmino()

	// ModuleCdc is the module-level codec backed by a fresh interface registry.
	ModuleCdc = codec.NewProtoCodec(cdctypes.NewInterfaceRegistry())
)

func init() {
	RegisterCodec(Amino)
}
