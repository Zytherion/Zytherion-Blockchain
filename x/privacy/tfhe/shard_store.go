// shard_store.go — Disk-based local storage for TFHE ciphertext shards.
//
// Each node stores the shards it is responsible for at:
//   ~/.zytherion/tfhe_shards/<hex(commitmentHash)>/<shardIndex>.shard
//
// Metadata (which shards live on which nodes) is stored in the Cosmos SDK
// KV store and accessed via the Keeper's TFHEMeta methods.
package tfhe

import (
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// ShardStore manages disk storage of TFHE ciphertext shards for the local node.
type ShardStore struct {
	// baseDir is the root directory for shard storage.
	// Default: ~/.zytherion/tfhe_shards/
	baseDir string
}

// NewShardStore creates a ShardStore rooted at baseDir.
// The directory is created if it does not exist.
func NewShardStore(baseDir string) (*ShardStore, error) {
	if err := os.MkdirAll(baseDir, 0o700); err != nil {
		return nil, fmt.Errorf("shard_store: failed to create base dir %q: %w", baseDir, err)
	}
	return &ShardStore{baseDir: baseDir}, nil
}

// shardDir returns the directory path for all shards belonging to a commitment.
func (s *ShardStore) shardDir(commitmentHash []byte) string {
	return filepath.Join(s.baseDir, hex.EncodeToString(commitmentHash))
}

// shardPath returns the full path for a specific shard.
func (s *ShardStore) shardPath(commitmentHash []byte, shardIndex int) string {
	return filepath.Join(s.shardDir(commitmentHash), fmt.Sprintf("%02d.shard", shardIndex))
}

// StoreShard saves a shard to disk under the commitment's directory.
// Overwrites any existing shard at that index.
func (s *ShardStore) StoreShard(commitmentHash []byte, shardIndex int, data []byte) error {
	if len(commitmentHash) == 0 {
		return errors.New("shard_store: commitmentHash must not be empty")
	}
	if shardIndex < 0 || shardIndex >= ErasureTotalShards {
		return fmt.Errorf("shard_store: invalid shard index %d (must be 0..%d)",
			shardIndex, ErasureTotalShards-1)
	}
	if len(data) == 0 {
		return errors.New("shard_store: shard data must not be empty")
	}

	dir := s.shardDir(commitmentHash)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("shard_store: failed to create shard directory: %w", err)
	}

	path := s.shardPath(commitmentHash, shardIndex)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("shard_store: failed to write shard %d: %w", shardIndex, err)
	}
	return nil
}

// LoadShard reads a shard from disk.
// Returns (nil, false) if the shard does not exist on this node.
func (s *ShardStore) LoadShard(commitmentHash []byte, shardIndex int) ([]byte, bool) {
	path := s.shardPath(commitmentHash, shardIndex)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	return data, true
}

// HasShard reports whether this node stores a given shard.
func (s *ShardStore) HasShard(commitmentHash []byte, shardIndex int) bool {
	path := s.shardPath(commitmentHash, shardIndex)
	_, err := os.Stat(path)
	return err == nil
}

// DeleteShards removes all shards for a commitment from disk.
// Used when a commitment expires or is pruned.
func (s *ShardStore) DeleteShards(commitmentHash []byte) error {
	dir := s.shardDir(commitmentHash)
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("shard_store: failed to delete shards for %x: %w", commitmentHash, err)
	}
	return nil
}

// ListLocalShards returns the shard indices stored locally for a commitment.
func (s *ShardStore) ListLocalShards(commitmentHash []byte) ([]int, error) {
	dir := s.shardDir(commitmentHash)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("shard_store: failed to list shards: %w", err)
	}

	var indices []int
	for _, e := range entries {
		var idx int
		if _, err := fmt.Sscanf(e.Name(), "%02d.shard", &idx); err == nil {
			indices = append(indices, idx)
		}
	}
	return indices, nil
}

// ── Dilithium5 Signature Storage ──────────────────────────────────────────────

// signaturePath returns the file path for the Dilithium5 signature of a shard.
// Signatures are stored as <index>.sig alongside <index>.shard.
func (s *ShardStore) signaturePath(commitmentHash []byte, shardIndex int) string {
	return filepath.Join(s.shardDir(commitmentHash), fmt.Sprintf("%02d.sig", shardIndex))
}

// StoreShardSignature persists the Dilithium5 signature for a shard to disk.
// sig should be exactly pqc.DilithiumSignatureSize bytes (4595 bytes).
// No-op if sig is empty (signing not configured).
func (s *ShardStore) StoreShardSignature(commitmentHash []byte, shardIndex int, sig []byte) error {
	if len(sig) == 0 {
		return nil
	}
	dir := s.shardDir(commitmentHash)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("shard_store: failed to create shard directory for signature: %w", err)
	}
	path := s.signaturePath(commitmentHash, shardIndex)
	if err := os.WriteFile(path, sig, 0o600); err != nil {
		return fmt.Errorf("shard_store: failed to write signature for shard %d: %w", shardIndex, err)
	}
	return nil
}

// LoadShardSignature reads the Dilithium5 signature for a shard from disk.
// Returns (nil, false) if no signature file exists (signing was not configured
// when the shard was stored, or the shard was stored by an older node).
func (s *ShardStore) LoadShardSignature(commitmentHash []byte, shardIndex int) ([]byte, bool) {
	path := s.signaturePath(commitmentHash, shardIndex)
	sig, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	return sig, true
}

// ── TFHEMetadata ──────────────────────────────────────────────────────────────

// Shard represents a single erasure shard with authenticity metadata.
// Validators verify Signature and MerkleProof before accepting storage.
type Shard struct {
	// Index is the shard position in [0, ErasureTotalShards).
	Index uint8

	// Data is the raw shard payload bytes.
	Data []byte

	// Signature is the Dilithium5 (ML-DSA-87) signature of SHA-256(Data)
	// produced by the block proposer's Dilithium key pair.
	// Absent in stub/testing builds (len == 0).
	Signature []byte

	// MerkleProof proves that this shard belongs to the committed Merkle tree.
	// Serialised as a flat concatenation of 32-byte sibling hashes (leaf→root order).
	MerkleProof []byte
}

// TFHEShardMeta holds metadata about where shards for a commitment are stored.
// This is stored in the Cosmos KV store so all nodes can look up shard locations.
type TFHEShardMeta struct {
	// CommitmentHash is the 32-byte SHA-256 hash of the original ciphertext.
	CommitmentHash []byte

	// OriginalLen is the byte length of the original ciphertext before erasure coding.
	OriginalLen int

	// ShardNodeMap maps shardIndex → list of nodeID strings that hold that shard.
	// Each shard is replicated to ReplicationFactor nodes.
	ShardNodeMap map[int][]string

	// MerkleRoot is the 32-byte root of the Merkle tree built over all 16 shard hashes.
	// Stored on-chain so validators can verify individual shard proofs.
	MerkleRoot []byte

	// ProposerPubkey is the Dilithium5 public key of the block proposer that
	// submitted this ciphertext. Used to verify shard Signatures.
	ProposerPubkey []byte
}

// ReplicationFactor is how many nodes store each shard.
// Increased from 3 to 4 in v0.4 for higher fault tolerance.
const ReplicationFactor = 4
