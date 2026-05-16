// lwr_proposal.go — ABCI 2.0 PrepareProposal / ProcessProposal handlers
// for the LWR-SHA3-Hybrid block commitment anchor and PoVL sequence.
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
	// lwrMarkerUint32 identifies the injected PoVL sentinel in the first position.
	// ASCII "LWRH" in big-endian = 0x4C575248.
	lwrMarkerUint32 uint32 = 0x4C575248

	// lwrVersion is the format version of the injected sentinel.
	lwrVersion uint32 = 0x00000001

	// Sizes
	lwrHeaderSize = 4 + 4
	lwrHashSize   = pqc.LWRHashSize // 96
	povlRootSize  = pqc.PoVLStateSize // 32
	povlStepSize  = 8 // uint64
	
	// The length of the PoVL ZK Proof can vary slightly (Groth16 BN254 is typically ~128 bytes in binary format).
	// We will length-prefix the proof.
)

// ── LWRPrepareProposal ────────────────────────────────────────────────────────

func (app *App) LWRPrepareProposal(
	ctx sdk.Context,
	req abci.RequestPrepareProposal,
) abci.ResponsePrepareProposal {
	txs := req.Txs
	txPayload := buildTxPayload(txs)
	prevHash := app.getPrevPQCHash(ctx)

	// 1. Compute deterministic LWR block hash
	lwrHash, err := pqc.GenerateLWRBlockHash(txPayload, prevHash)
	if err != nil {
		app.Logger().Error("LWRPrepareProposal: LWR hash failed", "error", err)
		lwrHash = make([]byte, pqc.LWRHashSize)
		copy(lwrHash, pqc.GenerateBlockHash(pqc.BlockHashInput{
			Height:       ctx.BlockHeight(),
			PrevHash:     prevHash,
			Transactions: [][]byte{txPayload},
		}))
	}

	// 2. Generate PoVL (Simulated for PrepareProposal)
	// In production, the proposer would run the VDF and gnark prover off-chain.
	// Here we generate the valid state deterministically to fulfill the structural requirement.
	steps := uint64(10) // Example N=10 steps for the VDF
	
	app.Logger().Info("LWRPrepareProposal: computing PoVL sequence...", "steps", steps, "height", ctx.BlockHeight())
	
	povlRoot := pqc.ComputePoVLChain(lwrHash[:32], steps) // Use seed as initial state
	povlProof := []byte("mock-groth16-proof-for-povl") // Mock proof for now

	app.Logger().Info("LWRPrepareProposal: PoVL sequence complete", "povl_root", hex.EncodeToString(povlRoot[:8]))

	// 3. Encode into a deterministic binary payload
	sentinel := buildPoVLSentinel(lwrHash, povlRoot, steps, povlProof)

	// 4. Inject as Tx[0]
	// Unlike the previous buggy bypass, we properly return it so the chain enforces it.
	// The LWETxDecoderWrapper (now LWRTxDecoderWrapper) handles preventing protobuf crashes.
	finalTxs := make([][]byte, 0, len(txs)+1)
	finalTxs = append(finalTxs, sentinel)
	finalTxs = append(finalTxs, txs...)

	return abci.ResponsePrepareProposal{Txs: finalTxs}
}

// ── LWRProcessProposal ────────────────────────────────────────────────────────

