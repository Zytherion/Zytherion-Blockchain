// signature.go — Post-Quantum signature primitives for Zytherion validators.
//
// # Algorithm choice: Dilithium5 (ML-DSA Level 5)
//
// We use Dilithium5 via github.com/cloudflare/circl/sign/dilithium/mode5.
// Dilithium is a NIST PQC round-3 winner (standardized as ML-DSA in FIPS 204)
// whose security rests on the hardness of Module-LWE and Module-SIS on
// structured (NTT-friendly) lattices.
//
// Dilithium2 vs Dilithium3 vs Dilithium5:
//   - Dilithium2: NIST cat-2 (~128-bit PQ)  — smallest keys & sigs.
//   - Dilithium3: NIST cat-3 (~192-bit PQ)  — intermediate.
//   - Dilithium5: NIST cat-5 (~256-bit PQ)  — CHOSEN: maximum security margin.
//
// Dilithium5 is selected for Zytherion because the chain is pre-mainnet and
// can absorb the larger key/signature overhead in exchange for the strongest
// available post-quantum security level, future-proofing against quantum
// adversaries with improved fault tolerance.
//
// Key and signature sizes (Dilithium5 / ML-DSA Level 5):
//   - Public key:  2592 bytes  (mode5.PublicKeySize)
//   - Private key: 4864 bytes  (mode5.PrivateKeySize)
//   - Signature:   4595 bytes  (mode5.SignatureSize)
package pqc

import (
	"fmt"

	mode5 "github.com/cloudflare/circl/sign/dilithium/mode5"
)

// KeyPair holds a Dilithium5 key pair as raw byte slices.
// Raw bytes make keys easy to persist in Cosmos SDK key stores and transmit
// over gRPC without introducing package-level type coupling.
type KeyPair struct {
	// PublicKey is the 2592-byte Dilithium5 public verification key.
	PublicKey []byte

	// PrivateKey is the 4864-byte Dilithium5 private signing key.
	// Treat this as a secret; clear it from memory when no longer needed.
	PrivateKey []byte
}

// GenerateKeyPair generates a fresh Dilithium5 key pair using the OS CSPRNG
// (crypto/rand, via circl's internal implementation).
//
// Returns an error only if the system entropy source fails, which is extremely
// rare and typically indicates a serious OS-level problem.
func GenerateKeyPair() (KeyPair, error) {
	pub, priv, err := mode5.GenerateKey(nil) // nil → crypto/rand
	if err != nil {
		return KeyPair{}, fmt.Errorf("dilithium5 keygen: %w", err)
	}
	return KeyPair{
		PublicKey:  pub.Bytes(),
		PrivateKey: priv.Bytes(),
	}, nil
}

// Sign produces a deterministic Dilithium5 signature over message using the
// given private key bytes.
//
// Dilithium5 signing is deterministic: for the same (message, privateKey)
// pair the output signature is always byte-for-byte identical. This property
// is required by Green-BFT so that validators can cache and compare proposals.
//
// Returns an error if privKeyBytes does not have the expected length
// (mode5.PrivateKeySize = 4864 bytes).
func Sign(message, privKeyBytes []byte) ([]byte, error) {
	if len(privKeyBytes) != mode5.PrivateKeySize {
		return nil, fmt.Errorf("dilithium5 sign: invalid private key length %d (want %d)",
			len(privKeyBytes), mode5.PrivateKeySize)
	}

	var buf [mode5.PrivateKeySize]byte
	copy(buf[:], privKeyBytes)

	var sk mode5.PrivateKey
	sk.Unpack(&buf)

	sig := make([]byte, mode5.SignatureSize)
	mode5.SignTo(&sk, message, sig)
	return sig, nil
}

// Verify returns true if and only if signature is a valid Dilithium5 signature
// over message created with the private key paired with pubKeyBytes.
//
// Returns false (not an error) for invalid signatures or malformed public keys,
// making it safe to call directly inside consensus validation hot paths.
func Verify(message, signature, pubKeyBytes []byte) bool {
	if len(pubKeyBytes) != mode5.PublicKeySize {
		return false // wrong size → cannot be a valid public key
	}
	if len(signature) != mode5.SignatureSize {
		return false // wrong sig length → definite reject
	}

	var buf [mode5.PublicKeySize]byte
	copy(buf[:], pubKeyBytes)

	var pk mode5.PublicKey
	pk.Unpack(&buf)

	return mode5.Verify(&pk, message, signature)
}

// DilithiumPublicKeySize is the expected byte-length of a Dilithium5 public key.
// Exported for use by consumers that want to validate key lengths up-front.
const DilithiumPublicKeySize = mode5.PublicKeySize

// DilithiumSignatureSize is the expected byte-length of a Dilithium5 signature.
const DilithiumSignatureSize = mode5.SignatureSize

// DilithiumPrivateKeySize is the expected byte-length of a Dilithium5 private key.
const DilithiumPrivateKeySize = mode5.PrivateKeySize

// DilithiumAlgorithm is the human-readable name of the signature algorithm.
const DilithiumAlgorithm = "ML-DSA-87 (Dilithium5, NIST Category 5, ~256-bit PQ security)"
