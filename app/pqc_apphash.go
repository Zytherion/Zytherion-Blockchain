// Package app — PQC App Hash Accessors
//
// This file provides app-level read helpers for the PQC block hash that is
// computed and stored by the privacy module's EndBlock every block.
//
// Integration flow (see x/privacy/module.go EndBlock for the write side):
//
//	EndBlock of B_n:
//	  H_n = LatticeHash(height_n || H_{n-1} || AppHash_{n-1} || DataHash_n)
//	  KVStore["pqc_hash/latest"] = H_n
//
//	Next block B_{n+1}:
//	  H_{n-1} is read from KVStore["pqc_hash/latest"] at the start of EndBlock
//
// The KVStore write happens before Commit(), so H_n becomes part of the
// privacy module's Merkle subtree and therefore part of the multistore root —
// i.e. it is embedded in the standard Cosmos AppHash without any consensus
// changes.
package app

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"zytherion/x/privacy/types"
)

// GetLatestPQCHash retrieves the PQC lattice hash of the most recently
// finalized block from the on-chain KVStore.
//
// Returns nil before the first block is committed (genesis state has no PQC
// hash yet).  After block 1, this is always a 32-byte SHA3-256 digest.
func (app *App) GetLatestPQCHash(ctx sdk.Context) []byte {
	store := ctx.KVStore(app.PrivacyKeeper.StoreKey())
	return store.Get([]byte(types.LatestPQCHashKey))
}

// LogPQCAppHash logs the current PQC hash to the application logger.
// Useful for debugging and block explorers that want to display the
// quantum-resistant hash alongside the standard AppHash.
func (app *App) LogPQCAppHash(ctx sdk.Context) {
	h := app.GetLatestPQCHash(ctx)
	if h == nil {
		app.Logger().Info("PQC app hash: not yet computed (pre-genesis)")
		return
	}
	app.Logger().Info("PQC app hash",
		"height", ctx.BlockHeight(),
		"pqc_hash", fmt.Sprintf("%x", h),
	)
}
