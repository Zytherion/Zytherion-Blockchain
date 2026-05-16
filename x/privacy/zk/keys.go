// keys.go — Verifying key (VK) management for Zytherion ZK subsystem.
//
// The verifying key is a public artifact generated once during trusted setup
// (tools/zksetup) and committed to the repository. Every validator loads it
// at startup; it never changes unless the circuit definition changes.
//
// Key lifecycle:
//   1. Developer runs `make zksetup` → generates verifying_key.bin + proving_key.bin
//   2. verifying_key.bin is committed to the repo under keys/
//   3. At node startup, LoadVerifyingKey reads the file from disk
//   4. The VK bytes are stored in the Keeper and used by VerifyTransferProof
package zk

import (
	"bytes"
	"errors"
	"fmt"
	"os"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark/constraint"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/frontend/cs/r1cs"
)

// DefaultVKPath is the default location for the verifying key in the repo.
const DefaultVKPath = "keys/verifying_key.bin"

// DefaultPKPath is the default location for the proving key.
const DefaultPKPath = "keys/proving_key.bin"

// LoadVerifyingKeyBytes reads the verifying key from disk and returns its
// raw serialized bytes. These bytes are stored in the Keeper and passed
// directly to VerifyTransferProof — no re-parsing needed per-transaction.
func LoadVerifyingKeyBytes(path string) ([]byte, error) {
	if path == "" {
		path = DefaultVKPath
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("zk: failed to read verifying key from %q: %w", path, err)
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("zk: verifying key file %q is empty", path)
	}
	// Quick sanity: try to parse it
	vk := groth16.NewVerifyingKey(ecc.BN254)
	if _, err := vk.ReadFrom(bytes.NewReader(data)); err != nil {
		return nil, fmt.Errorf("zk: verifying key file %q is not a valid Groth16 VK: %w", path, err)
	}
	return data, nil
}

// CompileCircuit compiles the TransferCircuit into a Groth16 constraint system.
// This is called by the trusted setup tool (tools/zksetup), not on-chain.
func CompileCircuit() (constraint.ConstraintSystem, error) {
	var circuit TransferCircuit
	cs, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, &circuit)
	if err != nil {
		return nil, fmt.Errorf("zk: circuit compilation failed: %w", err)
	}
	return cs, nil
}

// GenerateKeys runs the Groth16 trusted setup for the TransferCircuit.
// Returns (provingKey bytes, verifyingKey bytes, error).
// This is called ONCE by tools/zksetup — NOT on-chain.
func GenerateKeys() (pkBytes, vkBytes []byte, err error) {
	cs, err := CompileCircuit()
	if err != nil {
		return nil, nil, err
	}

	pk, vk, err := groth16.Setup(cs)
	if err != nil {
		return nil, nil, fmt.Errorf("zk: Groth16 setup failed: %w", err)
	}

	// Serialize PK
	var pkBuf bytes.Buffer
	if _, err := pk.WriteTo(&pkBuf); err != nil {
		return nil, nil, fmt.Errorf("zk: failed to serialize proving key: %w", err)
	}

	// Serialize VK
	var vkBuf bytes.Buffer
	if _, err := vk.WriteTo(&vkBuf); err != nil {
		return nil, nil, fmt.Errorf("zk: failed to serialize verifying key: %w", err)
	}

	return pkBuf.Bytes(), vkBuf.Bytes(), nil
}

// GenerateProof generates a Groth16 proof for the given (amount, blinding) pair.
// Returns (proofBytes, commitmentBytes, error).
// This is called by tools/zkprove — NOT on-chain.
func GenerateProof(pkBytes []byte, amount uint64, blinding []byte) (proofBytes, commitment []byte, err error) {
	if len(pkBytes) == 0 {
		return nil, nil, errors.New("zk: proving key bytes are empty")
	}

	// 1. Compute commitment
	commitment, err = Commit(amount, blinding)
	if err != nil {
		return nil, nil, fmt.Errorf("zk: commitment computation failed: %w", err)
	}

	// 2. Build the full witness
	assignment := &TransferCircuit{
		CommitmentX: new(ScalarFr).SetBytesReturn(commitment).BigInt(),
		CommitmentY: 0,
		Amount:      amount,
		Blinding:    new(ScalarFr).SetBytesReturn(blinding).BigInt(),
	}

	// 3. Compile circuit (cached in production; acceptable here for CLI tool)
	cs, err := CompileCircuit()
	if err != nil {
		return nil, nil, err
	}

	// 4. Build full witness
	witness, err := frontend.NewWitness(assignment, ecc.BN254.ScalarField())
	if err != nil {
		return nil, nil, fmt.Errorf("zk: failed to build witness: %w", err)
	}

	// 5. Deserialize PK
	pk := groth16.NewProvingKey(ecc.BN254)
	if _, err := pk.ReadFrom(bytes.NewReader(pkBytes)); err != nil {
		return nil, nil, fmt.Errorf("zk: failed to deserialize proving key: %w", err)
	}

	// 6. Generate Groth16 proof
	proof, err := groth16.Prove(cs, pk, witness)
	if err != nil {
		return nil, nil, fmt.Errorf("zk: Groth16 prove failed: %w", err)
	}

	// 7. Serialize proof
	var proofBuf bytes.Buffer
	if _, err := proof.WriteTo(&proofBuf); err != nil {
		return nil, nil, fmt.Errorf("zk: failed to serialize proof: %w", err)
	}

	return proofBuf.Bytes(), commitment, nil
}

// SetBytesReturn is a fluent helper that calls SetBytes and returns the receiver.
func (s *ScalarFr) SetBytesReturn(b []byte) *ScalarFr {
	s.SetBytes(b)
	return s
}
