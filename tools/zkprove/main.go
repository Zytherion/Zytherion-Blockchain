// tools/zkprove/main.go — Zytherion Off-chain ZK Prover Tool
//
// Generates a Groth16 proof for a private transfer commitment.
// Run this off-chain BEFORE submitting a transaction to the chain.
//
// Usage:
//
//	go run ./tools/zkprove \
//	  --amount 1000000 \
//	  --blinding <32-byte-hex> \
//	  --pk keys/proving_key.bin \
//	  --out proof.json
//
//	# Or generate a random blinding factor automatically:
//	go run ./tools/zkprove --amount 1000000 --pk keys/proving_key.bin --out proof.json
//
// The output proof.json can then be used with the CLI:
//
//	zytheriond tx privacy zk-transfer <recipient> --proof proof.json --from alice
package main

import (
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"zytherion/x/privacy/zk"
)

func main() {
	amount := flag.Uint64("amount", 0, "plaintext transfer amount (uint64, required)")
	blindingHex := flag.String("blinding", "", "32-byte hex blinding factor (auto-generated if empty)")
	pkPath := flag.String("pk", "keys/proving_key.bin", "path to Groth16 proving key")
	outPath := flag.String("out", "proof.json", "output path for proof JSON")
	flag.Parse()

	fmt.Println("╔══════════════════════════════════════════════════════════╗")
	fmt.Println("║           Zytherion Off-chain ZK Prover                 ║")
	fmt.Println("╚══════════════════════════════════════════════════════════╝")

	if *amount == 0 {
		fmt.Fprintln(os.Stderr, "ERROR: --amount must be > 0")
		flag.Usage()
		os.Exit(1)
	}

	// ── 1. Load proving key ───────────────────────────────────────────────────
	fmt.Printf("Loading proving key from %s... ", *pkPath)
	pkBytes, err := os.ReadFile(*pkPath)
	if err != nil {
		fmt.Println("FAILED")
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("ok (%d bytes)\n", len(pkBytes))

	// ── 2. Resolve blinding factor ────────────────────────────────────────────
	var blinding []byte
	if *blindingHex == "" {
		fmt.Print("Generating random blinding factor... ")
		blinding, err = zk.GenerateBlinding()
		if err != nil {
			fmt.Println("FAILED")
			fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("ok  (save this to spend later!)\n")
		fmt.Printf("  Blinding: %s\n", hex.EncodeToString(blinding))
	} else {
		blinding, err = hex.DecodeString(*blindingHex)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: invalid blinding hex: %v\n", err)
			os.Exit(1)
		}
		if len(blinding) < 16 {
			fmt.Fprintln(os.Stderr, "ERROR: blinding factor must be at least 16 bytes (32 recommended)")
			os.Exit(1)
		}
	}

	// ── 3. Generate proof ─────────────────────────────────────────────────────
	fmt.Printf("\nGenerating Groth16 proof for amount=%d... ", *amount)
	start := time.Now()

	proofBytes, commitment, err := zk.GenerateProof(pkBytes, *amount, blinding)
	if err != nil {
		fmt.Println("FAILED")
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}

	elapsed := time.Since(start)
	fmt.Printf("done (%s)\n", elapsed.Round(time.Millisecond))

	// ── 4. Compute public inputs ──────────────────────────────────────────────
	zeroPad := make([]byte, zk.CommitmentSize) // CommitmentY = 0
	publicInputs, err := zk.EncodePublicInputs(commitment, zeroPad)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: encoding public inputs: %v\n", err)
		os.Exit(1)
	}

	// ── 5. Build proof JSON ───────────────────────────────────────────────────
	proofJSON := zk.ProofJSON{
		Proof:        proofBytes,
		PublicInputs: [][]byte{commitment, zeroPad},
		Commitment:   commitment,
	}

	outBytes, err := json.MarshalIndent(proofJSON, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: marshaling proof JSON: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(*outPath, outBytes, 0600); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: writing proof JSON: %v\n", err)
		os.Exit(1)
	}

	// ── 6. Summary ────────────────────────────────────────────────────────────
	fmt.Printf("\n✅ Proof written to: %s\n", *outPath)
	fmt.Printf("   Commitment:    %s\n", hex.EncodeToString(commitment))
	fmt.Printf("   Public inputs: %s\n", hex.EncodeToString(publicInputs))
	fmt.Printf("   Proof size:    %d bytes\n\n", len(proofBytes))
	fmt.Println("Submit with:")
	fmt.Printf("  zytheriond tx privacy zk-transfer <recipient> --proof %s --from <key>\n", *outPath)

	_ = zeroPad // suppress unused warning
}
