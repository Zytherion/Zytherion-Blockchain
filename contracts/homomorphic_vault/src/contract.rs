#[cfg(not(feature = "library"))]
use cosmwasm_std::entry_point;

use cosmwasm_std::{
    from_binary, to_binary, Binary, Deps, DepsMut, Env, MessageInfo, QueryRequest, Response,
    StdResult,
};

use crate::error::ContractError;
use crate::msg::{
    EncryptedBalanceResponse, ExecuteMsg, InstantiateMsg, QueryMsg, TFHECiphertextResponse,
    TFHECustomQuery, VaultInfoResponse,
};
use crate::state::{VaultConfig, VaultStats, ENCRYPTED_BALANCE, VAULT_CONFIG, VAULT_STATS};

// ── Entry Points ──────────────────────────────────────────────────────────────

/// Instantiate the homomorphic vault.
///
/// Sets up vault configuration and zeroes the statistics counters.
/// No encrypted balance is stored yet — it is created on first deposit.
#[cfg_attr(not(feature = "library"), entry_point)]
pub fn instantiate(
    deps: DepsMut,
    _env: Env,
    _info: MessageInfo,
    msg: InstantiateMsg,
) -> Result<Response, ContractError> {
    // Validate the owner address
    deps.api.addr_validate(&msg.owner)?;

    VAULT_CONFIG.save(
        deps.storage,
        &VaultConfig {
            label: msg.label.clone(),
            owner: msg.owner.clone(),
        },
    )?;

    VAULT_STATS.save(deps.storage, &VaultStats::default())?;

    Ok(Response::new()
        .add_attribute("action", "instantiate")
        .add_attribute("label", msg.label)
        .add_attribute("owner", msg.owner))
}

/// Execute a state-changing operation on the vault.
#[cfg_attr(not(feature = "library"), entry_point)]
pub fn execute(
    deps: DepsMut,
    _env: Env,
    info: MessageInfo,
    msg: ExecuteMsg,
) -> Result<Response, ContractError> {
    match msg {
        ExecuteMsg::Deposit {
            encrypted_amount,
            memo,
        } => execute_deposit(deps, info, encrypted_amount, memo),
        ExecuteMsg::Transfer {
            to,
            encrypted_amount,
            memo,
        } => execute_transfer(deps, info, to, encrypted_amount, memo),
    }
}

/// Handle read-only queries.
#[cfg_attr(not(feature = "library"), entry_point)]
pub fn query(deps: Deps, _env: Env, msg: QueryMsg) -> StdResult<Binary> {
    match msg {
        QueryMsg::EncryptedBalance {} => to_binary(&query_encrypted_balance(deps)?),
        QueryMsg::VaultInfo {} => to_binary(&query_vault_info(deps)?),
        QueryMsg::HomomorphicAdd { ct1, ct2 } => {
            to_binary(&query_homomorphic_add(deps, ct1, ct2)?)
        }
    }
}

// ── Execute Handlers ──────────────────────────────────────────────────────────

/// Deposit an encrypted amount into the vault.
///
/// The vault accumulates encrypted balances using TFHE homomorphic addition.
/// If this is the first deposit, the ciphertext is stored directly.
/// For subsequent deposits the chain's TFHE subsystem adds the new ciphertext
/// to the running encrypted total — with no party learning the amounts.
fn execute_deposit(
    deps: DepsMut,
    _info: MessageInfo,
    encrypted_amount: Binary,
    memo: Option<String>,
) -> Result<Response, ContractError> {
    let new_ct = encrypted_amount.to_vec();
    if new_ct.is_empty() {
        return Err(ContractError::EmptyDeposit {});
    }

    // Accumulate encrypted balance
    let updated_balance = match ENCRYPTED_BALANCE.may_load(deps.storage)? {
        Some(existing_ct) => {
            // Homomorphic addition via the TFHE subsystem —
            // the chain evaluates Enc(a) + Enc(b) without learning a or b.
            let add_query = TFHECustomQuery::TfheAdd {
                ct1: Binary::from(existing_ct),
                ct2: Binary::from(new_ct),
            };
            let request: QueryRequest<TFHECustomQuery> = QueryRequest::Custom(add_query);
            let serialized_request = to_binary(&request).map_err(|e| ContractError::TFHEError {
                reason: format!("failed to serialize TFHE query: {}", e),
            })?;

            let raw_result = deps
                .querier
                .raw_query(&serialized_request)
                .unwrap()
                .map_err(|e| ContractError::TFHEError {
                    reason: format!("TFHE query failed: {}", e),
                })?;

            let result: TFHECiphertextResponse =
                from_binary(&raw_result).map_err(|e| ContractError::TFHEError {
                    reason: format!("failed to parse TFHE response: {}", e),
                })?;

            result.ciphertext.to_vec()
        }
        None => {
            // First deposit — store the ciphertext directly as the initial balance
            new_ct
        }
    };

    ENCRYPTED_BALANCE.save(deps.storage, &updated_balance)?;

    // Update deposit counter
    let mut stats = VAULT_STATS.load(deps.storage)?;
    stats.deposit_count += 1;
    VAULT_STATS.save(deps.storage, &stats)?;

    let mut resp = Response::new()
        .add_attribute("action", "deposit")
        .add_attribute("ct_size_bytes", updated_balance.len().to_string());

    if let Some(m) = memo {
        resp = resp.add_attribute("memo", m);
    }

    Ok(resp)
}

