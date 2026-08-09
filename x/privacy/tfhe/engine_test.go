// engine_test.go — Unit tests for the TFHE CGo engine.
//
// These tests require the Rust library to be compiled first:
//   cd x/privacy/tfhe/tfhe_c && cargo build --release
//
// Run with: go test -tags tfhe_cgo ./x/privacy/tfhe/ -v -timeout=600s
// (TFHE keygen is slow — allow ample timeout)

package tfhe_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	tfhepkg "zytherion/x/privacy/tfhe"
)

// TestTFHEEncryptDecryptRoundtrip tests the basic Encrypt → Decrypt round-trip.
// This is the most fundamental correctness test.
//
// Expected: Dec(Enc(x)) == x  for any 32-bit x.
func TestTFHEEncryptDecryptRoundtrip(t *testing.T) {
	t.Log("Generating TFHE keys (this may take 30-120 seconds)...")
	keys, err := tfhepkg.GenerateKeys()
	require.NoError(t, err, "GenerateKeys must succeed")
	require.NotEmpty(t, keys.ClientKey, "ClientKey must not be empty")
	require.NotEmpty(t, keys.ServerKey, "ServerKey must not be empty")

	testValues := []uint32{0, 1, 42, 1000, 0xDEAD, 0xFFFFFFFF}
	for _, v := range testValues {
		ct, err := tfhepkg.EncryptUint32(keys.ClientKey, v)
		require.NoError(t, err, "EncryptUint32(%d) must succeed", v)
		require.NotEmpty(t, ct, "Ciphertext must not be empty")

		// Verify ciphertext is approximately 21 KB (within 10-32 KB range)
		require.GreaterOrEqual(t, len(ct), 10*1024,
			"Ciphertext should be >= 10 KB (got %d bytes)", len(ct))
		require.LessOrEqual(t, len(ct), tfhepkg.CiphertextMaxBytes,
			"Ciphertext should be <= %d bytes", tfhepkg.CiphertextMaxBytes)

		plaintext, err := tfhepkg.DecryptUint32(keys.ClientKey, ct)
		require.NoError(t, err, "DecryptUint32 must succeed")
		require.Equal(t, v, plaintext, "Dec(Enc(%d)) must equal %d", v, v)
	}
}

// TestTFHEHomomorphicAdd verifies: Dec(Enc(a) + Enc(b)) == a + b (mod 2^32).
func TestTFHEHomomorphicAdd(t *testing.T) {
	t.Log("Generating TFHE keys for homomorphic add test...")
	keys, err := tfhepkg.GenerateKeys()
	require.NoError(t, err)

	var a uint32 = 100
	var b uint32 = 200

	ctA, err := tfhepkg.EncryptUint32(keys.ClientKey, a)
	require.NoError(t, err)

	ctB, err := tfhepkg.EncryptUint32(keys.ClientKey, b)
	require.NoError(t, err)

	// Homomorphic addition — runs on server side (no decryption needed here).
	ctSum, err := tfhepkg.AddUint32(keys.ServerKey, ctA, ctB)
	require.NoError(t, err, "AddUint32 must succeed")
	require.NotEmpty(t, ctSum, "Result ciphertext must not be empty")

	// Decrypt the result.
	result, err := tfhepkg.DecryptUint32(keys.ClientKey, ctSum)
	require.NoError(t, err)
	require.Equal(t, a+b, result, "Dec(Enc(%d) + Enc(%d)) must equal %d", a, b, a+b)
}

// TestTFHEScalarMultiply verifies: Dec(Enc(a) * scalar) == a * scalar (mod 2^32).
func TestTFHEScalarMultiply(t *testing.T) {
	t.Log("Generating TFHE keys for scalar multiply test...")
	keys, err := tfhepkg.GenerateKeys()
	require.NoError(t, err)

	var a uint32 = 7
	var scalar uint32 = 6

	ctA, err := tfhepkg.EncryptUint32(keys.ClientKey, a)
	require.NoError(t, err)

	ctResult, err := tfhepkg.MultiplyScalarUint32(keys.ServerKey, ctA, scalar)
	require.NoError(t, err, "MultiplyScalarUint32 must succeed")

	result, err := tfhepkg.DecryptUint32(keys.ClientKey, ctResult)
	require.NoError(t, err)
	require.Equal(t, a*scalar, result, "Dec(Enc(%d) * %d) must equal %d", a, scalar, a*scalar)
}

// TestTFHECiphertextSize verifies ciphertext is within the expected ~21 KB range.
func TestTFHECiphertextSize(t *testing.T) {
	keys, err := tfhepkg.GenerateKeys()
	require.NoError(t, err)

	ct, err := tfhepkg.EncryptUint32(keys.ClientKey, 42)
	require.NoError(t, err)

	t.Logf("Actual FheUint32 ciphertext size: %d bytes (%.1f KB)", len(ct), float64(len(ct))/1024.0)

	// Expect within 10 KB – 32 KB (tfhe-rs FheUint32 defaults to ~16–21 KB)
	require.GreaterOrEqual(t, len(ct), 10*1024, "Ciphertext too small (< 10 KB)")
	require.LessOrEqual(t, len(ct), 32*1024, "Ciphertext too large (> 32 KB)")
}

// BenchmarkTFHEEncrypt measures FheUint32 encryption throughput.
func BenchmarkTFHEEncrypt(b *testing.B) {
	keys, _ := tfhepkg.GenerateKeys()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tfhepkg.EncryptUint32(keys.ClientKey, uint32(i)) //nolint:errcheck
	}
}

// BenchmarkTFHEAdd measures FheUint32 homomorphic addition throughput.
func BenchmarkTFHEAdd(b *testing.B) {
	keys, _ := tfhepkg.GenerateKeys()
	ct1, _ := tfhepkg.EncryptUint32(keys.ClientKey, 42)
	ct2, _ := tfhepkg.EncryptUint32(keys.ClientKey, 58)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tfhepkg.AddUint32(keys.ServerKey, ct1, ct2) //nolint:errcheck
	}
}
