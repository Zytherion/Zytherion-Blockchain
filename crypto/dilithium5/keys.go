// Package dilithium5 implements the Dilithium5 (ML-DSA-87, NIST Category 5)
// post-quantum cryptography key type for Zytherion blockchain accounts.
//
// # Overview
//
// This package provides proto-compatible PubKey and PrivKey types that integrate
// natively with Cosmos SDK v0.47's codec and interface registry without requiring
// protoc code generation. Integration is achieved by:
//
//  1. Using `protobuf` struct tags so gogoproto can marshal/unmarshal binary
//  2. Calling proto.RegisterType in init() so InterfaceRegistry.RegisterImplementations works
//  3. Implementing cryptotypes.PubKey/PrivKey interfaces for transparent SDK integration
//
// # Security
//
// Dilithium5 provides ~256-bit post-quantum security (NIST Category 5).
// Key sizes: pubkey=2592B, privkey=4864B, signature=4595B.
// Signing is deterministic (same message+key → same signature, required by Green-BFT).
//
// # Type URLs (used by interface registry and Any encoding)
//
//	/zytherion.crypto.dilithium5.PubKey
//	/zytherion.crypto.dilithium5.PrivKey
package dilithium5

import (
	"bytes"
	"fmt"

	proto "github.com/cosmos/gogoproto/proto"
	"google.golang.org/protobuf/encoding/protowire"
)

const (
	// KeyType is the key algorithm name used in the keyring and CLI.
	KeyType = "dilithium5"

	// PubKeyTypeURL is the full protobuf type URL for Dilithium5 public keys.
	PubKeyTypeURL = "/zytherion.crypto.dilithium5.PubKey"

	// PrivKeyTypeURL is the full protobuf type URL for Dilithium5 private keys.
	PrivKeyTypeURL = "/zytherion.crypto.dilithium5.PrivKey"

	// Expected key sizes (from circl mode5 constants).
	PubKeySize  = 2592 // mode5.PublicKeySize
	PrivKeySize = 4864 // mode5.PrivateKeySize
	SigSize     = 4595 // mode5.SignatureSize
)

// ── PubKey ───────────────────────────────────────────────────────────────────

// PubKey is a Dilithium5 (ML-DSA-87) public key.
// It implements cryptotypes.PubKey and proto.Message.
//
// The protobuf wire encoding is:
//
//	message PubKey { bytes key = 1; }
//
// TypeURL: /zytherion.crypto.dilithium5.PubKey
type PubKey struct {
	// Key is the raw 2592-byte Dilithium5 public key.
	Key []byte `protobuf:"bytes,1,opt,name=key,proto3" json:"key,omitempty"`
}

// Ensure PubKey implements proto.Message.
var _ proto.Message = (*PubKey)(nil)

func (pk *PubKey) ProtoMessage()  {}
func (pk *PubKey) Reset()         { *pk = PubKey{} }
func (pk *PubKey) String() string {
	if len(pk.Key) < 4 {
		return "dilithium5.PubKey{<empty>}"
	}
	return fmt.Sprintf("dilithium5.PubKey{%X...}", pk.Key[:4])
}

// Marshal implements gogoproto Marshaler — encodes as protobuf bytes field 1.
func (pk *PubKey) Marshal() ([]byte, error) {
	return marshalBytesField1(pk.Key), nil
}

// MarshalTo writes the encoded form into dAtA and returns bytes written.
func (pk *PubKey) MarshalTo(dAtA []byte) (int, error) {
	b := marshalBytesField1(pk.Key)
	copy(dAtA, b)
	return len(b), nil
}

// MarshalToSizedBuffer writes the encoded form backwards into dAtA.
func (pk *PubKey) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	b := marshalBytesField1(pk.Key)
	copy(dAtA[len(dAtA)-len(b):], b)
	return len(b), nil
}

// Unmarshal decodes dAtA as a protobuf-encoded PubKey.
func (pk *PubKey) Unmarshal(dAtA []byte) error {
	key, err := unmarshalBytesField1(dAtA)
	if err != nil {
		return fmt.Errorf("dilithium5.PubKey.Unmarshal: %w", err)
	}
	pk.Key = key
	return nil
}

// Size returns the serialized byte size.
func (pk *PubKey) Size() int { return sizeField1(len(pk.Key)) }

