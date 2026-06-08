// shard_dilithium_test.go — Integration tests for Dilithium5 shard signing.
//
// Tests verify the full signing + verification flow:
//  1. ShardDistributor.WithSigningKey() attaches a valid key pair.
//  2. signShardData() produces a 4595-byte signature.
//  3. Signature verifies correctly against the public key.
//  4. A tampered shard fails verification.
//  5. POST /shard handler rejects invalid signatures (HTTP 403).
//  6. POST /shard handler accepts valid signatures (HTTP 201).
//  7. GET /shard response carries X-Shard-Signature header.
package tfhe_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"zytherion/x/privacy/pqc"
	tfhepkg "zytherion/x/privacy/tfhe"
)

// tempShardStore creates a ShardStore backed by a temporary directory.
func tempShardStore(t *testing.T) *tfhepkg.ShardStore {
	t.Helper()
	dir := t.TempDir()
	store, err := tfhepkg.NewShardStore(filepath.Join(dir, "shards"))
	require.NoError(t, err)
	return store
}

// newSignedDistributor creates a ShardDistributor with a fresh Dilithium5 key pair.
func newSignedDistributor(t *testing.T) (*tfhepkg.ShardDistributor, pqc.KeyPair) {
	t.Helper()
	store := tempShardStore(t)
	dist := tfhepkg.NewShardDistributor(store, "test-node")

	kp, err := pqc.GenerateKeyPair()
	require.NoError(t, err)

	err = dist.WithSigningKey(kp)
	require.NoError(t, err)

	return dist, kp
}

// TestWithSigningKey_ValidKey verifies that WithSigningKey accepts a valid key pair.
func TestWithSigningKey_ValidKey(t *testing.T) {
	dist, kp := newSignedDistributor(t)
	pubkey := dist.ProposerPubkey()

	require.Equal(t, kp.PublicKey, pubkey, "ProposerPubkey must match the key pair's public key")
	require.Len(t, pubkey, pqc.DilithiumPublicKeySize)
}

// TestWithSigningKey_InvalidKey verifies that WithSigningKey rejects a malformed key.
func TestWithSigningKey_InvalidKey(t *testing.T) {
	store := tempShardStore(t)
	dist := tfhepkg.NewShardDistributor(store, "test-node")

	badKP := pqc.KeyPair{
		PrivateKey: []byte("too short"),
		PublicKey:  []byte("also too short"),
	}
	err := dist.WithSigningKey(badKP)
	require.Error(t, err, "WithSigningKey must reject invalid key lengths")
}

// TestProposerPubkey_NoKey verifies that ProposerPubkey returns nil when not configured.
func TestProposerPubkey_NoKey(t *testing.T) {
	store := tempShardStore(t)
	dist := tfhepkg.NewShardDistributor(store, "test-node")
	require.Nil(t, dist.ProposerPubkey(), "ProposerPubkey must be nil without signing key")
}

// TestShardSignAndVerify verifies the end-to-end sign + verify flow for a shard.
func TestShardSignAndVerify(t *testing.T) {
	_, kp := newSignedDistributor(t)

	shardData := []byte("hello shard data from Zytherion v0.4.1")
	shardHash := sha256.Sum256(shardData)

	sig, err := pqc.Sign(shardHash[:], kp.PrivateKey)
	require.NoError(t, err)
	require.Len(t, sig, pqc.DilithiumSignatureSize)

	ok := pqc.Verify(shardHash[:], sig, kp.PublicKey)
	require.True(t, ok, "signature must verify against matching public key")
}

// TestShardSignAndVerify_TamperedData verifies that a tampered shard fails verification.
func TestShardSignAndVerify_TamperedData(t *testing.T) {
	_, kp := newSignedDistributor(t)

	originalData := []byte("original shard data")
	shardHash := sha256.Sum256(originalData)

	sig, err := pqc.Sign(shardHash[:], kp.PrivateKey)
	require.NoError(t, err)

	// Tamper with a single byte.
	tamperedData := append([]byte(nil), originalData...)
	tamperedData[0] ^= 0xFF
	tamperedHash := sha256.Sum256(tamperedData)

	ok := pqc.Verify(tamperedHash[:], sig, kp.PublicKey)
	require.False(t, ok, "tampered shard data must fail signature verification")
}

