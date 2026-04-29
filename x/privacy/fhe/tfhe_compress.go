//go:build !notfhe
// +build !notfhe

package fhe

import "fmt"

func compressCiphertext(serverKey []byte, ct []byte) ([]byte, error) {
	if len(serverKey) == 0 {
		return nil, fmt.Errorf("fhe/compress: server key is empty")
	}
	if len(ct) == 0 {
		return nil, fmt.Errorf("fhe/compress: ciphertext is empty")
	}
	return tfheCompress(serverKey, ct)
}

func decompressCiphertext(serverKey []byte, compressed []byte) ([]byte, error) {
	if len(serverKey) == 0 {
		return nil, fmt.Errorf("fhe/compress: server key is empty")
	}
	if len(compressed) == 0 {
		return nil, fmt.Errorf("fhe/compress: compressed bytes are empty")
	}
	return tfheDecompress(serverKey, compressed)
}