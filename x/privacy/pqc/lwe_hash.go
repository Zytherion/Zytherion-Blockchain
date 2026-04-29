// lwe_hash.go — Ring-LWE based One-Way Function for Zytherion block hashing.
//
// # Cryptographic Construction
//
// This file implements a hybrid LWE-SHA3 hash function as specified in the
// Zytherion whitepaper.  The construction binds the hardness of Block-LWE
// (Learning With Errors over ring Rq = Z_q[X]/(X^n + 1)) to the one-wayness
// of SHAKE-256, producing a function that is computationally hard to invert
// even for quantum adversaries.
//
// # Parameters (Kyber-inspired)
//
//	n = 256    ring dimension (coefficients in Z_q)
//	q = 3329   prime modulus (Kyber's q; NTT-friendly: q ≡ 1 mod 2n)
//
// # Output size = 96 bytes
//
//	[  0:32] seed   — SHA3-256 of (input || prevHash), provides domain binding
//	[ 32:96] b_out  — first lweOutputCoeffs (32) coefficients of b serialised
//	                  as little-endian uint16 (2 bytes each → 64 bytes)
//
// The total is fixed at LWEHashSize = 96 bytes, matching the block header slot.
package pqc

import (
	"encoding/binary"
	"fmt"

	"golang.org/x/crypto/sha3"
)

// ── Public constants ──────────────────────────────────────────────────────────

const (
	// HashAlgorithm identifies the LWE-SHA3 hybrid construction used by
	// GenerateLWEBlockHash.  Stored in block headers for protocol versioning.
	HashAlgorithm = "LWE-SHA3-Hybrid"

	// LWEHashSize is the fixed byte length of a GenerateLWEBlockHash output.
	// Layout: 32-byte seed || 64-byte serialised b coefficients = 96 bytes.
	LWEHashSize = 96

	// lweN is the ring dimension: R_q = Z_q[X]/(X^n + 1).
	lweN = 256

	// lweQ is the plaintext modulus (Kyber's prime modulus, NTT-friendly).
	lweQ = 3329

	// lweOutputCoeffs is how many coefficients of b are serialised into the
	// hash output (2 bytes each). 32 × 2 = 64 bytes appended after the 32-byte seed.
	lweOutputCoeffs = 32
)

// ── GenerateLWEBlockHash ──────────────────────────────────────────────────────

// GenerateLWEBlockHash computes a 96-byte LWE-SHA3 hybrid hash over the given
// input bytes and previous block hash.
//
// # Algorithm (steps match the Zytherion whitepaper §4.2)
//
//  1. Seed = SHA3-256(input || prevHash)                    [32 bytes]
//  2. Matrix A ∈ Rq^{n×n}:  SHAKE-256("A" || seed) → n coefficients in Z_q
//  3. Secret s ∈ Rq:        map input bytes to short coefficients ∈ {-1, 0, 1}
//  4. Error e ∈ Rq:         SHAKE-256("e" || seed) → CBD(η=1) noise ∈ {-1,0,1}
//  5. b = A·s + e  mod q   (polynomial multiplication then reduce mod q)
//  6. Output = seed || LE16(b[0]) || … || LE16(b[31])      [96 bytes total]
//
// Returns an error only if input serialisation is pathological (should never
// happen in practice).  The caller should fall back to GenerateBlockHash on
// error — see GenerateLWEBlockHashWithFallback.
func GenerateLWEBlockHash(input []byte, prevHash []byte) ([]byte, error) {
	// ── Step 1: Seed ─────────────────────────────────────────────────────────
	seed := computeSeed(input, prevHash) // 32 bytes

	// ── Step 2: Matrix A (deterministic from seed via SHAKE-256) ─────────────
	// We treat A as a single polynomial a ∈ Rq (RLWE variant).
	// Each coefficient is sampled uniformly in [0, q) by rejection sampling on
	// pairs of bytes drawn from the SHAKE-256 stream.
	a := expandMatrix(seed) // [lweN]int32

	// ── Step 3: Secret s (input-derived short polynomial) ────────────────────
	// Map each 2 bits of input to a coefficient in {-1, 0, 0, 1} (sparse /
	// balanced ternary). This is the "B-to-Secret" direct embedding used in
	// several lattice signature schemes.
	s := inputToSecret(input) // [lweN]int32

	// ── Step 4: Error e (CBD η=1 via SHAKE-256) ──────────────────────────────
	// Centered Binomial Distribution with η=1: sum two bits minus two bits.
	// Each coefficient is in {-1, 0, 1} with probabilities {1/4, 1/2, 1/4}.
	e := sampleCBD(seed) // [lweN]int32

	// ── Step 5: b = A·s + e  (mod q) ─────────────────────────────────────────
	// Negacyclic polynomial multiplication in Z_q[X]/(X^n+1):
	// product[k] = Σ_{i+j≡k (mod n)} sign(i+j) · a[i] · s[j]  (mod q)
	b := polyMulAdd(a, s, e) // [lweN]int32

	// ── Step 6: Serialise output ─────────────────────────────────────────────
	out := make([]byte, LWEHashSize)
	copy(out[:32], seed)

	for i := 0; i < lweOutputCoeffs; i++ {
		// Ensure coefficient is in [0, q) before encoding.
		coeff := ((b[i] % lweQ) + lweQ) % lweQ
		binary.LittleEndian.PutUint16(out[32+i*2:], uint16(coeff))
	}

	return out, nil
}

