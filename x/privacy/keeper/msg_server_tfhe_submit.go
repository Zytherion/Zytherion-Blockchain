// msg_server_tfhe_submit.go — TFHE ciphertext submission handler (v0.5.3).
//
// Transaction flow for tx/tfhe_submit:
//  1. Parse sender address.
//  2. Validate ciphertext size (must be within [MinCiphertextSize, MaxCiphertextSize]).
//  3. Quota check: each address may hold at most TFHEMaxActiveCommitments=1 commitment.
//  4. Compute commitment hash = SHA-256(ciphertext).
//  5. Split ciphertext into 16 Reed-Solomon shards (DataShards=12, Parity=4).
//  6. Build Merkle tree over shard hashes; compute MerkleRoot.
//  7. Distribute shards to random peers (ReplicationFactor=4 each).
//  8. Attach MerkleRoot to shard metadata; store on-chain.
//  9. Store commitment for sender.
// 10. Increment per-address quota counter.
// 11. Charge additional TFHE gas (1500 gas/KB).
// 12. Emit event.
package keeper

import (
	"context"
	"crypto/sha256"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"zytherion/x/privacy/tfhe"
	"zytherion/x/privacy/types"
)

// TFHEGasPerKB is the additional gas charged per KB of ciphertext data.
// Increased from 1000 to 1500 in v0.4 to account for Merkle tree computation.
const TFHEGasPerKB = 1500

// MinCiphertextSize is the minimum accepted ciphertext size in bytes.
const MinCiphertextSize = 1 * 1024 // 1 KB

// MaxCiphertextSize is the maximum accepted ciphertext size in bytes (32 KB).
const MaxCiphertextSize = tfhe.CiphertextMaxBytes

// TFHEMaxActiveCommitments is the maximum number of active TFHE commitments
// a single address may hold simultaneously.
// Each address may hold exactly 1 active commitment in v0.4.
const TFHEMaxActiveCommitments = 1

func (ms msgServer) TFHESubmit(
	goCtx context.Context,
	msg *types.MsgTFHESubmit,
) (*types.MsgTFHESubmitResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// ── 1. Parse sender address ────────────────────────────────────────────────
	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid sender — %s", types.ErrInvalidAddress, err)
	}

	// ── 2. Validate ciphertext ─────────────────────────────────────────────────
	if len(msg.Ciphertext) < MinCiphertextSize {
		return nil, fmt.Errorf("%w: ciphertext too small (%d B < %d B minimum)",
			types.ErrInvalidCiphertext, len(msg.Ciphertext), MinCiphertextSize)
	}
	if len(msg.Ciphertext) > MaxCiphertextSize {
		return nil, fmt.Errorf("%w: ciphertext too large (%d B > %d B maximum)",
			types.ErrInvalidCiphertext, len(msg.Ciphertext), MaxCiphertextSize)
	}

	// ── 4. Quota check ─────────────────────────────────────────────────────────
	// Each account may hold at most TFHEMaxActiveCommitments active TFHE
	// commitments. Prevents storage abuse and witholding attacks.
	if ms.GetTFHEQuota(ctx, senderAddr) >= TFHEMaxActiveCommitments {
		return nil, fmt.Errorf("%w: address %s already has %d active commitment(s)",
			types.ErrTFHEQuotaExceeded, msg.Sender, TFHEMaxActiveCommitments)
	}

	// ── 5. Compute commitment hash = SHA-256(ciphertext) ──────────────────────
	hashArr := sha256.Sum256(msg.Ciphertext)
	commitmentHash := hashArr[:]

	// ── 6. Split into erasure shards ───────────────────────────────────────────
	shards, err := tfhe.Split(msg.Ciphertext)
	if err != nil {
		return nil, fmt.Errorf("%w: erasure coding failed: %s",
			types.ErrShardOperationFailed, err)
	}

	// ── 7. Build Merkle tree over shard data ───────────────────────────────────
	// The Merkle root is stored on-chain so validators can verify individual
	// shard proofs without downloading all 16 shards.
	merkleTree, err := tfhe.BuildMerkleTree(shards)
	if err != nil {
		return nil, fmt.Errorf("%w: Merkle tree construction failed: %s",
			types.ErrShardOperationFailed, err)
	}
	merkleRoot := merkleTree.RootBytes()

	// ── 8. Distribute shards ───────────────────────────────────────────────────
	// Auto-discover active validator monikers from chain state for distributed sharding.
	peerAddrs := ms.GetValidatorMonikers(ctx)
	shardMeta, err := ms.ShardDistributor().DistributeShards(
		ctx.Context(),
		shards,
		commitmentHash,
		peerAddrs,
	)
	if err != nil {
		return nil, fmt.Errorf("%w: shard distribution failed: %s",
			types.ErrShardOperationFailed, err)
	}
	shardMeta.OriginalLen = len(msg.Ciphertext)

	// ── 9. Attach Merkle root + ProposerPubkey to metadata ────────────────────
	// ProposerPubkey is the Dilithium5 public key from the ShardDistributor.
	// It is nil when the node is running without a signing key (dev/test mode).
	// In production it is set via ShardDistributor.WithSigningKey() on startup.
	shardMeta.MerkleRoot = merkleRoot
	shardMeta.ProposerPubkey = ms.ShardDistributor().ProposerPubkey()

	// ── 10. Store shard metadata on-chain ─────────────────────────────────────
	if err := ms.SetTFHEMeta(ctx, shardMeta); err != nil {
		return nil, fmt.Errorf("failed to store shard metadata: %w", err)
	}

	// ── 11. Store commitment for sender ───────────────────────────────────────
	if err := ms.SetCommitment(ctx, senderAddr, commitmentHash); err != nil {
		return nil, fmt.Errorf("failed to set commitment: %w", err)
	}

	// ── 12. Increment per-address quota ───────────────────────────────────────
	ms.IncrTFHEQuota(ctx, senderAddr)

	// ── 13. Charge additional TFHE gas ────────────────────────────────────────
	// 1500 gas/KB covers erasure coding + Merkle tree construction overhead.
	ciphertextKB := uint64(len(msg.Ciphertext)+1023) / 1024
	extraGas := ciphertextKB * TFHEGasPerKB
	ctx.GasMeter().ConsumeGas(extraGas, "tfhe-ciphertext-storage")

	// ── 14. Emit event ─────────────────────────────────────────────────────────
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeTFHESubmit,
			sdk.NewAttribute(types.AttributeKeySender, msg.Sender),
			sdk.NewAttribute(types.AttributeKeyCommitmentHash, fmt.Sprintf("%x", commitmentHash)),
			sdk.NewAttribute("ciphertext_size_bytes", fmt.Sprintf("%d", len(msg.Ciphertext))),
			sdk.NewAttribute("shards_total", fmt.Sprintf("%d", tfhe.ErasureTotalShards)),
			sdk.NewAttribute("merkle_root", fmt.Sprintf("%x", merkleRoot)),
		),
	)

	ms.Logger(ctx).Info("TFHE ciphertext submitted",
		"sender", msg.Sender,
		"commitment", fmt.Sprintf("%x", commitmentHash[:4]),
		"ciphertext_bytes", len(msg.Ciphertext),
		"merkle_root", fmt.Sprintf("%x", merkleRoot[:4]),
		"extra_gas", extraGas,
	)

	return &types.MsgTFHESubmitResponse{
		CommitmentHash: commitmentHash,
		TotalShards:    uint32(tfhe.ErasureTotalShards),
	}, nil
}
