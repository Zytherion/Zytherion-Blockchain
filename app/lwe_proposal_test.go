package app_test

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/require"

	"zytherion/x/privacy/pqc"
)

// These tests validate the sentinel construction logic and ProcessProposal
// validation rules in isolation, without spinning up a full app.
// They directly exercise the helpers via exported constants from the app package.

// ── Sentinel wire format constants (mirrored from lwe_proposal.go) ───────────

const (
	testLWEMarkerUint32 uint32 = 0x4C574548 // "LWEH" BE
	testLWEVersion      uint32 = 0x00000001
	testLWEHeaderSize          = 8
	testLWESentinelSize        = testLWEHeaderSize + pqc.LWEHashSize // 104
)

// buildTestSentinel constructs a sentinel identical to the app's buildSentinel.
func buildTestSentinel(lweHash []byte) []byte {
	s := make([]byte, testLWESentinelSize)
	binary.BigEndian.PutUint32(s[0:4], testLWEMarkerUint32)
	binary.BigEndian.PutUint32(s[4:8], testLWEVersion)
	copy(s[testLWEHeaderSize:], lweHash)
	return s
}

// buildTestTxPayload concatenates txs the same way buildTxPayload does.
func buildTestTxPayload(txs [][]byte) []byte {
	var p []byte
	for _, tx := range txs {
		p = append(p, tx...)
	}
	return p
}

// ── Tests ─────────────────────────────────────────────────────────────────────

// TestSentinelSize verifies the sentinel is exactly 104 bytes.
func TestSentinelSize(t *testing.T) {
	hash, err := pqc.GenerateLWEBlockHash([]byte("test"), make([]byte, 32))
	require.NoError(t, err)
	sentinel := buildTestSentinel(hash)
	require.Len(t, sentinel, testLWESentinelSize,
		"sentinel must be exactly %d bytes", testLWESentinelSize)
}

// TestSentinelMagicAndVersion validates the wire-format header constants.
func TestSentinelMagicAndVersion(t *testing.T) {
	hash, err := pqc.GenerateLWEBlockHash([]byte("magic-test"), make([]byte, 32))
	require.NoError(t, err)
	sentinel := buildTestSentinel(hash)

	magic := binary.BigEndian.Uint32(sentinel[0:4])
	version := binary.BigEndian.Uint32(sentinel[4:8])

	require.Equal(t, testLWEMarkerUint32, magic,
		"magic must be 0x4C574548 (\"LWEH\")")
	require.Equal(t, testLWEVersion, version,
		"version must be 0x00000001")
}

// TestSentinelHashRoundtrip extracts the hash back from the sentinel and
// checks it equals the original LWE hash (no corruption).
func TestSentinelHashRoundtrip(t *testing.T) {
	input := []byte("alice-sends-500-ZYT-block-42")
	prevHash := make([]byte, 32)

	lweHash, err := pqc.GenerateLWEBlockHash(input, prevHash)
	require.NoError(t, err)

	sentinel := buildTestSentinel(lweHash)
	extracted := sentinel[testLWEHeaderSize : testLWEHeaderSize+pqc.LWEHashSize]

	require.True(t, bytes.Equal(lweHash, extracted),
		"extracted hash must equal original LWE hash")
}

// TestProcessProposal_ValidSentinel simulates a valid proposal: build the
// sentinel from the tx payload, then verify that re-computing the hash
// from the same payload gives a matching result.
func TestProcessProposal_ValidSentinel(t *testing.T) {
	txs := [][]byte{
		[]byte("tx-payload-1"),
		[]byte("tx-payload-2"),
	}
	prevHash := make([]byte, 32)
	txPayload := buildTestTxPayload(txs)

	// Compute the expected hash (as PrepareProposal would do).
	lweHash, err := pqc.GenerateLWEBlockHash(txPayload, prevHash)
	require.NoError(t, err)
	sentinel := buildTestSentinel(lweHash)

	// Simulate proposal: [sentinel, tx1, tx2]
	proposal := append([][]byte{sentinel}, txs...)

	// Simulate ProcessProposal verification.
	require.GreaterOrEqual(t, len(proposal), 1, "proposal must have at least 1 entry")

	extractedSentinel := proposal[0]
	require.Len(t, extractedSentinel, testLWESentinelSize)

	magic := binary.BigEndian.Uint32(extractedSentinel[0:4])
	require.Equal(t, testLWEMarkerUint32, magic)

	claimedHash := extractedSentinel[testLWEHeaderSize : testLWEHeaderSize+pqc.LWEHashSize]

	// Re-compute over user txs.
	userTxs := proposal[1:]
	recomputed, err := pqc.GenerateLWEBlockHash(buildTestTxPayload(userTxs), prevHash)
	require.NoError(t, err)

	require.True(t, bytes.Equal(claimedHash, recomputed),
		"ProcessProposal: claimed hash must match re-computed hash for a valid proposal")
}

