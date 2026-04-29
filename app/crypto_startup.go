// crypto_startup.go — Startup diagnostics for Zytherion's cryptographic subsystems.
//
// This file runs self-tests for both the FHE (BFV/Lattigo) and LWE (Ring-LWE)
// cryptographic subsystems when the node boots, printing a clear status banner
// to the node logger so operators can immediately confirm that all privacy
// primitives are operational.
//
// Integration: called from app.New() immediately after fheCtx is created.
package app

import (
	"encoding/hex"
	"fmt"
	"time"

	"github.com/cometbft/cometbft/libs/log"

	"zytherion/x/privacy/fhe"
	"zytherion/x/privacy/pqc"
)

// cryptoStatus holds the result of each subsystem check.
type cryptoStatus struct {
	name    string
	ok      bool
	detail  string
	elapsed time.Duration
}

// RunCryptoStartupChecks verifies that both the FHE and LWE subsystems are
// fully operational.  It logs a startup status banner showing:
//
//   - FHE (BFV/Lattigo): encrypt → add → decrypt round-trip test
//   - LWE (Ring-LWE hash): hash generation + avalanche sanity check
//
// This function MUST be called after the fheCtx has been successfully created
// in app.New().  It is a read-only self-test — it does NOT mutate any state.
func RunCryptoStartupChecks(logger log.Logger, fheCtx *fhe.Context) {
	results := []cryptoStatus{
		checkFHE(fheCtx),
		checkLWE(),
	}

	printStartupBanner(logger, results)
}

// ── FHE check ────────────────────────────────────────────────────────────────

// checkFHE performs an encrypt → homomorphic-add → decrypt round-trip using
// the already-initialised fheCtx.  This proves:
//  1. The BFV parameters are internally consistent.
//  2. The key pair (pk/sk) are valid.
//  3. The evaluator can perform AddNew without panicking.
func checkFHE(fheCtx *fhe.Context) cryptoStatus {
	start := time.Now()

	const a, b uint64 = 123_456_789, 987_654_321
	expected := a + b

	// Encrypt two values.
	ctA, err := fheCtx.Encrypt(a)
	if err != nil {
		return cryptoStatus{
			name:    "FHE (BFV/Lattigo)",
			ok:      false,
			detail:  fmt.Sprintf("Encrypt(%d) failed: %v", a, err),
			elapsed: time.Since(start),
		}
	}

	ctB, err := fheCtx.Encrypt(b)
	if err != nil {
		return cryptoStatus{
			name:    "FHE (BFV/Lattigo)",
			ok:      false,
			detail:  fmt.Sprintf("Encrypt(%d) failed: %v", b, err),
			elapsed: time.Since(start),
		}
	}

	// Homomorphic addition.
	ctSum, err := fheCtx.AddCiphertexts(ctA, ctB)
	if err != nil {
		return cryptoStatus{
			name:    "FHE (BFV/Lattigo)",
			ok:      false,
			detail:  fmt.Sprintf("AddCiphertexts failed: %v", err),
			elapsed: time.Since(start),
		}
	}

	// Decrypt and verify.
	result, err := fheCtx.Decrypt(ctSum)
	if err != nil {
		return cryptoStatus{
			name:    "FHE (BFV/Lattigo)",
			ok:      false,
			detail:  fmt.Sprintf("Decrypt failed: %v", err),
			elapsed: time.Since(start),
		}
	}

	if result != expected {
		return cryptoStatus{
			name:   "FHE (BFV/Lattigo)",
			ok:     false,
			detail: fmt.Sprintf("round-trip mismatch: got %d, want %d", result, expected),
		}
	}

	return cryptoStatus{
		name:    "FHE (TFHE-rs)",
		ok:      true,
		detail:  fmt.Sprintf("Enc(%d) + Enc(%d) -> Dec = %d ✓  |  TFHE Uint64", a, b, result),
		elapsed: time.Since(start),
	}
}

// ── LWE check ─────────────────────────────────────────────────────────────────