// Equal implements equality check for gogoproto.
func (pk *PubKey) Equal(other *PubKey) bool {
	return bytes.Equal(pk.Key, other.Key)
}

// ── PrivKey ──────────────────────────────────────────────────────────────────

// PrivKey is a Dilithium5 private key.
// It implements cryptotypes.PrivKey and proto.Message.
//
// The protobuf wire encoding is:
//
//	message PrivKey { bytes key = 1; }
//
// TypeURL: /zytherion.crypto.dilithium5.PrivKey
type PrivKey struct {
	// Key is the raw 4864-byte Dilithium5 private key.
	Key []byte `protobuf:"bytes,1,opt,name=key,proto3" json:"key,omitempty"`
}

var _ proto.Message = (*PrivKey)(nil)

func (sk *PrivKey) ProtoMessage()  {}
func (sk *PrivKey) Reset()         { *sk = PrivKey{} }
func (sk *PrivKey) String() string { return "dilithium5.PrivKey{***REDACTED***}" }

func (sk *PrivKey) Marshal() ([]byte, error)           { return marshalBytesField1(sk.Key), nil }
func (sk *PrivKey) MarshalTo(dAtA []byte) (int, error) {
	b := marshalBytesField1(sk.Key)
	copy(dAtA, b)
	return len(b), nil
}
func (sk *PrivKey) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	b := marshalBytesField1(sk.Key)
	copy(dAtA[len(dAtA)-len(b):], b)
	return len(b), nil
}
func (sk *PrivKey) Unmarshal(dAtA []byte) error {
	key, err := unmarshalBytesField1(dAtA)
	if err != nil {
		return fmt.Errorf("dilithium5.PrivKey.Unmarshal: %w", err)
	}
	sk.Key = key
	return nil
}
func (sk *PrivKey) Size() int { return sizeField1(len(sk.Key)) }

// ── Proto registration ────────────────────────────────────────────────────────

func init() {
	// Register Dilithium5 key types with the gogoproto v1 registry.
	// This enables gogoproto.MessageName() to return the correct type name,
	// which is required by InterfaceRegistry.RegisterImplementations to compute
	// the type URL: "/" + gogoproto.MessageName(impl).
	//
	// Without this, RegisterImplementations would panic with type URL "/".
	proto.RegisterType((*PubKey)(nil), "zytherion.crypto.dilithium5.PubKey")
	proto.RegisterType((*PrivKey)(nil), "zytherion.crypto.dilithium5.PrivKey")
}

// ── Wire encoding helpers ─────────────────────────────────────────────────────
//
// These implement field 1, type bytes (length-delimited) encoding.
// Protobuf tag: field_number=1, wire_type=2 → tag byte = (1<<3)|2 = 10 = 0x0a

// marshalBytesField1 encodes a []byte as protobuf field 1 (bytes type).
func marshalBytesField1(b []byte) []byte {
	if len(b) == 0 {
		return nil
	}
	tag := protowire.AppendTag(nil, 1, protowire.BytesType)
	return protowire.AppendBytes(tag, b)
}

// unmarshalBytesField1 decodes the first bytes field (field number 1) from dAtA.
func unmarshalBytesField1(dAtA []byte) ([]byte, error) {
	for len(dAtA) > 0 {
		num, typ, n := protowire.ConsumeTag(dAtA)
		if n < 0 {
			return nil, protowire.ParseError(n)
		}
		dAtA = dAtA[n:]

		if num == 1 && typ == protowire.BytesType {
			v, n := protowire.ConsumeBytes(dAtA)
			if n < 0 {
				return nil, protowire.ParseError(n)
			}
			result := make([]byte, len(v))
			copy(result, v)
			return result, nil
		}

		// Skip unknown fields
		n = protowire.ConsumeFieldValue(num, typ, dAtA)
		if n < 0 {
			return nil, protowire.ParseError(n)
		}
		dAtA = dAtA[n:]
	}
	return nil, nil
}

// sizeField1 returns the serialized size of a bytes field with field number 1.
func sizeField1(dataLen int) int {
	if dataLen == 0 {
		return 0
	}
	// 1 byte tag + varint(len) + data
	tagLen := protowire.SizeTag(1)
	return tagLen + int(protowire.SizeVarint(uint64(dataLen))) + dataLen
}
