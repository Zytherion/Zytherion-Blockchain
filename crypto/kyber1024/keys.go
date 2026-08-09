// Package kyber1024 implements CRYSTALS-Kyber (ML-KEM-1024, NIST FIPS 203)
// post-quantum Key Encapsulation Mechanism for Zytherion v0.7.
//
// # Overview
//
// Kyber1024 is the NIST Level 5 KEM (~256-bit post-quantum security).
// It is used in Zytherion for:
//
//  1. P2P SecretConnection hybrid key exchange (Kyber1024 + X25519)
//  2. TFHE ServerKey encrypted distribution
//  3. User-to-user encrypted messaging (future)
//
// # Key Sizes (Kyber1024)
//
//   - Public key:  1568 bytes
//   - Private key: 3168 bytes
//   - Ciphertext:  1568 bytes (KEM encapsulation output)
//   - Shared secret: 32 bytes
//
// # Type URLs (used by interface registry and Any encoding)
//
//	/zytherion.crypto.kyber1024.PublicKey
//	/zytherion.crypto.kyber1024.PrivateKey
package kyber1024

import (
	"bytes"
	"fmt"

	proto "github.com/cosmos/gogoproto/proto"
)

const (
	// KeyType is the key algorithm name used in CLI and keyring.
	KeyType = "kyber1024"

	// PubKeyTypeURL is the full protobuf type URL for Kyber1024 public keys.
	PubKeyTypeURL = "/zytherion.crypto.kyber1024.PublicKey"

	// PrivKeyTypeURL is the full protobuf type URL for Kyber1024 private keys.
	PrivKeyTypeURL = "/zytherion.crypto.kyber1024.PrivateKey"

	// Key sizes from cloudflare/circl kyber1024 constants.
	PublicKeySize  = 1568 // kyber1024.PublicKeySize
	PrivateKeySize = 3168 // kyber1024.PrivateKeySize
	CiphertextSize = 1568 // kyber1024.CiphertextSize (KEM encapsulation)
	SharedKeySize  = 32   // shared secret length
)

// ── PublicKey ─────────────────────────────────────────────────────────────────

// PublicKey is a Kyber1024 (ML-KEM-1024) public key used for KEM encapsulation.
// Recipients publish this key so senders can encrypt shared secrets to them.
//
// Wire encoding: protobuf bytes field 1.
// TypeURL: /zytherion.crypto.kyber1024.PublicKey
type PublicKey struct {
	Key []byte `protobuf:"bytes,1,opt,name=key,proto3" json:"key,omitempty"`
}

var _ proto.Message = (*PublicKey)(nil)

func (pk *PublicKey) ProtoMessage()  {}
func (pk *PublicKey) Reset()         { *pk = PublicKey{} }
func (pk *PublicKey) Descriptor() ([]byte, []int) { return nil, nil }
func (pk *PublicKey) String() string {
	if len(pk.Key) < 4 {
		return "kyber1024.PublicKey{<empty>}"
	}
	return fmt.Sprintf("kyber1024.PublicKey{%X...}", pk.Key[:4])
}

func (pk *PublicKey) Marshal() ([]byte, error) {
	return marshalBytesField1(pk.Key), nil
}
func (pk *PublicKey) MarshalTo(dAtA []byte) (int, error) {
	b := marshalBytesField1(pk.Key)
	copy(dAtA, b)
	return len(b), nil
}
func (pk *PublicKey) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	b := marshalBytesField1(pk.Key)
	copy(dAtA[len(dAtA)-len(b):], b)
	return len(b), nil
}
func (pk *PublicKey) Unmarshal(dAtA []byte) error {
	key, err := unmarshalBytesField1(dAtA)
	if err != nil {
		return fmt.Errorf("kyber1024.PublicKey.Unmarshal: %w", err)
	}
	pk.Key = key
	return nil
}
func (pk *PublicKey) Size() int { return len(marshalBytesField1(pk.Key)) }

// Bytes returns the raw key bytes.
func (pk *PublicKey) Bytes() []byte { return pk.Key }

// Equals returns true if two public keys are identical.
func (pk *PublicKey) Equals(other *PublicKey) bool {
	return bytes.Equal(pk.Key, other.Key)
}

