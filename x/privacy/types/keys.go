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

	// EncryptedBalanceKeyPrefix is the KVStore key prefix for encrypted balances.
	// Full key: EncryptedBalanceKeyPrefix | address_bytes
	// The value is a serialised EncryptedCiphertext protobuf.
	EncryptedBalanceKeyPrefix = "enc_balance/"

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

// EncryptedBalanceKey returns the full KVStore key for an account's encrypted
// balance.  Using the raw AccAddress bytes keeps the key compact and avoids
// any bech32 parsing at read time.
func EncryptedBalanceKey(addr []byte) []byte {
	prefix := KeyPrefix(EncryptedBalanceKeyPrefix)
	key := make([]byte, len(prefix)+len(addr))
	copy(key, prefix)
	copy(key[len(prefix):], addr)
	return key
}
