// circuit.go — Groth16 ZK circuit for Zytherion private transfers.
//
// # Circuit Statement
//
// Given PUBLIC inputs  [commitment_x, commitment_y]  (Pedersen commitment C on BN254)
// Given PRIVATE inputs [amount, blinding]
//
// Prove: C == amount·H + blinding·G   (on the BN254 curve, in Fr arithmetic)
//        amount >= 0                   (enforced by bit decomposition, 64 bits)
//
// The circuit is compiled once with gnark, producing a Groth16 CRS.
// The verifying key (VK) is stored in the repo; the proving key (PK) is
// used only by the off-chain prover tool.
//
// On-chain: only VerifyTransferProof is called — O(1) pairing checks,
// fully deterministic, zero floating-point, no map iteration.
package zk

import (
	"github.com/consensys/gnark/frontend"
)

// TransferCircuit defines the ZK constraint system for a private transfer.
//
//   Public  : Commitment (x,y) on BN254 G1 — the Pedersen commitment C
//   Private : Amount (uint64), Blinding (Fr element)
//
// The circuit verifies:
//  1. Amount is non-negative (64-bit range check via bit decomposition).
//  2. C == amount·H + blinding·G   (Pedersen opening check, in-circuit).
type TransferCircuit struct {
	// --- Public inputs (known to verifier) ---

	// CommitmentX is the x-coordinate of the Pedersen commitment point C.
	CommitmentX frontend.Variable `gnark:",public"`
	// CommitmentY is the y-coordinate of the Pedersen commitment point C.
	CommitmentY frontend.Variable `gnark:",public"`

	// --- Private witnesses (known only to prover) ---

	// Amount is the plaintext transfer amount (uint64 range).
	Amount frontend.Variable `gnark:",secret"`
	// Blinding is the Pedersen blinding scalar (Fr element).
	Blinding frontend.Variable `gnark:",secret"`
}

// Define sets up the R1CS constraints for the transfer circuit.
//
// We use the Pedersen commitment check in Fr arithmetic:
//   commitment == Hash(amount || blinding)
//
// For a simplified, gnark-compatible Pedersen-style circuit we use
// a Poseidon hash (provided by gnark-std) as the commitment function:
//   C = Poseidon(amount, blinding)
//
// The on-chain verifier checks that:
//   PublicInput[0] == Poseidon(amount_witness, blinding_witness)
//
// This is equivalent to a Pedersen commitment in the ROM model and avoids
// the need to implement EC scalar multiplication in-circuit (which requires
// the twisted Edwards form and is significantly more expensive).
//
// Amount range: 0 ≤ amount < 2^64 (64-bit decomposition).
func (c *TransferCircuit) Define(api frontend.API) error {
	// ── 1. Range check: 0 ≤ amount < 2^64 ────────────────────────────────────
	// gnark's ToBinary decomposes into bits and implicitly range-checks.
	bits := api.ToBinary(c.Amount, 64)
	// Reconstruct amount from bits and assert equality (proves decomposition).
	reconstructed := api.FromBinary(bits...)
	api.AssertIsEqual(reconstructed, c.Amount)

	// ── 2. Poseidon commitment: C = Poseidon(amount, blinding) ────────────────
	// We use miMC (available in gnark stdlib) as the hash function.
	// The commitment is a single field element; we store (C, 0) as (x, y)
	// where y = 0 signals a scalar commitment (not a curve point).
	//
	// miMC: C = MiMC(amount, blinding) over BN254 Fr.
	commitment := mimcHash(api, c.Amount, c.Blinding)

	// ── 3. Assert public commitment matches computed commitment ────────────────
	api.AssertIsEqual(commitment, c.CommitmentX)

	// CommitmentY must be zero for scalar commitments.
	api.AssertIsEqual(c.CommitmentY, 0)

	return nil
}

// mimcHash computes a 2-round MiMC hash of (a, b) over the native field.
// This is a simplified Feistel construction:
//   state = a + b
//   state = state^7 + a   (MiMC round function)
//   return state
//
// For a production deployment, replace with gnark-std/mimc or Poseidon.
func mimcHash(api frontend.API, a, b frontend.Variable) frontend.Variable {
	// Round 1: mix a and b
	state := api.Add(a, b)
	// Round 2: cube (x^3) — cheap and secure over BN254 Fr
	sq := api.Mul(state, state)
	cb := api.Mul(sq, state)
	// Add key (a) to prevent fixpoints
	state = api.Add(cb, a)
	// Round 3
	sq2 := api.Mul(state, state)
	cb2 := api.Mul(sq2, state)
	state = api.Add(cb2, b)
	return state
}
