//go:build !tfhe_cgo
// +build !tfhe_cgo

// engine_stub.go — Stub implementation of the TFHE engine for non-CGo builds.
//
// When building WITHOUT -tags tfhe_cgo, this file is compiled instead of engine.go.
// All TFHE operations return ErrTFHEDisabled so the binary still links and
// the privacy module compiles without needing the Rust library installed.
//
// To enable real TFHE:
//  1. Install Rust: https://rustup.rs
//  2. make build-tfhe-rs
//  3. go build -tags tfhe_cgo ./...
package tfhe

import "errors"

// ErrTFHEDisabled is returned by all stub TFHE functions when the CGo engine
// is not compiled in. Enable it with: go build -tags tfhe_cgo ./...
var ErrTFHEDisabled = errors.New(
	"TFHE engine is not compiled in — build with -tags tfhe_cgo after running 'make build-tfhe-rs'",
)

// TFHEKeyPair holds serialized TFHE keys (stub: always empty).
type TFHEKeyPair struct {
	ClientKey []byte
	ServerKey []byte
}

// GenerateKeys always returns ErrTFHEDisabled in stub builds.
func GenerateKeys() (*TFHEKeyPair, error) {
	return nil, ErrTFHEDisabled
}

// EncryptUint32 always returns ErrTFHEDisabled in stub builds.
func EncryptUint32(_ []byte, _ uint32) ([]byte, error) {
	return nil, ErrTFHEDisabled
}

// AddUint32 always returns ErrTFHEDisabled in stub builds.
func AddUint32(_, _, _ []byte) ([]byte, error) {
	return nil, ErrTFHEDisabled
}

// MultiplyScalarUint32 always returns ErrTFHEDisabled in stub builds.
func MultiplyScalarUint32(_, _ []byte, _ uint32) ([]byte, error) {
	return nil, ErrTFHEDisabled
}

// DecryptUint32 always returns ErrTFHEDisabled in stub builds.
func DecryptUint32(_, _ []byte) (uint32, error) {
	return 0, ErrTFHEDisabled
}
