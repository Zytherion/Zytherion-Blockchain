//! tfhe_c — C-compatible FFI wrapper for tfhe-rs FheUint32 operations.
//!
//! # Design
//!
//! All data crosses the FFI boundary as raw byte buffers (ptr + len).
//! This avoids any Rust ownership issues at the boundary and keeps the CGo
//! side simple. Rust objects (ClientKey, ServerKey, FheUint32) are
//! serialized with bincode before returning to Go and deserialized on
//! every call — a small overhead acceptable for a PoC.
//!
//! # Exported Functions (C signature)
//!
//! ```c
//! // Generate a ClientKey + ServerKey pair. Writes serialized bytes into
//! // caller-allocated buffers. Returns 0 on success, -1 on error.
//! int32_t tfhe_keygen(
//!     uint8_t *ck_out, uint64_t *ck_len,
//!     uint8_t *sk_out, uint64_t *sk_len
//! );
//!
//! // Encrypt a uint32 value. ciphertext_out must be pre-allocated to
//! // TFHE_CIPHERTEXT_MAX_BYTES. Returns actual ciphertext length or -1.
//! int64_t tfhe_encrypt_u32(
//!     const uint8_t *ck_bytes, uint64_t ck_len,
//!     uint32_t plaintext,
//!     uint8_t *ciphertext_out, uint64_t out_len
//! );
//!
//! // Homomorphic addition of two FheUint32 ciphertexts.
//! int64_t tfhe_add_u32(
//!     const uint8_t *sk_bytes, uint64_t sk_len,
//!     const uint8_t *c1_bytes, uint64_t c1_len,
//!     const uint8_t *c2_bytes, uint64_t c2_len,
//!     uint8_t *result_out, uint64_t out_len
//! );
//!
//! // Decrypt a FheUint32 ciphertext. Returns the plaintext value or -1 on error.
//! int64_t tfhe_decrypt_u32(
//!     const uint8_t *ck_bytes, uint64_t ck_len,
//!     const uint8_t *ciphertext_bytes, uint64_t ct_len
//! );
//!
//! // Returns the maximum ciphertext size in bytes (for buffer pre-allocation).
//! uint64_t tfhe_ciphertext_max_bytes(void);
//! ```

use std::slice;
use tfhe::{generate_keys, set_server_key, ConfigBuilder, FheUint32, CompressedPublicKey};
use tfhe::prelude::*;

// ── Constants ─────────────────────────────────────────────────────────────────

/// Maximum buffer size for a serialised FheUint32 ciphertext.
/// tfhe-rs 0.6 FheUint32 with default params serialises to ~200-400 KB.
/// We allocate 512 KB to leave headroom for all parameter variations.
const CIPHERTEXT_MAX_BYTES: u64 = 512 * 1024; // 512 KB

/// Maximum buffer size for a serialised ClientKey.
const CLIENT_KEY_MAX_BYTES: u64 = 16 * 1024 * 1024; // 16 MB

/// Maximum buffer size for a serialised ServerKey.
const SERVER_KEY_MAX_BYTES: u64 = 128 * 1024 * 1024; // 128 MB

// ── Helper macros ─────────────────────────────────────────────────────────────

/// Convert a raw pointer + length into a Rust byte slice.
/// SAFETY: caller must guarantee the pointer is valid for `len` bytes.
unsafe fn slice_from_raw(ptr: *const u8, len: u64) -> &'static [u8] {
    if ptr.is_null() || len == 0 {
        &[]
    } else {
        slice::from_raw_parts(ptr, len as usize)
    }
}

/// Convert a raw mut pointer + length into a mutable Rust byte slice.
unsafe fn slice_mut_from_raw(ptr: *mut u8, len: u64) -> &'static mut [u8] {
    if ptr.is_null() || len == 0 {
        &mut []
    } else {
        slice::from_raw_parts_mut(ptr, len as usize)
    }
}

// ── Exported FFI functions ────────────────────────────────────────────────────

/// Returns the maximum serialised ciphertext size in bytes.
/// CGo callers use this to pre-allocate output buffers.
#[no_mangle]
pub extern "C" fn tfhe_ciphertext_max_bytes() -> u64 {
    CIPHERTEXT_MAX_BYTES
}