/// Record a homomorphic transfer from this vault to another address.
///
/// Access control: only the vault owner may initiate a transfer.
///
/// In this implementation the transfer amount is recorded as an event
/// attribute (as an opaque ciphertext). A production system would also
/// subtract the amount from the vault's encrypted balance using TFHE
/// subtraction, and credit it to a recipient vault contract.
fn execute_transfer(
    deps: DepsMut,
    info: MessageInfo,
    to: String,
    encrypted_amount: Binary,
    memo: Option<String>,
) -> Result<Response, ContractError> {
    // Only the owner may transfer
    let config = VAULT_CONFIG.load(deps.storage)?;
    if info.sender.as_str() != config.owner {
        return Err(ContractError::Unauthorized {});
    }

    // Ensure the vault has a balance
    let balance_ct = ENCRYPTED_BALANCE
        .may_load(deps.storage)?
        .ok_or(ContractError::NoBalance {})?;
    if balance_ct.is_empty() {
        return Err(ContractError::NoBalance {});
    }

    let transfer_ct = encrypted_amount.to_vec();

    // Update transfer counter
    let mut stats = VAULT_STATS.load(deps.storage)?;
    stats.transfer_count += 1;
    VAULT_STATS.save(deps.storage, &stats)?;

    let mut resp = Response::new()
        .add_attribute("action", "transfer")
        .add_attribute("to", to)
        .add_attribute("transfer_ct_size", transfer_ct.len().to_string());

    if let Some(m) = memo {
        resp = resp.add_attribute("memo", m);
    }

    Ok(resp)
}

// ── Query Handlers ────────────────────────────────────────────────────────────

/// Return the vault's current encrypted balance ciphertext (never the plaintext).
fn query_encrypted_balance(deps: Deps) -> StdResult<EncryptedBalanceResponse> {
    let stats = VAULT_STATS.load(deps.storage)?;

    match ENCRYPTED_BALANCE.may_load(deps.storage)? {
        Some(ct) => Ok(EncryptedBalanceResponse {
            encrypted_balance: Binary::from(ct),
            deposit_count: stats.deposit_count,
            has_balance: true,
        }),
        None => Ok(EncryptedBalanceResponse {
            encrypted_balance: Binary::from(vec![]),
            deposit_count: 0,
            has_balance: false,
        }),
    }
}

/// Return vault metadata (label, owner, stats counters).
fn query_vault_info(deps: Deps) -> StdResult<VaultInfoResponse> {
    let config = VAULT_CONFIG.load(deps.storage)?;
    let stats = VAULT_STATS.load(deps.storage)?;
    Ok(VaultInfoResponse {
        label: config.label,
        owner: config.owner,
        deposit_count: stats.deposit_count,
        transfer_count: stats.transfer_count,
    })
}

/// Perform a homomorphic addition of two external ciphertexts.
///
/// This query demonstrates that arbitrary TFHE operations can be triggered
/// through the contract's query interface. Neither the contract, the caller,
/// nor any validator learns the plaintext values of ct1 or ct2.
fn query_homomorphic_add(
    deps: Deps,
    ct1: Binary,
    ct2: Binary,
) -> StdResult<TFHECiphertextResponse> {
    let add_query = TFHECustomQuery::TfheAdd { ct1, ct2 };
    let request: QueryRequest<TFHECustomQuery> = QueryRequest::Custom(add_query);
    let serialized = to_binary(&request)?;

    let raw_result = deps
        .querier
        .raw_query(&serialized)
        .unwrap()
        .map_err(|system_err| {
            cosmwasm_std::StdError::generic_err(format!(
                "TFHE homomorphic add query failed: {}",
                system_err
            ))
        })?;

    let result: TFHECiphertextResponse = from_binary(&raw_result)?;
    Ok(result)
}
