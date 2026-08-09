use cw_storage_plus::Item;
use serde::{Deserialize, Serialize};

/// Vault configuration stored at instantiation time.
#[derive(Serialize, Deserialize, Clone, Debug, PartialEq)]
pub struct VaultConfig {
    /// Human-readable label for this vault instance
    pub label: String,
    /// Owner bech32 address (the only account that can transfer out)
    pub owner: String,
}

/// Vault statistics counters.
#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, Default)]
pub struct VaultStats {
    /// Total number of successful deposits
    pub deposit_count: u64,
    /// Total number of transfer operations recorded
    pub transfer_count: u64,
}

// ── Storage Keys ──────────────────────────────────────────────────────────────

/// Vault configuration (label, owner).
pub const VAULT_CONFIG: Item<VaultConfig> = Item::new("vault_config");

/// Vault deposit/transfer statistics.
pub const VAULT_STATS: Item<VaultStats> = Item::new("vault_stats");

/// Encrypted balance stored as raw bytes (serialized FheUint32 ciphertext).
///
/// This is a homomorphically accumulated sum of all deposits.
/// The value is completely opaque to validators — only the holder of
/// the ClientKey can decrypt it.
pub const ENCRYPTED_BALANCE: Item<Vec<u8>> = Item::new("encrypted_balance");
