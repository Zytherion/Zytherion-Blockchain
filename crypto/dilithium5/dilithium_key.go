package dilithium5

import (
	"bytes"
	"fmt"

	"github.com/cloudflare/circl/sign/dilithium/mode5"
	cmtcrypto "github.com/cometbft/cometbft/crypto"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"zytherion/x/privacy/pqc"
)

// ── PubKey — cryptotypes.PubKey implementation ────────────────────────────────

// Ensure PubKey implements cryptotypes.PubKey at compile time.
var _ cryptotypes.PubKey = (*PubKey)(nil)

// Address returns the 20-byte Cosmos SDK address derived from the Dilithium5
// public key using SHA-256 truncated to 20 bytes.
//
// This matches the standard Cosmos address derivation convention:
//
//	address = SHA-256(pubkey)[0:20]
func (pk *PubKey) Address() cryptotypes.Address {
	if len(pk.Key) == 0 {
		return nil
	}
	return cmtcrypto.AddressHash(pk.Key)
}

// Bytes returns the raw 2592-byte Dilithium5 public key.
func (pk *PubKey) Bytes() []byte {
	return pk.Key
}

// VerifySignature verifies a Dilithium5 signature over msg.
// Uses the same Dilithium5/ML-DSA-87 primitives as the validator signing layer.
//
// Returns true only if:
//   - sig is exactly 4595 bytes (mode5.SignatureSize)
//   - pk.Key is exactly 2592 bytes (mode5.PublicKeySize)
//   - The signature is cryptographically valid
func (pk *PubKey) VerifySignature(msg []byte, sig []byte) bool {
	return pqc.Verify(msg, sig, pk.Key)
}

// Equals returns true if two public keys are identical.
func (pk *PubKey) Equals(other cryptotypes.PubKey) bool {
	if other == nil {
		return false
	}
	return bytes.Equal(pk.Key, other.Bytes())
}

// Type returns the key algorithm identifier.
func (pk *PubKey) Type() string {
	return KeyType
}

// ── PrivKey — cryptotypes.PrivKey implementation ──────────────────────────────

// Ensure PrivKey implements cryptotypes.PrivKey at compile time.
// NOTE: In SDK v0.47, PrivKey.Equals takes a LedgerPrivKey (subset interface).

// Sign signs msg with the Dilithium5 private key and returns the signature.
// Signing is DETERMINISTIC — same (msg, key) always produces the same signature.
// This is required by the Green-BFT consensus protocol for proposal caching.
func (sk *PrivKey) Sign(msg []byte) ([]byte, error) {
	sig, err := pqc.Sign(msg, sk.Key)
	if err != nil {
		return nil, fmt.Errorf("dilithium5 sign: %w", err)
	}
	return sig, nil
}

// PubKey derives the Dilithium5 public key from the private key using circl mode5.
//
// This is always safe to call — even if the public key was not stored,
// it can be recomputed from the private key bytes.
func (sk *PrivKey) PubKey() cryptotypes.PubKey {
	if len(sk.Key) != PrivKeySize {
		panic(fmt.Sprintf("dilithium5.PrivKey.PubKey: invalid private key size %d, expected %d", len(sk.Key), PrivKeySize))
	}

	// Unpack raw bytes into circl's PrivateKey type.
	var buf [mode5.PrivateKeySize]byte
	copy(buf[:], sk.Key)
	var privKey mode5.PrivateKey
	privKey.Unpack(&buf)

	// Derive the corresponding public key.
	pub := privKey.Public().(*mode5.PublicKey)
	return &PubKey{Key: pub.Bytes()}
}

// Bytes returns the raw 4864-byte Dilithium5 private key.
func (sk *PrivKey) Bytes() []byte {
	return sk.Key
}

// Equals returns true if two private keys are identical.
// The argument type is cryptotypes.LedgerPrivKey which is a subset of PrivKey.
func (sk *PrivKey) Equals(other cryptotypes.LedgerPrivKey) bool {
	if other == nil {
		return false
	}
	return bytes.Equal(sk.Key, other.Bytes())
}

// Type returns the key algorithm identifier.
func (sk *PrivKey) Type() string {
	return KeyType
}

// ── Key generation ────────────────────────────────────────────────────────────

// GenPrivKey generates a new random Dilithium5 private key using OS entropy.
// Use GenPrivKeyFromSecret for deterministic key generation from a seed.
func GenPrivKey() (*PrivKey, error) {
	_, priv, err := mode5.GenerateKey(nil) // nil = crypto/rand
	if err != nil {
		return nil, fmt.Errorf("dilithium5 key generation: %w", err)
	}
	return &PrivKey{Key: priv.Bytes()}, nil
}

// GenPrivKeyFromSecret derives a Dilithium5 private key deterministically from
// a 32+ byte secret (typically produced by HKDF from a BIP-39 mnemonic).
//
// The secret is used as the source of entropy for key generation — the same
// secret always produces the same keypair, enabling mnemonic-based recovery.
func GenPrivKeyFromSecret(secret []byte) (*PrivKey, error) {
	if len(secret) < 32 {
		return nil, fmt.Errorf("dilithium5 GenPrivKeyFromSecret: secret must be at least 32 bytes, got %d", len(secret))
	}

	// Use secret as a deterministic byte stream for Dilithium5 key generation.
	// mode5.GenerateKey reads from the given io.Reader.
	reader := newDeterministicReader(secret)
	_, priv, err := mode5.GenerateKey(reader)
	if err != nil {
		return nil, fmt.Errorf("dilithium5 deterministic keygen: %w", err)
	}

	return &PrivKey{Key: priv.Bytes()}, nil
}

// PrivKeyFromBytes wraps raw key bytes as a PrivKey (no copy; for deserialization).
func PrivKeyFromBytes(bz []byte) (*PrivKey, error) {
	if len(bz) != PrivKeySize {
		return nil, fmt.Errorf("dilithium5: invalid private key size %d, expected %d", len(bz), PrivKeySize)
	}
	key := make([]byte, PrivKeySize)
	copy(key, bz)
	return &PrivKey{Key: key}, nil
}

// PubKeyFromBytes wraps raw key bytes as a PubKey.
func PubKeyFromBytes(bz []byte) (*PubKey, error) {
	if len(bz) != PubKeySize {
		return nil, fmt.Errorf("dilithium5: invalid public key size %d, expected %d", len(bz), PubKeySize)
	}
	key := make([]byte, PubKeySize)
	copy(key, bz)
	return &PubKey{Key: key}, nil
}

// AccAddressFromPubKey returns the Cosmos bech32 account address for a PubKey.
func AccAddressFromPubKey(pk *PubKey) sdk.AccAddress {
	return sdk.AccAddress(pk.Address())
}
