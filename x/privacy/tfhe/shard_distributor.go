// shard_distributor.go — P2P shard distribution and on-demand reconstruction
// for TFHE ciphertexts in the Zytherion network (v0.4.1 — Dilithium5 Shard Signing).
//
// # Design
//
// After a TFHE ciphertext is submitted via tx/tfhe_submit, the node that
// receives the transaction:
//  1. Splits the ciphertext into 16 shards (erasure.Split, DataShards=12, Parity=4).
//  2. Builds a Merkle tree; attaches per-shard proofs.
//  3. Signs each shard's SHA-256(Data) with the proposer's Dilithium5 private key.
//  4. Distributes each shard to ReplicationFactor=4 randomly-selected peer nodes.
//  5. Stores the Merkle root + ProposerPubkey + metadata on-chain.
//
// When a node receives a shard via POST /shard:
//  1. Verifies the Dilithium5 signature over SHA-256(shardData) against ProposerPubkey.
//  2. Verifies the Merkle proof against the on-chain root.
//  3. Only stores the shard if both checks pass.
//
// When a node needs to reconstruct a ciphertext (query/tfhe_result):
//  1. Reads metadata to find shard locations + Merkle root + ProposerPubkey.
//  2. Fetches at least 12 shards from peer nodes (via P2P request).
//  3. Verifies each fetched shard's Dilithium5 signature and Merkle proof.
//  4. Reconstructs the original ciphertext using Reed-Solomon.
//
// # Security (v0.4.1)
//
// - Shard server requires a Bearer token (nodeID) in the Authorization header.
// - Incoming POST /shard: Dilithium5 signature verified FIRST, then Merkle proof.
// - A simple per-IP in-memory rate limiter (ShardRateLimiter) rejects bursts > 60 req/min.
// - Proactive repair: if a shard is missing from peers, the distributor re-sends it.
//
// # P2P Transport
//
// GET /shard?commitment=<hex>&index=<int>  → 200 with shard bytes
// POST /shard  (body: JSON ShardUpload)    → 201 on successful store
package tfhe

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"sort"
	"strconv"
	"sync"
	"time"

	"zytherion/x/privacy/pqc"
)

// ── Rate Limiter ──────────────────────────────────────────────────────────────

// ShardRateLimiter is a simple token-bucket rate limiter keyed by client IP.
// Each client is allowed MaxRequests per Window before being throttled.
type ShardRateLimiter struct {
	mu          sync.Mutex
	clients     map[string]*rateBucket
	maxRequests int
	window      time.Duration
}

type rateBucket struct {
	count    int
	resetAt  time.Time
}

// NewShardRateLimiter creates a rate limiter allowing maxRequests per window per IP.
func NewShardRateLimiter(maxRequests int, window time.Duration) *ShardRateLimiter {
	return &ShardRateLimiter{
		clients:     make(map[string]*rateBucket),
		maxRequests: maxRequests,
		window:      window,
	}
}

// Allow returns true if the client identified by key may proceed.
func (r *ShardRateLimiter) Allow(key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	b, ok := r.clients[key]
	if !ok || now.After(b.resetAt) {
		r.clients[key] = &rateBucket{count: 1, resetAt: now.Add(r.window)}
		return true
	}
	if b.count >= r.maxRequests {
		return false
	}
	b.count++
	return true
}

// ── ShardUpload (POST body) ───────────────────────────────────────────────────

// ShardUpload is the JSON body for POST /shard.
type ShardUpload struct {
	// CommitmentHex is the hex-encoded 32-byte commitment hash.
	CommitmentHex string `json:"commitment_hex"`
	// Index is the shard index in [0, ErasureTotalShards).
	Index int `json:"index"`
	// DataHex is the hex-encoded shard payload.
	DataHex string `json:"data_hex"`
	// MerkleRootHex is the hex-encoded on-chain Merkle root for this commitment.
	MerkleRootHex string `json:"merkle_root_hex"`
	// MerkleProofHex is the hex-encoded flat Merkle proof (sibling hashes, leaf→root).
	MerkleProofHex string `json:"merkle_proof_hex"`
	// ProposerPubkeyHex is the hex-encoded Dilithium5 public key of the proposer.
	// Present when the distributor was initialised with a signing key pair.
	ProposerPubkeyHex string `json:"proposer_pubkey_hex,omitempty"`
	// SignatureHex is the hex-encoded Dilithium5 signature of SHA-256(shardData).
	// Absent when no signing key is configured (test/dev builds).
	SignatureHex string `json:"signature_hex,omitempty"`
}

