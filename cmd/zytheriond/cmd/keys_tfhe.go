package cmd

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/spf13/cobra"
	tfhe "zytherion/x/privacy/tfhe"
)

// TFHEKeysCmd returns the parent command for all TFHE key and encryption utilities.
func TFHEKeysCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tfhe",
		Short: "TFHE utility commands (keygen, encrypt, decrypt) [v0.5.2]",
		Long:  `Generate TFHE keypairs, and encrypt/decrypt values locally on the client.`,
	}

	cmd.AddCommand(
		CmdTFHEKeygen(),
		CmdTFHEEncrypt(),
		CmdTFHEDecrypt(),
	)

	return cmd
}

// CmdTFHEKeygen generates a new ClientKey and ServerKey pair.
func CmdTFHEKeygen() *cobra.Command {
	return &cobra.Command{
		Use:   "keygen [output_dir]",
		Short: "Generate a new TFHE ClientKey and ServerKey pair",
		Long: `Generate a fresh TFHE keypair for homomorphic operations.
Keys are written to the output directory (defaults to ~/.zytherion/tfhe/).`,
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
				outDir = filepath.Join(home, ".zytherion", "tfhe")
			}

			if err := os.MkdirAll(outDir, 0700); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", outDir, err)
			}

			fmt.Println("Generating TFHE ClientKey and ServerKey...")
			fmt.Println("(This is a CPU-intensive operation and may take 10-60 seconds...)")

			kp, err := tfhe.GenerateKeys()
			if err != nil {
				return err
			}

			ckPath := filepath.Join(outDir, "client.key")
			skPath := filepath.Join(outDir, "server.key")

			if err := os.WriteFile(ckPath, kp.ClientKey, 0600); err != nil {
				return fmt.Errorf("failed to write client key: %w", err)
			}
			if err := os.WriteFile(skPath, kp.ServerKey, 0600); err != nil {
				return fmt.Errorf("failed to write server key: %w", err)
			}

			fmt.Printf("Keys generated and saved successfully!\n")
			fmt.Printf("  Client Key (secret): %s (%d bytes)\n", ckPath, len(kp.ClientKey))
			fmt.Printf("  Server Key (public): %s (%d bytes)\n", skPath, len(kp.ServerKey))
			return nil
		},
	}
}

// CmdTFHEEncrypt encrypts a uint32 value using a ClientKey.
func CmdTFHEEncrypt() *cobra.Command {
	return &cobra.Command{
		Use:   "encrypt <uint32_value> [client_key_path]",
		Short: "Encrypt a uint32 value into a base64 TFHE ciphertext",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			val64, err := strconv.ParseUint(args[0], 10, 32)
			if err != nil {
				return fmt.Errorf("invalid uint32 value: %w", err)
			}
			val := uint32(val64)

			ckPath := ""
			if len(args) > 1 {
				ckPath = args[1]
			} else {
				home, err := os.UserHomeDir()
				if err != nil {
					return err
				}
				ckPath = filepath.Join(home, ".zytherion", "tfhe", "client.key")
			}

			ck, err := os.ReadFile(ckPath)
			if err != nil {
				return fmt.Errorf("failed to read client key from %s: %w", ckPath, err)
			}

			ct, err := tfhe.EncryptUint32(ck, val)
			if err != nil {
				return fmt.Errorf("encryption failed: %w", err)
			}

			fmt.Println(base64.StdEncoding.EncodeToString(ct))
			return nil
		},
	}
}

// CmdTFHEDecrypt decrypts a base64 ciphertext using a ClientKey.
func CmdTFHEDecrypt() *cobra.Command {
	return &cobra.Command{
		Use:   "decrypt <base64_ciphertext> [client_key_path]",
		Short: "Decrypt a base64 TFHE ciphertext using a ClientKey",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ct, err := base64.StdEncoding.DecodeString(args[0])
			if err != nil {
				return fmt.Errorf("invalid base64 ciphertext: %w", err)
			}

			ckPath := ""
			if len(args) > 1 {
				ckPath = args[1]
			} else {
				home, err := os.UserHomeDir()
				if err != nil {
					return err
				}
				ckPath = filepath.Join(home, ".zytherion", "tfhe", "client.key")
			}

			ck, err := os.ReadFile(ckPath)
			if err != nil {
				return fmt.Errorf("failed to read client key from %s: %w", ckPath, err)
			}

			val, err := tfhe.DecryptUint32(ck, ct)
			if err != nil {
				return fmt.Errorf("decryption failed: %w", err)
			}

			fmt.Printf("Decrypted value: %d\n", val)
			return nil
		},
	}
}
