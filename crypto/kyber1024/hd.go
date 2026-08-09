// Package kyber1024 — HD key derivation for Cosmos SDK keyring.
//
// # Mnemonic Recovery for Kyber1024
//
// Standard BIP-39 mnemonics work with Kyber1024 via HKDF bridge:
//
//  1. BIP-39: mnemonic + passphrase → 512-bit seed (standard, unchanged)
//  2. HKDF-SHA256: seed + domain + HD path → 64-byte entropy
//  3. Kyber1024 DeriveKeyPair: entropy(64 bytes) → deterministic keypair
//
// # Domain Separation
//
// The HKDF info string "zytherion/kyber1024/v1" prevents cross-algorithm
// key reuse — a Kyber1024 key derived from a mnemonic will never collide
// with a Dilithium5 or secp256k1 key derived from the same mnemonic.
//
// # Important Note
//
// Kyber1024 is a KEM (Key Encapsulation), NOT a signature scheme.
// The Cosmos SDK keyring SignatureAlgo interface requires a Sign() method.
// For Kyber1024 in the keyring, we use an Ed25519 stub for signing
// while keeping the Kyber public key for KEM operations.
// The Kyber keypair is stored as auxiliary key material.
package kyber1024

import (
	"crypto/sha256"
	"fmt"
	"io"

	bip39 "github.com/cosmos/go-bip39"
	"golang.org/x/crypto/hkdf"
)

// hkdfDomainSeparator domain-separates Kyber1024 keys from all other algorithms.
const hkdfDomainSeparator = "zytherion/kyber1024/v1"

// DeriveFromMnemonic derives a Kyber1024 keypair from a BIP-39 mnemonic.
//
// This function is used directly (not via Cosmos SDK keyring SignatureAlgo)
// because Kyber is a KEM, not a signature scheme.
//
// Parameters:
//   - mnemonic: 24-word BIP-39 mnemonic
//   - bip39Passphrase: optional passphrase (use "" for none)
//   - hdPath: HD path string (e.g. "m/44'/118'/0'/0/0")
//
// Returns: pubBytes (1568 bytes), privBytes (3168 bytes)
func DeriveFromMnemonic(mnemonic, bip39Passphrase, hdPath string) (pubBytes, privBytes []byte, err error) {
	if !bip39.IsMnemonicValid(mnemonic) {
		return nil, nil, fmt.Errorf("kyber1024 derive: invalid BIP-39 mnemonic")
	}

	// Step 1: BIP-39 mnemonic → 512-bit seed.
	seed := bip39.NewSeed(mnemonic, bip39Passphrase)

	// Step 2: HKDF-SHA256 to produce 64 bytes of entropy for Kyber1024 DeriveKeyPair.
	// We need 64 bytes because circlkyber.DeriveKeyPair requires exactly 64 bytes.
	info := []byte(hkdfDomainSeparator + "/" + hdPath)
	r := hkdf.New(sha256.New, seed, nil, info)

	entropy := make([]byte, 64)
	if _, err = io.ReadFull(r, entropy); err != nil {
		return nil, nil, fmt.Errorf("kyber1024 derive hkdf: %w", err)
	}

	// Step 3: Deterministic Kyber1024 keygen from entropy.
	return GenKeyPairFromSeed(entropy)
}