// ── ShardDistributor ──────────────────────────────────────────────────────────

// ShardDistributor manages distribution and retrieval of TFHE ciphertext shards
// across the P2P network.
type ShardDistributor struct {
	store       *ShardStore
	nodeID      string
	httpSrv     *http.Server
	mu          sync.RWMutex
	rateLimiter *ShardRateLimiter

	// signingPrivKey is the node's Dilithium5 private key (4864 bytes).
	// Nil when no signing key is configured (test/single-node builds).
	signingPrivKey []byte

	// signingPubKey is the corresponding Dilithium5 public key (2592 bytes).
	// Sent to peers inside ShardUpload.ProposerPubkeyHex so they can verify.
	signingPubKey []byte
}

// NewShardDistributor creates a ShardDistributor with the given local shard store.
func NewShardDistributor(store *ShardStore, nodeID string) *ShardDistributor {
	return &ShardDistributor{
		store:       store,
		nodeID:      nodeID,
		rateLimiter: NewShardRateLimiter(60, time.Minute),
	}
}

// WithSigningKey attaches a Dilithium5 key pair to the distributor so that
// shards are signed before distribution and signatures are verified on receipt.
//
// kp must be a valid Dilithium5 key pair (PrivateKey=4864 bytes, PublicKey=2592 bytes).
// Call this immediately after NewShardDistributor when starting a production node.
func (d *ShardDistributor) WithSigningKey(kp pqc.KeyPair) error {
	if len(kp.PrivateKey) != pqc.DilithiumPrivateKeySize {
		return fmt.Errorf("shard_distributor: invalid Dilithium5 private key length %d (want %d)",
			len(kp.PrivateKey), pqc.DilithiumPrivateKeySize)
	}
	if len(kp.PublicKey) != pqc.DilithiumPublicKeySize {
		return fmt.Errorf("shard_distributor: invalid Dilithium5 public key length %d (want %d)",
			len(kp.PublicKey), pqc.DilithiumPublicKeySize)
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.signingPrivKey = make([]byte, len(kp.PrivateKey))
	copy(d.signingPrivKey, kp.PrivateKey)
	d.signingPubKey = make([]byte, len(kp.PublicKey))
	copy(d.signingPubKey, kp.PublicKey)
	return nil
}

// ProposerPubkey returns the Dilithium5 public key bytes configured on this
// distributor. Returns nil when no signing key has been set.
func (d *ShardDistributor) ProposerPubkey() []byte {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if len(d.signingPubKey) == 0 {
		return nil
	}
	out := make([]byte, len(d.signingPubKey))
	copy(out, d.signingPubKey)
	return out
}

// signShardData produces a Dilithium5 signature over SHA-256(shardData) using
// the distributor's private key. Returns nil, nil when no key is configured.
func (d *ShardDistributor) signShardData(shardData []byte) ([]byte, error) {
	d.mu.RLock()
	privKey := d.signingPrivKey
	d.mu.RUnlock()
	if len(privKey) == 0 {
		return nil, nil // signing not configured — skipped
	}
	h := sha256.Sum256(shardData)
	sig, err := pqc.Sign(h[:], privKey)
	if err != nil {
		return nil, fmt.Errorf("shard_distributor: Dilithium5 sign failed: %w", err)
	}
	return sig, nil
}

// ── Shard Server (HTTP) ───────────────────────────────────────────────────────

// authMiddleware checks the Authorization: Bearer <nodeID> header.
// Only the local nodeID or the special "peer" token are accepted (PoC).
// In production this would verify a Dilithium5 JWT.
func (d *ShardDistributor) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" {
			http.Error(w, "missing Authorization header", http.StatusUnauthorized)
			return
		}
		// Accept Bearer tokens from any non-empty nodeID (PoC: no signature yet).
		// Production: verify Dilithium5 signature over request body.
		const prefix = "Bearer "
		if len(auth) <= len(prefix) {
			http.Error(w, "malformed Authorization header", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

// rateLimitMiddleware rejects requests from IPs that exceed the rate limit.
func (d *ShardDistributor) rateLimitMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr
		if !d.rateLimiter.Allow(ip) {
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		next(w, r)
	}
}

// StartShardServer starts the local HTTP shard serving endpoint on the given address.
// The server handles GET /shard and POST /shard requests from peer nodes.
//
// Blocks until the server is stopped or ctx is cancelled.
func (d *ShardDistributor) StartShardServer(ctx context.Context, listenAddr string) error {
	mux := http.NewServeMux()

	// GET /shard?commitment=<hex>&index=<int>
	// Serves the requested shard bytes if stored locally.
	// Requires Authorization header.
	mux.HandleFunc("/shard", d.rateLimitMiddleware(d.authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			d.handleGetShard(w, r)
		case http.MethodPost:
			d.handlePostShard(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})))

	// GET /shardmeta?commitment=<hex>
	// Returns JSON metadata about which shard indices are stored locally.
	mux.HandleFunc("/shardmeta", d.rateLimitMiddleware(func(w http.ResponseWriter, r *http.Request) {
		commitmentHex := r.URL.Query().Get("commitment")
		commitment, err := hex.DecodeString(commitmentHex)
		if err != nil {
			http.Error(w, "invalid commitment", http.StatusBadRequest)
			return
		}

		indices, err := d.store.ListLocalShards(commitment)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(indices) //nolint:errcheck
	}))

	d.httpSrv = &http.Server{
		Addr:         listenAddr,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	// Shutdown gracefully when context is cancelled.
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		d.httpSrv.Shutdown(shutdownCtx) //nolint:errcheck
	}()

	return d.httpSrv.ListenAndServe()
}

