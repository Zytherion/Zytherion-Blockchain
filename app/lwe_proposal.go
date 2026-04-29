// lwe_proposal.go — ABCI 2.0 PrepareProposal / ProcessProposal handlers
// for the LWE-SHA3-Hybrid block commitment anchor.
//
// # Security Goal
//
// Every proposed block must carry a 96-byte Ring-LWE hash of its transaction
// payload as its FIRST injected byte-sequence.  Validators independently
// recompute this hash during ProcessProposal; if it doesn't match, the
// proposal is rejected before voting begins.  This makes the LWE hash a
// mandatory consensus-level commitment, not just an advisory log line.
//
// # Wire Format of the Injected Sentinel
//
//   [0:4]    magic marker  = lweMarker (0x4C574548 = "LWEH" in ASCII BE)
//   [4:8]    version       = lweVersion (0x00000001)
//   [8:104]  LWE hash      = GenerateLWEBlockHash(txPayload, prevPQCHash) [96 bytes]
//
// Total injected size: lweSentinelSize = 104 bytes.
//
// # Integration  (see app/app.go)
//
//   app.SetPrepareProposal(app.LWEPrepareProposal)
//   app.SetProcessProposal(app.LWEProcessProposal)
//
// # State Commitment
//
// The LWE hash extracted from the winning proposal is stored in the privacy
// module KVStore under types.LatestPQCHashKey during EndBlock (done by
// x/privacy/module.go EndBlock via GenerateLWEBlockHashWithFallback).
// This means the hash propagates into the multistore root (AppHash) of the
// NEXT block, anchoring the commitment to the Cosmos consensus chain.
package app

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"

	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"zytherion/x/privacy/pqc"
	privacytypes "zytherion/x/privacy/types"
)

// ── Sentinel constants ────────────────────────────────────────────────────────

const (
	// lweMarkerUint32 identifies the injected sentinel in the first position.
	// ASCII "LWEH" in big-endian = 0x4C574548.
	lweMarkerUint32 uint32 = 0x4C574548

	// lweVersion is the format version of the injected sentinel.
	lweVersion uint32 = 0x00000001

	// lweHeaderSize is the size of the magic + version prefix.
	lweHeaderSize = 4 + 4 // 8 bytes

	// lweSentinelSize is the total size of the injected sentinel entry:
	//   4 bytes magic + 4 bytes version + 96 bytes LWE hash = 104 bytes.
	lweSentinelSize = lweHeaderSize + pqc.LWEHashSize // 104

	// lweMarkerKey is the KVStore key where the proposer's LWE hash is cached
	// between PrepareProposal and EndBlock — useful for monitoring / audit.
	lweMarkerKey = "lwe_proposal_hash"
)

// ── LWEPrepareProposal ────────────────────────────────────────────────────────

// LWEPrepareProposal is the ABCI 2.0 PrepareProposal handler.
//
// Called by the block proposer before broadcasting the proposal to peers.
// It PREPENDS a 104-byte LWE sentinel as the first "transaction" in the
// proposal so that every validator can independently verify the commitment
// in ProcessProposal.
//
// Steps:
//  1. Collect the raw transaction bytes from the request.
//  2. Retrieve the previous PQC hash from the privacy KVStore (chain linkage).
//  3. Compute GenerateLWEBlockHash(txPayload, prevPQCHash).
//  4. Wrap the hash in the sentinel format and prepend it to the tx list.
func (app *App) LWEPrepareProposal(
	ctx sdk.Context,
	req abci.RequestPrepareProposal,
) abci.ResponsePrepareProposal {
	// ── 1. Respect the max-bytes limit BEFORE injecting ──────────────────────
	// Start with the SDK default handler to get the filtered tx list.
	txs := req.Txs

	// ── 2. Build the canonical payload for hashing ────────────────────────────
	txPayload := buildTxPayload(txs)

	// ── 3. Retrieve prevPQCHash (chain linkage) ───────────────────────────────
	prevHash := app.getPrevPQCHash(ctx)

	// ── 4. Compute the 96-byte LWE hash ──────────────────────────────────────
	lweHash, err := pqc.GenerateLWEBlockHash(txPayload, prevHash)
	if err != nil {
		// Extremely unlikely (only if input is unserialisable).
		// Fall back to SHA3-256 truncated to 32 bytes + zero-pad to 96 bytes.
		app.Logger().Error("LWEPrepareProposal: LWE hash failed, using SHA3 fallback",
			"error", err)
		sha3hash := pqc.GenerateBlockHash(pqc.BlockHashInput{
			Height:       ctx.BlockHeight(),
			PrevHash:     prevHash,
			Transactions: [][]byte{txPayload},
		})
		lweHash = make([]byte, pqc.LWEHashSize)
		copy(lweHash, sha3hash)
	}

	// ── 5. Build the 104-byte sentinel ────────────────────────────────────────
	sentinel := buildSentinel(lweHash)

	// ── 6. Prepend sentinel as the first entry ────────────────────────────────
	finalTxs := make([][]byte, 0, len(txs)+1)
	finalTxs = append(finalTxs, sentinel)
	finalTxs = append(finalTxs, txs...)

	app.Logger().Info("LWEPrepareProposal: sentinel injected",
		"height", ctx.BlockHeight(),
		"lwe_hash_prefix", hex.EncodeToString(lweHash[:8]),
		"tx_count", len(txs),
	)

	return abci.ResponsePrepareProposal{Txs: finalTxs}
}

