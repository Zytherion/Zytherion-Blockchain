//go:build tfhe_cgo
// +build tfhe_cgo

// engine.go — CGo wrapper for the tfhe_c Rust library.
//
// This file provides the Go API for TFHE (Fully Homomorphic Encryption)
// operations on 32-bit unsigned integers (FheUint32).
//
// # Build Requirements
//
//  1. Rust toolchain installed (rustup, cargo)
//  2. tfhe_c compiled: `cd x/privacy/tfhe/tfhe_c && cargo build --release`
//  3. The static lib must exist at the path referenced by LDFLAGS below.
//
// Build with: go build -tags tfhe_cgo ./...
// (Without -tags tfhe_cgo, engine_stub.go is used instead — all TFHE ops return ErrTFHEDisabled)
//
// # CGo Linking
//
// The CGo directives below instruct the Go toolchain to:
//   - Include the local cgo_bridge.h header
//   - Link the compiled libtfhe_c.a static library
//   - Link platform math/pthread libs required by the Rust binary
//
// # Thread Safety
//
// Each TFHE operation (especially Add, which sets a global server key) is
// guarded by a mutex to ensure only one goroutine calls into Rust at a time.
// This is required because tfhe-rs uses a thread-local-global ServerKey.

package tfhe

/*
#cgo CFLAGS: -I${SRCDIR}
#cgo LDFLAGS: -L${SRCDIR}/tfhe_c/target/release -ltfhe_c -lm -ldl -lpthread
#include "cgo_bridge.h"
#include <stdlib.h>
*/
import "C"
import (
	"errors"
	"fmt"
	"sync"
	"unsafe"
)

// ── Constants ──────────────────────────────────────────────────────────────────

// Note: CiphertextMaxBytes is defined in erasure.go to avoid CGo dependency.
// Use tfhe.CiphertextMaxBytes from that file directly.

// ── Global mutex ──────────────────────────────────────────────────────────────

// tfheMu serialises all calls into the Rust library.
// Required because set_server_key() in tfhe-rs sets a process-global key.
var tfheMu sync.Mutex

// ── KeyPair ───────────────────────────────────────────────────────────────────

// TFHEKeyPair holds the serialized TFHE ClientKey and ServerKey.
// ClientKey: used for Encrypt/Decrypt (must stay private).
// ServerKey: used for homomorphic evaluation (can be public/shared).
type TFHEKeyPair struct {
	ClientKey []byte
	ServerKey []byte
}

// GenerateKeys generates a fresh TFHE ClientKey + ServerKey pair.
//
// WARNING: Key generation is SLOW (~10–60 seconds depending on CPU).
// This should only be called once per node, with keys persisted to disk.
func GenerateKeys() (*TFHEKeyPair, error) {
	tfheMu.Lock()
	defer tfheMu.Unlock()

	ckMaxLen := C.tfhe_client_key_max_bytes()
	skMaxLen := C.tfhe_server_key_max_bytes()

	ckBuf := C.malloc(C.size_t(ckMaxLen))
	skBuf := C.malloc(C.size_t(skMaxLen))
	defer C.free(ckBuf)
	defer C.free(skBuf)

	var ckLen, skLen C.uint64_t

	ret := C.tfhe_keygen(
		(*C.uint8_t)(ckBuf), &ckLen,
		(*C.uint8_t)(skBuf), &skLen,
	)
	if ret != 0 {
		return nil, errors.New("tfhe: key generation failed in Rust library")
	}

	ck := C.GoBytes(ckBuf, C.int(ckLen))
	sk := C.GoBytes(skBuf, C.int(skLen))

	return &TFHEKeyPair{
		ClientKey: ck,
		ServerKey: sk,
	}, nil
}

// ── Encrypt ───────────────────────────────────────────────────────────────────

// EncryptUint32 encrypts a 32-bit unsigned integer using the TFHE ClientKey.
//
// Returns a serialised FheUint32 ciphertext (~16–21 KB in practice).
// The caller must supply the serialized ClientKey bytes from TFHEKeyPair.ClientKey.
func EncryptUint32(clientKey []byte, plaintext uint32) ([]byte, error) {
	if len(clientKey) == 0 {
		return nil, errors.New("tfhe: client key is empty")
	}

	tfheMu.Lock()
	defer tfheMu.Unlock()

	ctBuf := C.malloc(C.size_t(CiphertextMaxBytes))
	defer C.free(ctBuf)

	ckPtr := (*C.uint8_t)(unsafe.Pointer(&clientKey[0]))

	ctLen := C.tfhe_encrypt_u32(
		ckPtr, C.uint64_t(len(clientKey)),
		C.uint32_t(plaintext),
		(*C.uint8_t)(ctBuf), C.uint64_t(CiphertextMaxBytes),
	)
	if ctLen < 0 {
		return nil, fmt.Errorf("tfhe: encryption failed (Rust returned %d)", int(ctLen))
	}

	return C.GoBytes(ctBuf, C.int(ctLen)), nil
}