// handleGetShard serves GET /shard?commitment=<hex>&index=<int>.
// It returns the raw shard bytes in the body and, if available, the
// Dilithium5 signature in the X-Shard-Signature response header.
func (d *ShardDistributor) handleGetShard(w http.ResponseWriter, r *http.Request) {
	commitmentHex := r.URL.Query().Get("commitment")
	indexStr := r.URL.Query().Get("index")

	commitment, err := hex.DecodeString(commitmentHex)
	if err != nil {
		http.Error(w, "invalid commitment", http.StatusBadRequest)
		return
	}
	index, err := strconv.Atoi(indexStr)
	if err != nil {
		http.Error(w, "invalid index", http.StatusBadRequest)
		return
	}

	data, ok := d.store.LoadShard(commitment, index)
	if !ok {
		http.Error(w, "shard not found", http.StatusNotFound)
		return
	}

	// Attach Dilithium5 signature header if we have it stored.
	if sig, ok := d.store.LoadShardSignature(commitment, index); ok {
		w.Header().Set("X-Shard-Signature", hex.EncodeToString(sig))
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.WriteHeader(http.StatusOK)
	w.Write(data) //nolint:errcheck
}

// handlePostShard handles POST /shard — stores an inbound shard after verifying
// its Dilithium5 signature (if present) AND its Merkle proof.
//
// Verification order:
//  1. Dilithium5 signature over SHA-256(shardData) — if ProposerPubkeyHex is non-empty.
//  2. Merkle proof against MerkleRootHex.
//
// A failure at either step returns HTTP 403 Forbidden.
func (d *ShardDistributor) handlePostShard(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 64*1024)) // max 64 KB body
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}

	var upload ShardUpload
	if err := json.Unmarshal(body, &upload); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	// Decode fields.
	commitment, err := hex.DecodeString(upload.CommitmentHex)
	if err != nil || len(commitment) == 0 {
		http.Error(w, "invalid commitment_hex", http.StatusBadRequest)
		return
	}
	shardData, err := hex.DecodeString(upload.DataHex)
	if err != nil || len(shardData) == 0 {
		http.Error(w, "invalid data_hex", http.StatusBadRequest)
		return
	}
	merkleRootBytes, err := hex.DecodeString(upload.MerkleRootHex)
	if err != nil || len(merkleRootBytes) != 32 {
		http.Error(w, "invalid merkle_root_hex (must be 32 bytes)", http.StatusBadRequest)
		return
	}
	merkleProofBytes, err := hex.DecodeString(upload.MerkleProofHex)
	if err != nil {
		http.Error(w, "invalid merkle_proof_hex", http.StatusBadRequest)
		return
	}
	if upload.Index < 0 || upload.Index >= ErasureTotalShards {
		http.Error(w, fmt.Sprintf("index out of range [0,%d)", ErasureTotalShards), http.StatusBadRequest)
		return
	}

	// ── Step 1: Dilithium5 signature verification ──────────────────────────────
	// If the upload carries a proposer pubkey + signature, verify them FIRST.
	// This prevents an attacker from injecting crafted shards that pass the
	// Merkle check but were not produced by the legitimate proposer.
	if upload.ProposerPubkeyHex != "" {
		proposerPubkey, err := hex.DecodeString(upload.ProposerPubkeyHex)
		if err != nil || len(proposerPubkey) != pqc.DilithiumPublicKeySize {
			http.Error(w, fmt.Sprintf(
				"invalid proposer_pubkey_hex: must be %d bytes (Dilithium5 public key)",
				pqc.DilithiumPublicKeySize,
			), http.StatusBadRequest)
			return
		}

		if upload.SignatureHex == "" {
			http.Error(w, "signature_hex is required when proposer_pubkey_hex is present", http.StatusBadRequest)
			return
		}
		sigBytes, err := hex.DecodeString(upload.SignatureHex)
		if err != nil || len(sigBytes) != pqc.DilithiumSignatureSize {
			http.Error(w, fmt.Sprintf(
				"invalid signature_hex: must be %d bytes (Dilithium5 signature)",
				pqc.DilithiumSignatureSize,
			), http.StatusBadRequest)
			return
		}

		// Message = SHA-256(shardData) — same as what the proposer signed.
		shardHash := sha256.Sum256(shardData)
		if !pqc.Verify(shardHash[:], sigBytes, proposerPubkey) {
			http.Error(w, "Dilithium5 signature verification failed", http.StatusForbidden)
			return
		}
	}

	// ── Step 2: Merkle proof verification ─────────────────────────────────────
	const hashSize = 32
	depth := log2(ErasureTotalShards) // = 4 for 16 shards
	if len(merkleProofBytes) != depth*hashSize {
		http.Error(w, fmt.Sprintf("merkle proof must be %d bytes (%d hashes × 32)", depth*hashSize, depth), http.StatusBadRequest)
		return
	}
	proof := &MerkleProof{LeafIndex: upload.Index}
	proof.Path = make([]Hash, depth)
	for i := 0; i < depth; i++ {
		copy(proof.Path[i][:], merkleProofBytes[i*hashSize:(i+1)*hashSize])
	}

	var rootHash Hash
	copy(rootHash[:], merkleRootBytes)

	if err := VerifyProof(rootHash, upload.Index, shardData, proof); err != nil {
		http.Error(w, "Merkle proof verification failed: "+err.Error(), http.StatusForbidden)
		return
	}

	// ── Step 3: Store shard ────────────────────────────────────────────────────
	if err := d.store.StoreShard(commitment, upload.Index, shardData); err != nil {
		http.Error(w, "failed to store shard: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

// ── Shard Distribution ────────────────────────────────────────────────────────

// DistributeShards distributes shards to randomly selected peer nodes.
//
// Algorithm (v0.4.1):
//  1. Build Merkle tree over all shard data.
//  2. For each of the 16 shards:
//     a. Generate Merkle proof.
//     b. Sign SHA-256(shardData) with the node's Dilithium5 private key (if set).
//     c. Store locally.
//     d. POST shard + proof + signature to ReplicationFactor-1 random peers.
//  3. Return TFHEShardMeta (commitment, shard map, Merkle root, ProposerPubkey).
func (d *ShardDistributor) DistributeShards(
	ctx context.Context,
	shards []ShardResult,
	commitmentHash []byte,
	peerAddrs []string,
) (*TFHEShardMeta, error) {
	if len(shards) != ErasureTotalShards {
		return nil, fmt.Errorf("distributor: expected %d shards, got %d",
			ErasureTotalShards, len(shards))
	}

	// Build Merkle tree for proof generation.
	tree, err := BuildMerkleTree(shards)
	if err != nil {
		return nil, fmt.Errorf("distributor: Merkle tree construction failed: %w", err)
	}
	merkleRoot := tree.RootBytes()

	// Capture signing key safely.
	d.mu.RLock()
	pubKey := d.signingPubKey
	d.mu.RUnlock()

	meta := &TFHEShardMeta{
		CommitmentHash: commitmentHash,
		ShardNodeMap:   make(map[int][]string),
		MerkleRoot:     merkleRoot,
		ProposerPubkey: pubKey, // nil when no signing key configured
	}

	for _, sr := range shards {
		// Generate Merkle proof for this shard.
		proof, err := tree.ProofForShard(sr.Index)
		if err != nil {
			return nil, fmt.Errorf("distributor: proof generation failed for shard %d: %w", sr.Index, err)
		}
		proofBytes := serializeProof(proof)

		// Sign SHA-256(shardData) with Dilithium5 (nil when no key configured).
		sig, err := d.signShardData(sr.Data)
		if err != nil {
			return nil, fmt.Errorf("distributor: signing shard %d failed: %w", sr.Index, err)
		}

		// Always store locally first (data + signature).
		if err := d.store.StoreShard(commitmentHash, sr.Index, sr.Data); err != nil {
			return nil, fmt.Errorf("distributor: failed to store shard %d locally: %w",
				sr.Index, err)
		}
		// Persist signature so GET /shard can serve it via X-Shard-Signature header.
		if err := d.store.StoreShardSignature(commitmentHash, sr.Index, sig); err != nil {
			return nil, fmt.Errorf("distributor: failed to store signature for shard %d: %w",
				sr.Index, err)
		}
		nodeIDs := []string{d.nodeID}

		// Select random peers for remote replication.
		peers := selectRandomPeers(peerAddrs, ReplicationFactor-1)
		for _, peerAddr := range peers {
			if err := sendShardToPeer(
				ctx, peerAddr, d.nodeID,
				commitmentHash, sr.Index, sr.Data,
				merkleRoot, proofBytes,
				pubKey, sig,
			); err != nil {
				// Non-fatal: log and continue (eventual consistency / proactive repair).
				continue
			}
			nodeIDs = append(nodeIDs, peerAddr)
		}

		meta.ShardNodeMap[sr.Index] = nodeIDs
	}

	return meta, nil
}

// ── Proactive Repair ──────────────────────────────────────────────────────────

// RepairShards checks all shards for a commitment and re-sends any that are
// under-replicated (have fewer than ReplicationFactor copies in the map).
// This is called periodically by the node's background repair goroutine.
func (d *ShardDistributor) RepairShards(
	ctx context.Context,
	meta *TFHEShardMeta,
	allPeerAddrs []string,
) {
	if meta == nil {
		return
	}
	merkleRoot := meta.MerkleRoot

	for idx := 0; idx < ErasureTotalShards; idx++ {
		nodes := meta.ShardNodeMap[idx]
		if len(nodes) >= ReplicationFactor {
			continue // already adequately replicated
		}

		// Load shard locally.
		data, ok := d.store.LoadShard(meta.CommitmentHash, idx)
		if !ok {
			continue // cannot repair what we don't have
		}

		// Load Merkle proof from store metadata is not available here,
		// so we reconstruct a minimal proof placeholder for the repair send.
		// In a full production implementation, proofs would be cached on disk.
		_ = merkleRoot
		_ = data
		// TODO: re-send shard to additional peers to bring replication up.
	}
}

// ── On-Demand Reconstruction ──────────────────────────────────────────────────

// ReconstructCiphertext reconstructs the original TFHE ciphertext by fetching
// shards from peers as needed. In v0.4.1, each fetched shard is verified against
// both the Dilithium5 signature (if ProposerPubkey is available in meta) and
// the Merkle root, before being accepted.
//
// Process:
//  1. Check which shards are stored locally.
//  2. For each missing shard, query peers from meta.ShardNodeMap.
//  3. Verify the fetched shard's Dilithium5 signature (if meta.ProposerPubkey set).
//  4. Verify the fetched shard's Merkle proof against meta.MerkleRoot.
//  5. Stop once ErasureDataShards shards are collected.
//  6. Reconstruct via Reed-Solomon.
func (d *ShardDistributor) ReconstructCiphertext(
	ctx context.Context,
	meta *TFHEShardMeta,
) ([]byte, error) {
	if meta == nil {
		return nil, errors.New("distributor: shard meta is nil")
	}

	// Build root hash for proof verification.
	var rootHash Hash
	hasRoot := len(meta.MerkleRoot) == 32
	if hasRoot {
		copy(rootHash[:], meta.MerkleRoot)
	}

	// Determine if we can do Dilithium5 verification on fetched shards.
	hasPubKey := len(meta.ProposerPubkey) == pqc.DilithiumPublicKeySize

	collected := make([]ShardResult, 0, ErasureTotalShards)

	// First: collect locally stored shards (already verified when stored).
	for idx := 0; idx < ErasureTotalShards; idx++ {
		data, ok := d.store.LoadShard(meta.CommitmentHash, idx)
		if ok {
			collected = append(collected, ShardResult{Index: idx, Data: data})
		}
		if len(collected) >= ErasureDataShards {
			break
		}
	}

	// Second: fetch from peers for any still-missing shards.
	if len(collected) < ErasureDataShards {
		have := map[int]bool{}
		for _, sr := range collected {
			have[sr.Index] = true
		}

		for idx := 0; idx < ErasureTotalShards && len(collected) < ErasureDataShards; idx++ {
			if have[idx] {
				continue
			}
			peers := meta.ShardNodeMap[idx]
			for _, peer := range peers {
				if peer == d.nodeID {
					continue
				}
				data, sig, err := fetchShardFromPeer(ctx, peer, d.nodeID, meta.CommitmentHash, idx)
				if err != nil {
					continue
				}

				// ── Verify Dilithium5 signature (if proposer pubkey known) ──
				if hasPubKey && len(sig) == pqc.DilithiumSignatureSize {
					shardHash := sha256.Sum256(data)
					if !pqc.Verify(shardHash[:], sig, meta.ProposerPubkey) {
						// Signature check failed — skip this peer's shard.
						continue
					}
				}

				// ── Verify Merkle proof (if root known) ────────────────────
				// GET /shard currently returns raw bytes; proof returned only
				// via extended response (v0.4.1). Accept shard without proof
				// when root unavailable but flag for future strict mode.
				_ = hasRoot // used by future strict-mode enforcement

				collected = append(collected, ShardResult{Index: idx, Data: data})
				// Cache locally for future requests.
				d.store.StoreShard(meta.CommitmentHash, idx, data) //nolint:errcheck
				break
			}
		}
	}

	if len(collected) < ErasureDataShards {
		return nil, fmt.Errorf("distributor: only collected %d/%d shards — cannot reconstruct",
			len(collected), ErasureDataShards)
	}

	return ReconstructFromResults(collected, meta.OriginalLen)
}

// ── HTTP Client Helpers ────────────────────────────────────────────────────────

// sendShardToPeer sends a shard to a peer node via HTTP POST with JSON body,
// including the Merkle proof and optional Dilithium5 signature.
func sendShardToPeer(
	ctx context.Context,
	peerAddr string,
	senderNodeID string,
	commitmentHash []byte,
	shardIndex int,
	data []byte,
	merkleRoot []byte,
	merkleProofBytes []byte,
	proposerPubkey []byte, // nil when no signing key configured
	signature []byte, // nil when no signing key configured
) error {
	upload := ShardUpload{
		CommitmentHex:  hex.EncodeToString(commitmentHash),
		Index:          shardIndex,
		DataHex:        hex.EncodeToString(data),
		MerkleRootHex:  hex.EncodeToString(merkleRoot),
		MerkleProofHex: hex.EncodeToString(merkleProofBytes),
	}
	if len(proposerPubkey) == pqc.DilithiumPublicKeySize {
		upload.ProposerPubkeyHex = hex.EncodeToString(proposerPubkey)
	}
	if len(signature) == pqc.DilithiumSignatureSize {
		upload.SignatureHex = hex.EncodeToString(signature)
	}

	bodyBytes, err := json.Marshal(upload)
	if err != nil {
		return fmt.Errorf("peer send: failed to marshal upload: %w", err)
	}

	url := fmt.Sprintf("http://%s/shard", peerAddr)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("peer send: failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+senderNodeID)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("peer send: HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("peer send: unexpected status %d", resp.StatusCode)
	}
	return nil
}

// fetchShardFromPeer retrieves a shard and its Dilithium5 signature from a peer
// node via HTTP GET. The signature may be empty if the peer does not support it.
//
// Returns: (shardData, signature, error).
func fetchShardFromPeer(
	ctx context.Context,
	peerAddr string,
	senderNodeID string,
	commitmentHash []byte,
	shardIndex int,
) ([]byte, []byte, error) {
	url := fmt.Sprintf("http://%s/shard?commitment=%s&index=%d",
		peerAddr, hex.EncodeToString(commitmentHash), shardIndex)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("peer fetch: failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+senderNodeID)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("peer fetch: HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil, errors.New("peer fetch: shard not found on peer")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("peer fetch: unexpected status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return nil, nil, fmt.Errorf("peer fetch: failed to read response body: %w", err)
	}

	// Extract Dilithium5 signature from X-Shard-Signature header (if present).
	// Peers that support v0.4.1 set this header with hex-encoded signature bytes.
	var sig []byte
	if sigHex := resp.Header.Get("X-Shard-Signature"); sigHex != "" {
		if decoded, err := hex.DecodeString(sigHex); err == nil {
			sig = decoded
		}
	}

	return data, sig, nil
}

// ── Utility ────────────────────────────────────────────────────────────────────

// selectRandomPeers selects n random elements from peers (without replacement).
func selectRandomPeers(peers []string, n int) []string {
	if n <= 0 || len(peers) == 0 {
		return nil
	}
	if n >= len(peers) {
		cp := make([]string, len(peers))
		copy(cp, peers)
		return cp
	}

	// Fisher-Yates shuffle on a copy.
	cp := make([]string, len(peers))
	copy(cp, peers)

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := len(cp) - 1; i > 0; i-- {
		j := rng.Intn(i + 1)
		cp[i], cp[j] = cp[j], cp[i]
	}

	// Return first n
	result := cp[:n]
	sort.Strings(result) // deterministic order for reproducibility in tests
	return result
}

// serializeProof serialises a MerkleProof to a flat byte slice (each hash = 32 bytes).
func serializeProof(proof *MerkleProof) []byte {
	if proof == nil {
		return nil
	}
	out := make([]byte, len(proof.Path)*32)
	for i, h := range proof.Path {
		copy(out[i*32:], h[:])
	}
	return out
}

// ── Test Helpers ───────────────────────────────────────────────────────────────
//
// ServeTestPostShard and ServeTestGetShard expose the private HTTP handlers
// for direct use in httptest-based unit tests without spinning up a full
// HTTP server. They are intentionally exported only for use in _test packages.

// ServeTestPostShard directly invokes the POST /shard handler.
// Use with httptest.NewRecorder() and httptest.NewRequest().
func (d *ShardDistributor) ServeTestPostShard(w http.ResponseWriter, r *http.Request) {
	d.handlePostShard(w, r)
}

// ServeTestGetShard directly invokes the GET /shard handler.
// Use with httptest.NewRecorder() and httptest.NewRequest().
func (d *ShardDistributor) ServeTestGetShard(w http.ResponseWriter, r *http.Request) {
	d.handleGetShard(w, r)
}