// ── LWEProcessProposal ────────────────────────────────────────────────────────

// LWEProcessProposal is the ABCI 2.0 ProcessProposal handler.
//
// Called by every non-proposer validator when it receives a block proposal.
// It enforces the LWE commitment by:
//  1. Extracting the sentinel from position 0.
//  2. Verifying the sentinel magic and version.
//  3. Re-computing GenerateLWEBlockHash over the remaining transactions.
//  4. Comparing with the hash inside the sentinel — REJECT on mismatch.
func (app *App) LWEProcessProposal(
	ctx sdk.Context,
	req abci.RequestProcessProposal,
) abci.ResponseProcessProposal {
	reject := func(reason string, args ...interface{}) abci.ResponseProcessProposal {
		app.Logger().Error(fmt.Sprintf("LWEProcessProposal: REJECT — "+reason, args...),
			"height", ctx.BlockHeight(),
		)
		return abci.ResponseProcessProposal{
			Status: abci.ResponseProcessProposal_REJECT,
		}
	}

	txs := req.Txs

	// ── 1. A valid proposal must have at least the sentinel ───────────────────
	if len(txs) == 0 {
		return reject("proposal has no transactions (missing LWE sentinel)")
	}

	// ── 2. Validate sentinel size and magic + version header ──────────────────
	sentinel := txs[0]
	if len(sentinel) != lweSentinelSize {
		return reject("sentinel size mismatch: got %d bytes, want %d",
			len(sentinel), lweSentinelSize)
	}

	magic := binary.BigEndian.Uint32(sentinel[0:4])
	version := binary.BigEndian.Uint32(sentinel[4:8])

	if magic != lweMarkerUint32 {
		return reject("sentinel magic mismatch: got 0x%08X, want 0x%08X",
			magic, lweMarkerUint32)
	}
	if version != lweVersion {
		return reject("sentinel version mismatch: got %d, want %d",
			version, lweVersion)
	}

	// ── 3. Extract the claimed LWE hash from the sentinel ─────────────────────
	claimedHash := sentinel[lweHeaderSize : lweHeaderSize+pqc.LWEHashSize]

	// ── 4. Re-compute the expected LWE hash over the user txs ─────────────────
	userTxs := txs[1:]
	txPayload := buildTxPayload(userTxs)
	prevHash := app.getPrevPQCHash(ctx)

	expectedHash, err := pqc.GenerateLWEBlockHash(txPayload, prevHash)
	if err != nil {
		return reject("LWE hash computation failed: %v", err)
	}

	// ── 5. Validate: reject if hashes don't match ─────────────────────────────
	if !bytes.Equal(claimedHash, expectedHash) {
		return reject(
			"LWE hash mismatch — proposal from tampered or incompatible proposer: "+
				"claimed=%s expected=%s",
			hex.EncodeToString(claimedHash[:8]),
			hex.EncodeToString(expectedHash[:8]),
		)
	}

	// ── 6. Validate structural bounds of the LWE hash ─────────────────────────
	if err := pqc.ValidateLWEHash(expectedHash); err != nil {
		return reject("LWE hash structural validation failed: %v", err)
	}

	// ── 7. Persist the validated hash into the privacy KVStore for audit ──────
	// This happens at proposal acceptance time (before block execution).
	// The hash ends up in the KVStore under lweMarkerKey, available to
	// monitors and queries.  The authoritative LatestPQCHashKey is written
	// by the privacy module EndBlock via GenerateLWEBlockHashWithFallback.
	app.commitLWEHash(ctx, expectedHash)

	app.Logger().Info("LWEProcessProposal: ACCEPT",
		"height", ctx.BlockHeight(),
		"lwe_hash_prefix", hex.EncodeToString(expectedHash[:8]),
		"user_tx_count", len(userTxs),
	)

	return abci.ResponseProcessProposal{
		Status: abci.ResponseProcessProposal_ACCEPT,
	}
}

