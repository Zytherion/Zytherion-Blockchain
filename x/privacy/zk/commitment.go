// commitment.go — Pedersen/MiMC commitment helpers for Zytherion.
//
// This file provides the off-chain commitment computation that mirrors
// the in-circuit TransferCircuit.Define() logic.
//
// The commitment scheme:
//   C = MiMC(amount, blinding)  over BN254 Fr
//
// This matches the circuit's mimcHash function exactly, ensuring that
// a valid (amount, blinding) pair produces the correct public input C.
//
// These helpers are used by:
//   - tools/zkprove  (off-chain prover)
//   - CLI client     (commitment generation before proof submission)
//
// They are NOT called on-chain. Only VerifyTransferProof is called on-chain.
package zk

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
)

// CommitmentSize is the canonical on-chain size of a commitment (BN254 Fr element).
const CommitmentSize = 32

// ScalarFr is a BN254 scalar field element (32 bytes, big-endian).
// We use big.Int internally for portability (no CGo dependency).
type ScalarFr struct {
	v *big.Int
}

// BN254FrModulus is the BN254 scalar field modulus.
var BN254FrModulus, _ = new(big.Int).SetString(
	"21888242871839275222246405745257275088548364400416034343698204186575808495617",
	10,
)

// NewScalarFr creates a new ScalarFr from a big.Int. Reduces modulo Fr.
func NewScalarFr(n *big.Int) *ScalarFr {
	v := new(big.Int).Mod(n, BN254FrModulus)
	return &ScalarFr{v: v}
}

// SetBytes sets the scalar from big-endian bytes. Reduces modulo Fr.
func (s *ScalarFr) SetBytes(b []byte) {
	n := new(big.Int).SetBytes(b)
	s.v = new(big.Int).Mod(n, BN254FrModulus)
}

// Bytes returns the canonical 32-byte big-endian encoding (zero-padded).
func (s *ScalarFr) Bytes() []byte {
	b := s.v.Bytes()
	out := make([]byte, CommitmentSize)
	copy(out[CommitmentSize-len(b):], b)
	return out
}

// BigInt returns the underlying big.Int (a copy).
func (s *ScalarFr) BigInt() *big.Int {
	return new(big.Int).Set(s.v)
}

// IsZero returns true if the scalar is zero.
func (s *ScalarFr) IsZero() bool {
	return s.v.Sign() == 0
}

// Compute computes the 3-round MiMC hash of (a, b) over BN254 Fr,
// matching the in-circuit mimcHash function exactly.
//
// Round function: state = state^3 + key
func mimcHashNative(a, b *big.Int) *big.Int {
	q := BN254FrModulus

	// Round 1: mix a+b
	state := new(big.Int).Add(a, b)
	state.Mod(state, q)
	// state^3
	sq := new(big.Int).Mul(state, state)
	sq.Mod(sq, q)
	cb := new(big.Int).Mul(sq, state)
	cb.Mod(cb, q)
	// + a (key)
	state = new(big.Int).Add(cb, a)
	state.Mod(state, q)

	// Round 2
	sq2 := new(big.Int).Mul(state, state)
	sq2.Mod(sq2, q)
	cb2 := new(big.Int).Mul(sq2, state)
	cb2.Mod(cb2, q)
	state = new(big.Int).Add(cb2, b)
	state.Mod(state, q)

	return state
}

// Commit computes the MiMC commitment C = MiMC(amount, blinding) over BN254 Fr.
//
// Returns the canonical 32-byte big-endian encoding of C.
// This is the value that goes on-chain as the commitment, and as
// CommitmentX in the public inputs passed to VerifyTransferProof.
func Commit(amount uint64, blinding []byte) ([]byte, error) {
	if len(blinding) < 16 {
		return nil, errors.New("zk: blinding factor must be at least 16 bytes")
	}

	// Convert amount to Fr
	amountFr := new(big.Int).SetUint64(amount)
	amountFr.Mod(amountFr, BN254FrModulus)

	// Convert blinding to Fr (reduce mod q)
	blindingFr := new(big.Int).SetBytes(blinding)
	blindingFr.Mod(blindingFr, BN254FrModulus)

	// Compute MiMC hash
	c := mimcHashNative(amountFr, blindingFr)

	// Return 32-byte canonical encoding
	out := make([]byte, CommitmentSize)
	cBytes := c.Bytes()
	copy(out[CommitmentSize-len(cBytes):], cBytes)
	return out, nil
}

// GenerateBlinding generates a cryptographically random 32-byte blinding factor.
// The blinding factor MUST be kept secret by the user — it is never stored on-chain.
func GenerateBlinding() ([]byte, error) {
	b := make([]byte, CommitmentSize)
	if _, err := rand.Read(b); err != nil {
		return nil, fmt.Errorf("zk: failed to generate blinding factor: %w", err)
	}
	return b, nil
}

// ValidateCommitmentBytes performs stateless validation on a raw commitment.
//   - Must be exactly CommitmentSize (32) bytes.
//   - Must represent a value in [0, Fr_modulus).
//   - Must not be zero (zero commitment is invalid — it would allow trivial forgeries).
func ValidateCommitmentBytes(c []byte) error {
	if len(c) != CommitmentSize {
		return fmt.Errorf("zk: commitment must be %d bytes, got %d", CommitmentSize, len(c))
	}
	v := new(big.Int).SetBytes(c)
	if v.Sign() == 0 {
		return errors.New("zk: commitment must not be zero")
	}
	if v.Cmp(BN254FrModulus) >= 0 {
		return errors.New("zk: commitment exceeds BN254 Fr modulus")
	}
	return nil
}

// NullifierForTransfer computes a deterministic nullifier for a transfer.
// The nullifier is used to prevent double-spending of a commitment.
//
// nullifier = MiMC(commitment, blinding) — unique per (commitment, blinding) pair.
// The blinding factor is kept off-chain; the nullifier is stored on-chain after use.
func NullifierForTransfer(commitment, blinding []byte) ([]byte, error) {
	if len(commitment) != CommitmentSize {
		return nil, fmt.Errorf("zk: commitment must be %d bytes", CommitmentSize)
	}
	if len(blinding) < 16 {
		return nil, errors.New("zk: blinding factor must be at least 16 bytes")
	}

	cFr := new(big.Int).SetBytes(commitment)
	cFr.Mod(cFr, BN254FrModulus)

	bFr := new(big.Int).SetBytes(blinding)
	bFr.Mod(bFr, BN254FrModulus)

	n := mimcHashNative(cFr, bFr)
	out := make([]byte, CommitmentSize)
	nBytes := n.Bytes()
	copy(out[CommitmentSize-len(nBytes):], nBytes)
	return out, nil
}

// AmountToBytes encodes a uint64 amount as big-endian 8 bytes.
// Used for canonical serialization in public inputs.
func AmountToBytes(amount uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, amount)
	return b
}
