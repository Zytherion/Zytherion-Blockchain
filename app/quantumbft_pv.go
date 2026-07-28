// quantumbft_pv.go — QuantumBFT PrivValidator injection for Zytherion v0.6.
//
// When a node starts (via ignite chain serve or zytheriond start), this file's
// LoadQuantumPrivValidator function checks if priv_validator_key.json (or quantum_validator_key.json)
// contains a QuantumBFT Dilithium5 key. If found, it loads the QuantumFilePV and returns it.
package app

import (
	"os"
	"path/filepath"

	cmtcrypto "github.com/cometbft/cometbft/crypto"
	"github.com/cometbft/cometbft/libs/log"
	cmttypes "github.com/cometbft/cometbft/types"

	"zytherion/quantumbft"
)

// LoadQuantumPrivValidator checks for a QuantumBFT key file in the node's home directory.
// Returns (pv, true) if QuantumBFT key is present and loaded.
// Returns (nil, false) if no QuantumBFT key is found (standby/fallback mode).
func LoadQuantumPrivValidator(home string, logger log.Logger) (cmttypes.PrivValidator, bool) {
	if home == "" {
		home = DefaultNodeHome
	}

	keyPath := filepath.Join(home, "config", quantumbft.DefaultKeyFileName)
	legacyPath := filepath.Join(home, "config", quantumbft.LegacyKeyFileName)

	targetPath := ""
	if quantumbft.IsQuantumKeyFile(keyPath) {
		targetPath = keyPath
	} else if quantumbft.IsQuantumKeyFile(legacyPath) {
		targetPath = legacyPath
	} else {
		// Auto-generate QuantumBFT Dilithium5 key in quantum_validator_key.json.
		// This keeps priv_validator_key.json compatible with gentx/ignite init while
		// CometBFT consensus engine signs all proposals & votes with Dilithium5!
		targetPath = legacyPath
		if logger != nil {
			logger.Info("QuantumBFT: auto-generating Dilithium5 consensus key...", "path", targetPath)
		}
		if _, err := quantumbft.GenerateValidatorKey(targetPath); err != nil {
			if logger != nil {
				logger.Error("QuantumBFT: auto-generation failed", "error", err)
			}
			return nil, false
		}
	}

	// Load the QuantumBFT FilePV.
	pv, err := quantumbft.LoadValidatorKey(targetPath)
	if err != nil {
		logger.Error("QuantumBFT: failed to load validator key", "path", targetPath, "error", err)
		return nil, false
	}

	pubKey, err := pv.GetPubKey()
	if err != nil {
		logger.Error("QuantumBFT: failed to get pubkey from QuantumFilePV", "error", err)
		return nil, false
	}

	logger.Info("QuantumBFT: Dilithium5 validator key loaded",
		"address", pubKey.Address(),
		"key_file", pv.KeyFilePath(),
		"algorithm", "Dilithium5 (ML-DSA-87, NIST Category 5)",
	)

	return pv, true
}

// PatchGenesisWithQuantumPubKey updates genesis.json validator set to use Dilithium5 pubkey.
// Called explicitly by CLI commands (e.g. quantumbft init), NOT during app.New().
func PatchGenesisWithQuantumPubKey(genesisPath string, pubKey cmtcrypto.PubKey, logger log.Logger) error {
	genDoc, err := cmttypes.GenesisDocFromFile(genesisPath)
	if err != nil {
		return nil // genesis file doesn't exist yet
	}

	modified := false
	for i := range genDoc.Validators {
		if genDoc.Validators[i].PubKey == nil || genDoc.Validators[i].PubKey.Type() != quantumbft.KeyType {
			genDoc.Validators[i].PubKey = pubKey
			genDoc.Validators[i].Address = pubKey.Address()
			modified = true
		}
	}

	if modified {
		if err := genDoc.SaveAs(genesisPath); err != nil {
			return err
		}
		if logger != nil {
			logger.Info("QuantumBFT: updated genesis.json validator set to tendermint/PubKeyDilithium5",
				"address", pubKey.Address(),
				"pubkey_type", quantumbft.PubKeyName,
			)
		}
	}

	return nil
}
