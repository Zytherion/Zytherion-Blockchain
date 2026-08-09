use cosmwasm_std::Binary;
use schemars::JsonSchema;
use serde::{Deserialize, Serialize};

/// Instantiation message
#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct InstantiateMsg {
    /// Human-readable label for this vault
    pub label: String,
    /// The owner who can initiate transfers (bech32 address)
    pub owner: String,
}

/// Execute messages
#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
#[serde(rename_all = "snake_case")]
pub enum ExecuteMsg {
    /// Deposit an encrypted amount into the vault.
    ///
    /// The caller provides a ciphertext (FheUint32 encrypted with the node's
    /// client key). The vault accumulates encrypted balances using homomorphic
    /// addition — no validator or observer can see the actual amount.
    Deposit {
        /// Encrypted amount (base64-encoded FheUint32 ciphertext)
        encrypted_amount: Binary,
        /// Human-readable tag for this deposit (NOT secret — stored as an event)
        memo: Option<String>,
    },

    /// Request a homomorphic transfer to another account.
    ///
    /// The transfer amount is encrypted — neither validators nor node operators
    /// can learn the transferred value.
    Transfer {
        /// Recipient address (bech32)
        to: String,
        /// Encrypted transfer amount ciphertext
        encrypted_amount: Binary,
        /// Optional public memo (visible on-chain)
        memo: Option<String>,
    },
}

/// Query messages
#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
#[serde(rename_all = "snake_case")]
pub enum QueryMsg {
    /// Get the vault's current encrypted balance.
    /// Returns the raw ciphertext — NOT the plaintext.
    EncryptedBalance {},

    /// Get vault metadata (label, owner, counts).
    VaultInfo {},

    /// Compute the homomorphic sum of two external ciphertexts.
    /// Demonstrates that any caller can request TFHE addition through the vault.
    HomomorphicAdd { ct1: Binary, ct2: Binary },
}

// ── TFHE Custom Query Types ───────────────────────────────────────────────────

/// Envelope sent to the chain's TFHE subsystem via QueryRequest::Custom.
/// Exactly one variant must be set.
#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
#[serde(rename_all = "snake_case")]
pub enum TFHECustomQuery {
    /// Encrypt a uint32 plaintext using the node's client key.
    TfheEncrypt { value: u32 },

    /// Homomorphically add two FheUint32 ciphertexts.
    TfheAdd { ct1: Binary, ct2: Binary },

    /// Multiply a ciphertext by a plaintext scalar.
    TfheMulScalar { ciphertext: Binary, scalar: u32 },

    /// Decrypt a ciphertext (testing/demo only — reveals plaintext!).
    TfheDecrypt { ciphertext: Binary },

    /// Verify that a ciphertext is structurally well-formed.
    TfheVerify { ciphertext: Binary },
}

// ── TFHE Response Types ───────────────────────────────────────────────────────

/// Response returned by tfhe_encrypt, tfhe_add, tfhe_mul_scalar.
#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct TFHECiphertextResponse {
    /// Serialized FheUint32 ciphertext (base64 in JSON)
    pub ciphertext: Binary,
    /// Size of the ciphertext in bytes
    pub size_bytes: u64,
}

/// Response returned by tfhe_decrypt.
#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct TFHEPlaintextResponse {
    /// The decrypted uint32 value
    pub value: u32,
}

/// Response returned by tfhe_verify.
#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct TFHEVerifyResponse {
    /// Whether the ciphertext appears valid
    pub valid: bool,
    /// Size of the ciphertext in bytes
    pub size_bytes: u64,
    /// Human-readable validation message
    pub message: Option<String>,
}

// ── Contract Response Types ───────────────────────────────────────────────────

/// Response for the EncryptedBalance query.
#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct EncryptedBalanceResponse {
    /// Ciphertext of the current balance (FheUint32) — NOT the plaintext!
    pub encrypted_balance: Binary,
    /// Number of deposits accumulated into this vault
    pub deposit_count: u64,
    /// Whether the vault has ever received a deposit
    pub has_balance: bool,
}

/// Response for the VaultInfo query.
#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct VaultInfoResponse {
    pub label: String,
    pub owner: String,
    pub deposit_count: u64,
    pub transfer_count: u64,
}
