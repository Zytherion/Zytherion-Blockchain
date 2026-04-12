// signature.go — Post-Quantum signature primitives for Zytherion validators.
//
// # Algorithm choice: Dilithium3
//
// We use Dilithium3 via github.com/cloudflare/circl/sign/dilithium/mode3.
// Dilithium is a NIST PQC round-3 winner whose security rests on the hardness
// of Module-LWE and Module-SIS on structured (NTT-friendly) lattices.
//
// Dilithium2 vs Dilithium3 vs Dilithium5:
//   - Dilithium2: NIST cat-2 (~128-bit PQ)  — smallest keys & sigs.
//   - Dilithium3: NIST cat-3 (~192-bit PQ)  — chosen here for better margin.
//   - Dilithium5: NIST cat-5 (~256-bit PQ)  — largest keys.
//
// Dilithium3 is chosen because it provides meaningful security headroom over
// Dilithium2 while keeping key/signature sizes manageable for a blockchain
// where every validator signature must be included in each block.
//
// Key and signature sizes (Dilithium3):
//   - Public key:  1952 bytes  (mode3.PublicKeySize)
//   - Private key: 4000 bytes  (mode3.PrivateKeySize)
//   - Signature:   3293 bytes  (mode3.SignatureSize)
package pqc

import (
	"fmt"

	mode3 "github.com/cloudflare/circl/sign/dilithium/mode3"
)

// KeyPair holds a Dilithium3 key pair as raw byte slices.
// Raw bytes make keys easy to persist in Cosmos SDK key stores and transmit
// over gRPC without introducing package-level type coupling.
type KeyPair struct {
	// PublicKey is the 1952-byte Dilithium3 public verification key.
	PublicKey []byte

	// PrivateKey is the 4000-byte Dilithium3 private signing key.
	// Treat this as a secret; clear it from memory when no longer needed.
	PrivateKey []byte
}

// GenerateKeyPair generates a fresh Dilithium3 key pair using the OS CSPRNG
// (crypto/rand, via circl's internal implementation).
//
// Returns an error only if the system entropy source fails, which is extremely
// rare and typically indicates a serious OS-level problem.
func GenerateKeyPair() (KeyPair, error) {
	pub, priv, err := mode3.GenerateKey(nil) // nil → crypto/rand
	if err != nil {
		return KeyPair{}, fmt.Errorf("dilithium3 keygen: %w", err)
	}
	return KeyPair{
		PublicKey:  pub.Bytes(),
		PrivateKey: priv.Bytes(),
	}, nil
}

// Sign produces a deterministic Dilithium3 signature over message using the
// given private key bytes.
//
// Dilithium3 signing is deterministic: for the same (message, privateKey)
// pair the output signature is always byte-for-byte identical. This property
// is required by Green-BFT so that validators can cache and compare proposals.
//
// Returns an error if privKeyBytes does not have the expected length
// (mode3.PrivateKeySize = 4000 bytes).
func Sign(message, privKeyBytes []byte) ([]byte, error) {
	if len(privKeyBytes) != mode3.PrivateKeySize {
		return nil, fmt.Errorf("dilithium3 sign: invalid private key length %d (want %d)",
			len(privKeyBytes), mode3.PrivateKeySize)
	}

	var buf [mode3.PrivateKeySize]byte
	copy(buf[:], privKeyBytes)

	var sk mode3.PrivateKey
	sk.Unpack(&buf)

	sig := make([]byte, mode3.SignatureSize)
	mode3.SignTo(&sk, message, sig)
	return sig, nil
}

// Verify returns true if and only if signature is a valid Dilithium3 signature
// over message created with the private key paired with pubKeyBytes.
//
// Returns false (not an error) for invalid signatures or malformed public keys,
// making it safe to call directly inside consensus validation hot paths.
func Verify(message, signature, pubKeyBytes []byte) bool {
	if len(pubKeyBytes) != mode3.PublicKeySize {
		return false // wrong size → cannot be a valid public key
	}
	if len(signature) != mode3.SignatureSize {
		return false // wrong sig length → definite reject
	}

	var buf [mode3.PublicKeySize]byte
	copy(buf[:], pubKeyBytes)

	var pk mode3.PublicKey
	pk.Unpack(&buf)

	return mode3.Verify(&pk, message, signature)
}

// DilithiumPublicKeySize is the expected byte-length of a Dilithium3 public key.
// Exported for use by consumers that want to validate key lengths up-front.
const DilithiumPublicKeySize = mode3.PublicKeySize

// DilithiumSignatureSize is the expected byte-length of a Dilithium3 signature.
const DilithiumSignatureSize = mode3.SignatureSize


