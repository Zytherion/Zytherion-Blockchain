// block.go — Block and SignedBlock data structures for Zytherion.
//
// These types represent the core blockchain data unit and its validator-signed
// form. Hashing uses GenerateBlockHash (SHA3-256 + domain separation) and
// signing uses Dilithium3 (see signature.go).
package pqc

import "fmt"

// Block is the canonical unit of data commitment in the Zytherion chain.
//
// The Hash field is computed from all other fields via GenerateBlockHash and
// must be recalculated whenever any field changes.
type Block struct {
	// Height is the block's position in the chain (1-indexed, 0 = genesis).
	Height int64

	// PrevHash is the GenerateBlockHash output of the previous block.
	// At height 1, this is a 32-byte zero slice referencing the genesis state.
	PrevHash []byte

	// AppHash is the multistore Merkle root committed by the previous block
	// (identical to the Cosmos SDK AppHash).
	AppHash []byte

	// Transactions holds the raw transaction bytes in proposal order.
	Transactions [][]byte

	// Hash is the SHA3-256 block hash computed by GenerateBlockHash.
	// It commits this block's content to the chain and is stored in the
	// next block's PrevHash field.
	Hash []byte
}

// SignedBlock pairs a Block with the validator's Dilithium3 signature over
// its Hash field. Consensus nodes verify this signature before accepting a
// block proposal.
type SignedBlock struct {
	Block

	// ValidatorSignature is the 3293-byte Dilithium3 signature of Block.Hash,
	// produced by the proposer's private key.
	ValidatorSignature []byte
}

// NewBlock constructs a Block and immediately computes its hash.
//
// prevHash should be HashSize (32) bytes; shorter slices are zero-padded.
// Pass a nil or zero slice for prevHash at height 1 (first block after genesis).
func NewBlock(height int64, prevHash, appHash []byte, txs [][]byte) Block {
	b := Block{
		Height:       height,
		PrevHash:     zeroPad32(prevHash),
		AppHash:      appHash,
		Transactions: txs,
	}
	b.Hash = b.computeHash()
	return b
}

// computeHash recomputes and returns the SHA3-256 block hash.
// Called internally by NewBlock; call again only after mutating fields.
func (b *Block) computeHash() []byte {
	return GenerateBlockHash(BlockHashInput{
		Height:       b.Height,
		PrevHash:     b.PrevHash,
		AppHash:      b.AppHash,
		Transactions: b.Transactions,
	})
}

// SignBlock creates a SignedBlock by signing Block.Hash with the given
// Dilithium3 private key bytes.
//
// Returns an error if the private key is malformed or if Sign fails.
func SignBlock(b Block, privKeyBytes []byte) (SignedBlock, error) {
	if len(b.Hash) == 0 {
		return SignedBlock{}, fmt.Errorf("sign block: block hash is empty; call NewBlock first")
	}
	sig, err := Sign(b.Hash, privKeyBytes)
	if err != nil {
		return SignedBlock{}, fmt.Errorf("sign block (height=%d): %w", b.Height, err)
	}
	return SignedBlock{
		Block:              b,
		ValidatorSignature: sig,
	}, nil
}

// VerifySignedBlock returns true if sb.ValidatorSignature is a valid
// Dilithium3 signature over sb.Block.Hash using pubKeyBytes.
//
// Callers should also independently recompute sb.Block.Hash from the block
// fields and compare, to ensure the hash itself has not been tampered with.
func VerifySignedBlock(sb SignedBlock, pubKeyBytes []byte) bool {
	if len(sb.Hash) == 0 || len(sb.ValidatorSignature) == 0 {
		return false
	}
	return Verify(sb.Hash, sb.ValidatorSignature, pubKeyBytes)
}
