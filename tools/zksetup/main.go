// tools/zksetup/main.go — Zytherion ZK Trusted Setup Tool
//
// Run ONCE to generate the Groth16 proving key (PK) and verifying key (VK)
// for the TransferCircuit. Commit verifying_key.bin to the repository.
// Keep proving_key.bin off the chain — it is only needed by provers.
//
// Usage:
//
//	go run ./tools/zksetup --out ./keys
//	make zksetup
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"zytherion/x/privacy/zk"
)

func main() {
	outDir := flag.String("out", "keys", "output directory for proving_key.bin and verifying_key.bin")
	flag.Parse()

	fmt.Println("╔══════════════════════════════════════════════════════════╗")
	fmt.Println("║          Zytherion ZK Trusted Setup (Groth16)           ║")
	fmt.Println("╚══════════════════════════════════════════════════════════╝")
	fmt.Printf("Output directory: %s\n\n", *outDir)

	// Ensure output directory exists.
	if err := os.MkdirAll(*outDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: cannot create output directory: %v\n", err)
		os.Exit(1)
	}

	fmt.Print("⏳ Compiling TransferCircuit and running Groth16 setup... ")
	start := time.Now()

	pkBytes, vkBytes, err := zk.GenerateKeys()
	if err != nil {
		fmt.Println("FAILED")
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}

	elapsed := time.Since(start)
	fmt.Printf("done (%s)\n\n", elapsed.Round(time.Millisecond))

	// Write proving key.
	pkPath := filepath.Join(*outDir, "proving_key.bin")
	if err := os.WriteFile(pkPath, pkBytes, 0600); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: writing proving key: %v\n", err)
		os.Exit(1)
	}

	// Write verifying key.
	vkPath := filepath.Join(*outDir, "verifying_key.bin")
	if err := os.WriteFile(vkPath, vkBytes, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: writing verifying key: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Proving key  → %s  (%d bytes)\n", pkPath, len(pkBytes))
	fmt.Printf("✅ Verifying key → %s  (%d bytes)\n\n", vkPath, len(vkBytes))

	fmt.Println("Next steps:")
	fmt.Println("  1. Commit verifying_key.bin to the repository:")
	fmt.Println("       git add keys/verifying_key.bin && git commit -m 'zk: add trusted setup VK'")
	fmt.Println("  2. Keep proving_key.bin secure — share only with authorized provers.")
	fmt.Println("  3. Start the node — it will load keys/verifying_key.bin at startup.")
}
