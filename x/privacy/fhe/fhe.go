// Package fhe provides Fully Homomorphic Encryption (FHE) primitives for the
// Zytherion blockchain, built on top of the Lattigo v4 BFV scheme.
//
// The BFV scheme supports exact arithmetic on integers modulo the plaintext
// modulus t. This package uses a custom parameter set with a 40-bit prime
// t = 0x10000048001 = 1,099,511,955,457 (~1.1 trillion), which is large
// enough to represent any token amount with 6 decimal places up to ~1 million
// whole units, or raw microtoken amounts in the billions — without modular
// wrapping. Security remains at 128-bit classical (logN=13, logQP≈218 bits).
package fhe

import (
	"fmt"

	"github.com/tuneinsight/lattigo/v4/bfv"
	"github.com/tuneinsight/lattigo/v4/rlwe"
)

// plaintextModulus is the BFV plaintext modulus t.
//
// Chosen as a 40-bit NTT-friendly prime: t = 0x10000048001 = 1,099,511,955,457.
// This is significantly larger than 2^32 (4,294,967,296), allowing values up
// to ~1.1 trillion to be encrypted and computed upon without wrapping.
//
// Security: the surrounding ciphertext parameters use logN=13 and logQP≈218
// bits, providing ≥128-bit classical security per the Lattigo defaults.
const plaintextModulus uint64 = 0x10000048001 // 40-bit prime, ~1.1 trillion

// customParams is a BFV ParametersLiteral that takes the well-tested 128-bit
// secure ciphertext modulus from PN13QP218 and substitutes a much larger
// plaintext modulus t so that currency-scale integers (up to ~1.1 trillion)
// can be encrypted without modular reduction.
var customParams = bfv.ParametersLiteral{
	// N = 2^13 = 8192 ring dimension — same as PN13QP218.
	LogN: 13,
	// Ciphertext modulus Q: three 54-bit primes (total 162 bits).
	Q: []uint64{0x3fffffffef8001, 0x4000000011c001, 0x40000000120001},
	// Key-switching prime P: one 55-bit prime.
	P: []uint64{0x7ffffffffb4001},
	// Plaintext modulus t: 40-bit prime, large enough for currency amounts.
	// t = 1,099,511,955,457 — safely above 2^32 and below Q[0].
	T: plaintextModulus,
}

// Context holds the BFV scheme parameters and derived cryptographic objects.
// Create one with NewContext and reuse it — key generation is expensive.
type Context struct {
	params    bfv.Parameters
	encoder   bfv.Encoder
	evaluator bfv.Evaluator
	keygen    rlwe.KeyGenerator
	sk        *rlwe.SecretKey
	pk        *rlwe.PublicKey
	encryptor rlwe.Encryptor
	decryptor rlwe.Decryptor
}

// NewContext creates a Context using a custom 128-bit secure BFV parameter set
// with a 40-bit plaintext modulus (t = 1,099,511,955,457 ≈ 1.1 trillion).
// This allows encrypting and homomorphically computing on token amounts up to
// ~1.1 trillion without any modular wrapping.
//
// A fresh secret/public key pair is generated each time — callers are
// responsible for persisting keys across process restarts if needed.
func NewContext() (*Context, error) {
	params, err := bfv.NewParametersFromLiteral(customParams)
	if err != nil {
		return nil, fmt.Errorf("fhe: failed to create BFV parameters: %w", err)
	}

	keygen := bfv.NewKeyGenerator(params)
	sk, pk := keygen.GenKeyPair()

	encoder := bfv.NewEncoder(params)
	encryptor := bfv.NewEncryptor(params, pk)
	decryptor := bfv.NewDecryptor(params, sk)

	// EvaluationKey is empty (no relinearisation or rotation keys) — sufficient
	// for Add and scalar-multiplication operations. Provide a RelinearizationKey
	// if you need to relinearise after ciphertext × ciphertext multiplication.
	evaluator := bfv.NewEvaluator(params, rlwe.EvaluationKey{})

	return &Context{
		params:    params,
		encoder:   encoder,
		evaluator: evaluator,
		keygen:    keygen,
		sk:        sk,
		pk:        pk,
		encryptor: encryptor,
		decryptor: decryptor,
	}, nil
}

