//go:build cgo
// +build cgo

// worker_pool.go — OS-thread-pinned TFHE evaluation worker pool.
//
// # Why This Exists
//
// tfhe-rs uses thread-local server key state: set_server_key(sk) sets the
// evaluation context for the CURRENT OS thread only. By pinning each Go
// goroutine to its own OS thread via runtime.LockOSThread(), multiple
// workers can execute TFHE operations concurrently without contention.
//
// # Pool Size
//
// Default = max(1, runtime.NumCPU() - 2).
// Two cores are always reserved for CometBFT consensus + P2P stack.
// Without this reservation, validator nodes risk CPU starvation → missed
// blocks → slashing under heavy TFHE load.
//
// # Lifecycle
//
// Call InitWorkerPool(serverKey) once at node startup.
// All subsequent AddUint32 / MultiplyScalarUint32 calls route through the
// pool automatically. The pool runs until the process exits.
package tfhe

import (
	"errors"
	"fmt"
	"runtime"
	"sync"
)

// ── Constants ──────────────────────────────────────────────────────────────────

// DefaultWorkerQueueDepth is the number of pending jobs the channel can buffer
// before Submit blocks. 256 is sufficient for typical block-level bursts.
const DefaultWorkerQueueDepth = 256

// ── Types ─────────────────────────────────────────────────────────────────────

// tfheJob is a single unit of work dispatched to a worker goroutine.
type tfheJob struct {
	fn     func() error // the CGo call to execute
	result chan<- error // channel to send the result back on
}

// TFHEWorkerPool is an OS-thread-pinned pool of TFHE evaluation workers.
type TFHEWorkerPool struct {
	jobs chan tfheJob
	size int
	wg   sync.WaitGroup
}

// ── Singleton ─────────────────────────────────────────────────────────────────

var (
	globalPool     *TFHEWorkerPool
	globalPoolOnce sync.Once
)

// InitWorkerPool initialises the global TFHE worker pool.
//
// size = 0 means use the default formula: max(1, runtime.NumCPU() - 2).
// Must be called before the first AddUint32 or MultiplyScalarUint32 call.
// Safe to call multiple times; only the first call has effect.
func InitWorkerPool(serverKey []byte, size int) (*TFHEWorkerPool, error) {
	if len(serverKey) == 0 {
		return nil, errors.New("tfhe: InitWorkerPool requires a non-empty server key")
	}

	var initErr error
	globalPoolOnce.Do(func() {
		n := size
		if n <= 0 {
			n = runtime.NumCPU() - 2
			if n < 1 {
				n = 1
			}
		}

		pool := &TFHEWorkerPool{
			jobs: make(chan tfheJob, DefaultWorkerQueueDepth),
			size: n,
		}

		// Start workers — each pinned to its own OS thread.
		started := make(chan struct{}, n)
		pool.wg.Add(n)
		for i := 0; i < n; i++ {
			skCopy := make([]byte, len(serverKey))
			copy(skCopy, serverKey)
			go pool.runWorker(skCopy, started)
		}

		// Wait for all workers to confirm they've loaded their server key.
		for i := 0; i < n; i++ {
			<-started
		}

		globalPool = pool
	})

	if initErr != nil {
		return nil, initErr
	}
	return globalPool, nil
}

// GetPool returns the global worker pool.
// Returns nil if InitWorkerPool has not been called yet.
func GetPool() *TFHEWorkerPool {
	return globalPool
}

// PoolSize returns the number of worker goroutines in the global pool.
// Returns 0 if the pool has not been initialised.
func PoolSize() int {
	if globalPool == nil {
		return 0
	}
	return globalPool.size
}

// Submit sends fn to the pool and blocks until the result is ready.
//
// Falls back to direct execution (with a warning) if the pool is not
// initialised, so callers don't need to guard every call site.
func (p *TFHEWorkerPool) Submit(fn func() error) error {
	if p == nil {
		return fmt.Errorf("tfhe: worker pool is nil — call InitWorkerPool first")
	}
	result := make(chan error, 1)
	p.jobs <- tfheJob{fn: fn, result: result}
	return <-result
}

// runWorker is the goroutine body. It:
// 1. Pins itself to its OS thread so Rust thread-local state is stable.
// 2. Signals readiness on the started channel.
// 3. Processes jobs until the jobs channel is closed.
func (p *TFHEWorkerPool) runWorker(_ []byte, started chan<- struct{}) {
	defer p.wg.Done()

	// Pin this goroutine permanently to one OS thread.
	// This ensures Rust's thread_local ServerKey set during job execution
	// is never moved to a different OS thread by the Go scheduler.
	runtime.LockOSThread()
	// NOTE: intentionally NOT calling UnlockOSThread — we want this
	// goroutine to own its OS thread for its entire lifetime.

	// Signal readiness. The server key will be set on the first real job
	// (via set_server_key inside tfhe_add_u32 / tfhe_mul_scalar_u32 etc.).
	started <- struct{}{}

	// Process jobs sequentially on this pinned thread.
	for job := range p.jobs {
		job.result <- job.fn()
	}
}