/// Returns the maximum serialised ClientKey size.
#[no_mangle]
pub extern "C" fn tfhe_client_key_max_bytes() -> u64 {
    CLIENT_KEY_MAX_BYTES
}

/// Returns the maximum serialised ServerKey size.
#[no_mangle]
pub extern "C" fn tfhe_server_key_max_bytes() -> u64 {
    SERVER_KEY_MAX_BYTES
}

/// Generate a TFHE ClientKey + ServerKey pair.
///
/// Parameters:
///   ck_out  : caller-allocated buffer for serialised ClientKey (≥ CLIENT_KEY_MAX_BYTES)
///   ck_len  : pointer to u64; on success set to actual ClientKey byte count
///   sk_out  : caller-allocated buffer for serialised ServerKey (≥ SERVER_KEY_MAX_BYTES)
///   sk_len  : pointer to u64; on success set to actual ServerKey byte count
///
/// Returns 0 on success, -1 on any error.
#[no_mangle]
pub extern "C" fn tfhe_keygen(
    ck_out: *mut u8,
    ck_len: *mut u64,
    sk_out: *mut u8,
    sk_len: *mut u64,
) -> i32 {
    let result = std::panic::catch_unwind(|| -> Result<(), Box<dyn std::error::Error>> {
        // Build default TFHE configuration for FheUint32.
        let config = ConfigBuilder::default().build();
        let (client_key, server_key) = generate_keys(config);

        // Serialise ClientKey.
        let ck_bytes = bincode::serialize(&client_key)?;
        let ck_buf = unsafe { slice_mut_from_raw(ck_out, CLIENT_KEY_MAX_BYTES) };
        if ck_bytes.len() > ck_buf.len() {
            return Err("ClientKey serialisation exceeds buffer".into());
        }
        ck_buf[..ck_bytes.len()].copy_from_slice(&ck_bytes);
        unsafe { *ck_len = ck_bytes.len() as u64 };

        // Serialise ServerKey.
        let sk_bytes = bincode::serialize(&server_key)?;
        let sk_buf = unsafe { slice_mut_from_raw(sk_out, SERVER_KEY_MAX_BYTES) };
        if sk_bytes.len() > sk_buf.len() {
            return Err("ServerKey serialisation exceeds buffer".into());
        }
        sk_buf[..sk_bytes.len()].copy_from_slice(&sk_bytes);
        unsafe { *sk_len = sk_bytes.len() as u64 };

        Ok(())
    });

    match result {
        Ok(Ok(())) => 0,
        _ => -1,
    }
}

/// Encrypt a 32-bit unsigned integer using the TFHE ClientKey.
///
/// Parameters:
///   ck_bytes     : serialised ClientKey bytes
///   ck_len       : length of ck_bytes
///   plaintext    : the u32 value to encrypt
///   ciphertext_out : caller-allocated output buffer (≥ CIPHERTEXT_MAX_BYTES bytes)
///   out_len      : size of ciphertext_out buffer
///
/// Returns actual ciphertext length (> 0) on success, -1 on error.
#[no_mangle]
pub extern "C" fn tfhe_encrypt_u32(
    ck_bytes: *const u8,
    ck_len: u64,
    plaintext: u32,
    ciphertext_out: *mut u8,
    out_len: u64,
) -> i64 {
    let result = std::panic::catch_unwind(|| -> Result<i64, Box<dyn std::error::Error>> {
        let ck_slice = unsafe { slice_from_raw(ck_bytes, ck_len) };
        let client_key: tfhe::ClientKey = bincode::deserialize(ck_slice)?;

        let ciphertext: FheUint32 = FheUint32::encrypt(plaintext, &client_key);
        let ct_bytes = bincode::serialize(&ciphertext)?;

        let out_buf = unsafe { slice_mut_from_raw(ciphertext_out, out_len) };
        if ct_bytes.len() > out_buf.len() {
            return Err("Ciphertext exceeds output buffer".into());
        }
        out_buf[..ct_bytes.len()].copy_from_slice(&ct_bytes);
        Ok(ct_bytes.len() as i64)
    });

    match result {
        Ok(Ok(n)) => n,
        _ => -1,
    }
}

