package kyber1024

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
)

func init() {
	RegisterCodec(codec.NewLegacyAmino())
}

// RegisterCodec registers Kyber1024 key types with the legacy amino codec.
func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&PublicKey{}, "crypto/kyber1024/PublicKey", nil)
	cdc.RegisterConcrete(&PrivateKey{}, "crypto/kyber1024/PrivateKey", nil)
}

// RegisterInterfaces registers Kyber1024 key types with the interface registry.
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations((*interface{})(nil),
		&PublicKey{},
		&PrivateKey{},
	)
}
