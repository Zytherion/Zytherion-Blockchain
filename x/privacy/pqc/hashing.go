// Package pqc implements Post-Quantum Cryptographic (PQC) primitives for the
// Zytherion blockchain.
//
// # Design rationale
//
// Previous versions relied on a custom LWE-based lattice hash. While LWE is
// a sound hardness assumption, a bespoke hash construction without a formal
// security proof or NIST standardisation is inappropriate for production
// blockchains. This revision replaces it with:
//
//  1. SHA3-256 (FIPS 202) as the primary hash primitive — thoroughly audited,
//     hardware-accelerated, and already quantum-resistant as a hash function
//     (Grover's algorithm only halves the effective bit-security, leaving
//     SHA3-256 with ~128-bit PQ security).
//
//  2. SHAKE-256 as an optional pre-processing expansion layer. SHAKE-256 is
//     the same Keccak permutation as SHA3 in XOF mode, making the two
//     composable without introducing new assumptions.
//
//  3. Dilithium3 (see signature.go) for validator signatures — a NIST PQC
//     round-3 winner based on the Module-LWE / Module-SIS hard problems.
//
// Domain separation ("ZYTHERION_BLOCK_HASH_V1") ensures that hashes produced
// by this function are cryptographically isolated from hashes produced by any
// other sub-system, preventing cross-context collision attacks.
package pqc

import (
	"encoding/binary"

	"golang.org/x/crypto/sha3"
)

// HashSize is the byte length of a GenerateBlockHash output (SHA3-256 digest).
const HashSize = 32

// blockHashDomain is prepended to every hash input for domain separation.
// Changing this constant invalidates all previously computed block hashes,
// effectively requiring a hard fork. Do NOT change without a protocol upgrade.
const blockHashDomain = "ZYTHERION_BLOCK_HASH_V1"

// BlockHashInput holds the structured block data fed into GenerateBlockHash.
// All fields are serialised in canonical order before hashing, ensuring
// identical output across all validators.
type BlockHashInput struct {
	// Height is the current block height (≥ 1 for normal blocks, 0 only at genesis).
	Height int64

	// PrevHash is the GenerateBlockHash output of the previous block.
	// For block 1 (first block after genesis), this should be a 32-byte zero slice.
	PrevHash []byte

	// AppHash is the multistore commit hash from the previous block (the
	// standard Cosmos SDK AppHash), anchoring the PQC hash chain to the
	// conventional Merkle proof chain.
	AppHash []byte

	// Transactions is the flat list of raw transaction bytes in this block,
	// in the same order as they appear in the block proposal.
	Transactions [][]byte
}

// GenerateBlockHash produces a deterministic, domain-separated 32-byte block
// hash using SHA3-256, optionally pre-processed through SHAKE-256.
//
// # Algorithm
//
//  1. Canonical serialisation:
//     data = 8-byte-BE(height) ‖ 32-byte(prevHash) ‖ appHash ‖ tx₀ ‖ tx₁ ‖ …
//
//  2. SHAKE-256 expansion (128-byte pre-image padding):
//     expansion = SHAKE256(domain ‖ data)[0:128]
//     This step mixes the domain into the entropy before SHA3 and prevents
//     length-extension with different domain strings.
//
//  3. SHA3-256 final digest:
//     hash = SHA3-256(domain ‖ data ‖ expansion)
//
// # Properties
//   - Deterministic: given the same BlockHashInput, always returns the same hash.
//   - Domain-isolated: hashes are bound to "ZYTHERION_BLOCK_HASH_V1".
//   - Quantum-resistant: SHA3-256 retains ~128-bit security against Grover's.
//   - Thread-safe: no shared mutable state; each call is self-contained.
func GenerateBlockHash(input BlockHashInput) []byte {
	// ── Step 1: Canonical serialisation ──────────────────────────────────────
	data := canonicalise(input)

	// ── Step 2: SHAKE-256 expansion layer ────────────────────────────────────
	// Pre-hashing through SHAKE-256 with the domain prefix creates a fixed-size
	// "salt" that is fully data-dependent. This layer is an additional defense-
	// in-depth against length-extension and second-preimage attacks.
	const expansionSize = 128
	expansion := make([]byte, expansionSize)
	xof := sha3.NewShake256()
	xof.Write([]byte(blockHashDomain)) //nolint:errcheck // SHAKE never errors
	xof.Write(data)                    //nolint:errcheck
	xof.Read(expansion)               //nolint:errcheck

	// ── Step 3: SHA3-256 final digest ────────────────────────────────────────
	hasher := sha3.New256()
	hasher.Write([]byte(blockHashDomain)) //nolint:errcheck
	hasher.Write(data)                    //nolint:errcheck
	hasher.Write(expansion)               //nolint:errcheck
	return hasher.Sum(nil)
}

// canonicalise serialises a BlockHashInput into a deterministic byte slice.
//
// Layout:
//
//	[0:8]              height (big-endian int64)
//	[8:40]             prevHash (32 bytes, zero-padded if nil/short)
//	[40:40+len(ah)]    appHash
//	[40+len(ah):]      transactions concatenated in order
func canonicalise(input BlockHashInput) []byte {
	// Normalise prevHash to exactly 32 bytes.
	prevHash := zeroPad32(input.PrevHash)

	// Pre-compute total capacity to avoid re-allocation.
	total := 8 + HashSize + len(input.AppHash)
	for _, tx := range input.Transactions {
		total += len(tx)
	}

	buf := make([]byte, 0, total)

	// 8-byte big-endian height.
	var hb [8]byte
	binary.BigEndian.PutUint64(hb[:], uint64(input.Height))
	buf = append(buf, hb[:]...)

	// 32-byte previous block hash.
	buf = append(buf, prevHash...)

	// Application hash from commit of the previous block.
	buf = append(buf, input.AppHash...)

	// All transactions in proposal order.
	for _, tx := range input.Transactions {
		buf = append(buf, tx...)
	}

	return buf
}
