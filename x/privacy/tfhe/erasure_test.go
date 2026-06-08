// erasure_test.go — Unit tests for Reed-Solomon erasure coding.

package tfhe_test

import (
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/require"

	tfhepkg "zytherion/x/privacy/tfhe"
)

// TestErasureSplitAndReconstructAll verifies that all 16 shards reconstruct
// the original data correctly with no missing shards.
func TestErasureSplitAndReconstructAll(t *testing.T) {
	// Simulate a ~21 KB ciphertext.
	original := make([]byte, 21*1024)
	_, err := rand.Read(original)
	require.NoError(t, err)

	shardResults, err := tfhepkg.Split(original)
	require.NoError(t, err)
	require.Len(t, shardResults, tfhepkg.ErasureTotalShards)

	recovered, err := tfhepkg.ReconstructFromResults(shardResults, len(original))
	require.NoError(t, err)
	require.Equal(t, original, recovered, "Reconstructed data must match original")
}

// TestErasureReconstructWithMinimumShards verifies reconstruction with exactly
// the minimum required 12 shards (4 missing).
func TestErasureReconstructWithMinimumShards(t *testing.T) {
	original := make([]byte, 21*1024)
	_, err := rand.Read(original)
	require.NoError(t, err)

	shardResults, err := tfhepkg.Split(original)
	require.NoError(t, err)

	// Drop the last 4 shards (parity shards) — keep only 12 data shards.
	minShards := shardResults[:tfhepkg.ErasureDataShards]

	recovered, err := tfhepkg.ReconstructFromResults(minShards, len(original))
	require.NoError(t, err)
	require.Equal(t, original, recovered,
		"Must reconstruct from exactly %d shards", tfhepkg.ErasureDataShards)
}

// TestErasureReconstructWith4Missing verifies reconstruction when 4 arbitrary
// shards are missing (maximum tolerable loss with ParityShards=4).
func TestErasureReconstructWith4Missing(t *testing.T) {
	original := make([]byte, 21*1024)
	_, err := rand.Read(original)
	require.NoError(t, err)

	shardResults, err := tfhepkg.Split(original)
	require.NoError(t, err)

	// Drop shards at indices 0, 3, 7, 13 — arbitrary 4 shards.
	drop := map[int]bool{0: true, 3: true, 7: true, 13: true}
	var available []tfhepkg.ShardResult
	for _, sr := range shardResults {
		if !drop[sr.Index] {
			available = append(available, sr)
		}
	}
	require.Len(t, available, 12, "Must have 12 available shards")

	recovered, err := tfhepkg.ReconstructFromResults(available, len(original))
	require.NoError(t, err)
	require.Equal(t, original, recovered,
		"Must reconstruct with 4 arbitrary shards missing")
}

// TestErasureInsufficientShards verifies reconstruction fails when fewer than
// ErasureDataShards shards are available.
func TestErasureInsufficientShards(t *testing.T) {
	original := make([]byte, 21*1024)
	_, err := rand.Read(original)
	require.NoError(t, err)

	shardResults, err := tfhepkg.Split(original)
	require.NoError(t, err)

	// Provide only 11 shards (below minimum of 12).
	_, err = tfhepkg.ReconstructFromResults(shardResults[:11], len(original))
	require.Error(t, err, "Reconstruction with 11 shards must fail")
}

// TestErasureShardCount verifies Split produces exactly 16 shards with
// the v0.4 parameters (DataShards=12, ParityShards=4).
func TestErasureShardCount(t *testing.T) {
	require.Equal(t, 16, tfhepkg.ErasureTotalShards)
	require.Equal(t, 12, tfhepkg.ErasureDataShards)
	require.Equal(t, 4, tfhepkg.ErasureParityShards)
}

// BenchmarkErasureSplit measures sharding throughput for a 21 KB payload.
func BenchmarkErasureSplit(b *testing.B) {
	data := make([]byte, 21*1024)
	rand.Read(data) //nolint:errcheck
	b.ResetTimer()
	b.SetBytes(int64(len(data)))
	for i := 0; i < b.N; i++ {
		tfhepkg.Split(data) //nolint:errcheck
	}
}

// BenchmarkErasureReconstruct measures reconstruction throughput with all 16 shards.
func BenchmarkErasureReconstruct(b *testing.B) {
	data := make([]byte, 21*1024)
	rand.Read(data) //nolint:errcheck
	shards, _ := tfhepkg.Split(data)
	b.ResetTimer()
	b.SetBytes(int64(len(data)))
	for i := 0; i < b.N; i++ {
		tfhepkg.ReconstructFromResults(shards, len(data)) //nolint:errcheck
	}
}
