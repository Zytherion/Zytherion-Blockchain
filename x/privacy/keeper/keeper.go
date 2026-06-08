package keeper

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/cometbft/cometbft/libs/log"
	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	"zytherion/x/privacy/tfhe"
	"zytherion/x/privacy/types"
)

type (
	Keeper struct {
		cdc        codec.BinaryCodec
		storeKey   storetypes.StoreKey
		memKey     storetypes.StoreKey
		paramstore paramtypes.Subspace
		bankKeeper types.BankKeeper

		// tfheEnabled controls whether TFHE operations are accepted.
		// Governed by the --enable-tfhe startup flag (default: false).
		tfheEnabled bool

		// shardStore manages local disk storage of TFHE ciphertext shards.
		// Non-nil only when tfheEnabled == true.
		shardStore *tfhe.ShardStore

		// shardDistributor handles P2P shard distribution and reconstruction.
		// Non-nil only when tfheEnabled == true.
		shardDistributor *tfhe.ShardDistributor
	}
)

// NewKeeper creates a new Privacy module Keeper.
//
// Parameters:
//   - cdc, storeKey, memKey, ps: standard Cosmos SDK keeper params.
//   - bankKeeper: for coin transfers in deposit flows.
//   - enableTFHE: whether TFHE operations are enabled (--enable-tfhe flag).
//   - nodeHome: home directory for the node (~/.zytherion).
//   - nodeID: peer ID of this node for shard ownership metadata.
func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey,
	memKey storetypes.StoreKey,
	ps paramtypes.Subspace,
	bankKeeper types.BankKeeper,
	enableTFHE bool,
	nodeHome string,
	nodeID string,
) *Keeper {
	if !ps.HasKeyTable() {
		ps = ps.WithKeyTable(types.ParamKeyTable())
	}

	k := &Keeper{
		cdc:         cdc,
		storeKey:    storeKey,
		memKey:      memKey,
		paramstore:  ps,
		bankKeeper:  bankKeeper,
		tfheEnabled: enableTFHE,
	}

	if enableTFHE {
		shardDir := filepath.Join(nodeHome, "tfhe_shards")
		store, err := tfhe.NewShardStore(shardDir)
		if err != nil {
			// Non-fatal: disable TFHE gracefully if shard store cannot be initialised.
			k.tfheEnabled = false
		} else {
			k.shardStore = store
			k.shardDistributor = tfhe.NewShardDistributor(store, nodeID)
		}
	}

	return k
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

func (k Keeper) StoreKey() storetypes.StoreKey {
	return k.storeKey
}

// IsTFHEEnabled returns true if the TFHE subsystem is active.
func (k Keeper) IsTFHEEnabled() bool {
	return k.tfheEnabled
}

// ShardStore returns the local disk shard store. May be nil if TFHE is disabled.
func (k Keeper) ShardStore() *tfhe.ShardStore {
	return k.shardStore
}

// ShardDistributor returns the P2P shard distributor. May be nil if TFHE is disabled.
func (k Keeper) ShardDistributor() *tfhe.ShardDistributor {
	return k.shardDistributor
}

// ── TFHE Metadata store ────────────────────────────────────────────────────────

// SetTFHEMeta stores shard metadata for a commitment in the KV store.
func (k Keeper) SetTFHEMeta(ctx sdk.Context, meta *tfhe.TFHEShardMeta) error {
	if meta == nil || len(meta.CommitmentHash) == 0 {
		return fmt.Errorf("SetTFHEMeta: meta is nil or has empty commitment hash")
	}
	store := ctx.KVStore(k.storeKey)
	encoded, err := marshalTFHEMeta(meta)
	if err != nil {
		return fmt.Errorf("SetTFHEMeta: marshal failed: %w", err)
	}
	store.Set(types.TFHEMetaKey(meta.CommitmentHash), encoded)
	return nil
}

// GetTFHEMeta retrieves shard metadata for a commitment from the KV store.
// Returns (nil, false) if no metadata is stored for the commitment.
func (k Keeper) GetTFHEMeta(ctx sdk.Context, commitmentHash []byte) (*tfhe.TFHEShardMeta, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.TFHEMetaKey(commitmentHash))
	if bz == nil {
		return nil, false
	}
	meta, err := unmarshalTFHEMeta(bz)
	if err != nil {
		return nil, false
	}
	return meta, true
}

// ── TFHE Result store ──────────────────────────────────────────────────────────

// SetTFHEResult stores the serialised result ciphertext of a homomorphic operation.
func (k Keeper) SetTFHEResult(ctx sdk.Context, resultHash []byte, resultCiphertext []byte) {
	store := ctx.KVStore(k.storeKey)
	store.Set(types.TFHEResultKey(resultHash), resultCiphertext)
}

// GetTFHEResult retrieves a stored homomorphic operation result.
func (k Keeper) GetTFHEResult(ctx sdk.Context, resultHash []byte) ([]byte, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.TFHEResultKey(resultHash))
	if bz == nil {
		return nil, false
	}
	return bz, true
}