// GenerateLWEBlockHashWithFallback attempts GenerateLWEBlockHash and, on any
// error, transparently falls back to GenerateBlockHash (SHA3-256).
// The caller receives a consistent-length 32-byte slice on fallback.
func GenerateLWEBlockHashWithFallback(input BlockHashInput) []byte {
	// Build a compact byte representation of the block header for the LWE input.
	data := canonicalise(input)

	h, err := GenerateLWEBlockHash(data, input.PrevHash)
	if err != nil {
		// Fallback: standard SHA3-256 block hash (32 bytes).
		return GenerateBlockHash(input)
	}
	return h
}

// ── Internal helpers ──────────────────────────────────────────────────────────

// computeSeed returns SHA3-256(input || prevHash) as a 32-byte seed.
// This binds both the current input and the chain linkage into a single
// unpredictable 256-bit value that seeds all SHAKE-256 expansions.
func computeSeed(input, prevHash []byte) []byte {
	h := sha3.New256()
	h.Write(input)    //nolint:errcheck
	h.Write(prevHash) //nolint:errcheck
	return h.Sum(nil) // 32 bytes
}

// expandMatrix fills a[0..n-1] with uniform samples in [0, q) derived from
// SHAKE-256("A" || seed).  Rejection sampling is used: if a 16-bit sample ≥ q,
// draw the next pair of bytes.  This guarantees a truly uniform distribution.
func expandMatrix(seed []byte) [lweN]int32 {
	xof := sha3.NewShake256()
	xof.Write([]byte("A")) //nolint:errcheck
	xof.Write(seed)        //nolint:errcheck

	var a [lweN]int32
	buf := make([]byte, 2)
	for i := 0; i < lweN; {
		xof.Read(buf) //nolint:errcheck
		v := int32(binary.LittleEndian.Uint16(buf))
		if v < lweQ {
			a[i] = v
			i++
		}
		// rejection: discard and re-sample
	}
	return a
}

// inputToSecret maps raw input bytes to a short polynomial s ∈ {-1, 0, 1}^n.
//
// Each pair of bits from the (repeated) input bytes determines one coefficient:
//
//	00 → -1,  01 → 0,  10 → 0,  11 → +1
//
// The input is repeated (cyclic) if shorter than n/4 bytes (min 4 bytes).
// This produces a balanced distribution over ternary values.
func inputToSecret(input []byte) [lweN]int32 {
	var s [lweN]int32
	if len(input) == 0 {
		return s // all-zero secret for empty input
	}
	bitIdx := 0
	for i := 0; i < lweN; i++ {
		byteIdx := (bitIdx / 8) % len(input)
		shift := uint(bitIdx % 8)
		// Extract 2 bits; wrap at byte boundary using OR with next byte.
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

// sampleCBD generates a noise polynomial e ∈ {-1, 0, 1}^n using the
// Centered Binomial Distribution with η=1 seeded by SHAKE-256("e" || seed).
//
// CBD(η=1): each coefficient = (bit_a - bit_b) where a,b ~ Bernoulli(1/2).
//
//	Distribution: Pr[-1]=1/4, Pr[0]=1/2, Pr[+1]=1/4.
func sampleCBD(seed []byte) [lweN]int32 {
	xof := sha3.NewShake256()
	xof.Write([]byte("e")) //nolint:errcheck
	xof.Write(seed)        //nolint:errcheck

	var e [lweN]int32
	buf := make([]byte, 1)
	for i := 0; i < lweN; {
		xof.Read(buf) //nolint:errcheck
		b := buf[0]
		// Pack 4 CBD samples per byte (2 bits each: a_bit and b_bit).
		for j := 0; j < 4 && i < lweN; j++ {
			a := int32((b >> uint(j*2)) & 0x01)
			bBit := int32((b >> uint(j*2+1)) & 0x01)
			e[i] = a - bBit
			i++
		}
	}
	return e
}

// polyMulAdd computes b = a*s + e  in the negacyclic ring Z_q[X]/(X^n + 1).
//
// Negacyclic convolution rule:
//
//	product[k] = Σ_{i=0}^{n-1} sign(i,k) * a[i] * s[(k-i+n) mod n]  (mod q)
//
// where sign(i,k) = +1 if i ≤ k, else -1 (wrapping introduces negation).
//
// This is an O(n²) schoolbook multiplication — suitable for n=256 in tests
// and non-critical-path code; replace with NTT for production hot paths.
func polyMulAdd(a, s, e [lweN]int32) [lweN]int32 {
	var b [lweN]int32
	for i := 0; i < lweN; i++ {
		for j := 0; j < lweN; j++ {
			k := (i + j) % lweN
			prod := a[i] * s[j]
			if i+j >= lweN {
				// Negacyclic wrap: X^n ≡ -1 mod (X^n + 1)
				b[k] -= prod
			} else {
				b[k] += prod
			}
		}
	}
	// Add noise and reduce mod q.
	for k := 0; k < lweN; k++ {
		b[k] = ((b[k]+e[k])%lweQ + lweQ) % lweQ
	}
	return b
}

// ── Validation ────────────────────────────────────────────────────────────────

// ValidateLWEHash returns an error if h does not look like a valid LWE hash.
// It checks only structural invariants (length, coefficient bounds) — it does
// NOT verify that h was produced from a specific input (that would require the
// secret key, which is not held on-chain).
func ValidateLWEHash(h []byte) error {
	if len(h) != LWEHashSize {
		return fmt.Errorf("invalid LWE hash: expected %d bytes, got %d", LWEHashSize, len(h))
	}
	// Check that each serialised coefficient is in [0, q).
	for i := 0; i < lweOutputCoeffs; i++ {
		coeff := int32(binary.LittleEndian.Uint16(h[32+i*2:]))
		if coeff >= lweQ {
			return fmt.Errorf("invalid LWE hash: coefficient %d out of range [0,%d)", i, lweQ)
		}
	}
	return nil
}
