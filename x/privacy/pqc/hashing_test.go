package pqc_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"

	"zytherion/x/privacy/pqc"
)

// ─── GenerateBlockHash ────────────────────────────────────────────────────────

// TestBlockHashConsistency verifies that GenerateBlockHash is deterministic:
// the same BlockHashInput must always produce the same 32-byte output.
func TestBlockHashConsistency(t *testing.T) {
	input := pqc.BlockHashInput{
		Height:       42,
		PrevHash:     make([]byte, pqc.HashSize),
		AppHash:      []byte("apphash-0x1234"),
		Transactions: [][]byte{[]byte("tx1"), []byte("tx2")},
	}

	h1 := pqc.GenerateBlockHash(input)
	h2 := pqc.GenerateBlockHash(input)

	require.Equal(t, pqc.HashSize, len(h1), "output must be exactly 32 bytes")
	require.True(t, bytes.Equal(h1, h2),
		"GenerateBlockHash must be deterministic: same input must produce same output")
}

// TestBlockHashDomainIsolation verifies that changing the block height — even
// by a single increment — produces a completely different hash.  This ensures
// that domain-separation constants are actually mixed into the input.
func TestBlockHashDomainIsolation(t *testing.T) {
	base := pqc.BlockHashInput{
		Height:       100,
		PrevHash:     make([]byte, pqc.HashSize),
		AppHash:      []byte("static-apphash"),
		Transactions: nil,
	}
	next := base
	next.Height = 101

	hBase := pqc.GenerateBlockHash(base)
	hNext := pqc.GenerateBlockHash(next)

	require.False(t, bytes.Equal(hBase, hNext),
		"blocks at different heights must have different hashes")
}

// TestBlockHashAvalanche verifies that a single-byte change in a transaction
// produces a completely different block hash (avalanche effect).
func TestBlockHashAvalanche(t *testing.T) {
	tx := []byte("alice sends 100 ZYT to bob")
	modified := make([]byte, len(tx))
	copy(modified, tx)
	modified[0] ^= 0x01

	base := pqc.BlockHashInput{Height: 1, Transactions: [][]byte{tx}}
	alt := pqc.BlockHashInput{Height: 1, Transactions: [][]byte{modified}}

	hBase := pqc.GenerateBlockHash(base)
	hAlt := pqc.GenerateBlockHash(alt)

	require.False(t, bytes.Equal(hBase, hAlt),
		"single-byte change in transaction must alter block hash")

	diff := 0
	for i := 0; i < pqc.HashSize; i++ {
		if hBase[i] != hAlt[i] {
			diff++
		}
	}
	t.Logf("Avalanche: %d/%d bytes differ after single-byte tx change", diff, pqc.HashSize)
	require.Greater(t, diff, pqc.HashSize/4,
		"avalanche effect too weak (expected >%d differing bytes, got %d)", pqc.HashSize/4, diff)
}

// TestBlockHashPrevHashChaining verifies that changing PrevHash changes the
// block hash, ensuring the chain is properly linked.
func TestBlockHashPrevHashChaining(t *testing.T) {
	prev1 := make([]byte, pqc.HashSize)
	prev2 := make([]byte, pqc.HashSize)
	prev2[0] = 0xFF

	h1 := pqc.GenerateBlockHash(pqc.BlockHashInput{Height: 5, PrevHash: prev1})
	h2 := pqc.GenerateBlockHash(pqc.BlockHashInput{Height: 5, PrevHash: prev2})

	require.False(t, bytes.Equal(h1, h2),
		"different prevHash must produce different block hash")
}

// TestBlockHashNilPrevHash verifies that a nil PrevHash is treated identically
// to a zero-padded 32-byte slice (genesis block behaviour).
func TestBlockHashNilPrevHash(t *testing.T) {
	inputNil := pqc.BlockHashInput{Height: 1, PrevHash: nil}
	inputZero := pqc.BlockHashInput{Height: 1, PrevHash: make([]byte, pqc.HashSize)}

	hNil := pqc.GenerateBlockHash(inputNil)
	hZero := pqc.GenerateBlockHash(inputZero)

	require.True(t, bytes.Equal(hNil, hZero),
		"nil and zero-padded PrevHash must produce identical block hash")
}

// BenchmarkGenerateBlockHash measures hashing throughput for a typical block.
func BenchmarkGenerateBlockHash(b *testing.B) {
	input := pqc.BlockHashInput{
		Height:       1000,
		PrevHash:     make([]byte, pqc.HashSize),
		AppHash:      []byte("benchapphash"),
		Transactions: [][]byte{[]byte("tx-a"), []byte("tx-b"), []byte("tx-c")},
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pqc.GenerateBlockHash(input)
	}
}
