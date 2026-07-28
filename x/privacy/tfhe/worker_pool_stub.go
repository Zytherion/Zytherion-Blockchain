//go:build !cgo
// +build !cgo

package tfhe

import "errors"

type TFHEWorkerPool struct{}

func InitWorkerPool(_ []byte, _ int) (*TFHEWorkerPool, error) {
	return &TFHEWorkerPool{}, nil
}

func GetPool() *TFHEWorkerPool {
	return nil
}

func PoolSize() int {
	return 0
}

func (p *TFHEWorkerPool) Submit(fn func() error) error {
	if p == nil {
		return errors.New("tfhe: pool not available")
	}
	return fn()
}