// ── Commitment store ────────────────────────────────────────────────────────────

// SetCommitment stores a 32-byte commitment for an account.
func (k Keeper) SetCommitment(ctx sdk.Context, addr sdk.AccAddress, commitment []byte) error {
	if len(commitment) != 32 {
		return fmt.Errorf("SetCommitment: commitment must be 32 bytes, got %d", len(commitment))
	}
	store := ctx.KVStore(k.storeKey)
	store.Set(types.CommitmentKey(addr), commitment)
	k.Logger(ctx).Debug("commitment updated",
		"address", addr.String(),
		"commitment_prefix", fmt.Sprintf("%x", commitment[:4]),
	)
	return nil
}

// GetCommitment retrieves the 32-byte commitment for an account.
func (k Keeper) GetCommitment(ctx sdk.Context, addr sdk.AccAddress) ([]byte, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.CommitmentKey(addr))
	if bz == nil {
		return nil, false
	}
	return bz, true
}

// HasCommitment reports whether an account has a registered commitment.
func (k Keeper) HasCommitment(ctx sdk.Context, addr sdk.AccAddress) bool {
	return ctx.KVStore(k.storeKey).Has(types.CommitmentKey(addr))
}

// ── Encoding helpers ────────────────────────────────────────────────────────────

// shardMetaJSON is the JSON-serialisable form of TFHEShardMeta.
type shardMetaJSON struct {
	CommitmentHash string              `json:"commitment_hash"`
	OriginalLen    int                 `json:"original_len"`
	ShardNodeMap   map[string][]string `json:"shard_node_map"`
	MerkleRoot     string              `json:"merkle_root"`
	ProposerPubkey string              `json:"proposer_pubkey"`
}

func marshalTFHEMeta(meta *tfhe.TFHEShardMeta) ([]byte, error) {
	m := shardMetaJSON{
		CommitmentHash: fmt.Sprintf("%x", meta.CommitmentHash),
		OriginalLen:    meta.OriginalLen,
		ShardNodeMap:   make(map[string][]string),
		MerkleRoot:     hex.EncodeToString(meta.MerkleRoot),
		ProposerPubkey: hex.EncodeToString(meta.ProposerPubkey),
	}
	for idx, nodes := range meta.ShardNodeMap {
		m.ShardNodeMap[strconv.Itoa(idx)] = nodes
	}
	return json.Marshal(m)
}

func unmarshalTFHEMeta(bz []byte) (*tfhe.TFHEShardMeta, error) {
	var m shardMetaJSON
	if err := json.Unmarshal(bz, &m); err != nil {
		return nil, err
	}
	meta := &tfhe.TFHEShardMeta{
		OriginalLen:  m.OriginalLen,
		ShardNodeMap: make(map[int][]string),
	}
	// Decode commitment hash from hex string.
	ch := make([]byte, len(m.CommitmentHash)/2)
	fmt.Sscanf(m.CommitmentHash, "%x", &ch)
	meta.CommitmentHash = ch

	// Decode Merkle root and proposer pubkey.
	if mr, err := hex.DecodeString(m.MerkleRoot); err == nil {
		meta.MerkleRoot = mr
	}
	if pk, err := hex.DecodeString(m.ProposerPubkey); err == nil {
		meta.ProposerPubkey = pk
	}

	for idxStr, nodes := range m.ShardNodeMap {
		idx, err := strconv.Atoi(idxStr)
		if err != nil {
			continue
		}
		meta.ShardNodeMap[idx] = nodes
	}
	return meta, nil
}

// ── TFHE Quota store ────────────────────────────────────────────────────────────

// GetTFHEQuota returns the number of active TFHE submissions for an account (0 if none).
func (k Keeper) GetTFHEQuota(ctx sdk.Context, addr sdk.AccAddress) uint64 {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.TFHEQuotaKey(addr))
	if len(bz) != 8 {
		return 0
	}
	return binary.BigEndian.Uint64(bz)
}

// setTFHEQuota stores the quota counter for an account.
func (k Keeper) setTFHEQuota(ctx sdk.Context, addr sdk.AccAddress, count uint64) {
	store := ctx.KVStore(k.storeKey)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, count)
	store.Set(types.TFHEQuotaKey(addr), bz)
}

// IncrTFHEQuota increments the active submission count for addr by 1.
func (k Keeper) IncrTFHEQuota(ctx sdk.Context, addr sdk.AccAddress) {
	k.setTFHEQuota(ctx, addr, k.GetTFHEQuota(ctx, addr)+1)
}

// DecrTFHEQuota decrements the active submission count for addr by 1 (floor 0).
func (k Keeper) DecrTFHEQuota(ctx sdk.Context, addr sdk.AccAddress) {
	cur := k.GetTFHEQuota(ctx, addr)
	if cur > 0 {
		k.setTFHEQuota(ctx, addr, cur-1)
	}
}