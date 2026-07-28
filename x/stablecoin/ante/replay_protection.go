// Package ante provides Cosmos SDK AnteHandler decorators for the stablecoin module.
//
// # Forward-Only Replay Protection
//
// Every ZYTD message (MsgMintZYTD, MsgBurnZYTD, MsgLiquidate, MsgRegisterZYTDKey)
// carries an ExpirationBlockHeight field that the sender signs. This decorator
// enforces:
//
//	current_block_height <= ExpirationBlockHeight <= current_block_height + MaxExpirationWindow
//
// This prevents:
//   - Replay attacks: a captured tx cannot be replayed after its expiration block.
//   - Far-future abuse: tx cannot be pre-signed for a block far in the future.
//   - Past inclusion: tx signed for an already-passed block is always rejected.
//
// The window is forward-only (no minus tolerance) to give a strict replay guarantee
// while allowing ~5 blocks (~30 seconds) of network jitter before the tx expires.
package ante

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	stablecointypes "zytherion/x/stablecoin/types"
)

// MaxExpirationWindow is the maximum number of blocks ahead the expiration
// block height may be set relative to the current block height.
// With ~6 s/block this gives a ~30-second validity window.
const MaxExpirationWindow int64 = 5

// ErrExpirationTooFar is returned when ExpirationBlockHeight exceeds
// current_height + MaxExpirationWindow.
var ErrExpirationTooFar = errorsmod.Register(
	stablecointypes.ModuleName, 1600,
	fmt.Sprintf("expiration_block_height exceeds current block + %d (max forward window)", MaxExpirationWindow),
)

// ErrExpirationPast is returned when ExpirationBlockHeight < current_height.
var ErrExpirationPast = errorsmod.Register(
	stablecointypes.ModuleName, 1601,
	"expiration_block_height is in the past — transaction has expired",
)

// ZYTDReplayProtectionDecorator is an AnteHandler decorator that validates
// the ExpirationBlockHeight on all ZYTD stablecoin messages.
//
// Position in ante chain: AFTER signature verification, BEFORE msg execution.
type ZYTDReplayProtectionDecorator struct{}

// NewZYTDReplayProtectionDecorator returns a new decorator.
func NewZYTDReplayProtectionDecorator() ZYTDReplayProtectionDecorator {
	return ZYTDReplayProtectionDecorator{}
}

// AnteHandle implements sdk.AnteDecorator.
// Validates forward-only expiration window for every ZYTD message in the tx.
func (d ZYTDReplayProtectionDecorator) AnteHandle(
	ctx sdk.Context,
	tx sdk.Tx,
	simulate bool,
	next sdk.AnteHandler,
) (sdk.Context, error) {
	// Skip real height checks during gas estimation simulation.
	if simulate {
		return next(ctx, tx, simulate)
	}

	currentHeight := ctx.BlockHeight()

	for i, msg := range tx.GetMsgs() {
		expirer, ok := msg.(ExpirationBlockHeighter)
		if !ok {
			continue // not a ZYTD message — pass through
		}

		exp := expirer.GetExpirationBlockHeight()

		// Rule 1: reject txs whose expiration is in the past.
		if exp < currentHeight {
			return ctx, errorsmod.Wrapf(
				ErrExpirationPast,
				"msg[%d]: expiration_block_height=%d current_height=%d — tx expired",
				i, exp, currentHeight,
			)
		}

		// Rule 2: reject txs whose expiration exceeds the forward window.
		maxAllowed := currentHeight + MaxExpirationWindow
		if exp > maxAllowed {
			return ctx, errorsmod.Wrapf(
				ErrExpirationTooFar,
				"msg[%d]: expiration_block_height=%d max_allowed=%d (current=%d + window=%d)",
				i, exp, maxAllowed, currentHeight, MaxExpirationWindow,
			)
		}
	}

	return next(ctx, tx, simulate)
}

// ExpirationBlockHeighter is the interface implemented by every ZYTD message
// that carries an expiration block height. Using an interface decouples this
// decorator from concrete message types and allows easy future extension.
type ExpirationBlockHeighter interface {
	sdk.Msg
	GetExpirationBlockHeight() int64
}

// Compile-time assertions: all ZYTD messages must satisfy ExpirationBlockHeighter.
var _ ExpirationBlockHeighter = (*stablecointypes.MsgMintZYTD)(nil)
var _ ExpirationBlockHeighter = (*stablecointypes.MsgBurnZYTD)(nil)
var _ ExpirationBlockHeighter = (*stablecointypes.MsgLiquidate)(nil)
var _ ExpirationBlockHeighter = (*stablecointypes.MsgRegisterZYTDKey)(nil)

// ErrInvalidExpirationHeight is a convenience sentinel for tests.
var ErrInvalidExpirationHeight = sdkerrors.ErrUnauthorized