/// Encrypt a uint32 using a CompressedPublicKey (user-held).
///
/// FIX 2 (CVE-ZYTH-002): This function uses the USER's registered
/// CompressedPublicKey instead of the node's ClientKey. The resulting
/// ciphertext can ONLY be decrypted by the holder of the matching ClientKey.
/// Node operators cannot decrypt balances encrypted with this function.
///
/// Returns actual ciphertext length (> 0) on success, -1 on error.
#[no_mangle]
pub extern "C" fn tfhe_encrypt_u32_pk(
    pk_bytes: *const u8,
    pk_len: u64,
    plaintext: u32,
    ciphertext_out: *mut u8,
    out_len: u64,
) -> i64 {
    let result = std::panic::catch_unwind(|| -> Result<i64, Box<dyn std::error::Error>> {
        let pk_slice = unsafe { slice_from_raw(pk_bytes, pk_len) };
        let public_key: CompressedPublicKey = bincode::deserialize(pk_slice)?;

        // Encrypt using the CompressedPublicKey — no ClientKey required.
        // The ciphertext is semantically secure: even the encryptor cannot
        // reverse it without the matching ClientKey.
        let ciphertext: FheUint32 = FheUint32::encrypt(plaintext, &public_key);
        let ct_bytes = bincode::serialize(&ciphertext)?;

        let out_buf = unsafe { slice_mut_from_raw(ciphertext_out, out_len) };
        if ct_bytes.len() > out_buf.len() {
            return Err("Ciphertext exceeds output buffer".into());
        }
        out_buf[..ct_bytes.len()].copy_from_slice(&ct_bytes);
        Ok(ct_bytes.len() as i64)
    });

    match result {
        Ok(Ok(n)) => n,
        _ => -1,
    }
}

/// Homomorphic addition: result = c1 + c2 (mod 2^32).
///
/// Both c1 and c2 must have been encrypted with the same key configuration.
/// The ServerKey is set as the global evaluation key for this thread.
///
/// Returns actual result ciphertext length on success, -1 on error.
#[no_mangle]
pub extern "C" fn tfhe_add_u32(
    sk_bytes: *const u8,
    sk_len: u64,
    c1_bytes: *const u8,
    c1_len: u64,
    c2_bytes: *const u8,
    c2_len: u64,
    result_out: *mut u8,
    out_len: u64,
) -> i64 {
    let result = std::panic::catch_unwind(|| -> Result<i64, Box<dyn std::error::Error>> {
        let sk_slice = unsafe { slice_from_raw(sk_bytes, sk_len) };
        let server_key: tfhe::ServerKey = bincode::deserialize(sk_slice)?;

        // Set the server key as the global evaluation key for this thread.
        set_server_key(server_key);

        let c1_slice = unsafe { slice_from_raw(c1_bytes, c1_len) };
        let c2_slice = unsafe { slice_from_raw(c2_bytes, c2_len) };

        let ct1: FheUint32 = bincode::deserialize(c1_slice)?;
        let ct2: FheUint32 = bincode::deserialize(c2_slice)?;

        // Homomorphic addition — runs under the server key.
        let ct_result = ct1 + ct2;

        let res_bytes = bincode::serialize(&ct_result)?;
        let out_buf = unsafe { slice_mut_from_raw(result_out, out_len) };
        if res_bytes.len() > out_buf.len() {
            return Err("Result ciphertext exceeds output buffer".into());
        }
        out_buf[..res_bytes.len()].copy_from_slice(&res_bytes);
        Ok(res_bytes.len() as i64)
    });

    match result {
        Ok(Ok(n)) => n,
        _ => -1,
    }
}