// checkLWE generates two LWE block hashes from inputs that differ by a single
// bit and confirms:
//  1. The output is exactly LWEHashSize (96) bytes.
//  2. All 32 b-coefficients are in [0, q).
//  3. A 1-bit input change causes at least 25% of output bytes to differ
//     (avalanche effect).
func checkLWE() cryptoStatus {
	start := time.Now()

	input1 := []byte("zytherion-lwe-startup-probe-v1")
	input2 := make([]byte, len(input1))
	copy(input2, input1)
	input2[0] ^= 0x01 // flip 1 bit

	prevHash := make([]byte, 32)

	h1, err := pqc.GenerateLWEBlockHash(input1, prevHash)
	if err != nil {
		return cryptoStatus{
			name:    "LWE (Ring-LWE / SHAKE-256)",
			ok:      false,
			detail:  fmt.Sprintf("GenerateLWEBlockHash failed: %v", err),
			elapsed: time.Since(start),
		}
	}

	if err := pqc.ValidateLWEHash(h1); err != nil {
		return cryptoStatus{
			name:    "LWE (Ring-LWE / SHAKE-256)",
			ok:      false,
			detail:  fmt.Sprintf("ValidateLWEHash failed: %v", err),
			elapsed: time.Since(start),
		}
	}

	h2, err := pqc.GenerateLWEBlockHash(input2, prevHash)
	if err != nil {
		return cryptoStatus{
			name:    "LWE (Ring-LWE / SHAKE-256)",
			ok:      false,
			detail:  fmt.Sprintf("GenerateLWEBlockHash (alt input) failed: %v", err),
			elapsed: time.Since(start),
		}
	}

	// Avalanche check: ≥ 25% of output bytes must differ.
	diffBytes := 0
	for i := 0; i < pqc.LWEHashSize; i++ {
		if h1[i] != h2[i] {
			diffBytes++
		}
	}
	avalanchePct := diffBytes * 100 / pqc.LWEHashSize
	if avalanchePct < 25 {
		return cryptoStatus{
			name:    "LWE (Ring-LWE / SHAKE-256)",
			ok:      false,
			detail:  fmt.Sprintf("weak avalanche: only %d%% bytes differ (want ≥25%%)", avalanchePct),
			elapsed: time.Since(start),
		}
	}

	return cryptoStatus{
		name: "LWE (Ring-LWE / SHAKE-256)",
		ok:   true,
		detail: fmt.Sprintf(
			"n=%d q=%d size=%dB seed=%s…  avalanche=%d%%/1-bit ✓",
			256, 3329, pqc.LWEHashSize,
			hex.EncodeToString(h1[:4]),
			avalanchePct,
		),
		elapsed: time.Since(start),
	}
}

// ── Banner printer ────────────────────────────────────────────────────────────

// printStartupBanner logs a formatted startup status block to the node logger.
// Operators see an unambiguous OK / FAIL for every crypto subsystem.
func printStartupBanner(logger log.Logger, results []cryptoStatus) {
	allOK := true
	for _, r := range results {
		if !r.ok {
			allOK = false
			break
		}
	}

	sep := "═══════════════════════════════════════════════════════════"

	logger.Info(sep)
	logger.Info("  ⚛  ZYTHERION CRYPTOGRAPHIC SUBSYSTEM STARTUP REPORT  ⚛")
	logger.Info(sep)

	for _, r := range results {
		status := "✅ OK  "
		if !r.ok {
			status = "❌ FAIL"
		}
		logger.Info(fmt.Sprintf("  [%s] %s", status, r.name),
			"detail", r.detail,
			"elapsed", r.elapsed.Round(time.Millisecond).String(),
		)
	}

	logger.Info(sep)
	if allOK {
		logger.Info("  ✅ ALL CRYPTO SUBSYSTEMS OPERATIONAL — node is READY")
		logger.Info("     FHE encrypted transfers: ACTIVE")
		logger.Info("     LWE block hashing:        ACTIVE  (algo=" + pqc.HashAlgorithm + ")")
	} else {
		logger.Error("  ❌ ONE OR MORE CRYPTO SUBSYSTEMS FAILED — CHECK LOGS ABOVE")
		logger.Error("     The node will continue but affected features may not work.")
	}
	logger.Info(sep)
}
