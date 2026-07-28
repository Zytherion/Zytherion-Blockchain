// node_keys.go — Exported key management helper for the tfhe package.
//
// EnsureNodeKeys provides a process-level singleton for loading or generating
// the node's TFHE key pair. It is called by both:
//   - x/privacy/keeper.NewKeeper (to init the worker pool at startup)
//   - x/privacy/tfhe/cosmwasm.NewTFHEQueryPlugin (to serve CosmWasm queries)
//
// Keys are persisted to disk so they survive node restarts without costly
// re-generation (~10–60 seconds for FheUint32 parameter set).
//
// Key file locations (default):
//
//	~/.zytherion_tfhe_client.key  (permission 0600 — owner only)
//	~/.zytherion_tfhe_server.key  (permission 0600 — owner only)
//
// If nodeHome is provided, keys are stored alongside the node data instead:
//
//	<nodeHome>/tfhe_client.key
//	<nodeHome>/tfhe_server.key
package tfhe

import (
	"fmt"
	"os"
	"sync"
)

// ── Singleton cache ────────────────────────────────────────────────────────────

var (
	nodeKeysMu        sync.Mutex
	nodeClientKeyCache []byte
	nodeServerKeyCache []byte
)

const (
	defaultClientKeyFile = ".zytherion_tfhe_client.key"
	defaultServerKeyFile = ".zytherion_tfhe_server.key"
)

// EnsureNodeKeys loads or generates the node's TFHE key pair.
//
// Parameters:
//   - nodeHome: if non-empty, keys are stored in <nodeHome>/tfhe_{client,server}.key.
//     If empty, the user home directory (~/) is used.
//
// Returns (clientKey, serverKey, error).
// Subsequent calls return the cached keys without disk I/O.
//
// Thread-safe: uses a mutex to prevent concurrent key generation.
func EnsureNodeKeys(nodeHome string) (clientKey, serverKey []byte, err error) {
	nodeKeysMu.Lock()
	defer nodeKeysMu.Unlock()

	if len(nodeClientKeyCache) > 0 {
		return nodeClientKeyCache, nodeServerKeyCache, nil
	}

	ckPath, skPath := keyPaths(nodeHome)

	ck, errCK := os.ReadFile(ckPath)
	sk, errSK := os.ReadFile(skPath)

	if errCK == nil && errSK == nil && len(ck) > 0 && len(sk) > 0 {
		nodeClientKeyCache = ck
		nodeServerKeyCache = sk
		return ck, sk, nil
	}

	// Keys not found — generate a new pair.
	// This is slow (~10–60 seconds); it happens only once per node lifecycle.
	fmt.Println("[INFO] tfhe: generating new TFHE key pair (this takes 10–60 seconds)...")
	kp, genErr := GenerateKeys()
	if genErr != nil {
		return nil, nil, fmt.Errorf("tfhe: key generation failed: %w", genErr)
	}
	fmt.Println("[INFO] tfhe: TFHE key pair generated and cached.")

	// Persist to disk.
	if writeErr := os.WriteFile(ckPath, kp.ClientKey, 0600); writeErr != nil {
		fmt.Printf("[WARN] tfhe: could not persist client key to %s: %v\n", ckPath, writeErr)
	}
	if writeErr := os.WriteFile(skPath, kp.ServerKey, 0600); writeErr != nil {
		fmt.Printf("[WARN] tfhe: could not persist server key to %s: %v\n", skPath, writeErr)
	}

	nodeClientKeyCache = kp.ClientKey
	nodeServerKeyCache = kp.ServerKey
	return kp.ClientKey, kp.ServerKey, nil
}

// keyPaths returns the file paths for client and server keys.
func keyPaths(nodeHome string) (ckPath, skPath string) {
	if nodeHome != "" {
		return nodeHome + "/tfhe_client.key", nodeHome + "/tfhe_server.key"
	}
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return home + "/" + defaultClientKeyFile, home + "/" + defaultServerKeyFile
}