/// Decrypt a FheUint32 ciphertext using the ClientKey.
///
/// Returns the decrypted u32 value cast to i64 (≥ 0) on success, or -1 on error.
/// Note: if the actual plaintext value happens to equal the error indicator,
/// use the error-free variant that writes to an out pointer instead.
#[no_mangle]
pub extern "C" fn tfhe_decrypt_u32(
    ck_bytes: *const u8,
    ck_len: u64,
    ciphertext_bytes: *const u8,
    ct_len: u64,
) -> i64 {
    let result = std::panic::catch_unwind(|| -> Result<i64, Box<dyn std::error::Error>> {
        let ck_slice = unsafe { slice_from_raw(ck_bytes, ck_len) };
        let client_key: tfhe::ClientKey = bincode::deserialize(ck_slice)?;

        let ct_slice = unsafe { slice_from_raw(ciphertext_bytes, ct_len) };
        let ciphertext: FheUint32 = bincode::deserialize(ct_slice)?;

        let plaintext: u32 = ciphertext.decrypt(&client_key);
        Ok(plaintext as i64)
    });

    match result {
        Ok(Ok(v)) => v,
        _ => -1,
    }
}

/// Multiply a FheUint32 ciphertext by a plaintext scalar.
///
/// This is a "scalar multiplication" — scalar is NOT encrypted.
/// Returns actual result ciphertext length on success, -1 on error.
#[no_mangle]
pub extern "C" fn tfhe_mul_scalar_u32(
    sk_bytes: *const u8,
    sk_len: u64,
    ct_bytes: *const u8,
    ct_len: u64,
    scalar: u32,
    result_out: *mut u8,
    out_len: u64,
) -> i64 {
    let result = std::panic::catch_unwind(|| -> Result<i64, Box<dyn std::error::Error>> {
        let sk_slice = unsafe { slice_from_raw(sk_bytes, sk_len) };
        let server_key: tfhe::ServerKey = bincode::deserialize(sk_slice)?;
        set_server_key(server_key);

        let ct_slice = unsafe { slice_from_raw(ct_bytes, ct_len) };
        let ciphertext: FheUint32 = bincode::deserialize(ct_slice)?;

        let ct_result = ciphertext * scalar;

        let res_bytes = bincode::serialize(&ct_result)?;
        let out_buf = unsafe { slice_mut_from_raw(result_out, out_len) };
        if res_bytes.len() > out_buf.len() {
            return Err("Result ciphertext exceeds output buffer".into());
        }
        out_buf[..res_bytes.len()].copy_from_slice(&res_bytes);
        Ok(res_bytes.len() as i64)
    });

    match result {
        Ok(Ok(n)) => n,
        _ => -1,
    }
}

/// Homomorphic subtraction: result = c1 - c2 (mod 2^32).
///
/// Both c1 and c2 must have been encrypted with the same key configuration.
/// The ServerKey is set as the thread-local evaluation key.
///
/// Returns actual result ciphertext length on success, -1 on error.
#[no_mangle]
pub extern "C" fn tfhe_sub_u32(
    sk_bytes: *const u8,
    sk_len: u64,
    c1_bytes: *const u8,
    c1_len: u64,
    c2_bytes: *const u8,
    c2_len: u64,
    result_out: *mut u8,
    out_len: u64,
) -> i64 {
    let result = std::panic::catch_unwind(|| -> Result<i64, Box<dyn std::error::Error>> {
        let sk_slice = unsafe { slice_from_raw(sk_bytes, sk_len) };
        let server_key: tfhe::ServerKey = bincode::deserialize(sk_slice)?;
        set_server_key(server_key);

        let c1_slice = unsafe { slice_from_raw(c1_bytes, c1_len) };
        let c2_slice = unsafe { slice_from_raw(c2_bytes, c2_len) };

        let ct1: FheUint32 = bincode::deserialize(c1_slice)?;
        let ct2: FheUint32 = bincode::deserialize(c2_slice)?;

        // Homomorphic subtraction — runs under the server key.
        let ct_result = ct1 - ct2;

        let res_bytes = bincode::serialize(&ct_result)?;
        let out_buf = unsafe { slice_mut_from_raw(result_out, out_len) };
        if res_bytes.len() > out_buf.len() {
            return Err("Result ciphertext exceeds output buffer".into());
        }
        out_buf[..res_bytes.len()].copy_from_slice(&res_bytes);
        Ok(res_bytes.len() as i64)
    });

    match result {
        Ok(Ok(n)) => n,
        _ => -1,
    }
}

