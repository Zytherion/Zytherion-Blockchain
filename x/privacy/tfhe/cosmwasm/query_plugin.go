// query_plugin.go — CosmWasm custom query plugin for TFHE operations.
//
// This file is compiled ONLY when -tags tfhe_cgo is set.
// It exposes TFHE operations (encrypt, decrypt, add, mul_scalar) as CosmWasm
// custom queries that smart contracts can call via QueryRequest::Custom.
//
// # Usage from CosmWasm contracts (Rust)
//
//	// Encrypt a value
//	let query = TFHECustomQuery::TfheEncrypt { value: 42 };
//	let response: TFHECiphertextResponse = deps.querier.custom(&query)?;
//
//	// Add two ciphertexts homomorphically (result is still encrypted!)
//	let sum_query = TFHECustomQuery::TfheAdd { ct1, ct2 };
//	let sum: TFHECiphertextResponse = deps.querier.custom(&sum_query)?;
package cosmwasm

import (
	"encoding/json"
	"fmt"
	"os"

	sdk "github.com/cosmos/cosmos-sdk/types"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	tfhe "zytherion/x/privacy/tfhe"
)

// Process-level cache of the node's TFHE key pair.
// Generated once at first use and stored in the user's home directory for reuse
// across restarts.
var (
	cachedClientKey []byte
	cachedServerKey []byte
)

const (
	tfheClientKeyFile = ".zytherion_tfhe_client.key"
	tfheServerKeyFile = ".zytherion_tfhe_server.key"
)

// ensureKeys loads or generates the node's TFHE key pair.
// Keys are persisted in the user's home directory for reuse across restarts.
// Key generation is slow (~10-60 seconds); it happens only once per node lifecycle.
func ensureKeys() (clientKey, serverKey []byte, err error) {
	if len(cachedClientKey) > 0 {
		return cachedClientKey, cachedServerKey, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}

	ckPath := home + "/" + tfheClientKeyFile
	skPath := home + "/" + tfheServerKeyFile

	ck, errCK := os.ReadFile(ckPath)
	sk, errSK := os.ReadFile(skPath)

	if errCK == nil && errSK == nil && len(ck) > 0 && len(sk) > 0 {
		// Found existing keys on disk — use them
		cachedClientKey = ck
		cachedServerKey = sk
		return ck, sk, nil
	}

	// Generate new keys (slow — happens only once per node)
	kp, err := tfhe.GenerateKeys()
	if err != nil {
		return nil, nil, fmt.Errorf("tfhe: key generation failed: %w", err)
	}

	// Persist to disk (permission 0600 — only owner can read)
	if writeErr := os.WriteFile(ckPath, kp.ClientKey, 0600); writeErr != nil {
		fmt.Printf("[WARN] tfhe: could not persist client key to %s: %v\n", ckPath, writeErr)
	}
	if writeErr := os.WriteFile(skPath, kp.ServerKey, 0600); writeErr != nil {
		fmt.Printf("[WARN] tfhe: could not persist server key to %s: %v\n", skPath, writeErr)
	}

	cachedClientKey = kp.ClientKey
	cachedServerKey = kp.ServerKey
	return kp.ClientKey, kp.ServerKey, nil
}

// NewTFHEQueryPlugin returns a CosmWasm CustomQuerier that handles TFHE queries.
//
// Supported queries (JSON envelope, exactly one field non-null):
//
//	{"tfhe_encrypt":{"value":42}}                      → TFHECiphertextResponse
//	{"tfhe_decrypt":{"ciphertext":"<base64>"}}         → TFHEPlaintextResponse
//	{"tfhe_add":{"ct1":"<base64>","ct2":"<base64>"}}   → TFHECiphertextResponse
//	{"tfhe_mul_scalar":{"ciphertext":"<b64>","scalar":3}} → TFHECiphertextResponse
//	{"tfhe_verify":{"ciphertext":"<base64>"}}           → TFHEVerifyResponse
func NewTFHEQueryPlugin() wasmkeeper.CustomQuerier {
	return func(ctx sdk.Context, request json.RawMessage) ([]byte, error) {
		var query TFHECustomQuery
		if err := json.Unmarshal(request, &query); err != nil {
			return nil, fmt.Errorf("tfhe: invalid custom query JSON: %w", err)
		}

		switch {
		case query.Encrypt != nil:
			return handleEncrypt(query.Encrypt)

		case query.Decrypt != nil:
			return handleDecrypt(query.Decrypt)

		case query.AddCT != nil:
			return handleAdd(query.AddCT)

		case query.MulScalar != nil:
			return handleMulScalar(query.MulScalar)

		case query.VerifyCT != nil:
			return handleVerify(query.VerifyCT)

		default:
			return nil, fmt.Errorf(
				"tfhe: unknown custom query — supported: " +
					"tfhe_encrypt, tfhe_decrypt, tfhe_add, tfhe_mul_scalar, tfhe_verify",
			)
		}
	}
}

