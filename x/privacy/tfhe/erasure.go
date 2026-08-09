// erasure.go — Reed-Solomon erasure coding for TFHE ciphertext sharding.
//
// # Scheme
//
// A TFHE ciphertext (~21 KB) is split into N=16 shards using Reed-Solomon
// erasure coding with the following parameters:
//
//	DataShards:   12  (minimum shards required for reconstruction)
//	ParityShards:  4  (redundancy shards, allows up to 4 missing shards)
//	TotalShards:  16  (= DataShards + ParityShards)
//
// This means any 12 of the 16 distributed shards are sufficient to reconstruct
// the original ciphertext — tolerating the loss or unavailability of up to 4 nodes.
//
// # Storage sizing
//
// For a 21 KB ciphertext:
//   - Each shard: ~1.75 KB   (21 KB / 12 data shards)
//   - Total storage across all nodes: 16 × 1.75 KB ≈ 28 KB
//
// # Library
//
// Uses github.com/klauspost/reedsolomon — a pure-Go, battle-tested
// Reed-Solomon implementation with optional SIMD acceleration.
package tfhe

import (
	"errors"
	"fmt"

	"github.com/klauspost/reedsolomon"
)

const (
	// ErasureDataShards is the minimum number of shards required to reconstruct
	// the original ciphertext.
	ErasureDataShards = 12

	// ErasureParityShards is the number of redundancy shards. Up to this many
	// shards can be missing or corrupt and reconstruction still succeeds.
	ErasureParityShards = 4

	// ErasureTotalShards is the total number of shards produced by Split.
	ErasureTotalShards = ErasureDataShards + ErasureParityShards

	// CiphertextMaxBytes is the maximum allowed TFHE FheUint32 ciphertext size.
	// Also used as the pre-allocation buffer size for Rust FFI calls.
	// tfhe-rs 0.6 with default params serialises FheUint32 to ~200-400 KB;
	// 512 KB provides safe headroom for all parameter variations.
	CiphertextMaxBytes = 512 * 1024
)

// newEncoder returns a cached Reed-Solomon encoder.
// In production, consider caching this at package init for performance.
func newEncoder() (reedsolomon.Encoder, error) {
	enc, err := reedsolomon.New(ErasureDataShards, ErasureParityShards)
	if err != nil {
		return nil, fmt.Errorf("erasure: failed to create Reed-Solomon encoder: %w", err)
	}
	return enc, nil
}

// ShardResult holds a single shard produced by Split.
type ShardResult struct {
	// Index is the shard index in [0, ErasureTotalShards).
	// Indices 0..11 are data shards; 12..15 are parity shards.
	Index int

	// Data is the shard payload bytes.
	Data []byte
}

// Split divides data into ErasureTotalShards=16 Reed-Solomon shards.
//
// Returns a slice of ErasureTotalShards ShardResult values.
// The caller is responsible for distributing shards to peers.
//
// Thread-safe: each call creates its own encoder and shard set.
func Split(data []byte) ([]ShardResult, error) {
	if len(data) == 0 {
		return nil, errors.New("erasure: data must not be empty")
	}

	enc, err := newEncoder()
	if err != nil {
		return nil, err
	}

	// Split into data shards. This pads the data to make it divisible by
	// ErasureDataShards and allocates the parity shards as nil slices.
	shards, err := enc.Split(data)
	if err != nil {
		return nil, fmt.Errorf("erasure: Split failed: %w", err)
	}

	// Compute parity shards in-place.
	if err := enc.Encode(shards); err != nil {
		return nil, fmt.Errorf("erasure: Encode (parity computation) failed: %w", err)
	}

	// Wrap into ShardResult for transport.
	results := make([]ShardResult, ErasureTotalShards)
	for i, s := range shards {
		cp := make([]byte, len(s))
		copy(cp, s)
		results[i] = ShardResult{
			Index: i,
			Data:  cp,
		}
	}

	return results, nil
}

// Reconstruct recovers the original data from available shards.
//
// shards must be a slice of exactly ErasureTotalShards entries.
// Missing shards must be represented as nil (not zero-length slices).
// At least ErasureDataShards shards must be non-nil for reconstruction to succeed.
//
// Returns the original data (same byte content as passed to Split, without padding).
func Reconstruct(shards [][]byte, originalLen int) ([]byte, error) {
	if len(shards) != ErasureTotalShards {
		return nil, fmt.Errorf("erasure: expected %d shards, got %d",
			ErasureTotalShards, len(shards))
	}

	// Count available shards.
	available := 0
	for _, s := range shards {
		if s != nil {
			available++
		}
	}
	if available < ErasureDataShards {
		return nil, fmt.Errorf("erasure: insufficient shards: have %d, need at least %d",
			available, ErasureDataShards)
	}

	enc, err := newEncoder()
	if err != nil {
		return nil, err
	}

	// Reconstruct missing shards.
	if err := enc.Reconstruct(shards); err != nil {
		return nil, fmt.Errorf("erasure: Reconstruct failed: %w", err)
	}

	// Verify data integrity.
	ok, err := enc.Verify(shards)
	if err != nil {
		return nil, fmt.Errorf("erasure: Verify failed: %w", err)
	}
	if !ok {
		return nil, errors.New("erasure: reconstructed shards failed integrity check")
	}

	// Join data shards and trim to originalLen.
	var out []byte
	for i := 0; i < ErasureDataShards; i++ {
		out = append(out, shards[i]...)
	}
	if originalLen > 0 && originalLen <= len(out) {
		out = out[:originalLen]
	}

	return out, nil
}

// ReconstructFromResults reconstructs original data from a collection of ShardResults.
//
// results must include at least ErasureDataShards entries (any subset of the 16 shards).
// Missing shards are filled with nil automatically.
func ReconstructFromResults(results []ShardResult, originalLen int) ([]byte, error) {
	shards := make([][]byte, ErasureTotalShards)
	for _, r := range results {
		if r.Index < 0 || r.Index >= ErasureTotalShards {
			return nil, fmt.Errorf("erasure: invalid shard index %d", r.Index)
		}
		shards[r.Index] = r.Data
	}
	return Reconstruct(shards, originalLen)
}
