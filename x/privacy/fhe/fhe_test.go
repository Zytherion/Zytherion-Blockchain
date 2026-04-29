//go:build !notfhe
// +build !notfhe

package fhe_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"zytherion/x/privacy/fhe"
)

func TestNewContext(t *testing.T) {
	ctx, err := fhe.NewContext()
	require.NoError(t, err)
	require.NotNil(t, ctx)
	ct, err := ctx.Encrypt(0)
	require.NoError(t, err)
	require.NotEmpty(t, ct)
}

func TestEncryptDecrypt(t *testing.T) {
	ctx, err := fhe.NewContext()
	require.NoError(t, err)

	tests := []struct {
		name  string
		value uint64
	}{
		{"zero", 0},
		{"one", 1},
		{"large", 65000},
		{"one_billion", 1_000_000_000},
		// NOTE: FheUint32 supports values up to 4,294,967,295 (2^32 - 1).
		{"max_u32", 4_294_967_295},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ct, err := ctx.Encrypt(tt.value)
			require.NoError(t, err)
			require.NotEmpty(t, ct)
			got, err := ctx.Decrypt(ct)
			require.NoError(t, err)
			require.Equal(t, tt.value, got)
		})
	}
}

func TestCompressDecompress(t *testing.T) {
	ctx, err := fhe.NewContext()
	require.NoError(t, err)

	original := uint64(999_888_777)
	compressed, err := ctx.Encrypt(original)
	require.NoError(t, err)

	decrypted, err := ctx.Decrypt(compressed)
	require.NoError(t, err)
	require.Equal(t, original, decrypted)

	require.Less(t, len(compressed), 12_000,
		"compressed ciphertext must be < 12KB (was %d bytes)", len(compressed))
}

func TestHomomorphicAdd(t *testing.T) {
	ctx, err := fhe.NewContext()
	require.NoError(t, err)

	tests := []struct{ a, b, want uint64 }{
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
		require.NoError(t, err)
		got, err := ctx.Decrypt(ctSum)
		require.NoError(t, err)
		require.Equal(t, tt.want, got)
	}
}

func TestHomomorphicSub(t *testing.T) {
	ctx, err := fhe.NewContext()
	require.NoError(t, err)

	a := uint64(1_000_000)
	b := uint64(250_000)
	ctA, _ := ctx.Encrypt(a)
	ctB, _ := ctx.Encrypt(b)
	ctResult, err := ctx.SubCiphertexts(ctA, ctB)
	require.NoError(t, err)
	result, err := ctx.Decrypt(ctResult)
	require.NoError(t, err)
	require.Equal(t, a-b, result)
}

func TestNilCiphertextErrors(t *testing.T) {
	ctx, err := fhe.NewContext()
	require.NoError(t, err)

	ct, _ := ctx.Encrypt(42)

	_, err = ctx.AddCiphertexts(nil, ct)
	require.Error(t, err)
	_, err = ctx.AddCiphertexts(ct, nil)
	require.Error(t, err)
	_, err = ctx.Decrypt(nil)
	require.Error(t, err)
	_, err = ctx.SubCiphertexts(nil, ct)
	require.Error(t, err)
	_, err = ctx.SubCiphertexts(ct, nil)
	require.Error(t, err)
}