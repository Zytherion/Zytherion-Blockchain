// crypto_startup.go — Startup diagnostics for Zytherion's cryptographic subsystems.
//
// Runs self-tests for the Dilithium5 (ML-DSA Level 5) signature and LWR hashing
// subsystems when the node boots, printing a clear status banner to the node logger.
//
// # v0.3 Changes
//
// ZK (Groth16/BN254) subsystem has been REMOVED in v0.3.
// The startup check now verifies:
//   - Dilithium5 sign/verify round-trip (replaces ZK verifier check)
//   - LWR (Ring-LWR / SHAKE-256) block hashing with avalanche property
//   - TFHE subsystem status (enabled/disabled report)
package app

import (
	"encoding/hex"
	"fmt"
	"time"

	"github.com/cometbft/cometbft/libs/log"

	"zytherion/x/privacy/pqc"
)

// cryptoStatus holds the result of each subsystem check.
type cryptoStatus struct {
	name    string
	ok      bool
	detail  string
	elapsed time.Duration
}

// RunCryptoStartupChecks verifies that the Dilithium5 signer and LWR subsystems
// are fully operational. It logs a startup status banner showing:
//
//   - Dilithium5 (ML-DSA Level 5): keygen + sign/verify self-test
//   - LWR (Ring-LWR hash): hash generation + avalanche sanity check
//   - TFHE: enabled/disabled status (no functional check at startup — too slow)
//
// This function is a read-only self-test — it does NOT mutate any state.
func RunCryptoStartupChecks(logger log.Logger, tfheEnabled bool) {
	results := []cryptoStatus{
		checkDilithium5(),
		checkLWR(),
		checkTFHEStatus(tfheEnabled),
	}

	printStartupBanner(logger, results)
}

// ── Dilithium5 self-test ──────────────────────────────────────────────────────

// checkDilithium5 performs a complete sign/verify round-trip with Dilithium5.
// This verifies the circl library is functioning correctly at startup.
func checkDilithium5() cryptoStatus {
	start := time.Now()

	kp, err := pqc.GenerateKeyPair()
	if err != nil {
		return cryptoStatus{
			name:    "Dilithium5 (ML-DSA Level 5)",
			ok:      false,
			detail:  fmt.Sprintf("GenerateKeyPair failed: %v", err),
			elapsed: time.Since(start),
		}
	}

	testMsg := []byte("zytherion-dilithium5-startup-probe-v03")
	sig, err := pqc.Sign(testMsg, kp.PrivateKey)
	if err != nil {
		return cryptoStatus{
			name:    "Dilithium5 (ML-DSA Level 5)",
			ok:      false,
			detail:  fmt.Sprintf("Sign failed: %v", err),
			elapsed: time.Since(start),
		}
	}

	if !pqc.Verify(testMsg, sig, kp.PublicKey) {
		return cryptoStatus{
			name:    "Dilithium5 (ML-DSA Level 5)",
			ok:      false,
			detail:  "Verify returned false for a freshly signed message — CRITICAL BUG",
			elapsed: time.Since(start),
		}
	}

	// Verify tampered message fails.
	tampered := make([]byte, len(testMsg))
	copy(tampered, testMsg)
	tampered[0] ^= 0xFF
	if pqc.Verify(tampered, sig, kp.PublicKey) {
		return cryptoStatus{
			name:    "Dilithium5 (ML-DSA Level 5)",
			ok:      false,
			detail:  "Verify accepted a tampered message — CRITICAL SECURITY BUG",
			elapsed: time.Since(start),
		}
	}

	return cryptoStatus{
		name: "Dilithium5 (ML-DSA Level 5)",
		ok:   true,
		detail: fmt.Sprintf(
			"pubkey=%dB privkey=%dB sig=%dB  sign=✓ verify=✓ tamper-reject=✓",
			pqc.DilithiumPublicKeySize, pqc.DilithiumPrivateKeySize, pqc.DilithiumSignatureSize,
		),
		elapsed: time.Since(start),
	}
}

// ── LWR check ─────────────────────────────────────────────────────────────────

