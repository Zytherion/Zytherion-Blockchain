//go:build notfhe
// +build notfhe

package fhe

func tfheGenerateKeys() ([]byte, []byte, error)          { return nil, nil, nil }
func tfheEncrypt(_ []byte, _ uint64) ([]byte, error)     { return nil, nil }
func tfheDecrypt(_ []byte, _ []byte) (uint64, error)     { return 0, nil }
func tfheAdd(_, _, _ []byte) ([]byte, error)             { return nil, nil }
func tfheSub(_, _, _ []byte) ([]byte, error)             { return nil, nil }
func tfheCompress(_, _ []byte) ([]byte, error)           { return nil, nil }
func tfheDecompress(_, _ []byte) ([]byte, error)         { return nil, nil }