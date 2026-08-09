package dilithium5

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
)

// RegisterInterfaces registers the Dilithium5 PubKey and PrivKey types with
// the Cosmos SDK interface registry.
//
// This enables:
//   - Protobuf Any encoding/decoding of Dilithium5 keys
//   - Account pubkey storage as /zytherion.crypto.dilithium5.PubKey
//   - CLI output showing the correct type URL
//
// Must be called before starting the node (in app.MakeEncodingConfig).
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations(
		(*cryptotypes.PubKey)(nil),
		&PubKey{},
	)
	registry.RegisterImplementations(
		(*cryptotypes.PrivKey)(nil),
		&PrivKey{},
	)
}

// RegisterAmino registers the Dilithium5 key types with the legacy Amino codec.
//
// This enables Amino JSON encoding (used by legacy transaction signing and
// some CLI outputs) to correctly identify Dilithium5 keys.
func RegisterAmino(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&PubKey{}, "crypto/dilithium5/PubKey", nil)
	cdc.RegisterConcrete(&PrivKey{}, "crypto/dilithium5/PrivKey", nil)
}
