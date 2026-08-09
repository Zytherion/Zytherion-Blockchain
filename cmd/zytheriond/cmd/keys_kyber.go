package cmd

import (
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	kyber "zytherion/crypto/kyber1024"
)

// KyberCmd returns the parent command for all Kyber1024 KEM utilities.
func KyberCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kyber",
		Short: "Kyber1024 (ML-KEM) post-quantum key encapsulation commands [v0.7]",
		Long: `CRYSTALS-Kyber1024 (ML-KEM-1024, NIST FIPS 203) utilities.

Kyber1024 is a post-quantum Key Encapsulation Mechanism (KEM).
It replaces classical ECDH/X25519 for quantum-safe key exchange.

Security level: NIST Level 5 (~256-bit post-quantum security)
Key sizes:
  Public key:   1568 bytes
  Private key:  3168 bytes
  Ciphertext:   1568 bytes (KEM output)
  Shared secret: 32 bytes (used as AES-256-GCM key)`,
	}

	cmd.AddCommand(
		CmdKyberKeygen(),
		CmdKyberEncapsulate(),
		CmdKyberDecapsulate(),
		CmdKyberEncryptFile(),
		CmdKyberDecryptFile(),
	)

	return cmd
}

// CmdKyberKeygen generates a new Kyber1024 keypair.
func CmdKyberKeygen() *cobra.Command {
	return &cobra.Command{
		Use:   "keygen [output_dir]",
		Short: "Generate a new Kyber1024 (ML-KEM-1024) keypair",
		Long: `Generate a fresh Kyber1024 keypair for post-quantum key encapsulation.

Keys are saved to output_dir (default: ~/.zytherion/kyber/).
  kyber.pub  — public key (1568 bytes, safe to share)
  kyber.priv — private key (3168 bytes, keep secret!)`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			outDir := ""
			if len(args) > 0 {
				outDir = args[0]
			} else {
				home, err := os.UserHomeDir()
				if err != nil {
					return err
				}
				outDir = filepath.Join(home, ".zytherion", "kyber")
			}

			if err := os.MkdirAll(outDir, 0700); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", outDir, err)
			}

			fmt.Println("Generating Kyber1024 keypair...")
			pubBytes, privBytes, err := kyber.GenKeyPair()
			if err != nil {
				return err
			}

			pubPath := filepath.Join(outDir, "kyber.pub")
			privPath := filepath.Join(outDir, "kyber.priv")

			if err := os.WriteFile(pubPath, pubBytes, 0644); err != nil {
				return fmt.Errorf("failed to write public key: %w", err)
			}
			if err := os.WriteFile(privPath, privBytes, 0600); err != nil {
				return fmt.Errorf("failed to write private key: %w", err)
			}

			fmt.Printf("✅ Kyber1024 keypair generated!\n")
			fmt.Printf("  Public key  (share):  %s (%d bytes)\n", pubPath, len(pubBytes))
			fmt.Printf("  Private key (secret): %s (%d bytes)\n", privPath, len(privBytes))
			fmt.Printf("  Public key hex: %s\n", hex.EncodeToString(pubBytes))
			return nil
		},
	}
}

// CmdKyberEncapsulate encapsulates a shared secret to a Kyber1024 public key.
func CmdKyberEncapsulate() *cobra.Command {
	return &cobra.Command{
		Use:   "encapsulate <pubkey-hex-or-file>",
		Short: "Encapsulate a 32-byte shared secret to a Kyber1024 public key",
		Long: `Perform KEM encapsulation: generate a random shared secret and encrypt
it to the recipient's Kyber1024 public key.

Output:
  ciphertext (hex) — send this to the recipient
  shared_secret (hex) — use this as AES-256 key for payload encryption

The recipient decapsulates the ciphertext with their private key to get
the same shared secret.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pubBytes, err := loadKeyArg(args[0])
			if err != nil {
				return err
			}

			ct, ss, err := kyber.Encapsulate(pubBytes)
			if err != nil {
				return err
			}

			fmt.Printf("KEM Encapsulation Result:\n")
			fmt.Printf("  ciphertext (1568 bytes): %s\n", hex.EncodeToString(ct))
			fmt.Printf("  shared_secret (32 bytes): %s\n", hex.EncodeToString(ss))
			fmt.Printf("\n⚠️  Keep shared_secret private! Send only the ciphertext to recipient.\n")
			return nil
		},
	}
}

// CmdKyberDecapsulate decapsulates a shared secret from a Kyber1024 ciphertext.
func CmdKyberDecapsulate() *cobra.Command {
	return &cobra.Command{
		Use:   "decapsulate <privkey-file> <ciphertext-hex>",
		Short: "Decapsulate a shared secret using your Kyber1024 private key",
		Long: `Recover the 32-byte shared secret from a KEM ciphertext using your
Kyber1024 private key.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			privBytes, err := os.ReadFile(args[0])
			if err != nil {
				return fmt.Errorf("failed to read private key from %s: %w", args[0], err)
			}

			ct, err := hex.DecodeString(args[1])
			if err != nil {
				return fmt.Errorf("invalid ciphertext hex: %w", err)
			}

			ss, err := kyber.Decapsulate(privBytes, ct)
			if err != nil {
				return err
			}

			fmt.Printf("Decapsulated shared_secret (32 bytes): %s\n", hex.EncodeToString(ss))
			return nil
		},
	}
}

