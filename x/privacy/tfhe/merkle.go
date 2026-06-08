// merkle.go — Binary Merkle Tree over TFHE erasure shards.
//
// # Purpose
//
// After splitting a TFHE ciphertext into 16 Reed-Solomon shards, we build a
// Merkle tree over the shard data so that:
//
//  1. The Merkle root is stored on-chain inside TFHEShardMeta.
//  2. Each shard receiver gets a MerkleProof, allowing them to verify their
//     shard is authentic without downloading all 16 shards.
//  3. The proposer signs the root with Dilithium5, binding the entire shard
//     set to their identity.
//
// # Tree Structure
//
//	Leaves  : SHA-256(shard[i].Data)  for i in [0, ErasureTotalShards)
//	Interior: SHA-256(left || right)
//	Padding : If ErasureTotalShards is not a power of two, leaves are padded
//	          with SHA-256([]byte{0x00}) until the next power-of-two boundary.
//
// ErasureTotalShards=16 is already a power of two, so no padding is needed.
package tfhe

import (
	"crypto/sha256"
	"errors"
	"fmt"
)

// MerkleTree represents a binary Merkle tree over shard hashes.
type MerkleTree struct {
	// nodes holds all tree nodes level by level, starting from the root.
	// nodes[0] = root (1 node)
	// nodes[1] = level 1 (2 nodes)
	// ...
	// nodes[depth] = leaves (ErasureTotalShards nodes)
	nodes [][]Hash

	// depth is log2(numLeaves).
	depth int

	// numLeaves is the number of leaves (padded to power of two).
	numLeaves int
}

// Hash is a 32-byte SHA-256 digest.
type Hash [32]byte

// BuildMerkleTree constructs a binary Merkle tree from a slice of shard results.
//
// shards must have exactly ErasureTotalShards entries.
// Returns the tree with the root hash accessible via Root().
func BuildMerkleTree(shards []ShardResult) (*MerkleTree, error) {
	if len(shards) != ErasureTotalShards {
		return nil, fmt.Errorf("merkle: expected %d shards, got %d", ErasureTotalShards, len(shards))
	}

	// Compute leaf hashes = SHA-256(shard.Data).
	leaves := make([]Hash, ErasureTotalShards)
	for _, sr := range shards {
		if sr.Index < 0 || sr.Index >= ErasureTotalShards {
			return nil, fmt.Errorf("merkle: shard index %d out of range", sr.Index)
		}
		leaves[sr.Index] = sha256.Sum256(sr.Data)
	}

	// ErasureTotalShards=16 is already a power of two.
	numLeaves := ErasureTotalShards
	depth := log2(numLeaves)

	// Build tree bottom-up. Store levels in reverse (leaves → root).
	levels := make([][]Hash, depth+1)
	levels[depth] = leaves

	for lvl := depth - 1; lvl >= 0; lvl-- {
		childLevel := levels[lvl+1]
		parentSize := len(childLevel) / 2
		parentLevel := make([]Hash, parentSize)
		for i := 0; i < parentSize; i++ {
			parentLevel[i] = hashPair(childLevel[2*i], childLevel[2*i+1])
		}
		levels[lvl] = parentLevel
	}

	// Reverse so that nodes[0] = root, nodes[depth] = leaves.
	return &MerkleTree{
		nodes:     levels,
		depth:     depth,
		numLeaves: numLeaves,
	}, nil
}

// Root returns the Merkle root hash of the tree.
func (t *MerkleTree) Root() Hash {
	return t.nodes[0][0]
}

// RootBytes returns the Merkle root as a 32-byte slice.
func (t *MerkleTree) RootBytes() []byte {
	r := t.Root()
	return r[:]
}

// MerkleProof is a list of sibling hashes required to verify a leaf.
// The proof is ordered from the leaf level up to the root.
type MerkleProof struct {
	// LeafIndex is the index of the leaf being proved.
	LeafIndex int

	// Path contains sibling hashes from leaf level to just below root.
	// len(Path) == depth of the tree.
	Path []Hash
}

// ProofForShard returns the Merkle proof for shard at shardIndex.
func (t *MerkleTree) ProofForShard(shardIndex int) (*MerkleProof, error) {
	if shardIndex < 0 || shardIndex >= t.numLeaves {
		return nil, fmt.Errorf("merkle: shard index %d out of range [0, %d)", shardIndex, t.numLeaves)
	}

	proof := &MerkleProof{
		LeafIndex: shardIndex,
		Path:      make([]Hash, t.depth),
	}

	idx := shardIndex
	for lvl := t.depth; lvl > 0; lvl-- {
		siblingIdx := idx ^ 1 // XOR with 1 flips even↔odd
		proof.Path[t.depth-lvl] = t.nodes[lvl][siblingIdx]
		idx = idx / 2
	}

	return proof, nil
}

// VerifyProof verifies that shardData at shardIndex belongs to a tree with
// the given root. Returns nil if valid, error otherwise.
func VerifyProof(root Hash, shardIndex int, shardData []byte, proof *MerkleProof) error {
	if proof == nil {
		return errors.New("merkle: proof is nil")
	}
	if proof.LeafIndex != shardIndex {
		return fmt.Errorf("merkle: proof.LeafIndex (%d) != shardIndex (%d)", proof.LeafIndex, shardIndex)
	}

	current := sha256.Sum256(shardData)
	idx := shardIndex

	for _, sibling := range proof.Path {
		if idx%2 == 0 {
			// current is left child
			current = hashPair(current, sibling)
		} else {
			// current is right child
			current = hashPair(sibling, current)
		}
		idx /= 2
	}

	if current != root {
		return fmt.Errorf("merkle: proof verification failed for shard %d: computed root %x != expected %x",
			shardIndex, current, root)
	}
	return nil
}

// ── Internal helpers ──────────────────────────────────────────────────────────

// hashPair computes SHA-256(left || right).
func hashPair(left, right Hash) Hash {
	data := make([]byte, 64)
	copy(data[:32], left[:])
	copy(data[32:], right[:])
	return sha256.Sum256(data)
}

// log2 returns the base-2 logarithm for powers of two.
// Panics if n is not a power of two.
func log2(n int) int {
	if n <= 0 || (n&(n-1)) != 0 {
		panic(fmt.Sprintf("merkle: log2: %d is not a positive power of two", n))
	}
	d := 0
	for n > 1 {
		n >>= 1
		d++
	}
	return d
}
