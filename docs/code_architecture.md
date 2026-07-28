# Zytherion v0.4 — Code Architecture Guide

This document describes the codebase structure, APIs, and execution pipelines of the Zytherion Blockchain (v0.4).

---

## 1. Module Structure (`x/privacy`)

Zytherion uses the Cosmos SDK module layout. The core cryptographic logic, storage pipelines, and CLI endpoints are encapsulated inside the `x/privacy` module.

```
x/privacy/
├── client/cli/                 # Command Line Interface endpoints
│   ├── query.go                # Registers query subcommands
│   ├── tx.go                   # Registers transaction subcommands
│   ├── tx_deposit.go           # MsgInitCommitment CLI handler
│   └── tx_encrypted_transfer.go# MsgTFHESubmit CLI handler
│
├── keeper/                     # State-transition and database access logic
│   ├── keeper.go               # KVStore read/writes, TFHE meta/result helpers, quota CRUD (v0.4)
│   ├── msg_server_tfhe_submit.go  # MsgTFHESubmit handler (quota check, Merkle build, shards)
│   ├── query_tfhe_result.go    # QueryTFHEResult handler (reconstructs ciphertext)
│   └── msg_server_deposit.go   # MsgInitCommitment handler
│
├── pqc/                        # Post-Quantum Cryptography primitives
│   ├── signature.go            # CRYSTALS-Dilithium5 FIPS 204 wrappers
│   ├── lwr_hash.go             # Ring-LWR hashing algorithm
│   └── povl.go                 # Sequential VDF (Proof of Verifiable Lattices)
│
├── tfhe/                       # Fully Homomorphic Encryption engine
│   ├── tfhe_c/                 # Rust FFI library (static build)
│   │   ├── src/lib.rs          # Rust FFI wrapper functions
│   │   └── Cargo.toml          # Rust dependencies (tfhe-rs)
│   ├── engine.go               # Go CGo bridge linking libtfhe_c.a
│   ├── engine_stub.go          # Fallback stub for compiling without CGo
│   ├── erasure.go              # Reed-Solomon coding (12+4=16 shards) [v0.4]
│   ├── merkle.go               # Binary SHA-256 Merkle tree over shards [NEW v0.4]
│   ├── shard_store.go          # Disk-based shard persistence + Shard struct [v0.4]
│   └── shard_distributor.go    # Shard server (HTTP) + auth + rate limiter + POST/Merkle [v0.4]
│
└── types/                      # Protobuf types and interface definitions
    ├── codec.go                # Amino and Proto codec registration
    ├── errors.go               # Domain-specific error codes (ErrTFHEQuotaExceeded 1205, ErrShardAuthFailed 1206)
    ├── keys.go                 # KVStore prefix keys, event attributes, TFHEQuotaKeyPrefix [v0.4]
    └── tx.pb.go                # Generated protobuf structs and interfaces
```

---

## 2. Rust-Go CGo Integration

The homomorphic operations are evaluated using Zama's `tfhe-rs` Rust library. Because the main codebase is written in Go, Zytherion links the compiled Rust library (`libtfhe_c.a`) statically via CGo.

### FFI Header & Linking Directive (`x/privacy/tfhe/engine.go`)
```go
//go:build tfhe_cgo
// +build tfhe_cgo

package tfhe

/*
#cgo CFLAGS: -I${SRCDIR}
#cgo LDFLAGS: -L${SRCDIR}/tfhe_c/target/release -ltfhe_c -lm -ldl -lpthread
#include "cgo_bridge.h"
#include <stdlib.h>
*/
import "C"
```

### Mutex Serialization
Because `tfhe-rs` sets a thread-local global `ServerKey` for homomorphic evaluations, calling into the Rust FFI concurrently will lead to race conditions. To prevent this, all CGo operations are serialized using a package-level mutex:

```go
var tfheMu sync.Mutex

func AddUint32(serverKey, ct1, ct2 []byte) ([]byte, error) {
	tfheMu.Lock()
	defer tfheMu.Unlock()
    
	// Call into Rust CGo bridge...
}
```

---

## 3. Reed-Solomon Erasure Coding Pipeline (v0.4)

