// Package kyber1024 — KEM operations (Encapsulate / Decapsulate / key generation).
//
// # KEM Flow
//
//  1. Receiver generates keypair: GenKeyPair() → (PublicKey, PrivateKey)
//  2. Receiver publishes PublicKey
//  3. Sender calls: Encapsulate(pubKey) → (ciphertext, sharedSecret)
//  4. Sender encrypts payload with sharedSecret (e.g. AES-256-GCM)
//  5. Sender transmits: ciphertext + encrypted payload
//  6. Receiver calls: Decapsulate(privKey, ciphertext) → sharedSecret
//  7. Receiver decrypts payload with sharedSecret
//
// Both parties now share the same 32-byte secret without it ever appearing
// in plaintext on the wire — quantum-safe because Kyber1024 is hard under LWE.
package kyber1024

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"

	circlkyber "github.com/cloudflare/circl/kem/kyber/kyber1024"
)

// GenKeyPair generates a fresh Kyber1024 keypair from a random source.
// pubBytes: 1568 bytes, privBytes: 3168 bytes.
func GenKeyPair() (pubBytes, privBytes []byte, err error) {
	pub, priv, err := circlkyber.GenerateKeyPair(rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("kyber1024: key generation failed: %w", err)
	}

	pubBytes, err = pub.MarshalBinary()
	if err != nil {
		return nil, nil, fmt.Errorf("kyber1024: marshal public key: %w", err)
	}

	privBytes, err = priv.MarshalBinary()
	if err != nil {
		return nil, nil, fmt.Errorf("kyber1024: marshal private key: %w", err)
	}

	return pubBytes, privBytes, nil
}

// GenKeyPairFromSeed generates a deterministic Kyber1024 keypair from 64 bytes of entropy.
// Used by HD derivation (BIP-39 mnemonic → deterministic keypair).
func GenKeyPairFromSeed(seed []byte) (pubBytes, privBytes []byte, err error) {
	if len(seed) < 64 {
		return nil, nil, fmt.Errorf("kyber1024: seed must be at least 64 bytes, got %d", len(seed))
	}

	// circlkyber Scheme().DeriveKeyPair requires exactly 64 bytes of seed.
	scheme := circlkyber.Scheme()
	pub, priv := scheme.DeriveKeyPair(seed[:64])

	pubBytes, err = pub.MarshalBinary()
	if err != nil {
		return nil, nil, fmt.Errorf("kyber1024: marshal public key: %w", err)
	}

	privBytes, err = priv.MarshalBinary()
	if err != nil {
		return nil, nil, fmt.Errorf("kyber1024: marshal private key: %w", err)
	}

	return pubBytes, privBytes, nil
}

// PublicKeyFromPrivate extracts the public key bytes from a serialised private key.
func PublicKeyFromPrivate(privBytes []byte) ([]byte, error) {
	scheme := circlkyber.Scheme()
	priv, err := scheme.UnmarshalBinaryPrivateKey(privBytes)
	if err != nil {
		return nil, fmt.Errorf("kyber1024: unmarshal private key: %w", err)
	}
	pub := priv.Public()
	b, err := pub.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("kyber1024: marshal public key: %w", err)
	}
	return b, nil
}

// Encapsulate performs KEM encapsulation against a serialised Kyber1024 public key.
//
// Returns:
//   - ciphertext (1568 bytes): transmit this to the receiver
//   - sharedSecret (32 bytes): use this as symmetric key (e.g. AES-256-GCM)
func Encapsulate(pubBytes []byte) (ciphertext []byte, sharedSecret []byte, err error) {
	scheme := circlkyber.Scheme()
	pub, err := scheme.UnmarshalBinaryPublicKey(pubBytes)
	if err != nil {
		return nil, nil, fmt.Errorf("kyber1024: unmarshal public key: %w", err)
	}

	ct, ss, err := scheme.Encapsulate(pub)
	if err != nil {
		return nil, nil, fmt.Errorf("kyber1024: encapsulate: %w", err)
	}

	return ct, ss, nil
}

