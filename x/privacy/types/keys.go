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

	// CommitmentKeyPrefix is the KVStore key prefix for privacy commitments.
	// Full key: CommitmentKeyPrefix | address_bytes
	// The value is a raw 32-byte SHA-256 commitment hash.
	CommitmentKeyPrefix = "commitment/"

	// PQCBlockHashKeyPrefix is the KVStore key prefix for PQC block hashes.
	// Full key: PQCBlockHashKeyPrefix | big-endian int64 block height
	// Each value is a 32-byte LWR lattice hash produced at EndBlock.
	PQCBlockHashKeyPrefix = "pqc_hash/"

	// LatestPQCHashKey is the KVStore key that always holds the most recently
	// stored PQC hash (i.e. the hash from the last finalized block).
	LatestPQCHashKey = "pqc_hash/latest"

	// ── TFHE key prefixes ────────────────────────────────────────────────────

	// TFHEMetaKeyPrefix is the KVStore key prefix for TFHE shard metadata.
	// Full key: TFHEMetaKeyPrefix | commitmentHash (32 bytes)
	// Value: JSON-encoded TFHEShardMeta (commitmentHash → shardIndex → nodeIDs).
	TFHEMetaKeyPrefix = "tfhe_meta/"

	// TFHEResultKeyPrefix is the KVStore key prefix for cached TFHE evaluation results.
	// Full key: TFHEResultKeyPrefix | resultHash (32 bytes)
	// Value: raw ciphertext bytes of the homomorphic operation result.
	TFHEResultKeyPrefix = "tfhe_result/"

	// TFHEQuotaKeyPrefix is the KVStore key prefix for per-address TFHE submission quotas.
	// Full key: TFHEQuotaKeyPrefix | address_bytes
	// Value: big-endian uint64 count of active TFHE submissions for the address.
	// In v0.4 the per-account limit is 1 active commitment at a time.
	TFHEQuotaKeyPrefix = "tfhe_quota/"

	// ── Event type constants ──────────────────────────────────────────────────

	// EventTypeTFHESubmit is emitted when a TFHE ciphertext is submitted.
	EventTypeTFHESubmit = "tfhe_submit"

	// AttributeKeySender is the event attribute for the transaction sender.
	AttributeKeySender = "sender"

	// AttributeKeyRecipient is the event attribute for the transaction recipient.
	AttributeKeyRecipient = "recipient"

	// AttributeKeyCommitmentHash is the event attribute for the TFHE commitment hash.
	AttributeKeyCommitmentHash = "commitment_hash"

	// EventTypeInitCommitment is emitted when a privacy commitment is initialised.
	EventTypeInitCommitment = "init_commitment"

	// AttributeKeyCreator is the event attribute for the commitment creator.
	AttributeKeyCreator = "creator"

	// AttributeKeyDepositDenom is the event attribute for the deposit coin denomination.
	AttributeKeyDepositDenom = "deposit_denom"
)

func KeyPrefix(p string) []byte {
	return []byte(p)
}

// CommitmentKey returns the full KVStore key for an account's commitment.
func CommitmentKey(addr []byte) []byte {
	prefix := KeyPrefix(CommitmentKeyPrefix)
	key := make([]byte, len(prefix)+len(addr))
	copy(key, prefix)
	copy(key[len(prefix):], addr)
	return key
}

// TFHEMetaKey returns the full KVStore key for TFHE shard metadata.
// commitmentHash must be 32 bytes.
func TFHEMetaKey(commitmentHash []byte) []byte {
	prefix := KeyPrefix(TFHEMetaKeyPrefix)
	key := make([]byte, len(prefix)+len(commitmentHash))
	copy(key, prefix)
	copy(key[len(prefix):], commitmentHash)
	return key
}

// TFHEResultKey returns the full KVStore key for a cached TFHE result ciphertext.
// resultHash must be 32 bytes.
func TFHEResultKey(resultHash []byte) []byte {
	prefix := KeyPrefix(TFHEResultKeyPrefix)
	key := make([]byte, len(prefix)+len(resultHash))
	copy(key, prefix)
	copy(key[len(prefix):], resultHash)
	return key
}

// TFHEQuotaKey returns the full KVStore key tracking the active TFHE submission
// count for a given account address.
func TFHEQuotaKey(addr []byte) []byte {
	prefix := KeyPrefix(TFHEQuotaKeyPrefix)
	key := make([]byte, len(prefix)+len(addr))
	copy(key, prefix)
	copy(key[len(prefix):], addr)
	return key
}
