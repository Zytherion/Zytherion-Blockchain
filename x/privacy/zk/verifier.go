// verifier.go — Deterministic Groth16 proof verifier for Zytherion.
//
// # Design Invariants (required for Cosmos SDK consensus safety)
//
//  1. NO goroutines — verification is synchronous.
//  2. NO floating-point arithmetic.
//  3. NO map iteration — all loops are over slices.
//  4. Constant-time proof rejection (returns error, never panics).
//  5. The Verifying Key (VK) is loaded once at node startup and reused.
//
// On-chain calls: VerifyTransferProof(proofBytes, publicInputs, vkBytes)
// This is the ONLY entry point for on-chain verification.
package zk

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark/frontend"
)

// ProofJSON is the wire format for a ZK proof submitted in a transaction.
// JSON is used for human-readability in the CLI; the keeper decodes this
// before calling VerifyTransferProof.
type ProofJSON struct {
	// Proof is the base64-encoded Groth16 proof bytes.
	Proof []byte `json:"proof"`
	// PublicInputs contains the public witness values in canonical order:
	//   [0] CommitmentX  (Fr element, big-endian 32 bytes)
	//   [1] CommitmentY  (Fr element, big-endian 32 bytes — always 0 for scalar commitments)
	PublicInputs [][]byte `json:"public_inputs"`
	// Commitment is the canonical 32-byte commitment (= CommitmentX) stored on-chain.
	Commitment []byte `json:"commitment"`
}

// VerifyTransferProof verifies a Groth16 proof against the chain's verifying key.
//
// Parameters:
//   - proofBytes   : raw Groth16 proof (gnark binary serialization)
//   - publicInputs : canonical public witness — must be exactly 2 Fr elements (64 bytes total)
//   - vkBytes      : serialized Groth16 verifying key (gnark binary serialization)
//
// Returns nil on successful verification, error on any failure.
// This function is deterministic, allocation-bounded, and safe to call concurrently.
func VerifyTransferProof(proofBytes, publicInputs, vkBytes []byte) error {
	// ── 1. Structural validation ──────────────────────────────────────────────
	if len(proofBytes) == 0 {
		return errors.New("zk: proof bytes are empty")
	}
	if len(publicInputs) != 64 {
		return fmt.Errorf("zk: public inputs must be exactly 64 bytes (2×32), got %d", len(publicInputs))
	}
	if len(vkBytes) == 0 {
		return errors.New("zk: verifying key bytes are empty")
	}

	// ── 2. Deserialize verifying key ─────────────────────────────────────────
	vk := groth16.NewVerifyingKey(ecc.BN254)
	if _, err := vk.ReadFrom(bytes.NewReader(vkBytes)); err != nil {
		return fmt.Errorf("zk: failed to deserialize verifying key: %w", err)
	}

	// ── 3. Deserialize proof ──────────────────────────────────────────────────
	proof := groth16.NewProof(ecc.BN254)
	if _, err := proof.ReadFrom(bytes.NewReader(proofBytes)); err != nil {
		return fmt.Errorf("zk: failed to deserialize proof: %w", err)
	}

	// ── 4. Build public witness ───────────────────────────────────────────────
	// Public inputs layout (64 bytes total):
	//   bytes  0–31 : CommitmentX (BN254 Fr, big-endian)
	//   bytes 32–63 : CommitmentY (BN254 Fr, big-endian — expected to be 0)
	commitmentX := new(ScalarFr)
	commitmentY := new(ScalarFr)
	commitmentX.SetBytes(publicInputs[0:32])
	commitmentY.SetBytes(publicInputs[32:64])

	// ── 5. Reject non-zero CommitmentY (scalar commitment invariant) ──────────
	if !commitmentY.IsZero() {
		return errors.New("zk: CommitmentY must be zero for scalar commitments")
	}

	// ── 6. Build the gnark witness ────────────────────────────────────────────
	assignment := &TransferCircuit{
		CommitmentX: commitmentX.BigInt(),
		CommitmentY: 0,
	}
	witness, err := frontend.NewWitness(assignment, ecc.BN254.ScalarField(), frontend.PublicOnly())
	if err != nil {
		return fmt.Errorf("zk: failed to build public witness: %w", err)
	}

	// ── 7. Groth16 verification ───────────────────────────────────────────────
	if err := groth16.Verify(proof, vk, witness); err != nil {
		return fmt.Errorf("zk: proof verification failed: %w", err)
	}

	return nil
}

// DecodeProofJSON parses a ProofJSON from raw JSON bytes.
// Used by the CLI and keeper to decode transaction payloads.
func DecodeProofJSON(raw []byte) (*ProofJSON, error) {
	if len(raw) == 0 {
		return nil, errors.New("zk: proof JSON is empty")
	}
	var p ProofJSON
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, fmt.Errorf("zk: failed to unmarshal proof JSON: %w", err)
	}
	if len(p.Proof) == 0 {
		return nil, errors.New("zk: proof field is empty in proof JSON")
	}
	if len(p.Commitment) == 0 {
		return nil, errors.New("zk: commitment field is empty in proof JSON")
	}
	return &p, nil
}

// EncodePublicInputs encodes two 32-byte scalars into the canonical 64-byte
// public inputs format expected by VerifyTransferProof.
func EncodePublicInputs(commitmentX, commitmentY []byte) ([]byte, error) {
	if len(commitmentX) != 32 {
		return nil, fmt.Errorf("zk: commitmentX must be 32 bytes, got %d", len(commitmentX))
	}
	if len(commitmentY) != 32 {
		return nil, fmt.Errorf("zk: commitmentY must be 32 bytes, got %d", len(commitmentY))
	}
	out := make([]byte, 64)
	copy(out[0:32], commitmentX)
	copy(out[32:64], commitmentY)
	return out, nil
}