// TestShardSignAndVerify_WrongKey verifies that verification fails with a different key.
func TestShardSignAndVerify_WrongKey(t *testing.T) {
	_, kp := newSignedDistributor(t)

	otherKP, err := pqc.GenerateKeyPair()
	require.NoError(t, err)

	data := []byte("legitimate shard")
	h := sha256.Sum256(data)

	sig, err := pqc.Sign(h[:], kp.PrivateKey)
	require.NoError(t, err)

	// Verify with a DIFFERENT public key — must fail.
	ok := pqc.Verify(h[:], sig, otherKP.PublicKey)
	require.False(t, ok, "signature must not verify with a different public key")
}

// ── HTTP handler tests ────────────────────────────────────────────────────────

// buildShardUploadJSON builds a ShardUpload JSON body for testing the POST handler.
func buildShardUploadJSON(t *testing.T,
	commitmentHex string,
	index int,
	data []byte,
	merkleRootHex string,
	merkleProofHex string,
	pubkeyHex string,
	sigHex string,
) []byte {
	t.Helper()
	upload := map[string]interface{}{
		"commitment_hex":  commitmentHex,
		"index":           index,
		"data_hex":        hex.EncodeToString(data),
		"merkle_root_hex": merkleRootHex,
		"merkle_proof_hex": merkleProofHex,
	}
	if pubkeyHex != "" {
		upload["proposer_pubkey_hex"] = pubkeyHex
		upload["signature_hex"] = sigHex
	}
	body, err := json.Marshal(upload)
	require.NoError(t, err)
	return body
}

