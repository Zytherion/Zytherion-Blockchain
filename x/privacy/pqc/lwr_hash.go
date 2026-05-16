// lwr_hash.go — Ring-LWR based One-Way Function for Zytherion block hashing.
//
// # Cryptographic Construction
//
// This file implements a hybrid LWR-SHA3 hash function as specified in the
// Zytherion whitepaper. The construction binds the hardness of Block-LWR
// (Learning With Rounding over ring Rq = Z_q[X]/(X^n + 1)) to the one-wayness
// of SHAKE-256. This deterministic rounding guarantees bit-identical outputs
// across all node architectures.
//
// # Parameters
//
//	n = 256    ring dimension (coefficients in Z_q)
//	q = 3329   prime modulus (Kyber's q)
//	p = 256    rounding modulus
//
// # Output size = 96 bytes
//
//	[  0:32] seed   — SHA3-256 of (input || prevHash)
//	[ 32:96] b_out  — first lwrOutputCoeffs (64) coefficients of b
//	                  as 1 byte each → 64 bytes
package pqc

import (
	"encoding/binary"
	"fmt"

	"golang.org/x/crypto/sha3"
)

// ── Public constants ──────────────────────────────────────────────────────────

const (
	// HashAlgorithm identifies the LWR-SHA3 hybrid construction used by
	// GenerateLWRBlockHash. Stored in block headers for protocol versioning.
	HashAlgorithm = "LWR-SHA3-Hybrid"

	// LWRHashSize is the fixed byte length of a GenerateLWRBlockHash output.
	// Layout: 32-byte seed || 64-byte serialised b coefficients = 96 bytes.
	LWRHashSize = 96

	// lwrN is the ring dimension: R_q = Z_q[X]/(X^n + 1).
	lwrN = 256

	// lwrQ is the plaintext modulus (Kyber's prime modulus).
	lwrQ = 3329

	// lwrP is the rounding modulus. Since p=256, rounded coefficients fit in 1 byte.
	lwrP = 256

	// lwrOutputCoeffs is how many coefficients of b are serialised into the
	// hash output. 64 × 1 = 64 bytes appended after the 32-byte seed.
	lwrOutputCoeffs = 64
)

// ── GenerateLWRBlockHash ──────────────────────────────────────────────────────

// GenerateLWRBlockHash computes a 96-byte LWR-SHA3 hybrid hash.
func GenerateLWRBlockHash(input []byte, prevHash []byte) ([]byte, error) {
	seed := computeSeed(input, prevHash)
	a := expandMatrix(seed)
	s := inputToSecret(input)
	
	// b = round(A·s) mod p
	// Computed as b = ((A·s) * p) / q mod p
	b := polyMulRound(a, s)

	out := make([]byte, LWRHashSize)
	copy(out[:32], seed)

	for i := 0; i < lwrOutputCoeffs; i++ {
		out[32+i] = byte(b[i])
	}

	return out, nil
}

func GenerateLWRBlockHashWithFallback(input BlockHashInput) []byte {
	data := canonicalise(input)
	h, err := GenerateLWRBlockHash(data, input.PrevHash)
	if err != nil {
		return GenerateBlockHash(input)
	}
	return h
}

func computeSeed(input, prevHash []byte) []byte {
	h := sha3.New256()
	h.Write(input)
	h.Write(prevHash)
	return h.Sum(nil)
}

func expandMatrix(seed []byte) [lwrN]int32 {
	xof := sha3.NewShake256()
	xof.Write([]byte("A"))
	xof.Write(seed)

	var a [lwrN]int32
	buf := make([]byte, 2)
	for i := 0; i < lwrN; {
		xof.Read(buf)
		v := int32(binary.LittleEndian.Uint16(buf))
		if v < lwrQ {
			a[i] = v
			i++
		}
	}
	return a
}

func inputToSecret(input []byte) [lwrN]int32 {
	var s [lwrN]int32
	if len(input) == 0 {
		return s
	}
	bitIdx := 0
	for i := 0; i < lwrN; i++ {
		byteIdx := (bitIdx / 8) % len(input)
		shift := uint(bitIdx % 8)
		b0 := (input[byteIdx] >> shift) & 0x01
		b1 := (input[(byteIdx+1)%len(input)] >> ((shift + 1) % 8)) & 0x01
		two := int32(b0) | (int32(b1) << 1)
		switch two {
		case 0:
			s[i] = -1
		case 3:
			s[i] = 1
		default:
			s[i] = 0
		}
		bitIdx += 2
	}
	return s
}

func polyMulRound(a, s [lwrN]int32) [lwrN]int32 {
	var b [lwrN]int32
	for i := 0; i < lwrN; i++ {
		for j := 0; j < lwrN; j++ {
			k := (i + j) % lwrN
			prod := a[i] * s[j]
			if i+j >= lwrN {
				b[k] -= prod
			} else {
				b[k] += prod
			}
		}
	}
	// Round mod p without floats: (x * p) / q
	for k := 0; k < lwrN; k++ {
		// First reduce mod q to keep it bounded
		val := ((b[k] % lwrQ) + lwrQ) % lwrQ
		
		// Deterministic rounding
		rounded := (val * lwrP) / lwrQ
		
		b[k] = rounded % lwrP
	}
	return b
}

func ValidateLWRHash(h []byte) error {
	if len(h) != LWRHashSize {
		return fmt.Errorf("invalid LWR hash: expected %d bytes, got %d", LWRHashSize, len(h))
	}
	for i := 0; i < lwrOutputCoeffs; i++ {
		coeff := h[32+i]
		if int32(coeff) >= lwrP {
			return fmt.Errorf("invalid LWR hash: coefficient %d out of range [0,%d)", i, lwrP)
		}
	}
	return nil
}