// PlaintextModulus returns the plaintext modulus t for this Context.
// All encrypted values are integers in [0, t). Values ≥ t wrap modulo t.
func (c *Context) PlaintextModulus() uint64 {
	return c.params.T()
}

// Params returns the BFV parameters for this Context.
// External callers use this to construct or deserialise ciphertexts that are
// compatible with this context (e.g. NewCiphertext, UnmarshalBinary).
func (c *Context) Params() bfv.Parameters {
	return c.params
}

// SecretKey returns the secret key for this Context.
// Keep this private — never expose it on-chain.
func (c *Context) SecretKey() *rlwe.SecretKey {
	return c.sk
}

// PublicKey returns the public key for this Context.
// This can be published on-chain so other parties can encrypt values
// that only the key holder can decrypt.
func (c *Context) PublicKey() *rlwe.PublicKey {
	return c.pk
}

// Encrypt encodes a single uint64 value into BFV slot 0 and encrypts it
// under the Context's public key, returning a ciphertext.
// The value must be in [0, PlaintextModulus()). Values outside this range
// are silently reduced modulo t.
func (c *Context) Encrypt(value uint64) (*rlwe.Ciphertext, error) {
	slots := make([]uint64, c.params.N())
	slots[0] = value

	pt := bfv.NewPlaintext(c.params, c.params.MaxLevel())
	// Encode is void in Lattigo v4.
	c.encoder.Encode(slots, pt)

	// EncryptNew returns *rlwe.Ciphertext only (no error) in Lattigo v4.
	ct := c.encryptor.EncryptNew(pt)
	return ct, nil
}

// Decrypt decrypts the given ciphertext using the Context's secret key and
// returns the uint64 value stored in slot 0.
func (c *Context) Decrypt(ct *rlwe.Ciphertext) (uint64, error) {
	if ct == nil {
		return 0, fmt.Errorf("fhe: nil ciphertext")
	}

	pt := c.decryptor.DecryptNew(ct)
	// DecodeUintNew is the idiomatic Lattigo v4 path for []uint64 results.
	slots := c.encoder.DecodeUintNew(pt)
	if len(slots) == 0 {
		return 0, fmt.Errorf("fhe: decode returned empty slice")
	}
	return slots[0], nil
}

// AddCiphertexts performs homomorphic addition of two ciphertexts, returning
// a new ciphertext whose decryption equals (Decrypt(a) + Decrypt(b)) mod t.
//
// This is a core FHE primitive: addition on encrypted data without any
// knowledge of the underlying values.
func (c *Context) AddCiphertexts(a, b *rlwe.Ciphertext) (*rlwe.Ciphertext, error) {
	if a == nil || b == nil {
		return nil, fmt.Errorf("fhe: nil ciphertext in AddCiphertexts")
	}
	// AddNew returns *rlwe.Ciphertext only in Lattigo v4 BFV evaluator.
	result := c.evaluator.AddNew(a, b)
	return result, nil
}

// MulCiphertexts performs homomorphic multiplication of two ciphertexts.
// Note: ciphertext × ciphertext multiplication produces a degree-2 ciphertext
// that must be relinearised before further operations. Provide a
// RelinearizationKey via rlwe.EvaluationKey if you plan to chain multiplications.
func (c *Context) MulCiphertexts(a, b *rlwe.Ciphertext) (*rlwe.Ciphertext, error) {
	if a == nil || b == nil {
		return nil, fmt.Errorf("fhe: nil ciphertext in MulCiphertexts")
	}
	// MulNew returns *rlwe.Ciphertext only in Lattigo v4 BFV evaluator.
	result := c.evaluator.MulNew(a, b)
	return result, nil
}
