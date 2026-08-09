package cosmwasm

import "encoding/json"

// ── Custom Query Types (contracts → chain via QueryRequest::Custom) ─────────

// TFHECustomQuery is the top-level custom query envelope.
// Exactly one field must be set.
type TFHECustomQuery struct {
	Encrypt   *TFHEEncryptQuery   `json:"tfhe_encrypt,omitempty"`
	Decrypt   *TFHEDecryptQuery   `json:"tfhe_decrypt,omitempty"`
	AddCT     *TFHEAddCTQuery     `json:"tfhe_add,omitempty"`
	MulScalar *TFHEMulScalarQuery `json:"tfhe_mul_scalar,omitempty"`
	VerifyCT  *TFHEVerifyCTQuery  `json:"tfhe_verify,omitempty"`
}

// TFHEEncryptQuery encrypts a plaintext uint32 using the node's client key.
// Returns: TFHECiphertextResponse
type TFHEEncryptQuery struct {
	Value uint32 `json:"value"`
}

// TFHEDecryptQuery decrypts a ciphertext and returns the plaintext.
// CAUTION: For demonstration/testing only — reveals plaintext on-chain!
// Returns: TFHEPlaintextResponse
type TFHEDecryptQuery struct {
	Ciphertext []byte `json:"ciphertext"` // base64-encoded in JSON
}

// TFHEAddCTQuery homomorphically adds two FheUint32 ciphertexts.
// Both ciphertexts must have been encrypted with the same client key.
// Returns: TFHECiphertextResponse (encrypted sum, nobody knows the value)
type TFHEAddCTQuery struct {
	CT1 []byte `json:"ct1"` // base64-encoded in JSON
	CT2 []byte `json:"ct2"` // base64-encoded in JSON
}

// TFHEMulScalarQuery multiplies an encrypted value by a plaintext scalar.
// Returns: TFHECiphertextResponse
type TFHEMulScalarQuery struct {
	Ciphertext []byte `json:"ciphertext"` // base64-encoded in JSON
	Scalar     uint32 `json:"scalar"`
}

// TFHEVerifyCTQuery checks that a ciphertext blob is well-formed.
// Returns: TFHEVerifyResponse
type TFHEVerifyCTQuery struct {
	Ciphertext []byte `json:"ciphertext"`
}

// ── Response Types ─────────────────────────────────────────────────────────

// TFHECiphertextResponse wraps a serialised FheUint32 ciphertext.
type TFHECiphertextResponse struct {
	Ciphertext []byte `json:"ciphertext"` // base64-encoded in JSON
	SizeBytes  int    `json:"size_bytes"`
}

// TFHEPlaintextResponse wraps a decrypted uint32 value.
type TFHEPlaintextResponse struct {
	Value uint32 `json:"value"`
}

// TFHEVerifyResponse is the result of verifying a ciphertext.
type TFHEVerifyResponse struct {
	Valid     bool   `json:"valid"`
	SizeBytes int    `json:"size_bytes"`
	Message   string `json:"message,omitempty"`
}

// MarshalJSON helpers so []byte is base64 in JSON (standard encoding/json behavior).
// encoding/json already handles []byte as base64, so no extra work needed.

// ── TFHE Engine Interface (allows mocking in tests) ─────────────────────────

// TFHEEngine is the interface for TFHE cryptographic operations.
// Implemented by the live CGo engine when -tags tfhe_cgo is used.
type TFHEEngine interface {
	// GetKeys returns the node's TFHE key pair (clientKey, serverKey)
	GetKeys() (clientKey []byte, serverKey []byte, err error)
	// Encrypt encrypts a uint32 using the node client key
	Encrypt(clientKey []byte, value uint32) ([]byte, error)
	// Decrypt decrypts a ciphertext using the node client key
	Decrypt(clientKey []byte, ct []byte) (uint32, error)
	// Add homomorphically adds two ciphertexts (requires serverKey)
	Add(serverKey, ct1, ct2 []byte) ([]byte, error)
	// MulScalar multiplies a ciphertext by a plaintext scalar (requires serverKey)
	MulScalar(serverKey, ct []byte, scalar uint32) ([]byte, error)
}

// MustMarshal marshals v to JSON, panicking on error. Use only in contexts where
// JSON marshaling cannot fail (known-good types).
func MustMarshal(v interface{}) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic("tfhe/cosmwasm: impossible JSON marshal failure: " + err.Error())
	}
	return b
}
