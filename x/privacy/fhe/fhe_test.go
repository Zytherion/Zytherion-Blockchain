package fhe_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"zytherion/x/privacy/fhe"
)

// TestNewContext verifies that a Context can be created without errors.
func TestNewContext(t *testing.T) {
	ctx, err := fhe.NewContext()
	require.NoError(t, err)
	require.NotNil(t, ctx)
	require.NotNil(t, ctx.SecretKey())
	require.NotNil(t, ctx.PublicKey())
}

// TestEncryptDecrypt verifies the round-trip: encrypt a value, decrypt it,
// and assert the original value is recovered exactly.
//
// Note: BFV arithmetic is modular with respect to the plaintext modulus t.
// For PN12QP109, t = 65537, so values must be in [0, t). Values >= t are
// automatically reduced modulo t — see TestModularWrapping below.
func TestEncryptDecrypt(t *testing.T) {
	ctx, err := fhe.NewContext()
	require.NoError(t, err)

	tests := []struct {
		name  string
		value uint64
	}{
		{"zero", 0},
		{"one", 1},
		// Use a value safely within PN12QP109's plaintext modulus t=65537.
		{"large_within_t", 65000},
		// The following would have failed with the old PN12QP109 params (t=65537)
		// but succeeds with our custom 40-bit prime t≈1.1 trillion.
		{"one_billion", 1_000_000_000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ct, err := ctx.Encrypt(tt.value)
			require.NoError(t, err, "Encrypt should not fail")
			require.NotNil(t, ct)

			got, err := ctx.Decrypt(ct)
			require.NoError(t, err, "Decrypt should not fail")
			require.Equal(t, tt.value, got, "decrypted value must match plaintext")
		})
	}
}

// TestModularWrapping documents BFV's modular arithmetic: values >= t (plaintext
// modulus) wrap around. With our custom params, t = 0x10000048001 = 1,099,511,955,457.
// Values in [0, t) are exact; Encrypt(t) decrypts to 0.
func TestModularWrapping(t *testing.T) {
	ctx, err := fhe.NewContext()
	require.NoError(t, err)

	// Retrieve t from the context itself so this test stays in sync with
	// any future parameter changes.
	plaintextModulus := ctx.PlaintextModulus()
	require.EqualValues(t, uint64(0x10000048001), plaintextModulus,
		"plaintext modulus should be the 40-bit prime 0x10000048001")

	ct, err := ctx.Encrypt(plaintextModulus)
	require.NoError(t, err)
	got, err := ctx.Decrypt(ct)
	require.NoError(t, err)
	// Encrypt(t) ≡ 0 (mod t)
	require.Equal(t, uint64(0), got, "Enc(t) should decrypt to 0 (mod t)")
}

// TestHomomorphicAdd verifies that adding two ciphertexts homomorphically
// produces a ciphertext whose decryption equals the sum of the plaintexts.
func TestHomomorphicAdd(t *testing.T) {
	ctx, err := fhe.NewContext()
	require.NoError(t, err)

	tests := []struct {
		a, b, want uint64
	}{
		{0, 0, 0},
		{1, 2, 3},
		{100, 200, 300},
		{999, 1, 1000},
	}

	for _, tt := range tests {
		ctA, err := ctx.Encrypt(tt.a)
		require.NoError(t, err)

		ctB, err := ctx.Encrypt(tt.b)
		require.NoError(t, err)

		ctSum, err := ctx.AddCiphertexts(ctA, ctB)
		require.NoError(t, err, "AddCiphertexts should not fail")

		got, err := ctx.Decrypt(ctSum)
		require.NoError(t, err)
		require.Equal(t, tt.want, got,
			"Decrypt(Enc(%d) + Enc(%d)) should equal %d", tt.a, tt.b, tt.want)
	}
}

// TestNilCiphertextErrors verifies graceful error handling for nil inputs.
func TestNilCiphertextErrors(t *testing.T) {
	ctx, err := fhe.NewContext()
	require.NoError(t, err)

	ct, err := ctx.Encrypt(42)
	require.NoError(t, err)

	_, err = ctx.AddCiphertexts(nil, ct)
	require.Error(t, err, "nil first argument should return error")

	_, err = ctx.AddCiphertexts(ct, nil)
	require.Error(t, err, "nil second argument should return error")

	_, err = ctx.Decrypt(nil)
	require.Error(t, err, "nil ciphertext decrypt should return error")
}