// handleEncrypt encrypts a plaintext uint32 using the node's client key.
func handleEncrypt(q *TFHEEncryptQuery) ([]byte, error) {
	ck, _, err := ensureKeys()
	if err != nil {
		return nil, fmt.Errorf("tfhe_encrypt: key error: %w", err)
	}

	ct, err := tfhe.EncryptUint32(ck, q.Value)
	if err != nil {
		return nil, fmt.Errorf("tfhe_encrypt: encryption failed: %w", err)
	}

	resp := TFHECiphertextResponse{
		Ciphertext: ct,
		SizeBytes:  len(ct),
	}
	return json.Marshal(resp)
}

// handleDecrypt decrypts a FheUint32 ciphertext using the node's client key.
// CAUTION: This reveals plaintext. Use only for testing/demo purposes.
func handleDecrypt(q *TFHEDecryptQuery) ([]byte, error) {
	if len(q.Ciphertext) == 0 {
		return nil, fmt.Errorf("tfhe_decrypt: ciphertext is empty")
	}

	ck, _, err := ensureKeys()
	if err != nil {
		return nil, fmt.Errorf("tfhe_decrypt: key error: %w", err)
	}

	value, err := tfhe.DecryptUint32(ck, q.Ciphertext)
	if err != nil {
		return nil, fmt.Errorf("tfhe_decrypt: decryption failed: %w", err)
	}

	resp := TFHEPlaintextResponse{Value: value}
	return json.Marshal(resp)
}

// handleAdd performs homomorphic addition of two FheUint32 ciphertexts.
// Uses the node's server key for evaluation.
func handleAdd(q *TFHEAddCTQuery) ([]byte, error) {
	if len(q.CT1) == 0 || len(q.CT2) == 0 {
		return nil, fmt.Errorf("tfhe_add: both ct1 and ct2 must be non-empty")
	}

	_, sk, err := ensureKeys()
	if err != nil {
		return nil, fmt.Errorf("tfhe_add: key error: %w", err)
	}

	result, err := tfhe.AddUint32(sk, q.CT1, q.CT2)
	if err != nil {
		return nil, fmt.Errorf("tfhe_add: homomorphic addition failed: %w", err)
	}

	resp := TFHECiphertextResponse{
		Ciphertext: result,
		SizeBytes:  len(result),
	}
	return json.Marshal(resp)
}

// handleMulScalar multiplies a ciphertext by a plaintext scalar.
// Uses the node's server key for evaluation.
func handleMulScalar(q *TFHEMulScalarQuery) ([]byte, error) {
	if len(q.Ciphertext) == 0 {
		return nil, fmt.Errorf("tfhe_mul_scalar: ciphertext is empty")
	}

	_, sk, err := ensureKeys()
	if err != nil {
		return nil, fmt.Errorf("tfhe_mul_scalar: key error: %w", err)
	}

	result, err := tfhe.MultiplyScalarUint32(sk, q.Ciphertext, q.Scalar)
	if err != nil {
		return nil, fmt.Errorf("tfhe_mul_scalar: multiplication failed: %w", err)
	}

	resp := TFHECiphertextResponse{
		Ciphertext: result,
		SizeBytes:  len(result),
	}
	return json.Marshal(resp)
}

// handleVerify checks whether a ciphertext blob is structurally well-formed.
// Validation is based on size heuristics for FheUint32 ciphertexts.
func handleVerify(q *TFHEVerifyCTQuery) ([]byte, error) {
	if len(q.Ciphertext) == 0 {
		resp := TFHEVerifyResponse{
			Valid:     false,
			SizeBytes: 0,
			Message:   "ciphertext is empty",
		}
		return json.Marshal(resp)
	}

	// A FheUint32 ciphertext is "valid" if its size is within expected bounds.
	// These are empirical bounds from the tfhe-rs library for default parameters.
	const minSize = 8 * 1024       // 8 KB minimum reasonable size for FheUint32
	const maxSize = 32 * 1024 * 1024 // 32 MB absolute maximum

	if len(q.Ciphertext) < minSize || len(q.Ciphertext) > maxSize {
		resp := TFHEVerifyResponse{
			Valid:     false,
			SizeBytes: len(q.Ciphertext),
			Message: fmt.Sprintf(
				"ciphertext size %d bytes out of expected range [%d, %d]",
				len(q.Ciphertext), minSize, maxSize,
			),
		}
		return json.Marshal(resp)
	}

	resp := TFHEVerifyResponse{
		Valid:     true,
		SizeBytes: len(q.Ciphertext),
		Message:   "ciphertext appears well-formed",
	}
	return json.Marshal(resp)
}
