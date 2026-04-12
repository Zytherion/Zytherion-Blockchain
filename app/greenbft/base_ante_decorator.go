// Package greenbft — BaseAnteDecorator wrapper
//
// This file provides a thin wrapper that adapts a standard sdk.AnteHandler
// (as returned by ante.NewAnteHandler) into an sdk.AnteDecorator so it can
// be composed with sdk.ChainAnteDecorators alongside the PQCAnteDecorator.
//
// It also increments the AdaptiveTimeoutManager's per-block tx counter after
// each successful transaction so that EndBlocker has an accurate count.

package greenbft

import sdk "github.com/cosmos/cosmos-sdk/types"

// BaseAnteDecorator wraps a pre-built sdk.AnteHandler so it can participate
// in an sdk.ChainAnteDecorators chain as a final step and simultaneously track
// how many transactions have passed ante checks in the current block.
type BaseAnteDecorator struct {
	handler sdk.AnteHandler

	// manager is used to increment the per-block transaction counter.
	// If nil, counting is skipped (safe default for tests that don't need it).
	manager *AdaptiveTimeoutManager
}

// NewBaseAnteDecorator wraps h as an sdk.AnteDecorator.
// Pass the AdaptiveTimeoutManager so accepted transactions are counted
// for the adaptive timeout rolling window.
func NewBaseAnteDecorator(h sdk.AnteHandler, mgr *AdaptiveTimeoutManager) BaseAnteDecorator {
	return BaseAnteDecorator{handler: h, manager: mgr}
}

// AnteHandle implements sdk.AnteDecorator.
// It runs the full standard Cosmos ante chain. On success it increments the
// AdaptiveTimeoutManager's per-block counter so EndBlocker knows how many
// transactions cleared ante in this block without accessing req.Txs (which
// does not exist on RequestEndBlock in CometBFT v0.37).
func (d BaseAnteDecorator) AnteHandle(
	ctx sdk.Context,
	tx sdk.Tx,
	simulate bool,
	_ sdk.AnteHandler, // next is intentionally ignored; wrapped handler is terminal
) (sdk.Context, error) {
	newCtx, err := d.handler(ctx, tx, simulate)
	if err != nil {
		return newCtx, err
	}
	// Only count non-simulation txs (delivery mode) that actually pass.
	if !simulate && d.manager != nil {
		d.manager.AddTx()
	}
	return newCtx, nil
}