To avoid storing the full 21 KB TFHE ciphertext on every node, Zytherion splits it into 16 shards using Reed-Solomon erasure coding.

- **Data Shards:** 12 (minimum required for reconstruction) **[v0.4 updated from 10]**
- **Parity Shards:** 4 (redundancy overhead) **[v0.4 updated from 6]**
- **Total Shards:** 16

### Pipeline Execution Flow (`x/privacy/tfhe/erasure.go`)

```
          [Original Ciphertext (~21 KB)]
                        │
                        ▼
             Split into 12 Data Shards
                        │
                        ▼
            Generate 4 Parity Shards
                        │
                        ▼
      [16 Shards Result (~1.75 KB each)]
```

If any 12 shards are retrieved from peer nodes, the original ciphertext is fully reconstructed:
```go
original, err := tfhe.Reconstruct(shards, originalLen)
```

---

## 4. Merkle Tree Shard Integrity (NEW v0.4)

After sharding, a binary SHA-256 Merkle tree is built over all 16 shard hashes. The Merkle root is stored on-chain inside `TFHEShardMeta`. Receiving peers verify each shard's proof before storing it to disk.

### Tree Structure (`x/privacy/tfhe/merkle.go`)

```
       Root (SHA-256)
      /               \
   H(0-7)           H(8-15)
   /     \           /     \
 H(0-3) H(4-7)  H(8-11) H(12-15)
  / \    / \     / \      / \
 H0 H1 H2 H3   H8 H9  H12 H13 ...

Leaf Hᵢ = SHA-256(shard[i].Data)
Tree depth = log₂(16) = 4
```

### Key API

```go
// Build Merkle tree over all 16 shards
tree, err := tfhe.BuildMerkleTree(shards)
root := tree.RootBytes()          // stored on-chain in TFHEShardMeta.MerkleRoot

// Generate proof for a specific shard
proof, _ := tree.ProofForShard(i) // 4 × 32 = 128 bytes

// Verify on receiver side before storing
err = tfhe.VerifyProof(rootHash, i, shardData, proof)  // returns error if tampered
```

---

## 5. P2P Shard Storage & Gossip (v0.4 Hardened)

The P2P network distributes shards to peers using an out-of-band HTTP-based protocol. In v0.4, the server requires authentication and verifies Merkle proofs.

```
Proposer Node (MsgTFHESubmit)
  │
  ├─► Stores all 16 shards locally (for single-node backup)
  ├─► Selects (ReplicationFactor - 1) = 3 random peers for each shard index
  ├─► Attaches Merkle proof to each shard
  │
  └─► POST /shard  (JSON: {data_hex, merkle_root_hex, merkle_proof_hex})
       │
       └─► Peer Node:
            1. Verify Auth: Bearer <nodeID> header required
            2. Rate check: reject if >60 req/min from this IP
            3. Verify Merkle proof: VerifyProof(root, index, data, proof)
            4. Store to local disk on success (HTTP 201)
```

### Retrieval & On-Demand Reconstruction
When a validator needs to perform an evaluation (e.g. during block processing or a REST query):
1. The validator queries the on-chain metadata mapping `commitmentHash -> ShardNodeMap + MerkleRoot`.
2. The validator polls the local `ShardStore`.
3. If local shards are fewer than 12, the validator requests missing indexes from peers listed in the mapping using `GET /shard?commitment=<hex>&index=<idx>`.
4. Once 12 shards are collected, the validator runs `Reconstruct` to recover the original ciphertext.

---

## 6. Post-Quantum Validator Signatures (Dilithium5)

Validator signing keys use **CRYSTALS-Dilithium5 (ML-DSA-87)**.

- **Public Key Size:** 2592 bytes
- **Private Key Size:** 4864 bytes
- **Signature Size:** 4595 bytes
- **Library:** `github.com/cloudflare/circl/sign/dilithium/mode5`

The node initialization process (`app/crypto_startup.go`) runs a self-test of the Dilithium5 verification engine at startup. If the signature engine fails to sign/verify a test payload, the node panics and halts immediately.

---

## 7. Per-Address TFHE Quota (NEW v0.4)

To prevent storage abuse and withholding attacks, each account may hold at most **1 active TFHE commitment** at any time.

