// cgo_bridge.h — C header for the tfhe_c Rust static library.
//
// This file is included by engine.go via CGo's `#include` directive.
// It declares the C-compatible FFI functions exported from the Rust crate.
//
// Build instructions:
//   1. cd x/privacy/tfhe/tfhe_c && cargo build --release
//   2. The static library is at: target/release/libtfhe_c.a
//   3. CGo links against it via the LDFLAGS in engine.go
//
// All buffer sizes must be pre-allocated by the caller using the
// tfhe_*_max_bytes() functions.

#ifndef TFHE_CGO_BRIDGE_H
#define TFHE_CGO_BRIDGE_H

#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

// ── Buffer size queries ───────────────────────────────────────────────────────

/// Returns the maximum serialised ciphertext size in bytes.
/// Use this to pre-allocate ciphertext output buffers.
uint64_t tfhe_ciphertext_max_bytes(void);

/// Returns the maximum serialised ClientKey size in bytes.
uint64_t tfhe_client_key_max_bytes(void);

/// Returns the maximum serialised ServerKey size in bytes.
uint64_t tfhe_server_key_max_bytes(void);

// ── Key generation ────────────────────────────────────────────────────────────

/// Generate a TFHE ClientKey + ServerKey pair.
///
/// @param ck_out   Buffer for serialised ClientKey (allocate tfhe_client_key_max_bytes bytes)
/// @param ck_len   On success, set to actual ClientKey byte length
/// @param sk_out   Buffer for serialised ServerKey (allocate tfhe_server_key_max_bytes bytes)
/// @param sk_len   On success, set to actual ServerKey byte length
/// @return 0 on success, -1 on error
int32_t tfhe_keygen(
    uint8_t *ck_out, uint64_t *ck_len,
    uint8_t *sk_out, uint64_t *sk_len
);

// ── Encryption ────────────────────────────────────────────────────────────────

/// Encrypt a uint32 plaintext value using the ClientKey (node-held).
///
/// @param ck_bytes      Serialised ClientKey bytes
/// @param ck_len        Length of ck_bytes
/// @param plaintext     The u32 value to encrypt
/// @param ct_out        Buffer for ciphertext (allocate tfhe_ciphertext_max_bytes bytes)
/// @param out_len       Size of ct_out buffer
/// @return Actual ciphertext length in bytes (> 0), or -1 on error
int64_t tfhe_encrypt_u32(
    const uint8_t *ck_bytes, uint64_t ck_len,
    uint32_t plaintext,
    uint8_t *ct_out, uint64_t out_len
);

/// Encrypt a uint32 plaintext using a CompressedPublicKey (user-held).
///
/// FIX 2 (CVE-ZYTH-002): Unlike tfhe_encrypt_u32, this function uses the
/// USER's own registered CompressedPublicKey. The resulting ciphertext can
/// ONLY be decrypted by the user's matching ClientKey — not the node.
///
/// @param pk_bytes      Serialised CompressedPublicKey bytes (registered on-chain)
/// @param pk_len        Length of pk_bytes
/// @param plaintext     The u32 value to encrypt
/// @param ct_out        Buffer for ciphertext (allocate tfhe_ciphertext_max_bytes bytes)
/// @param out_len       Size of ct_out buffer
/// @return Actual ciphertext length in bytes (> 0), or -1 on error
int64_t tfhe_encrypt_u32_pk(
    const uint8_t *pk_bytes, uint64_t pk_len,
    uint32_t plaintext,
    uint8_t *ct_out, uint64_t out_len
);

// ── Homomorphic operations ────────────────────────────────────────────────────

/// Homomorphic addition: result_ct = c1 + c2 (mod 2^32).
///
/// @param sk_bytes   Serialised ServerKey bytes
/// @param sk_len     Length of sk_bytes
/// @param c1_bytes   First ciphertext bytes
/// @param c1_len     Length of c1_bytes
/// @param c2_bytes   Second ciphertext bytes
/// @param c2_len     Length of c2_bytes
/// @param result_out Buffer for result ciphertext (allocate tfhe_ciphertext_max_bytes bytes)
/// @param out_len    Size of result_out buffer
/// @return Actual result ciphertext length (> 0), or -1 on error
int64_t tfhe_add_u32(
    const uint8_t *sk_bytes, uint64_t sk_len,
    const uint8_t *c1_bytes, uint64_t c1_len,
    const uint8_t *c2_bytes, uint64_t c2_len,
    uint8_t *result_out, uint64_t out_len
);

/// Homomorphic subtraction: result_ct = c1 - c2 (mod 2^32).
///
/// @param sk_bytes   Serialised ServerKey bytes
/// @param sk_len     Length of sk_bytes
/// @param c1_bytes   First ciphertext bytes (minuend)
/// @param c1_len     Length of c1_bytes
/// @param c2_bytes   Second ciphertext bytes (subtrahend)
/// @param c2_len     Length of c2_bytes
/// @param result_out Buffer for result ciphertext (allocate tfhe_ciphertext_max_bytes bytes)
/// @param out_len    Size of result_out buffer
/// @return Actual result ciphertext length (> 0), or -1 on error
int64_t tfhe_sub_u32(
    const uint8_t *sk_bytes, uint64_t sk_len,
    const uint8_t *c1_bytes, uint64_t c1_len,
    const uint8_t *c2_bytes, uint64_t c2_len,
    uint8_t *result_out, uint64_t out_len
);

/// Scalar multiplication: result_ct = ct * scalar (mod 2^32).
///
/// @param sk_bytes   Serialised ServerKey bytes
/// @param sk_len     Length of sk_bytes
/// @param ct_bytes   Input ciphertext bytes
/// @param ct_len     Length of ct_bytes
/// @param scalar     Plaintext scalar (NOT encrypted)
/// @param result_out Buffer for result ciphertext (allocate tfhe_ciphertext_max_bytes bytes)
/// @param out_len    Size of result_out buffer
/// @return Actual result ciphertext length (> 0), or -1 on error
int64_t tfhe_mul_scalar_u32(
    const uint8_t *sk_bytes, uint64_t sk_len,
    const uint8_t *ct_bytes, uint64_t ct_len,
    uint32_t scalar,
    uint8_t *result_out, uint64_t out_len
);

// ── Decryption ────────────────────────────────────────────────────────────────

/// Decrypt a FheUint32 ciphertext.
///
/// @param ck_bytes    Serialised ClientKey bytes
/// @param ck_len      Length of ck_bytes
/// @param ct_bytes    Ciphertext bytes to decrypt
/// @param ct_len      Length of ct_bytes
/// @return Decrypted u32 value as int64_t (≥ 0), or -1 on error
int64_t tfhe_decrypt_u32(
    const uint8_t *ck_bytes, uint64_t ck_len,
    const uint8_t *ct_bytes, uint64_t ct_len
);

#ifdef __cplusplus
}
#endif

#endif // TFHE_CGO_BRIDGE_H
