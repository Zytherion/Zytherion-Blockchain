package pqc

import (
	"bytes"
	"testing"
)

func TestGenerateLWRBlockHash_Determinism(t *testing.T) {
	input1 := []byte("hello zytherion lwr block 1")
	prevHash := make([]byte, 32) // all zeros

	h1, err := GenerateLWRBlockHash(input1, prevHash)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	h2, err := GenerateLWRBlockHash(input1, prevHash)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !bytes.Equal(h1, h2) {
		t.Errorf("LWR hash is not deterministic. h1 != h2")
	}

	if err := ValidateLWRHash(h1); err != nil {
		t.Errorf("ValidateLWRHash failed on valid hash: %v", err)
	}
}

func TestGenerateLWRBlockHash_Avalanche(t *testing.T) {
	input1 := []byte("zytherion test data")
	input2 := []byte("zytherion test data")
	input2[0] ^= 0x01 // flip 1 bit

	prevHash := make([]byte, 32)

	h1, _ := GenerateLWRBlockHash(input1, prevHash)
	h2, _ := GenerateLWRBlockHash(input2, prevHash)

	if bytes.Equal(h1, h2) {
		t.Errorf("LWR hash did not change after 1-bit flip")
	}

	// Calculate difference in output bytes (just for test assertion)
	diffBytes := 0
	for i := 0; i < LWRHashSize; i++ {
		if h1[i] != h2[i] {
			diffBytes++
		}
	}
	
	if diffBytes < 10 { // expect at least some avalanche
		t.Errorf("Weak avalanche: only %d bytes differ", diffBytes)
	}
}
