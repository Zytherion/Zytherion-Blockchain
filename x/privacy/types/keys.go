package types

const (
	// ModuleName defines the module name
	ModuleName = "privacy"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// RouterKey defines the module's message routing key
	RouterKey = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_privacy"

	// CommitmentKeyPrefix is the KVStore key prefix for ZK commitments.
	// Full key: CommitmentKeyPrefix | address_bytes
	// The value is a raw 32-byte MiMC commitment (BN254 Fr element).
	CommitmentKeyPrefix = "commitment/"

	// NullifierKeyPrefix is the KVStore key prefix for spent nullifiers.
	// Full key: NullifierKeyPrefix | nullifier_bytes (32 bytes)
	// The value is a 1-byte sentinel ([]byte{1}) — we only need existence.
	// Nullifiers prevent double-spending of the same commitment.
	NullifierKeyPrefix = "nullifier/"

	// PQCBlockHashKeyPrefix is the KVStore key prefix for PQC block hashes.
	// Full key: PQCBlockHashKeyPrefix | big-endian int64 block height
	// Each value is a 32-byte LWE lattice hash produced at EndBlock.
	PQCBlockHashKeyPrefix = "pqc_hash/"

	// LatestPQCHashKey is the KVStore key that always holds the most recently
	// stored PQC hash (i.e. the hash from the last finalized block).
	// This allows O(1) access to the "previous PQC hash" at EndBlock without
	// needing to know the previous block height.
	LatestPQCHashKey = "pqc_hash/latest"
)

func KeyPrefix(p string) []byte {
	return []byte(p)
}

// CommitmentKey returns the full KVStore key for an account's ZK commitment.
// Using the raw AccAddress bytes keeps the key compact and avoids
// any bech32 parsing at read time.
func CommitmentKey(addr []byte) []byte {
	prefix := KeyPrefix(CommitmentKeyPrefix)
	key := make([]byte, len(prefix)+len(addr))
	copy(key, prefix)
	copy(key[len(prefix):], addr)
	return key
}

// NullifierKey returns the full KVStore key for a spent nullifier.
// The nullifier is 32 bytes (BN254 Fr element).
func NullifierKey(nullifier []byte) []byte {
	prefix := KeyPrefix(NullifierKeyPrefix)
	key := make([]byte, len(prefix)+len(nullifier))
	copy(key, prefix)
	copy(key[len(prefix):], nullifier)
	return key
}
