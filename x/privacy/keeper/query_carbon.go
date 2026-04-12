// query_carbon.go — Green Badge: Estimated Carbon Saved per Transaction
//
// This file implements the carbon-savings estimation query for the Zytherion
// privacy module. The model is based on publicly available energy consumption
// figures and maps them to CO₂ equivalents using the world-average grid carbon
// intensity published by the IEA.
//
// References:
//   - Cambridge Centre for Alternative Finance (CCAF): Bitcoin energy per tx
//     https://ccaf.io/cbnsi/cbeci
//   - Cosmos SDK / CometBFT validator benchmarks: ~0.00017 kWh per tx
//   - IEA 2024 world average grid carbon intensity: ~475 gCO₂ / kWh
//
// The values are intentionally conservative estimates; real savings will vary
// depending on the energy mix of validators and the network load.

package keeper

import sdk "github.com/cosmos/cosmos-sdk/types"

// ─── Carbon model constants ──────────────────────────────────────────────────

const (
	// powEnergyPerTxKWh is the estimated energy consumed per Bitcoin PoW
	// transaction in kWh, based on CCAF 2024 data.
	powEnergyPerTxKWh = 900.0

	// bftEnergyPerTxKWh is the estimated energy consumed per Zytherion BFT
	// (CometBFT) transaction in kWh, based on Cosmos SDK validator benchmarks.
	bftEnergyPerTxKWh = 0.00017

	// carbonIntensityGPerKWh is the world-average grid carbon intensity in
	// grams of CO₂ per kWh (IEA 2024).
	carbonIntensityGPerKWh = 475.0
)

// CarbonReport contains the estimated carbon savings per Zytherion transaction
// compared to a Bitcoin Proof-of-Work chain, together with the underlying
// model parameters for full transparency.
type CarbonReport struct {
	// PoW reference chain parameters.
	PoWChainName          string  `json:"pow_chain_name"`
	PoWEnergyPerTxKWh     float64 `json:"pow_energy_per_tx_kwh"`

	// Zytherion BFT parameters.
	BFTEnergyPerTxKWh     float64 `json:"bft_energy_per_tx_kwh"`

	// Shared carbon intensity model.
	CarbonIntensityGPerKWh float64 `json:"carbon_intensity_g_per_kwh"`

	// Derived savings.
	SavedEnergyPerTxKWh float64 `json:"saved_energy_per_tx_kwh"`
	SavedGramsCO2PerTx  float64 `json:"saved_grams_co2_per_tx"`
	SavedKgCO2PerTx     float64 `json:"saved_kg_co2_per_tx"`

	// Metadata.
	Note string `json:"note"`

	// Cumulative savings since genesis (if block height is provided).
	EstimatedTotalTxs      int64   `json:"estimated_total_txs,omitempty"`
	CumulativeSavedKgCO2   float64 `json:"cumulative_saved_kg_co2,omitempty"`
}

// CarbonSavedPerTx returns the estimated grams of CO₂ saved per Zytherion
// transaction compared to a Bitcoin Proof-of-Work chain.
//
// The query is pure-computation: it does not read the KVStore and can be
// called at any block height. For cumulative stats, pass a non-zero ctx.
func (k Keeper) CarbonSavedPerTx(ctx sdk.Context) CarbonReport {
	savedEnergyKWh := powEnergyPerTxKWh - bftEnergyPerTxKWh
	savedGrams := savedEnergyKWh * carbonIntensityGPerKWh
	savedKg := savedGrams / 1000.0

	report := CarbonReport{
		PoWChainName:           "Bitcoin (PoW)",
		PoWEnergyPerTxKWh:      powEnergyPerTxKWh,
		BFTEnergyPerTxKWh:      bftEnergyPerTxKWh,
		CarbonIntensityGPerKWh: carbonIntensityGPerKWh,
		SavedEnergyPerTxKWh:    savedEnergyKWh,
		SavedGramsCO2PerTx:     savedGrams,
		SavedKgCO2PerTx:        savedKg,
		Note: "Model: CCAF 2024 Bitcoin energy est. vs Cosmos SDK BFT benchmark. " +
			"Carbon intensity: IEA 2024 world average (475 gCO₂/kWh). " +
			"Actual savings depend on validator energy mix and network load.",
	}

	// Optionally compute cumulative savings using block height as a proxy
	// for total transactions (assumes avg 1 tx/block for a conservative estimate).
	if ctx.BlockHeight() > 0 {
		// Use block height as an approximate lower bound on total transactions.
		report.EstimatedTotalTxs = ctx.BlockHeight()
		report.CumulativeSavedKgCO2 = float64(ctx.BlockHeight()) * savedKg
	}

	return report
}
