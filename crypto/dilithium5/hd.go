// Package dilithium5 — HD key derivation for Cosmos SDK keyring.
//
// # Mnemonic Recovery for Dilithium5
//
// Standard BIP-39 mnemonics work seamlessly with Dilithium5 via a two-step bridge:
//
//  1. BIP-39: mnemonic + passphrase → 512-bit seed  (standard, unchanged)
//  2. HKDF-SHA256: seed + domain + HD path → 32-byte entropy
//  3. Dilithium5 keygen: deterministic reader(entropy) → keypair
//
// The same 24-word mnemonic always produces the same Dilithium5 keypair.
// This means users can recover their quantum wallet from their mnemonic,
// just like they would with secp256k1.
//
// # Domain Separation
//
// The HKDF info string "zytherion/dilithium5/v1" prevents cross-algorithm
// key reuse — a Dilithium5 key derived from a mnemonic will never collide
// with a secp256k1 key derived from the same mnemonic.
package dilithium5

import (
	"crypto/sha256"
	"fmt"
	"io"

	bip39 "github.com/cosmos/go-bip39"
	"golang.org/x/crypto/hkdf"

	"github.com/cosmos/cosmos-sdk/crypto/hd"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
)

// hkdfDomainSeparator is the HKDF info string that domain-separates Dilithium5
// keys from secp256k1 keys derived from the same mnemonic.
const hkdfDomainSeparator = "zytherion/dilithium5/v1"

// Dilithium5Algo is the signature algorithm for Dilithium5 keys.
// Register it with the keyring to enable:
//
//	zytheriond keys add alice --key-type dilithium5
//	zytheriond keys add alice --key-type dilithium5 --recover  (mnemonic recovery)
var Dilithium5Algo dilithium5Algo = dilithium5Algo{}



// Expose methods on the concrete type for testing without the interface.
type dilithium5Algo struct{}

// Name returns the algorithm name used in --key-type flag and key info.
func (a dilithium5Algo) Name() hd.PubKeyType {
	return hd.PubKeyType(KeyType) // "dilithium5"
}

// Derive returns the derivation function: mnemonic + passphrase + path → 32-byte raw key.
//
// The returned bytes are passed directly to Generate() to produce the keypair.
// The HD path is embedded in HKDF's "info" parameter so different paths produce
// different keys from the same mnemonic (standard HD wallet behaviour).
func (a dilithium5Algo) Derive() hd.DeriveFn {
	return func(mnemonic, bip39Passphrase, hdPath string) ([]byte, error) {
		if !bip39.IsMnemonicValid(mnemonic) {
			return nil, fmt.Errorf("dilithium5 derive: invalid BIP-39 mnemonic")
		}

		// Step 1: BIP-39 mnemonic → 512-bit seed (exactly as secp256k1 does it).
		seed := bip39.NewSeed(mnemonic, bip39Passphrase)

		// Step 2: HKDF-SHA256 to produce 32 bytes of entropy for Dilithium5 keygen.
		// - ikm (input key material): 512-bit BIP-39 seed
		// - salt: nil (HKDF spec allows this; salt defaults to zero bytes)
		// - info (domain + path): prevents key reuse across algorithms and paths
		info := []byte(hkdfDomainSeparator + "/" + hdPath)
		r := hkdf.New(sha256.New, seed, nil, info)

		entropy := make([]byte, 32)
		if _, err := io.ReadFull(r, entropy); err != nil {
			return nil, fmt.Errorf("dilithium5 derive hkdf: %w", err)
		}

		return entropy, nil
	}
}

// Generate returns the key generation function: 32-byte entropy → PrivKey.
//
// The entropy (produced by Derive) is used as a deterministic seed for
// Dilithium5 key generation, making the keypair reproducible from the mnemonic.
func (a dilithium5Algo) Generate() hd.GenerateFn {
	return func(bz []byte) cryptotypes.PrivKey {
		privKey, err := GenPrivKeyFromSecret(bz)
		if err != nil {
			// This should never happen because Derive guarantees 32 bytes.
			panic(fmt.Sprintf("dilithium5 Generate: %v", err))
		}
		return privKey
	}
}
