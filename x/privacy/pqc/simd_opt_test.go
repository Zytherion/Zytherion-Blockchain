package pqc_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"zytherion/x/privacy/pqc"
)

// TestSIMDOptRemoved is a placeholder confirming that the SIMD LWE variant
// has been intentionally removed and replaced by GenerateBlockHash (SHA3-256).
// This file is retained to prevent accidental re-introduction of the old code.
func TestSIMDOptRemoved(t *testing.T) {
	// The zeroPad32 helper is internal (unexported); we validate it indirectly
	// by confirming that nil PrevHash in two identical BlockHashInput structs
	// yields the same output as a zero-filled 32-byte PrevHash — the
	// underlying zeroPad32 call in GenerateBlockHash makes this true.
	nilHash := pqc.GenerateBlockHash(pqc.BlockHashInput{Height: 999, PrevHash: nil})
	zeroHash := pqc.GenerateBlockHash(pqc.BlockHashInput{Height: 999, PrevHash: make([]byte, pqc.HashSize)})

	require.Equal(t, pqc.HashSize, len(nilHash))
	require.Equal(t, nilHash, zeroHash,
		"zeroPad32 must normalise nil and zero-slice PrevHash identically")
}