// ── Homomorphic Add ───────────────────────────────────────────────────────────

// AddUint32 performs homomorphic addition of two FheUint32 ciphertexts.
//
//	result_ct = Enc(a) + Enc(b)   →   when decrypted: Dec(result_ct) == (a + b) mod 2^32
//
// The ServerKey is required to evaluate the circuit. Both ciphertexts must
// have been encrypted under the same configuration.
func AddUint32(serverKey, ct1, ct2 []byte) ([]byte, error) {
	if len(serverKey) == 0 {
		return nil, errors.New("tfhe: server key is empty")
	}
	if len(ct1) == 0 || len(ct2) == 0 {
		return nil, errors.New("tfhe: ciphertext inputs must not be empty")
	}

	tfheMu.Lock()
	defer tfheMu.Unlock()

	resultBuf := C.malloc(C.size_t(CiphertextMaxBytes))
	defer C.free(resultBuf)

	skPtr := (*C.uint8_t)(unsafe.Pointer(&serverKey[0]))
	c1Ptr := (*C.uint8_t)(unsafe.Pointer(&ct1[0]))
	c2Ptr := (*C.uint8_t)(unsafe.Pointer(&ct2[0]))

	resLen := C.tfhe_add_u32(
		skPtr, C.uint64_t(len(serverKey)),
		c1Ptr, C.uint64_t(len(ct1)),
		c2Ptr, C.uint64_t(len(ct2)),
		(*C.uint8_t)(resultBuf), C.uint64_t(CiphertextMaxBytes),
	)
	if resLen < 0 {
		return nil, fmt.Errorf("tfhe: homomorphic add failed (Rust returned %d)", int(resLen))
	}

	return C.GoBytes(resultBuf, C.int(resLen)), nil
}

// ── Scalar Multiply ───────────────────────────────────────────────────────────

// MultiplyScalarUint32 multiplies a ciphertext by an unencrypted scalar.
//
//	result_ct = Enc(a) * scalar   →   when decrypted: Dec(result_ct) == (a * scalar) mod 2^32
func MultiplyScalarUint32(serverKey, ct []byte, scalar uint32) ([]byte, error) {
	if len(serverKey) == 0 {
		return nil, errors.New("tfhe: server key is empty")
	}
	if len(ct) == 0 {
		return nil, errors.New("tfhe: ciphertext must not be empty")
	}

	tfheMu.Lock()
	defer tfheMu.Unlock()

	resultBuf := C.malloc(C.size_t(CiphertextMaxBytes))
	defer C.free(resultBuf)

	skPtr := (*C.uint8_t)(unsafe.Pointer(&serverKey[0]))
	ctPtr := (*C.uint8_t)(unsafe.Pointer(&ct[0]))

	resLen := C.tfhe_mul_scalar_u32(
		skPtr, C.uint64_t(len(serverKey)),
		ctPtr, C.uint64_t(len(ct)),
		C.uint32_t(scalar),
		(*C.uint8_t)(resultBuf), C.uint64_t(CiphertextMaxBytes),
	)
	if resLen < 0 {
		return nil, fmt.Errorf("tfhe: scalar multiply failed (Rust returned %d)", int(resLen))
	}

	return C.GoBytes(resultBuf, C.int(resLen)), nil
}

// ── Decrypt ───────────────────────────────────────────────────────────────────

// DecryptUint32 decrypts a FheUint32 ciphertext using the ClientKey.
//
// Returns the plaintext uint32 value on success.
func DecryptUint32(clientKey, ciphertext []byte) (uint32, error) {
	if len(clientKey) == 0 {
		return 0, errors.New("tfhe: client key is empty")
	}
	if len(ciphertext) == 0 {
		return 0, errors.New("tfhe: ciphertext is empty")
	}

	tfheMu.Lock()
	defer tfheMu.Unlock()

	ckPtr := (*C.uint8_t)(unsafe.Pointer(&clientKey[0]))
	ctPtr := (*C.uint8_t)(unsafe.Pointer(&ciphertext[0]))

	result := C.tfhe_decrypt_u32(
		ckPtr, C.uint64_t(len(clientKey)),
		ctPtr, C.uint64_t(len(ciphertext)),
	)
	if result < 0 {
		return 0, fmt.Errorf("tfhe: decryption failed (Rust returned %d)", int(result))
	}

	return uint32(result), nil
}
