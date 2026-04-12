// greenbft.go — Green-BFT validator simulation for Zytherion.
//
// # Concept
//
// Green-BFT is a consensus efficiency extension: validators that are idle
// (no pending block proposal) redirect their compute resources toward useful
// background tasks — in the production design, these tasks are Fully
// Homomorphic Encryption (FHE) computations that support the privacy module.
//
// # This file: simulation only
//
// Real FHE is computationally expensive and requires specialised libraries
// (e.g., lattigo). This stub simulates the scheduling behaviour so the
// consensus layer can be wired up and tested without importing FHE dependencies.
// Replace the body of runFHETask with a real FHE call when ready.
package pqc

import (
	"context"
	"log"
	"sync"
	"time"
)

// Validator represents a single consensus node participating in Green-BFT.
//
// Each validator maintains an "idle window" channel: when the node is not
// actively proposing or voting on a block, it signals the idle channel,
// triggering background FHE computation.
type Validator struct {
	// ID is a human-readable identifier (e.g., validator address or moniker).
	ID string

	// PublicKey is the validator's Dilithium3 public key in serialised form.
	PublicKey []byte

	// idleWindowCh receives a signal whenever the validator enters an idle state.
	// It is a buffered channel (capacity 1) to avoid blocking the consensus loop.
	idleWindowCh chan struct{}

	// mu guards the running flag.
	mu      sync.Mutex
	running bool
}

// NewValidator constructs a Validator ready to participate in Green-BFT.
func NewValidator(id string, pubKey []byte) *Validator {
	return &Validator{
		ID:           id,
		PublicKey:    pubKey,
		idleWindowCh: make(chan struct{}, 1),
	}
}

// NotifyIdle signals that the validator has entered an idle window.
//
// Call this from the consensus engine whenever a round ends without a new
// block proposal (e.g., proposal timeout, no pending transactions).
// If the channel already has a pending signal, the call is a no-op rather
// than blocking, preventing back-pressure on the consensus goroutine.
func (v *Validator) NotifyIdle() {
	select {
	case v.idleWindowCh <- struct{}{}:
		// Signal enqueued.
	default:
		// Already one pending idle signal; skip to avoid blocking consensus.
	}
}

// ScheduleFHEComputation starts a background goroutine that waits for idle
// windows and runs mock FHE tasks during them.
//
// The goroutine runs until ctx is cancelled (e.g., node shutdown). It is safe
// to call ScheduleFHEComputation multiple times; only the first call starts
// the goroutine — subsequent calls are no-ops.
//
// To trigger an FHE task, call v.NotifyIdle() from the consensus loop.
func (v *Validator) ScheduleFHEComputation(ctx context.Context) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.running {
		return // already scheduled
	}
	v.running = true

	go func() {
		log.Printf("[GreenBFT] validator %s: FHE scheduler started", v.ID)
		for {
			select {
			case <-ctx.Done():
				log.Printf("[GreenBFT] validator %s: FHE scheduler stopped (context cancelled)", v.ID)
				return
			case <-v.idleWindowCh:
				log.Printf("[GreenBFT] validator %s: idle window detected, running FHE task", v.ID)
				runFHETask(ctx, v.ID)
			}
		}
	}()
}

// runFHETask is a mock FHE computation executed during a validator's idle window.
//
// Production replacement: substitute this with a real lattigo or TFHE call,
// e.g., evaluating a homomorphic circuit over encrypted user balances.
//
// The mock sleeps for a short duration to simulate realistic CPU usage without
// consuming real resources during testing.
func runFHETask(ctx context.Context, validatorID string) {
	const mockDuration = 50 * time.Millisecond

	select {
	case <-ctx.Done():
		// Context cancelled mid-task; abort gracefully.
		return
	case <-time.After(mockDuration):
		// Simulated FHE computation complete.
		log.Printf("[GreenBFT] validator %s: FHE task complete (mock, %.0fms)", validatorID, mockDuration.Seconds()*1000)
	}
}
