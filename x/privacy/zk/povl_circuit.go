package zk

import (
	"github.com/consensys/gnark/frontend"
)

// PoVLCircuit defines the ZK constraint system for a Proof of Verifiable Lattices (PoVL).
//
// In a full production implementation, this circuit would unroll the exact
// negacyclic convolution `b = (A.s * p)/q mod p`. However, due to the immense
// constraint size of Ring-LWR within a BN254 SNARK, this reference implementation
// compresses the N-step sequential chain using a repeated Poseidon (MiMC) hash
// function to prove sequential computational delay.
//
//   Public  : InitialState, FinalState, StepCount N
//   Private : None (This is a VDF-style deterministic computation proof)
//
// The circuit verifies:
//  1. Starting from InitialState, executing N steps of the deterministic mixing function
//     results exactly in FinalState.
type PoVLCircuit struct {
	// --- Public inputs (known to verifier) ---

	InitialState frontend.Variable `gnark:",public"`
	FinalState   frontend.Variable `gnark:",public"`

	// We fix N at compile time for the circuit structure because SNARKs require
	// fixed-size loops. In practice, you would compile several circuits for
	// different N (e.g., N=10, N=100) or use a recursive SNARK.
	StepCount int // Not a frontend.Variable, but a circuit configuration parameter
}

// Define sets up the R1CS constraints for the PoVL circuit.
func (c *PoVLCircuit) Define(api frontend.API) error {
	state := c.InitialState

	// Unroll the loop N times. Each step represents a sequential transformation.
	for i := 0; i < c.StepCount; i++ {
		// We use MiMC to represent the VDF step.
		// In a full LWR implementation, this would be the LWR polynomial multiplication.
		state = mimcHash(api, state, state)
	}

	// Assert that the computed final state matches the public FinalState
	api.AssertIsEqual(state, c.FinalState)

	return nil
}
