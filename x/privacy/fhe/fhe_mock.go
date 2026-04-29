//go:build notfhe
// +build notfhe

package fhe

import (
	"fmt"
	"sync"
)

type Context struct {
	mu sync.Mutex
}

func NewContext() (*Context, error) {
	return nil, fmt.Errorf("fhe: TFHE-rs not available (notfhe build). Run make build-tfhe first")
}

func NewContextFromKeys(_, _ []byte) (*Context, error) {
	return nil, fmt.Errorf("fhe: not available (notfhe build tag)")
}

func (c *Context) Encrypt(_ uint64) ([]byte, error) {
	return nil, fmt.Errorf("fhe: not available (notfhe build tag)")
}

func (c *Context) Decrypt(_ []byte) (uint64, error) {
	return 0, fmt.Errorf("fhe: not available (notfhe build tag)")
}

func (c *Context) AddCiphertexts(_, _ []byte) ([]byte, error) {
	return nil, fmt.Errorf("fhe: not available (notfhe build tag)")
}

func (c *Context) SubCiphertexts(_, _ []byte) ([]byte, error) {
	return nil, fmt.Errorf("fhe: not available (notfhe build tag)")
}