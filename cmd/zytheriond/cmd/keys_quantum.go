package cmd

import (
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"

	dilithium5 "zytherion/crypto/dilithium5"
)

// dilithium5KeyringOption returns a keyring.Option that adds Dilithium5 to the
// list of supported signing algorithms.
//
// Apply this option when opening a keyring to enable:
//
//	zytheriond keys add alice --key-type dilithium5
//	zytheriond keys add alice --key-type dilithium5 --recover
func dilithium5KeyringOption() keyring.Option {
	return func(options *keyring.Options) {
		options.SupportedAlgos = keyring.SigningAlgoList{
			hd.Secp256k1,
			dilithium5.Dilithium5Algo,
		}
		options.SupportedAlgosLedger = keyring.SigningAlgoList{
			hd.Secp256k1, // Ledger only supports classical secp256k1
		}
	}
}

// Dilithium5KeyringOptions is the slice of keyring options that registers
// Dilithium5 as an available signing algorithm. Pass to keyring.New().
var Dilithium5KeyringOptions = []keyring.Option{dilithium5KeyringOption()}
