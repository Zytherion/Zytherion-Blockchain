package dilithium5

import (
	"io"
)

// deterministicReader implements io.Reader using a seeded byte stream.
// It is used to make Dilithium5 key generation reproducible from a fixed secret,
// enabling mnemonic-based recovery (same secret → same keypair always).
//
// Stream construction: the secret is repeated cyclically.
// For Dilithium5 key generation (mode5.GenerateKey), this provides sufficient
// deterministic entropy without requiring a CSPRNG.
type deterministicReader struct {
	seed []byte
	pos  int
}

// newDeterministicReader creates a cyclic reader from the given seed bytes.
// The seed must be at least 32 bytes.
func newDeterministicReader(seed []byte) io.Reader {
	buf := make([]byte, len(seed))
	copy(buf, seed)
	return &deterministicReader{seed: buf}
}

// Read fills p with bytes from the cyclic seed stream.
func (r *deterministicReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = r.seed[r.pos%len(r.seed)]
		r.pos++
	}
	return len(p), nil
}
