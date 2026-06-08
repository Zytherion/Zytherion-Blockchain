package pqc_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"

	"zytherion/x/privacy/pqc"
)

// ─── KeyPair Generation ───────────────────────────────────────────────────────

// TestDilithiumKeyPairGeneration verifies that GenerateKeyPair succeeds and
// returns non-empty keys of the expected lengths for Dilithium5 (ML-DSA Level 5).
func TestDilithiumKeyPairGeneration(t *testing.T) {
	kp, err := pqc.GenerateKeyPair()
	require.NoError(t, err, "GenerateKeyPair must not return an error")

	// Dilithium5 public key = mode5.PublicKeySize = 2592 bytes.
	require.Equal(t, pqc.DilithiumPublicKeySize, len(kp.PublicKey),
		"Dilithium5 public key must be %d bytes", pqc.DilithiumPublicKeySize)
	// Dilithium5 private key = mode5.PrivateKeySize = 4864 bytes.
	require.Equal(t, pqc.DilithiumPrivateKeySize, len(kp.PrivateKey),
		"Dilithium5 private key must be %d bytes", pqc.DilithiumPrivateKeySize)
	require.NotEmpty(t, kp.PrivateKey, "private key must not be empty")
}

// ─── Sign & Verify ────────────────────────────────────────────────────────────

// TestDilithiumSignVerify is the core round-trip test:
// sign a message and verify the signature — must succeed.
func TestDilithiumSignVerify(t *testing.T) {
	kp, err := pqc.GenerateKeyPair()
	require.NoError(t, err)

	message := []byte("height=1 prevhash=0x0000 apphash=0xdeadbeef")

	sig, err := pqc.Sign(message, kp.PrivateKey)
	require.NoError(t, err, "Sign must not return an error")
	require.NotEmpty(t, sig, "signature must not be empty")

	// Dilithium5 signature = mode5.SignatureSize = 4595 bytes.
	require.Equal(t, pqc.DilithiumSignatureSize, len(sig),
		"Dilithium5 signature must be %d bytes", pqc.DilithiumSignatureSize)

	ok := pqc.Verify(message, sig, kp.PublicKey)
	require.True(t, ok, "Verify must return true for a valid Dilithium5 signature")
}

// TestDilithiumDeterminism verifies that signing the same message twice with
// the same key produces the same signature (deterministic signing per ML-DSA spec).
func TestDilithiumDeterminism(t *testing.T) {
	kp, err := pqc.GenerateKeyPair()
	require.NoError(t, err)

	message := []byte("determinism-test-payload")

	sig1, err := pqc.Sign(message, kp.PrivateKey)
	require.NoError(t, err)

	sig2, err := pqc.Sign(message, kp.PrivateKey)
	require.NoError(t, err)

	require.True(t, bytes.Equal(sig1, sig2),
		"Dilithium5 signing must be deterministic: same (key, message) must produce identical signatures")
}

// TestDilithiumWrongKeyVerify verifies that a signature produced by key A
// does not verify under key B. This catches any accidental key confusion.
func TestDilithiumWrongKeyVerify(t *testing.T) {
	kpA, err := pqc.GenerateKeyPair()
	require.NoError(t, err)
	kpB, err := pqc.GenerateKeyPair()
	require.NoError(t, err)

	message := []byte("payload signed with key A")

	sig, err := pqc.Sign(message, kpA.PrivateKey)
	require.NoError(t, err)

	// Must NOT verify under key B.
	ok := pqc.Verify(message, sig, kpB.PublicKey)
	require.False(t, ok,
		"signature produced with key A must not verify under key B")
}

// TestDilithiumTamperedMessage verifies that modifying the message after
// signing causes Dilithium5 verification to fail.
func TestDilithiumTamperedMessage(t *testing.T) {
	kp, err := pqc.GenerateKeyPair()
	require.NoError(t, err)

	original := []byte("original block hash bytes ...")
	sig, err := pqc.Sign(original, kp.PrivateKey)
	require.NoError(t, err)

	tampered := make([]byte, len(original))
	copy(tampered, original)
	tampered[0] ^= 0xFF

	ok := pqc.Verify(tampered, sig, kp.PublicKey)
	require.False(t, ok,
		"tampered message must fail Dilithium5 verification")
}

// TestDilithiumInvalidKeyBytes verifies that Sign and Verify handle
// malformed key bytes gracefully rather than panicking.
func TestDilithiumInvalidKeyBytes(t *testing.T) {
	garbage := []byte("not a valid dilithium5 key")

	_, err := pqc.Sign([]byte("msg"), garbage)
	require.Error(t, err, "Sign with invalid private key must return an error")

	ok := pqc.Verify([]byte("msg"), []byte("fake-sig"), garbage)
	require.False(t, ok, "Verify with invalid public key must return false")
}

// TestDilithiumKeySizes confirms exported constants match expected ML-DSA Level 5 sizes.
func TestDilithiumKeySizes(t *testing.T) {
	require.Equal(t, 2592, pqc.DilithiumPublicKeySize, "ML-DSA-87 public key must be 2592 bytes")
	require.Equal(t, 4864, pqc.DilithiumPrivateKeySize, "ML-DSA-87 private key must be 4864 bytes")
	require.Equal(t, 4595, pqc.DilithiumSignatureSize, "ML-DSA-87 signature must be 4595 bytes")
}

// ─── Benchmark ────────────────────────────────────────────────────────────────

// BenchmarkDilithiumSign measures Dilithium5 signing throughput.
func BenchmarkDilithiumSign(b *testing.B) {
	kp, _ := pqc.GenerateKeyPair()
	msg := make([]byte, 32)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pqc.Sign(msg, kp.PrivateKey) //nolint:errcheck
	}
}

// BenchmarkDilithiumVerify measures Dilithium5 verification throughput.
func BenchmarkDilithiumVerify(b *testing.B) {
	kp, _ := pqc.GenerateKeyPair()
	msg := make([]byte, 32)
	sig, _ := pqc.Sign(msg, kp.PrivateKey)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pqc.Verify(msg, sig, kp.PublicKey)
	}
}
