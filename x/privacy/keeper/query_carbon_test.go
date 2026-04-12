package keeper_test

import (
	"testing"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/stretchr/testify/require"
)


// TestCarbonSavedPerTx verifies the carbon savings model returns plausible values.
func TestCarbonSavedPerTx(t *testing.T) {
	k, ctx := newTestKeeper(t)

	// Override context with a real block height so cumulative stats are computed.
	ctx = ctx.WithBlockHeader(tmproto.Header{Height: 1000})

	report := k.CarbonSavedPerTx(ctx)

	// Basic sanity: per-tx savings should be in the range 400–500 kg CO₂.
	require.InDelta(t, 427.419, report.SavedKgCO2PerTx, 10.0,
		"per-tx CO₂ savings should be ~427 kg (±10 kg)")

	// Energy savings must be positive and much less than PoW.
	require.Greater(t, report.SavedEnergyPerTxKWh, 0.0,
		"saved energy must be positive")
	require.Less(t, report.BFTEnergyPerTxKWh, report.PoWEnergyPerTxKWh,
		"BFT energy must be significantly less than PoW")

	// Grams and kg should be consistent.
	require.InDelta(t, report.SavedGramsCO2PerTx/1000.0, report.SavedKgCO2PerTx, 0.001,
		"SavedKgCO2PerTx must equal SavedGramsCO2PerTx / 1000")

	// Cumulative stats should use block height.
	require.Equal(t, int64(1000), report.EstimatedTotalTxs,
		"EstimatedTotalTxs should match block height")
	require.InDelta(t, 427419.0, report.CumulativeSavedKgCO2, 1000.0,
		"cumulative savings at height 1000 should be ~427 tonne")

	// Note must be non-empty.
	require.NotEmpty(t, report.Note)
	require.Equal(t, "Bitcoin (PoW)", report.PoWChainName)
}

// TestCarbonSavedPerTx_ZeroHeight verifies that zero block height results in
// zero cumulative stats (no garbage values).
func TestCarbonSavedPerTx_ZeroHeight(t *testing.T) {
	k, ctx := newTestKeeper(t)
	// newTestKeeper produces a context with Height 0.
	report := k.CarbonSavedPerTx(ctx)

	require.EqualValues(t, 0, report.EstimatedTotalTxs,
		"at genesis height cumulative stats should be zero")
	require.EqualValues(t, 0.0, report.CumulativeSavedKgCO2,
		"cumulative CO₂ saving at height 0 must be 0")
}

// TestCarbonSavedPerTx_ModelConstants verifies that the physical model
// constants are within expected ranges — prevents accidental changes.
func TestCarbonSavedPerTx_ModelConstants(t *testing.T) {
	k, ctx := newTestKeeper(t)
	report := k.CarbonSavedPerTx(ctx)

	// PoW energy: between 500 and 1500 kWh per tx.
	require.InDelta(t, 900.0, report.PoWEnergyPerTxKWh, 400.0,
		"PoW energy estimate out of expected range")

	// BFT energy: vanishingly small compared to PoW.
	require.Less(t, report.BFTEnergyPerTxKWh, 0.001,
		"BFT energy per tx must be < 0.001 kWh")

	// Carbon intensity: IEA 2024 range roughly 400–600 g/kWh.
	require.InDelta(t, 475.0, report.CarbonIntensityGPerKWh, 100.0,
		"carbon intensity out of IEA expected range")
}
