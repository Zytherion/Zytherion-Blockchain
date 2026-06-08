// test_helpers_test.go — Shared test helper for x/privacy/keeper package tests.
package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	keepertest "zytherion/testutil/keeper"
	"zytherion/x/privacy/keeper"
)

// newTestKeeper creates a Keeper and Context suitable for unit testing.
// TFHE is disabled (no Rust library required).
func newTestKeeper(t testing.TB) (*keeper.Keeper, sdk.Context) {
	t.Helper()
	return keepertest.PrivacyKeeper(t)
}
