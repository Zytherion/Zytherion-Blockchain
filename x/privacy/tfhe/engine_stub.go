//go:build !cgo
// +build !cgo

package tfhe

import "errors"

var ErrTFHEDisabled = errors.New(
	"TFHE engine is not compiled in — CGo is disabled or GCC is missing during compilation",
)

type TFHEKeyPair struct {
	ClientKey []byte
	ServerKey []byte
}

func GenerateKeys() (*TFHEKeyPair, error) {
	return nil, ErrTFHEDisabled
}

func EncryptUint32(_ []byte, _ uint32) ([]byte, error) {
	return nil, ErrTFHEDisabled
}

func EncryptWithPublicKey(_ []byte, _ uint32) ([]byte, error) {
	return nil, ErrTFHEDisabled
}

func EncryptWithServerKey(_ []byte, _ uint32) ([]byte, error) {
	return nil, ErrTFHEDisabled
}

func AddUint32(_, _, _ []byte) ([]byte, error) {
	return nil, ErrTFHEDisabled
}

func SubUint32(_, _, _ []byte) ([]byte, error) {
	return nil, ErrTFHEDisabled
}

func MultiplyScalarUint32(_, _ []byte, _ uint32) ([]byte, error) {
	return nil, ErrTFHEDisabled
}

func DecryptUint32(_, _ []byte) (uint32, error) {
	return 0, ErrTFHEDisabled
}