func (app *App) LWRProcessProposal(
	ctx sdk.Context,
	req abci.RequestProcessProposal,
) abci.ResponseProcessProposal {
	reject := func(reason string, args ...interface{}) abci.ResponseProcessProposal {
		app.Logger().Error(fmt.Sprintf("LWRProcessProposal: REJECT — "+reason, args...),
			"height", ctx.BlockHeight(),
		)
		return abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}
	}

	txs := req.Txs
	if len(txs) == 0 {
		return reject("no transactions (missing PoVL sentinel)")
	}

	sentinel := txs[0]
	if len(sentinel) < lwrHeaderSize {
		return reject("sentinel too short")
	}

	magic := binary.BigEndian.Uint32(sentinel[0:4])
	if magic != lwrMarkerUint32 {
		return reject("sentinel magic mismatch")
	}

	// Extract Sentinel Data
	offset := lwrHeaderSize
	if len(sentinel) < offset+lwrHashSize+povlRootSize+povlStepSize {
		return reject("sentinel payload too short")
	}

	claimedHash := sentinel[offset : offset+lwrHashSize]
	offset += lwrHashSize

	claimedPoVLRoot := sentinel[offset : offset+povlRootSize]
	offset += povlRootSize

	claimedSteps := binary.BigEndian.Uint64(sentinel[offset : offset+povlStepSize])
	offset += povlStepSize

	proofLen := binary.BigEndian.Uint32(sentinel[offset : offset+4])
	offset += 4
	if len(sentinel) < offset+int(proofLen) {
		return reject("sentinel proof length exceeds bounds")
	}
	claimedProof := sentinel[offset : offset+int(proofLen)]

	// 1. Verify expected LWR block hash over remaining txs
	userTxs := txs[1:]
	txPayload := buildTxPayload(userTxs)
	prevHash := app.getPrevPQCHash(ctx)

	expectedHash, err := pqc.GenerateLWRBlockHash(txPayload, prevHash)
	if err != nil {
		return reject("LWR hash computation failed: %v", err)
	}

	if !bytes.Equal(claimedHash, expectedHash) {
		return reject(
			"LWR hash mismatch: claimed=%s expected=%s",
			hex.EncodeToString(claimedHash[:8]),
			hex.EncodeToString(expectedHash[:8]),
		)
	}

	if err := pqc.ValidateLWRHash(expectedHash); err != nil {
		return reject("LWR hash validation failed: %v", err)
	}

	// 2. Verify PoVL ZK Proof (Groth16)
	// Here we would call the ZK verifier for PoVL. We mock it for the test.
	if string(claimedProof) != "mock-groth16-proof-for-povl" {
		// In production:
		// vk := app.PrivacyKeeper.GetPoVLVerifyingKey()
		// publicInputs := []byte{...} // encode initialState + finalState + steps
		// err := zk.VerifyTransferProof(claimedProof, publicInputs, vk)
		return reject("PoVL proof verification failed")
	}

	// 3. Validate PoVL State Evolution (Fallback to local execution if proof is skipped/mocked)
	// This acts as a secondary check or substitute if the proof is not yet fully ZK-backed.
	if !pqc.VerifyPoVLChain(expectedHash[:32], claimedPoVLRoot, claimedSteps) {
		return reject("PoVL sequential state evolution mismatch")
	}

	// Persist
	app.commitLWRHash(ctx, expectedHash)

	app.Logger().Info("LWRProcessProposal: ACCEPT",
		"height", ctx.BlockHeight(),
		"lwr_hash_prefix", hex.EncodeToString(expectedHash[:8]),
		"povl_root_prefix", hex.EncodeToString(claimedPoVLRoot[:8]),
		"povl_steps", claimedSteps,
	)

	return abci.ResponseProcessProposal{
		Status: abci.ResponseProcessProposal_ACCEPT,
	}
}

func (app *App) commitLWRHash(ctx sdk.Context, lwrHash []byte) {
	store := ctx.KVStore(app.PrivacyKeeper.StoreKey())
	store.Set([]byte("lwr_proposal_hash"), lwrHash)
}

func buildPoVLSentinel(lwrHash, povlRoot []byte, steps uint64, proof []byte) []byte {
	size := lwrHeaderSize + lwrHashSize + povlRootSize + povlStepSize + 4 + len(proof)
	sentinel := make([]byte, size)
	
	binary.BigEndian.PutUint32(sentinel[0:4], lwrMarkerUint32)
	binary.BigEndian.PutUint32(sentinel[4:8], lwrVersion)
	
	offset := lwrHeaderSize
	copy(sentinel[offset:offset+lwrHashSize], lwrHash)
	offset += lwrHashSize
	
	copy(sentinel[offset:offset+povlRootSize], povlRoot)
	offset += povlRootSize
	
	binary.BigEndian.PutUint64(sentinel[offset:offset+povlStepSize], steps)
	offset += povlStepSize
	
	binary.BigEndian.PutUint32(sentinel[offset:offset+4], uint32(len(proof)))
	offset += 4
	
	copy(sentinel[offset:], proof)
	
	return sentinel
}

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

func (app *App) getPrevPQCHash(ctx sdk.Context) []byte {
	store := ctx.KVStore(app.PrivacyKeeper.StoreKey())
	h := store.Get([]byte(privacytypes.LatestPQCHashKey))
	if h == nil {
		return make([]byte, 32)
	}
	return h
}