```
KV Store key: tfhe_quota/<address_bytes>  (8-byte big-endian uint64)

On MsgTFHESubmit:
  if GetTFHEQuota(addr) >= 1  →  return ErrTFHEQuotaExceeded
  else:
    proceed with submission
    IncrTFHEQuota(addr)       (counter: 0 → 1)
```

When a commitment is revoked (planned for v0.4.1):
```
  DecrTFHEQuota(addr)         (counter: 1 → 0)
```

---

## 8. ABCI 2.0 Block Processing Pipeline

Zytherion uses CometBFT's ABCI 2.0 hooks to enforce block integrity:

```
[ Block Proposer ]
  │
  └─► PrepareProposal:
        1. Inject LWR Hash sentinel for current block data.
        2. Compute PoVL sequential VDF (N=10 steps).
        3. Pack block proposals with the PoVL VDF proof.
        ▼
[ Validator Nodes ]
  │
  └─► ProcessProposal:
        1. Extract block proposal and VDF proof.
        2. Verify VDF proof.
        3. If VDF is valid ──► ACCEPT Proposal & Vote
        4. If VDF is invalid ─► REJECT Proposal & Hault/Skip
```

This sequence prevents block stamp manipulation or timestamp manipulation attacks by malicious proposers.

---

## 9. Oracle Module (`x/oracle`)
The oracle module allows validators to submit off-chain price feeds. The module maintains a sliding block window (TWAP window) to compute median prices for whitelisted tokens (`ZYTC`, `axlUSDC`, `mUSDT`, `ATOM`, `wBTC`, `wETH`).

- **Median Pricing**: Median calculation prevents outliers from skewing the pricing feed, making it resistant to flash loan manipulation.
- **Active Pruning**: Old price entries outside of the TWAP window + max age are actively deleted at each block end to keep storage minimal.

```
Validator Nodes (tx submit-price)
   │
   ├─► Store PriceEntry in KVStore: price/<denom>/<height>
   │
   └─► BeginBlocker:
        1. Fetch all PriceEntries within the TWAP window.
        2. Compute median price.
        3. Save computed TWAP price to KVStore.
        4. EndBlocker: Prune price entries older than TwapWindowBlocks + MaxPriceAge.
```

---

## 10. IBC Collateral Module (`x/ibc-collateral`)
The `ibc-collateral` module manages the locking and unlocking of whitelisted IBC collateral assets. It acts as an ICS-20 transfer middleware.

- **Vaulting**: Locks assets in a module account named `ibc_collateral_vault`.
- **Position Tracking**: Tracks user collateral deposits via `CollateralPosition` records in the KVStore.
- **Middleware**: Intercepts incoming transfer packets via `OnRecvPacket`. If the token is a whitelisted collateral asset, it registers the deposit in the vault.

---

## 11. Stablecoin Module (`x/stablecoin`)
The `stablecoin` module implements the multi-collateral stablecoin engine. It interacts with both `x/oracle` (to fetch price TWAPs) and `x/ibc-collateral` (to manage collateral backing).

- **Minting ZYTD**: Verifies that the requested ZYTD to mint does not exceed the collateral's USD value divided by the minimum collateral ratio:
  $$\text{max\_mintable} = \frac{\text{collateral\_amount} \times \text{TWAP}}{\text{min\_ratio}}$$
- **Burning ZYTD**: Burns the user's ZYTD and returns a proportional amount of their locked collateral.
- **Liquidation**: If a position's collateral ratio drops below the liquidation threshold, any user can trigger liquidation. The liquidator pays the outstanding ZYTD debt and receives the locked collateral at a discount (seized collateral minus protocol fees).

---

## 12. CosmWasm Integration (`wasmd`)
The permissioned CosmWasm smart contract engine (`wasmd` v0.45.0 + `wasmvm` v1.5.2) allows developers to write custom dApps in Rust.
- **Permissioned Mode**: Only authorized accounts or governance proposals can upload smart contract bytecodes (`MsgStoreCode`), keeping the ecosystem secure.
- **Gas & Sandboxing**: CosmWasm runs inside a secure, gas-metered WebAssembly sandbox, ensuring smart contract execution cannot stall validator nodes.
