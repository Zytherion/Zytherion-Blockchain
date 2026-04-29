#ifndef TFHE_CGO_H
#define TFHE_CGO_H
#include <stddef.h>
#include <stdint.h>
#ifdef __cplusplus
extern "C" {
#endif
int32_t tfhe_generate_keys(uint8_t **out_client_key, size_t *out_client_key_len,
                           uint8_t **out_server_key,
                           size_t *out_server_key_len);
int32_t tfhe_encrypt_u64(const uint8_t *client_key_bytes, size_t client_key_len,
                         uint64_t value, uint8_t **out_ct, size_t *out_ct_len);
int32_t tfhe_decrypt_u64(const uint8_t *client_key_bytes, size_t client_key_len,
                         const uint8_t *ct_bytes, size_t ct_len,
                         uint64_t *out_value);
int32_t tfhe_add(const uint8_t *server_key_bytes, size_t server_key_len,
                 const uint8_t *ct_a_bytes, size_t ct_a_len,
                 const uint8_t *ct_b_bytes, size_t ct_b_len, uint8_t **out_ct,
                 size_t *out_ct_len);
int32_t tfhe_sub(const uint8_t *server_key_bytes, size_t server_key_len,
                 const uint8_t *ct_a_bytes, size_t ct_a_len,
                 const uint8_t *ct_b_bytes, size_t ct_b_len, uint8_t **out_ct,
                 size_t *out_ct_len);
int32_t tfhe_compress(const uint8_t *server_key_bytes, size_t server_key_len,
                      const uint8_t *ct_bytes, size_t ct_len, uint8_t **out_ct,
                      size_t *out_ct_len);
int32_t tfhe_decompress(const uint8_t *server_key_bytes, size_t server_key_len,
                        const uint8_t *compressed_bytes, size_t compressed_len,
                        uint8_t **out_ct, size_t *out_ct_len);
void tfhe_free_bytes(uint8_t *ptr, size_t len);
#ifdef __cplusplus
}
#endif
#endif