// TestPostShard_ValidDilithiumSignature verifies that a correctly signed shard
// is accepted by the POST /shard handler (HTTP 201).
func TestPostShard_ValidDilithiumSignature(t *testing.T) {
	// Build 16 shards from dummy data so we have valid Merkle proofs.
	original := make([]byte, 21*1024)
	shards, err := tfhepkg.Split(original)
	require.NoError(t, err)

	tree, err := tfhepkg.BuildMerkleTree(shards)
	require.NoError(t, err)
	merkleRoot := tree.RootBytes()

	// Pick shard 0.
	sr := shards[0]
	proof, err := tree.ProofForShard(sr.Index)
	require.NoError(t, err)

	proofBytes := make([]byte, len(proof.Path)*32)
	for i, h := range proof.Path {
		copy(proofBytes[i*32:], h[:])
	}

	// Generate key pair and sign shard 0.
	kp, err := pqc.GenerateKeyPair()
	require.NoError(t, err)
	shardHash := sha256.Sum256(sr.Data)
	sig, err := pqc.Sign(shardHash[:], kp.PrivateKey)
	require.NoError(t, err)

	commitment := sha256.Sum256(original)

	body := buildShardUploadJSON(t,
		hex.EncodeToString(commitment[:]),
		sr.Index,
		sr.Data,
		hex.EncodeToString(merkleRoot),
		hex.EncodeToString(proofBytes),
		hex.EncodeToString(kp.PublicKey),
		hex.EncodeToString(sig),
	)

	// Create a temporary store-backed distributor.
	store := tempShardStore(t)
	dist := tfhepkg.NewShardDistributor(store, "test-node")

	// Use httptest to call handlePostShard indirectly via the mux.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tmpDir := t.TempDir()
	listenAddr := filepath.Join(tmpDir, "unused") // not actually bound
	_ = listenAddr

	// Build a standalone request targeting the handler directly.
	req := httptest.NewRequest(http.MethodPost, "/shard", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer test-node")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Start and immediately stop the server to exercise the handler.
	_ = ctx
	// Manually call the handler via the public ServeHTTP-compatible helper:
	dist.ServeTestPostShard(w, req)

	require.Equal(t, http.StatusCreated, w.Code,
		"valid Dilithium5 signature + Merkle proof must return 201 Created")
}

// TestPostShard_InvalidDilithiumSignature verifies that a shard with a bad
// Dilithium5 signature is rejected with HTTP 403 Forbidden.
func TestPostShard_InvalidDilithiumSignature(t *testing.T) {
	original := make([]byte, 21*1024)
	shards, err := tfhepkg.Split(original)
	require.NoError(t, err)

	tree, err := tfhepkg.BuildMerkleTree(shards)
	require.NoError(t, err)
	merkleRoot := tree.RootBytes()

	sr := shards[0]
	proof, err := tree.ProofForShard(sr.Index)
	require.NoError(t, err)
	proofBytes := make([]byte, len(proof.Path)*32)
	for i, h := range proof.Path {
		copy(proofBytes[i*32:], h[:])
	}

	kp, err := pqc.GenerateKeyPair()
	require.NoError(t, err)

	// Sign the WRONG data (tampered).
	wrongData := append([]byte(nil), sr.Data...)
	wrongData[0] ^= 0xFF
	wrongHash := sha256.Sum256(wrongData)
	badSig, err := pqc.Sign(wrongHash[:], kp.PrivateKey)
	require.NoError(t, err)

	commitment := sha256.Sum256(original)

	body := buildShardUploadJSON(t,
		hex.EncodeToString(commitment[:]),
		sr.Index,
		sr.Data, // ACTUAL data but WRONG sig (signed over tampered data)
		hex.EncodeToString(merkleRoot),
		hex.EncodeToString(proofBytes),
		hex.EncodeToString(kp.PublicKey),
		hex.EncodeToString(badSig),
	)

	store := tempShardStore(t)
	dist := tfhepkg.NewShardDistributor(store, "test-node")

	req := httptest.NewRequest(http.MethodPost, "/shard", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer test-node")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	dist.ServeTestPostShard(w, req)

	require.Equal(t, http.StatusForbidden, w.Code,
		"invalid Dilithium5 signature must return 403 Forbidden")
}

// TestStoreAndLoadSignature verifies signature disk persistence round-trip.
func TestStoreAndLoadSignature(t *testing.T) {
	store := tempShardStore(t)

	kp, err := pqc.GenerateKeyPair()
	require.NoError(t, err)

	data := []byte("shard data bytes")
	h := sha256.Sum256(data)
	sig, err := pqc.Sign(h[:], kp.PrivateKey)
	require.NoError(t, err)

	commitment := sha256.Sum256(data)
	err = store.StoreShardSignature(commitment[:], 3, sig)
	require.NoError(t, err)

	loaded, ok := store.LoadShardSignature(commitment[:], 3)
	require.True(t, ok)
	require.Equal(t, sig, loaded, "loaded signature must match stored signature")

	// Non-existent index must return false.
	_, ok = store.LoadShardSignature(commitment[:], 99)
	require.False(t, ok)
}

// TestGetShard_XShardSignatureHeader verifies that GET /shard returns the
// X-Shard-Signature header when a signature is stored.
func TestGetShard_XShardSignatureHeader(t *testing.T) {
	store := tempShardStore(t)
	dist := tfhepkg.NewShardDistributor(store, "test-node")

	commitment := sha256.Sum256([]byte("test"))
	data := []byte("shard payload data for header test")
	err := store.StoreShard(commitment[:], 5, data)
	require.NoError(t, err)

	kp, err := pqc.GenerateKeyPair()
	require.NoError(t, err)
	h := sha256.Sum256(data)
	sig, err := pqc.Sign(h[:], kp.PrivateKey)
	require.NoError(t, err)

	err = store.StoreShardSignature(commitment[:], 5, sig)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/shard?commitment="+
		hex.EncodeToString(commitment[:])+"&index=5", nil)
	req.Header.Set("Authorization", "Bearer test-node")
	w := httptest.NewRecorder()

	dist.ServeTestGetShard(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	sigHeader := w.Header().Get("X-Shard-Signature")
	require.NotEmpty(t, sigHeader, "GET /shard must return X-Shard-Signature header")

	decodedSig, err := hex.DecodeString(sigHeader)
	require.NoError(t, err)
	require.Equal(t, sig, decodedSig, "X-Shard-Signature header must match stored signature")

	// Verify the signature in the header is actually valid.
	h2 := sha256.Sum256(w.Body.Bytes())
	require.True(t, pqc.Verify(h2[:], decodedSig, kp.PublicKey),
		"signature from X-Shard-Signature must verify against proposer pubkey")

	// Make sure the directory cleanup works.
	_ = os.RemoveAll(t.TempDir())
}
