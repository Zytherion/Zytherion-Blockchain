package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
)

// RegisterCodec registers the module's concrete message types on the legacy
// Amino codec so they can be used in transactions signed with Amino.
func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgLockCollateral{}, "ibccollateral/LockCollateral", nil)
	cdc.RegisterConcrete(&MsgUnlockCollateral{}, "ibccollateral/UnlockCollateral", nil)
}

// RegisterInterfaces is a no-op for ibc-collateral — messages are Amino-routed only.
// RegisterImplementations requires v2 proto-generated types; our manual structs
// do NOT have a protoreflect descriptor, so calling it would panic at startup.
func RegisterInterfaces(_ cdctypes.InterfaceRegistry) {}

var (
	// Amino is the legacy amino codec instance shared across the module.
	Amino = codec.NewLegacyAmino()

	// ModuleCdc is the module-level codec backed by a fresh interface registry.
	ModuleCdc = codec.NewProtoCodec(cdctypes.NewInterfaceRegistry())
)

func init() {
	RegisterCodec(Amino)
}
