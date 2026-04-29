//go:build !notfhe
// +build !notfhe

// Package fhe provides Fully Homomorphic Encryption (FHE) primitives for the
// Zytherion blockchain, built on top of TFHE (Torus FHE, LWE-based) via
// Zama TFHE-rs v0.7 through a CGo binding.
//
// Scheme:        TFHE / Torus FHE (LWE-based, as referenced in the whitepaper)
// Library:       Zama TFHE-rs v0.7 via CGo (libtfhe_cgo.a)
// Security:      128-bit default TFHE-rs parameters
// Plaintext type: FheUint64 (unsigned 64-bit integer, full [0, 2^64) range)
// Compression:   Active by default â€” Encrypt() returns compressed bytes (~1-5 KB)
package fhe

import (
	"fmt"
	"sync"
)

// Context holds the TFHE key material and provides the public FHE interface.
// Create one with NewContext and reuse it â€” key generation takes several seconds.
// All methods are safe for concurrent use.
type Context struct {
	mu             sync.Mutex
	clientKeyBytes []byte // SECRET â€” never expose outside this package
	serverKeyBytes []byte // public evaluation key
}

// NewContext generates a fresh TFHE client/server key pair and returns a Context.
func NewContext() (*Context, error) {
	ck, sk, err := tfheGenerateKeys()
	if err != nil {
		return nil, fmt.Errorf("fhe: key generation failed: %w", err)
	}
	return &Context{clientKeyBytes: ck, serverKeyBytes: sk}, nil
}

// NewContextFromKeys reconstructs a Context from previously persisted key bytes.
func NewContextFromKeys(clientKeyBytes, serverKeyBytes []byte) (*Context, error) {
	if len(clientKeyBytes) == 0 || len(serverKeyBytes) == 0 {
		return nil, fmt.Errorf("fhe: key bytes must not be empty")
	}
	ck := make([]byte, len(clientKeyBytes))
	sk := make([]byte, len(serverKeyBytes))
	copy(ck, clientKeyBytes)
	copy(sk, serverKeyBytes)
	return &Context{clientKeyBytes: ck, serverKeyBytes: sk}, nil
}

// Encrypt encrypts value and returns a COMPRESSED ciphertext (~1-5 KB).
// TFHE FheUint64 supports the full [0, 2^64) range without modular wrapping.
func (c *Context) Encrypt(value uint64) ([]byte, error) {
	raw, err := tfheEncrypt(c.clientKeyBytes, value)
	if err != nil {
		return nil, fmt.Errorf("fhe: encrypt failed: %w", err)
	}
	compressed, err := compressCiphertext(c.serverKeyBytes, raw)
	if err != nil {
		return nil, fmt.Errorf("fhe: compression failed: %w", err)
	}
	return compressed, nil
}

// Decrypt decompresses and decrypts a compressed TFHE ciphertext.
func (c *Context) Decrypt(compressed []byte) (uint64, error) {
	if len(compressed) == 0 {
		return 0, fmt.Errorf("fhe: nil or empty ciphertext")
	}
	raw, err := decompressCiphertext(c.serverKeyBytes, compressed)
	if err != nil {
		return 0, fmt.Errorf("fhe: decompression failed: %w", err)
	}
	return tfheDecrypt(c.clientKeyBytes, raw)
}

// AddCiphertexts performs homomorphic addition on two compressed ciphertexts.
// Returns a new compressed ciphertext encoding (Decrypt(a) + Decrypt(b)).
func (c *Context) AddCiphertexts(a, b []byte) ([]byte, error) {
	if len(a) == 0 {
		return nil, fmt.Errorf("fhe: nil or empty ciphertext a in AddCiphertexts")
	}
	if len(b) == 0 {
		return nil, fmt.Errorf("fhe: nil or empty ciphertext b in AddCiphertexts")
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	rawA, err := decompressCiphertext(c.serverKeyBytes, a)
	if err != nil {
		return nil, fmt.Errorf("fhe: decompress a: %w", err)
	}
	rawB, err := decompressCiphertext(c.serverKeyBytes, b)
	if err != nil {
		return nil, fmt.Errorf("fhe: decompress b: %w", err)
	}
	rawResult, err := tfheAdd(c.serverKeyBytes, rawA, rawB)
	if err != nil {
		return nil, fmt.Errorf("fhe: add: %w", err)
	}
	return compressCiphertext(c.serverKeyBytes, rawResult)
}

// SubCiphertexts performs homomorphic subtraction, returning Enc(a - b).
func (c *Context) SubCiphertexts(a, b []byte) ([]byte, error) {
	if len(a) == 0 {
		return nil, fmt.Errorf("fhe: nil or empty ciphertext a in SubCiphertexts")
	}
	if len(b) == 0 {
		return nil, fmt.Errorf("fhe: nil or empty ciphertext b in SubCiphertexts")
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	rawA, err := decompressCiphertext(c.serverKeyBytes, a)
	if err != nil {
		return nil, fmt.Errorf("fhe: decompress a: %w", err)
	}
	rawB, err := decompressCiphertext(c.serverKeyBytes, b)
	if err != nil {
		return nil, fmt.Errorf("fhe: decompress b: %w", err)
	}
	rawResult, err := tfheSub(c.serverKeyBytes, rawA, rawB)
	if err != nil {
		return nil, fmt.Errorf("fhe: sub: %w", err)
	}
	return compressCiphertext(c.serverKeyBytes, rawResult)
}