use cosmwasm_std::StdError;
use thiserror::Error;

#[derive(Error, Debug)]
pub enum ContractError {
    #[error("{0}")]
    Std(#[from] StdError),

    #[error("Unauthorized: only the vault owner can perform this action")]
    Unauthorized {},

    #[error("TFHE operation failed: {reason}")]
    TFHEError { reason: String },

    #[error("Invalid ciphertext: {reason}")]
    InvalidCiphertext { reason: String },

    #[error("No encrypted balance — deposit at least once before transferring")]
    NoBalance {},

    #[error("Empty deposit: encrypted_amount ciphertext must not be empty")]
    EmptyDeposit {},
}