// ── PrivateKey ────────────────────────────────────────────────────────────────

// PrivateKey is a Kyber1024 private key used for KEM decapsulation.
// Must be kept secret — only the holder can recover the shared secret.
//
// Wire encoding: protobuf bytes field 1.
// TypeURL: /zytherion.crypto.kyber1024.PrivateKey
type PrivateKey struct {
	Key []byte `protobuf:"bytes,1,opt,name=key,proto3" json:"key,omitempty"`
}

var _ proto.Message = (*PrivateKey)(nil)

func (sk *PrivateKey) ProtoMessage()  {}
func (sk *PrivateKey) Reset()         { *sk = PrivateKey{} }
func (sk *PrivateKey) Descriptor() ([]byte, []int) { return nil, nil }
func (sk *PrivateKey) String() string { return "kyber1024.PrivateKey{<redacted>}" }

func (sk *PrivateKey) Marshal() ([]byte, error) {
	return marshalBytesField1(sk.Key), nil
}
func (sk *PrivateKey) MarshalTo(dAtA []byte) (int, error) {
	b := marshalBytesField1(sk.Key)
	copy(dAtA, b)
	return len(b), nil
}
func (sk *PrivateKey) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	b := marshalBytesField1(sk.Key)
	copy(dAtA[len(dAtA)-len(b):], b)
	return len(b), nil
}
func (sk *PrivateKey) Unmarshal(dAtA []byte) error {
	key, err := unmarshalBytesField1(dAtA)
	if err != nil {
		return fmt.Errorf("kyber1024.PrivateKey.Unmarshal: %w", err)
	}
	sk.Key = key
	return nil
}
func (sk *PrivateKey) Size() int { return len(marshalBytesField1(sk.Key)) }

// Bytes returns the raw private key bytes.
func (sk *PrivateKey) Bytes() []byte { return sk.Key }

// Public returns the corresponding Kyber1024 public key.
func (sk *PrivateKey) Public() *PublicKey {
	pub, err := PublicKeyFromPrivate(sk.Key)
	if err != nil {
		panic(fmt.Sprintf("kyber1024.PrivateKey.Public: %v", err))
	}
	return &PublicKey{Key: pub}
}

// ── Protobuf wire helpers (varint + bytes field 1) ────────────────────────────

// marshalBytesField1 encodes []byte as protobuf field 1, type LEN (wire type 2).
func marshalBytesField1(b []byte) []byte {
	if len(b) == 0 {
		return nil
	}
	// Field 1, wire type 2: tag = (1 << 3) | 2 = 0x0a
	tag := []byte{0x0a}
	length := encodeVarint(uint64(len(b)))
	result := make([]byte, 0, 1+len(length)+len(b))
	result = append(result, tag...)
	result = append(result, length...)
	result = append(result, b...)
	return result
}

// unmarshalBytesField1 decodes a protobuf field 1 LEN value.
func unmarshalBytesField1(dAtA []byte) ([]byte, error) {
	if len(dAtA) == 0 {
		return nil, nil
	}
	// Expect tag 0x0a (field 1, wire type 2)
	if dAtA[0] != 0x0a {
		return nil, fmt.Errorf("unexpected tag byte 0x%02x (expected 0x0a)", dAtA[0])
	}
	dAtA = dAtA[1:]
	length, n := decodeVarint(dAtA)
	if n <= 0 {
		return nil, fmt.Errorf("invalid varint for length")
	}
	dAtA = dAtA[n:]
	if uint64(len(dAtA)) < length {
		return nil, fmt.Errorf("buffer too short: need %d bytes, have %d", length, len(dAtA))
	}
	return dAtA[:length], nil
}

func encodeVarint(v uint64) []byte {
	var buf [10]byte
	n := 0
	for v >= 0x80 {
		buf[n] = byte(v) | 0x80
		v >>= 7
		n++
	}
	buf[n] = byte(v)
	return buf[:n+1]
}

func decodeVarint(b []byte) (uint64, int) {
	var x uint64
	var s uint
	for i, c := range b {
		if i == 10 {
			return 0, -1
		}
		if c < 0x80 {
			return x | uint64(c)<<s, i + 1
		}
		x |= uint64(c&0x7f) << s
		s += 7
	}
	return 0, 0
}