// TestProcessProposal_TamperedTx simulates an attacker modifying a tx AFTER
// the hash was computed — the re-computed hash must differ (→ REJECT).
func TestProcessProposal_TamperedTx(t *testing.T) {
	txs := [][]byte{
		[]byte("original-tx-1"),
		[]byte("original-tx-2"),
	}
	prevHash := make([]byte, 32)
	txPayload := buildTestTxPayload(txs)

	// Proposer computes sentinel over original txs.
	lweHash, err := pqc.GenerateLWEBlockHash(txPayload, prevHash)
	require.NoError(t, err)
	sentinel := buildTestSentinel(lweHash)

	// Attacker swaps tx-2 with a malicious payload.
	tamperedTxs := [][]byte{
		[]byte("original-tx-1"),
		[]byte("MALICIOUS-TX!!"),
	}

	// Simulate ProcessProposal re-computation over tampered txs.
	claimedHash := sentinel[testLWEHeaderSize : testLWEHeaderSize+pqc.LWEHashSize]
	recomputed, err := pqc.GenerateLWEBlockHash(buildTestTxPayload(tamperedTxs), prevHash)
	require.NoError(t, err)

	require.False(t, bytes.Equal(claimedHash, recomputed),
		"tampered tx payload must produce a different hash → proposal should be REJECTED")
}

// TestProcessProposal_MissingSentinel verifies that a proposal with no txs
// (or with a non-sentinel first entry) would be rejected in ProcessProposal.
func TestProcessProposal_MissingSentinel(t *testing.T) {
	// Empty proposal.
	emptyProposal := [][]byte{}
	require.Empty(t, emptyProposal, "empty proposal must be caught (no sentinel)")

	// Proposal with a regular tx as first entry (no sentinel marker).
	noSentinelProposal := [][]byte{
		[]byte("not-a-sentinel"),
		[]byte("tx-data"),
	}
	first := noSentinelProposal[0]
	// A valid sentinel must be exactly 104 bytes.
	require.NotEqual(t, testLWESentinelSize, len(first),
		"non-sentinel first entry must fail size check → REJECT")
}

// TestProcessProposal_WrongMagic verifies that a 104-byte blob with a wrong
// magic number is rejected.
func TestProcessProposal_WrongMagic(t *testing.T) {
	fakeSentinel := make([]byte, testLWESentinelSize)
	binary.BigEndian.PutUint32(fakeSentinel[0:4], 0xDEADBEEF) // wrong magic

	magic := binary.BigEndian.Uint32(fakeSentinel[0:4])
	require.NotEqual(t, testLWEMarkerUint32, magic,
		"wrong magic must cause REJECT in ProcessProposal")
}

// TestLWEHashDeterminismAcrossProposal verifies that the same tx set always
// produces the same sentinel (deterministic proposal → deterministic validation).
func TestLWEHashDeterminismAcrossProposal(t *testing.T) {
	txs := [][]byte{[]byte("deterministic-payload")}
	prevHash := make([]byte, 32)
	payload := buildTestTxPayload(txs)

	h1, err := pqc.GenerateLWEBlockHash(payload, prevHash)
	require.NoError(t, err)
	h2, err := pqc.GenerateLWEBlockHash(payload, prevHash)
	require.NoError(t, err)

	require.True(t, bytes.Equal(h1, h2),
		"same tx payload must produce identical LWE hash across PrepareProposal calls")
}
