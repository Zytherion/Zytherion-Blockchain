// simd_opt.go — internal utility helpers shared by hashing.go and signature.go.
//
// Note: The previous SIMD-unrolled LatticeHash variant has been removed as
// part of the cryptographic refactor. The new GenerateBlockHash (SHA3-256)
// is already accelerated by the Go runtime's native SHA-3 implementation,
// which uses hardware intrinsics on supported platforms.
package pqc

// zeroPad32 returns a copy of b padded (or truncated) to exactly 32 bytes.
// This is used when normalising PrevHash fields that may arrive as nil or
// with an unexpected length (e.g., at genesis).
func zeroPad32(b []byte) []byte {
	out := make([]byte, HashSize)
	copy(out, b) // copy fills up to min(len(b), HashSize) bytes
	return out
}
