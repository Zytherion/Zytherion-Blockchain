//go:build notfhe
// +build notfhe

package fhe

import "fmt"

func compressCiphertext(_, _ []byte) ([]byte, error) {
	return nil, fmt.Errorf("fhe: not available (notfhe build tag)")
}

func decompressCiphertext(_, _ []byte) ([]byte, error) {
	return nil, fmt.Errorf("fhe: not available (notfhe build tag)")
}