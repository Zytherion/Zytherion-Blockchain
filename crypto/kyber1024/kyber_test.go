package kyber1024_test

import (
	"bytes"
	"testing"

	kyber "zytherion/crypto/kyber1024"
)

func TestKyber1024_KeyGenAndKEM(t *testing.T) {
	// 1. Keygen
	pubBytes, privBytes, err := kyber.GenKeyPair()
	if err != nil {
		t.Fatalf("GenKeyPair failed: %v", err)
	}
	if len(pubBytes) != kyber.PublicKeySize {
		t.Errorf("expected pubKey size %d, got %d", kyber.PublicKeySize, len(pubBytes))
	}
	if len(privBytes) != kyber.PrivateKeySize {
		t.Errorf("expected privKey size %d, got %d", kyber.PrivateKeySize, len(privBytes))
	}

	// 2. Encapsulate
	ct, senderSS, err := kyber.Encapsulate(pubBytes)
	if err != nil {
		t.Fatalf("Encapsulate failed: %v", err)
	}
	if len(ct) != kyber.CiphertextSize {
		t.Errorf("expected ciphertext size %d, got %d", kyber.CiphertextSize, len(ct))
	}
	if len(senderSS) != kyber.SharedKeySize {
		t.Errorf("expected shared secret size %d, got %d", kyber.SharedKeySize, len(senderSS))
	}

	// 3. Decapsulate
	receiverSS, err := kyber.Decapsulate(privBytes, ct)
	if err != nil {
		t.Fatalf("Decapsulate failed: %v", err)
	}

	// 4. Verify shared secrets match
	if !bytes.Equal(senderSS, receiverSS) {
		t.Fatalf("Shared secrets do not match!\nSender:   %x\nReceiver: %x", senderSS, receiverSS)
	}
}

func TestKyber1024_EncryptDecryptFile(t *testing.T) {
	pubBytes, privBytes, err := kyber.GenKeyPair()
	if err != nil {
		t.Fatalf("GenKeyPair failed: %v", err)
	}

	originalData := []byte("Zytherion v0.7 — Quantum-Safe Message Payload with Kyber1024 + AES-256-GCM")

	encrypted, err := kyber.EncryptFile(pubBytes, originalData)
	if err != nil {
		t.Fatalf("EncryptFile failed: %v", err)
	}

	decrypted, err := kyber.DecryptFile(privBytes, encrypted)
	if err != nil {
		t.Fatalf("DecryptFile failed: %v", err)
	}

	if !bytes.Equal(originalData, decrypted) {
		t.Fatalf("Decrypted data does not match original!\nOriginal:  %s\nDecrypted: %s", originalData, decrypted)
	}
}

func TestKyber1024_MnemonicDerivation(t *testing.T) {
	mnemonic := "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon art"
	hdPath := "m/44'/118'/0'/0/0"

	pub1, priv1, err := kyber.DeriveFromMnemonic(mnemonic, "", hdPath)
	if err != nil {
		t.Fatalf("DeriveFromMnemonic failed: %v", err)
	}

	// Derive again with same mnemonic & path -> must be deterministic
	pub2, priv2, err := kyber.DeriveFromMnemonic(mnemonic, "", hdPath)
	if err != nil {
		t.Fatalf("DeriveFromMnemonic 2 failed: %v", err)
	}

	if !bytes.Equal(pub1, pub2) || !bytes.Equal(priv1, priv2) {
		t.Fatalf("Deterministic derivation failed: derived different keys from same mnemonic!")
	}
}
