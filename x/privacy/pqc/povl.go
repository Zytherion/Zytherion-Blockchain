package pqc

import (
	"bytes"

	"golang.org/x/crypto/sha3"
)

// PoVLStateSize is the size of the PoVL state commitment in bytes.
const PoVLStateSize = 32

// GeneratePoVLState computes a single step of the PoVL sequential VDF-like clock.
// It applies a deterministic LWR transformation to the input state.
//
// state_n = SHA3-256( LWRHash(state_{n-1}) || state_{n-1} )
func GeneratePoVLState(prevState []byte) []byte {
	if len(prevState) == 0 {
		prevState = make([]byte, PoVLStateSize)
	}
	
	// Ensure the state is exactly PoVLStateSize
	currentState := zeroPad32(prevState)

	// We use the LWR hash function as the core mixing primitive for PoVL
	// using the previous state as both the input and the "prevHash" to
	// bind it tightly.
	lwrMix, _ := GenerateLWRBlockHash(currentState, currentState)

	// The next state is the SHA3-256 of the LWR mix concatenated with the prev state
	h := sha3.New256()
	h.Write(lwrMix)
	h.Write(currentState)
	return h.Sum(nil)
}

// ComputePoVLChain computes N sequential steps of the PoVL clock, starting
// from the initialState. Returns the final state commitment.
func ComputePoVLChain(initialState []byte, nSteps uint64) []byte {
	state := zeroPad32(initialState)
	for i := uint64(0); i < nSteps; i++ {
		state = GeneratePoVLState(state)
	}
	return state
}

// VerifyPoVLChain verifies that computing nSteps from initialState
// results exactly in expectedFinalState.
func VerifyPoVLChain(initialState, expectedFinalState []byte, nSteps uint64) bool {
	computedFinalState := ComputePoVLChain(initialState, nSteps)
	return bytes.Equal(computedFinalState, expectedFinalState)
}