// CmdKyberEncryptFile encrypts a file using Kyber1024 KEM + AES-256-GCM.
func CmdKyberEncryptFile() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "encrypt-file <pubkey-hex-or-file> <input-file>",
		Short: "Encrypt a file using Kyber1024 KEM + AES-256-GCM",
		Long: `Encrypt any file (e.g. TFHE server.key) to a recipient's Kyber1024 public key.
Only the holder of the matching private key can decrypt.

Example (encrypt TFHE server.key for on-chain distribution):
  zytheriond keys kyber encrypt-file ~/.zytherion/kyber/kyber.pub ~/.zytherion/tfhe/server.key`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			outFile, _ := cmd.Flags().GetString("out")

			pubBytes, err := loadKeyArg(args[0])
			if err != nil {
				return err
			}

			plaintext, err := os.ReadFile(args[1])
			if err != nil {
				return fmt.Errorf("failed to read input file %s: %w", args[1], err)
			}

			fmt.Printf("Encrypting %s (%d bytes) with Kyber1024...\n", args[1], len(plaintext))

			encrypted, err := kyber.EncryptFile(pubBytes, plaintext)
			if err != nil {
				return err
			}

			if outFile == "" {
				outFile = args[1] + ".kyber"
			}

			if err := os.WriteFile(outFile, encrypted, 0644); err != nil {
				return fmt.Errorf("failed to write encrypted file: %w", err)
			}

			fmt.Printf("✅ Encrypted → %s (%d bytes)\n", outFile, len(encrypted))
			fmt.Printf("   Original: %d bytes → Encrypted: %d bytes\n", len(plaintext), len(encrypted))
			return nil
		},
	}
	cmd.Flags().String("out", "", "Output file path (default: input.kyber)")
	return cmd
}

// CmdKyberDecryptFile decrypts a Kyber1024-encrypted file.
func CmdKyberDecryptFile() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "decrypt-file <privkey-file> <encrypted-file>",
		Short: "Decrypt a file encrypted with Kyber1024 KEM + AES-256-GCM",
		Long: `Decrypt a file that was encrypted with 'zytheriond keys kyber encrypt-file'.

Example (decrypt TFHE server.key):
  zytheriond keys kyber decrypt-file ~/.zytherion/kyber/kyber.priv server.key.kyber`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			outFile, _ := cmd.Flags().GetString("out")

			privBytes, err := os.ReadFile(args[0])
			if err != nil {
				return fmt.Errorf("failed to read private key from %s: %w", args[0], err)
			}

			blob, err := os.ReadFile(args[1])
			if err != nil {
				return fmt.Errorf("failed to read encrypted file %s: %w", args[1], err)
			}

			fmt.Printf("Decrypting %s (%d bytes) with Kyber1024...\n", args[1], len(blob))

			plaintext, err := kyber.DecryptFile(privBytes, blob)
			if err != nil {
				return err
			}

			if outFile == "" {
				// Strip .kyber extension if present
				outFile = args[1]
				if len(outFile) > 6 && outFile[len(outFile)-6:] == ".kyber" {
					outFile = outFile[:len(outFile)-6]
				} else {
					outFile = outFile + ".dec"
				}
			}

			if err := os.WriteFile(outFile, plaintext, 0600); err != nil {
				return fmt.Errorf("failed to write decrypted file: %w", err)
			}

			fmt.Printf("✅ Decrypted → %s (%d bytes)\n", outFile, len(plaintext))
			return nil
		},
	}
	cmd.Flags().String("out", "", "Output file path (default: removes .kyber suffix)")
	return cmd
}

// loadKeyArg loads key bytes from either a hex string or a file path.
func loadKeyArg(arg string) ([]byte, error) {
	// Try as file first
	if _, err := os.Stat(arg); err == nil {
		return os.ReadFile(arg)
	}
	// Try as hex string
	b, err := hex.DecodeString(arg)
	if err != nil {
		return nil, fmt.Errorf("argument %q is neither a valid file path nor valid hex: %w", arg, err)
	}
	return b, nil
}