// Decapsulate performs KEM decapsulation using a serialised Kyber1024 private key.
//
// Returns the same 32-byte sharedSecret that the sender obtained via Encapsulate.
func Decapsulate(privBytes, ciphertext []byte) (sharedSecret []byte, err error) {
	scheme := circlkyber.Scheme()
	priv, err := scheme.UnmarshalBinaryPrivateKey(privBytes)
	if err != nil {
		return nil, fmt.Errorf("kyber1024: unmarshal private key: %w", err)
	}

	ss, err := scheme.Decapsulate(priv, ciphertext)
	if err != nil {
		return nil, fmt.Errorf("kyber1024: decapsulate: %w", err)
	}

	return ss, nil
}

// EncryptFile encrypts arbitrary plaintext using Kyber1024 KEM + AES-256-GCM.
//
// Output format (all concatenated):
//
//	[2 bytes: KEM ciphertext length (big-endian uint16)]
//	[N bytes: KEM ciphertext (1568 bytes for Kyber1024)]
//	[12 bytes: AES-GCM nonce]
//	[M bytes: AES-GCM ciphertext + tag]
//
// The receiver uses their Kyber1024 PrivateKey to decapsulate the shared secret,
// then uses it as the AES-256-GCM key to decrypt the payload.
func EncryptFile(pubBytes, plaintext []byte) ([]byte, error) {
	// Step 1: KEM encapsulate → ciphertext + sharedSecret
	kemCT, sharedSecret, err := Encapsulate(pubBytes)
	if err != nil {
		return nil, err
	}

	// Step 2: AES-256-GCM encrypt with sharedSecret (32 bytes = AES-256)
	block, err := aes.NewCipher(sharedSecret)
	if err != nil {
		return nil, fmt.Errorf("kyber1024: AES init: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("kyber1024: GCM init: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("kyber1024: nonce generation: %w", err)
	}

	encrypted := gcm.Seal(nil, nonce, plaintext, nil)

	// Step 3: Assemble output
	// [2 bytes KEM CT len][KEM CT][nonce][AES-GCM CT+tag]
	ctLen := len(kemCT)
	out := make([]byte, 2+ctLen+len(nonce)+len(encrypted))
	out[0] = byte(ctLen >> 8)
	out[1] = byte(ctLen)
	copy(out[2:], kemCT)
	copy(out[2+ctLen:], nonce)
	copy(out[2+ctLen+len(nonce):], encrypted)

	return out, nil
}

// DecryptFile decrypts a blob produced by EncryptFile using a Kyber1024 private key.
func DecryptFile(privBytes, blob []byte) ([]byte, error) {
	if len(blob) < 2 {
		return nil, fmt.Errorf("kyber1024: blob too short")
	}

	ctLen := int(blob[0])<<8 | int(blob[1])
	if len(blob) < 2+ctLen+12 {
		return nil, fmt.Errorf("kyber1024: blob truncated (need %d, have %d)", 2+ctLen+12, len(blob))
	}

	kemCT := blob[2 : 2+ctLen]
	rest := blob[2+ctLen:]

	// Step 1: Decapsulate → sharedSecret
	sharedSecret, err := Decapsulate(privBytes, kemCT)
	if err != nil {
		return nil, err
	}

	// Step 2: AES-256-GCM decrypt
	block, err := aes.NewCipher(sharedSecret)
	if err != nil {
		return nil, fmt.Errorf("kyber1024: AES init: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("kyber1024: GCM init: %w", err)
	}

	if len(rest) < gcm.NonceSize() {
		return nil, fmt.Errorf("kyber1024: blob missing nonce")
	}

	nonce := rest[:gcm.NonceSize()]
	ciphertext := rest[gcm.NonceSize():]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("kyber1024: AES-GCM decrypt failed (wrong key or corrupted data): %w", err)
	}

	return plaintext, nil
}