// checkLWR generates two LWR block hashes and confirms avalanche properties.
func checkLWR() cryptoStatus {
	start := time.Now()

	input1 := []byte("zytherion-lwr-startup-probe-v1")
	input2 := make([]byte, len(input1))
	copy(input2, input1)
	input2[0] ^= 0x01 // flip 1 bit

	prevHash := make([]byte, 32)

	h1, err := pqc.GenerateLWRBlockHash(input1, prevHash)
	if err != nil {
		return cryptoStatus{
			name:    "LWR (Ring-LWR / SHAKE-256)",
			ok:      false,
			detail:  fmt.Sprintf("GenerateLWRBlockHash failed: %v", err),
			elapsed: time.Since(start),
		}
	}

	if err := pqc.ValidateLWRHash(h1); err != nil {
		return cryptoStatus{
			name:    "LWR (Ring-LWR / SHAKE-256)",
			ok:      false,
			detail:  fmt.Sprintf("ValidateLWRHash failed: %v", err),
			elapsed: time.Since(start),
		}
	}

	h2, err := pqc.GenerateLWRBlockHash(input2, prevHash)
	if err != nil {
		return cryptoStatus{
			name:    "LWR (Ring-LWR / SHAKE-256)",
			ok:      false,
			detail:  fmt.Sprintf("GenerateLWRBlockHash (alt input) failed: %v", err),
			elapsed: time.Since(start),
		}
	}

	// Avalanche check: ≥ 25% of output bytes must differ.
	diffBytes := 0
	for i := 0; i < pqc.LWRHashSize; i++ {
		if h1[i] != h2[i] {
			diffBytes++
		}
	}
	avalanchePct := diffBytes * 100 / pqc.LWRHashSize
	if avalanchePct < 25 {
		return cryptoStatus{
			name:    "LWR (Ring-LWR / SHAKE-256)",
			ok:      false,
			detail:  fmt.Sprintf("weak avalanche: only %d%% bytes differ (want ≥25%%)", avalanchePct),
			elapsed: time.Since(start),
		}
	}

	return cryptoStatus{
		name: "LWR (Ring-LWR / SHAKE-256)",
		ok:   true,
		detail: fmt.Sprintf(
			"n=%d q=%d size=%dB seed=%s…  avalanche=%d%%/1-bit ✓",
			256, 3329, pqc.LWRHashSize,
			hex.EncodeToString(h1[:4]),
			avalanchePct,
		),
		elapsed: time.Since(start),
	}
}

// ── TFHE status check ─────────────────────────────────────────────────────────

// checkTFHEStatus reports the TFHE subsystem enabled/disabled status.
// We do NOT perform a functional TFHE test at startup because key generation
// takes 30-120 seconds — unacceptable for a blockchain node startup.
func checkTFHEStatus(enabled bool) cryptoStatus {
	if enabled {
		return cryptoStatus{
			name:    "TFHE (tfhe-rs / FheUint32)",
			ok:      true,
			detail:  "subsystem ENABLED — erasure coding: 10+6=16 shards, replication=3×",
			elapsed: 0,
		}
	}
	return cryptoStatus{
		name:    "TFHE (tfhe-rs / FheUint32)",
		ok:      true, // not a failure — it's intentionally disabled
		detail:  "subsystem DISABLED (start with --enable-tfhe to activate)",
		elapsed: 0,
	}
}

// ── Banner printer ────────────────────────────────────────────────────────────

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
	logger.Info("  ⚛  ZYTHERION BLOCKCHAIN AND CRYPTOCURRENCY v0.3  ⚛")
	logger.Info("  ⚛  Founder: Rayhan Aziel Abbrar                    ⚛")
	logger.Info("  ⚛  CRYPTOGRAPHIC SUBSYSTEM STARTUP REPORT          ⚛")
	logger.Info(sep)

	for _, r := range results {
		status := "✅ OK  "
		if !r.ok {
			status = "❌ FAIL"
		}
		elapsed := ""
		if r.elapsed > 0 {
			elapsed = r.elapsed.Round(time.Millisecond).String()
		}
		logger.Info(fmt.Sprintf("  [%s] %s", status, r.name),
			"detail", r.detail,
			"elapsed", elapsed,
		)
	}

	logger.Info(sep)
	if allOK {
		logger.Info("  ✅ ALL CRYPTO SUBSYSTEMS OPERATIONAL — node is READY")
		logger.Info("     Signature algorithm: Dilithium5 (ML-DSA-87, NIST Cat-5, ~256-bit PQ)")
		logger.Info("     Block hashing:        LWR (Ring-LWR / SHAKE-256) ACTIVE")
		logger.Info("     PoVL sequential VDF:  ACTIVE (delay_steps=10)")
		logger.Info("     ZK-SNARK (Groth16):   REMOVED in v0.3")
		logger.Info("     TFHE homomorphic:     see TFHE status above")
	} else {
		logger.Error("  ❌ ONE OR MORE CRYPTO SUBSYSTEMS FAILED — CHECK LOGS ABOVE")
		logger.Error("     The node will continue but affected features may not work.")
	}
	logger.Info(sep)
}