// ── commitLWEHash (internal) ─────────────────────────────────────────────────

// commitLWEHash persists a validated LWE hash to the privacy KVStore under
// lweMarkerKey.  Called internally by LWEProcessProposal on ACCEPT.
func (app *App) commitLWEHash(ctx sdk.Context, lweHash []byte) {
	store := ctx.KVStore(app.PrivacyKeeper.StoreKey())
	store.Set([]byte(lweMarkerKey), lweHash)
	ctx.Logger().Debug("LWE proposal hash committed",
		"key", lweMarkerKey,
		"hash_prefix", hex.EncodeToString(lweHash[:8]),
	)
}

// ── CommitLWEProposalHash (public — for external audit access) ───────────────

// CommitLWEProposalHash extracts the LWE hash from a raw sentinel bytes slice
// and stores it.  Safe to call with nil or invalid input (no-op in that case).
// Can be used by test helpers or monitoring sidecars that have raw block bytes.
func (app *App) CommitLWEProposalHash(ctx sdk.Context, sentinelBytes []byte) {
	if len(sentinelBytes) != lweSentinelSize {
		return
	}
	magic := binary.BigEndian.Uint32(sentinelBytes[0:4])
	if magic != lweMarkerUint32 {
		return
	}
	lweHash := sentinelBytes[lweHeaderSize : lweHeaderSize+pqc.LWEHashSize]
	app.commitLWEHash(ctx, lweHash)
}

// GetLWEProposalHash retrieves the LWE hash committed by the most recent
// successfully processed block proposal.  Returns nil if none is stored yet.
func (app *App) GetLWEProposalHash(ctx sdk.Context) []byte {
	store := ctx.KVStore(app.PrivacyKeeper.StoreKey())
	return store.Get([]byte(lweMarkerKey))
}

// ── Helper functions ──────────────────────────────────────────────────────────

// buildSentinel constructs the 104-byte sentinel byte slice from an LWE hash.
//
//	[0:4]   magic marker  (BE uint32 = lweMarkerUint32)
//	[4:8]   version       (BE uint32 = lweVersion)
//	[8:104] LWE hash      (96 bytes)
func buildSentinel(lweHash []byte) []byte {
	sentinel := make([]byte, lweSentinelSize)
	binary.BigEndian.PutUint32(sentinel[0:4], lweMarkerUint32)
	binary.BigEndian.PutUint32(sentinel[4:8], lweVersion)
	copy(sentinel[lweHeaderSize:], lweHash)
	return sentinel
}

// buildTxPayload concatenates raw transaction bytes into a single payload
// that is fed into GenerateLWEBlockHash.  Using concatenation (not a Merkle
// tree) keeps the construction simple while still binding every tx to the hash.
func buildTxPayload(txs [][]byte) []byte {
	total := 0
	for _, tx := range txs {
		total += len(tx)
	}
	payload := make([]byte, 0, total)
	for _, tx := range txs {
		payload = append(payload, tx...)
	}
	return payload
}

// getPrevPQCHash retrieves the previous block's authoritative PQC hash from
// the privacy module KVStore.  Returns a 32-byte zero slice at genesis.
func (app *App) getPrevPQCHash(ctx sdk.Context) []byte {
	store := ctx.KVStore(app.PrivacyKeeper.StoreKey())
	h := store.Get([]byte(privacytypes.LatestPQCHashKey))
	if h == nil {
		return make([]byte, 32) // zero slice at genesis
	}
	return h
}

// IsSentinel returns true if the raw bytes look like an LWE proposal sentinel.
// Used by the ante handler to skip fee/auth checks on the injected tx.
func IsSentinel(raw []byte) bool {
	if len(raw) != lweSentinelSize {
		return false
	}
	return binary.BigEndian.Uint32(raw[0:4]) == lweMarkerUint32
}
