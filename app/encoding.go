package app

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/std"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"

	"zytherion/app/params"
	dilithium5 "zytherion/crypto/dilithium5"
	"zytherion/quantumbft"
)

// makeEncodingConfig creates an EncodingConfig for an amino based test configuration.
func makeEncodingConfig() params.EncodingConfig {
	amino := codec.NewLegacyAmino()
	interfaceRegistry := types.NewInterfaceRegistry()
	marshaler := codec.NewProtoCodec(interfaceRegistry)
	txCfg := tx.NewTxConfig(marshaler, tx.DefaultSignModes)

	return params.EncodingConfig{
		InterfaceRegistry: interfaceRegistry,
		Marshaler:         marshaler,
		TxConfig:          txCfg,
		Amino:             amino,
	}
}

// MakeEncodingConfig creates an EncodingConfig for testing
func MakeEncodingConfig() params.EncodingConfig {
	encodingConfig := makeEncodingConfig()
	std.RegisterLegacyAminoCodec(encodingConfig.Amino)
	std.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	ModuleBasics.RegisterLegacyAminoCodec(encodingConfig.Amino)
	ModuleBasics.RegisterInterfaces(encodingConfig.InterfaceRegistry)

	// Register Dilithium5 (ML-DSA-87) quantum-safe key types.
	// This enables accounts with Dilithium5 pubkeys to be stored and retrieved
	// from the auth module, and enables --key-type dilithium5 in the keyring.
	dilithium5.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	dilithium5.RegisterAmino(encodingConfig.Amino)

	// Register QuantumBFT consensus key types in amino.
	quantumbft.RegisterAmino(encodingConfig.Amino)

	return encodingConfig
}
