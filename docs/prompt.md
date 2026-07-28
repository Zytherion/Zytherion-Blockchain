# Zytherion v0.5.1 — Architecture Prompt

**Project:** Zytherion Blockchain and Cryptocurrency  
**Version:** 0.5.1  
**Founder:** Rayhan Aziel Abbrar  
**Repository:** https://github.com/Zytherion/Zytherion-Blockchain

---

## System Description

Zytherion is a Layer-1 blockchain built on **Cosmos SDK v0.47 + CometBFT v0.37** featuring post-quantum cryptographic primitives and a fully homomorphic encryption privacy module.

**Tech Stack:**
- Languages: **Go** (main codebase) + **Rust** (TFHE library via CGo binding)
- Framework: Cosmos SDK (IBC, Gov, Staking, Bank, Privacy module)
- Consensus: CometBFT BFT + GreenBFT extensions
- Hashing: Ring-LWR (post-quantum, pure integer arithmetic)
- Signatures: **Dilithium5 (ML-DSA Level 5)** — NIST Cat-5, ~256-bit PQ security
- Privacy: **TFHE via tfhe-rs** (Zama) — FheUint32 operations
- Storage: **Reed-Solomon erasure coding** (12+4=16 shards, v0.4)
- Integrity: **Binary Merkle Tree** (SHA-256) over shard set — root stored on-chain (v0.4)

---

## Cryptographic Architecture (v0.4)

### Layer 1: Block Hashing — Ring-LWR

**Files:** `x/privacy/pqc/lwr_hash.go`, `hashing.go`  
**Algorithm:** Learning With Rounding in the Ring $Rq = \mathbb{Z}_q[X]/(X^n+1)$  
**Parameters:** $n=256$, $q=3329$, $p=256$, output size $= 96$ bytes  
**Status:** ✅ Unchanged from v0.2  

```
b = floor((p/q) * A·s) mod p
H_n = SHA3-256(LWR(data_n) || H_{n-1})
```

### Layer 2: Sequential VDF — PoVL (Proof of Verifiable Lattices)

**Files:** `x/privacy/pqc/povl.go`, `block.go`  
**Algorithm:** N-step sequential hash chain (VDF)  
**Integration:** ABCI 2.0 PrepareProposal/ProcessProposal  
**Status:** ✅ Unchanged from v0.2  

### Layer 3: Validator Signatures — Dilithium5 (ML-DSA Level 5)

**File:** `x/privacy/pqc/signature.go`  
**Library:** `github.com/cloudflare/circl/sign/dilithium/mode5`  
**Status:** 🆕 **UPGRADE** from Dilithium3 → Dilithium5 in v0.3  

**Key Sizes:**
- Public key: **2592 bytes** (mode5.PublicKeySize)
- Private key: **4864 bytes** (mode5.PrivateKeySize)
- Signature: **4595 bytes** (mode5.SignatureSize)

**API (Go):**
```go
// Generate key pair
kp, err := pqc.GenerateKeyPair()  // -> KeyPair{PublicKey, PrivateKey}

// Sign
sig, err := pqc.Sign(message, kp.PrivateKey)  // -> []byte (4595 bytes)

// Verify
ok := pqc.Verify(message, sig, kp.PublicKey)  // -> bool
```

### Layer 4: TFHE Homomorphic Encryption

**Files:** `x/privacy/tfhe/engine.go` (CGo) + `x/privacy/tfhe/tfhe_c/src/lib.rs` (Rust)  
**Library:** Zama's `tfhe-rs` v0.6, feature = `integer + x86_64-unix`  
**Status:** 🆕 **NEW** in v0.3 (replaces the removed ZK-SNARK subsystem)  

**API (Go):**
```go
// Key generation (SLOW: 30-120s)
keys, err := tfhe.GenerateKeys()
// keys.ClientKey: used for Encrypt/Decrypt (PRIVATE)
// keys.ServerKey: used for Add/Multiply (can be shared)

// Encrypt
ct, err := tfhe.EncryptUint32(keys.ClientKey, value)  // -> []byte (~21 KB)

// Homomorphic add: Enc(a) + Enc(b) -> Enc(a+b)
ctSum, err := tfhe.AddUint32(keys.ServerKey, ct1, ct2)

// Homomorphic multiply: Enc(a) * scalar -> Enc(a*scalar)
ctMul, err := tfhe.MultiplyScalarUint32(keys.ServerKey, ct, scalar)

// Decrypt
plaintext, err := tfhe.DecryptUint32(keys.ClientKey, ct)  // -> uint32
```

**Key Notes:**
- Server keys are not secret and can be shared with evaluation nodes.
- CGo calls are serialized using a global mutex (`tfheMu`).
- Ciphertext size: 16–32 KB (default: ~21 KB for FheUint32).

### Layer 5: Erasure Coding

**File:** `x/privacy/tfhe/erasure.go`  
**Library:** `github.com/klauspost/reedsolomon` v1.12.1  
**Parameters:** DataShards=12, ParityShards=4, Total=16 (**v0.4 updated**)  
**Status:** 🔄 **UPDATED** in v0.4 (was 10+6 in v0.3)  

```go
// Split 21 KB ciphertext into 16 shards
shards, err := tfhe.Split(ciphertext)  // -> []ShardResult (16 items)

// Reconstruct from any 12+ shards
original, err := tfhe.ReconstructFromResults(shards[:12], originalLen)
```

### Layer 6: Merkle Tree Integrity (NEW v0.4)

**File:** `x/privacy/tfhe/merkle.go`  
**Algorithm:** Binary Merkle tree — leaves = SHA-256(shard.Data)  
**Tree Depth:** 4 (log2(16) — no padding needed)  
**Status:** 🆕 **NEW** in v0.4  

```go
// Build Merkle tree over all 16 shards
tree, err := tfhe.BuildMerkleTree(shards)   // -> *MerkleTree
root := tree.RootBytes()                    // -> []byte (32 bytes, stored on-chain)

// Generate proof for shard i
proof, err := tree.ProofForShard(i)         // -> *MerkleProof (4 × 32 bytes)

// Verify shard authenticity without downloading all shards
err = tfhe.VerifyProof(rootHash, i, shardData, proof)
```

### Layer 7: P2P Shard Distribution

**Files:** `x/privacy/tfhe/shard_store.go`, `shard_distributor.go`  
**Protocol:** HTTP with Bearer-token auth + rate limiting (60 req/min/IP) (v0.4)  
**Replication:** **4 copies** per shard across peer nodes (v0.4, was 3)  
**Status:** 🔄 **HARDENED** in v0.4  

---

## Features REMOVED in v0.3

| Component | Files Removed |
|---|---|
| ZK Circuit (Groth16) | `x/privacy/zk/circuit.go` |
| ZK Verifier | `x/privacy/zk/verifier.go` |
| ZK Keys | `x/privacy/zk/keys.go` |
| ZK Commitment | `x/privacy/zk/commitment.go` |
| PoVL ZK Circuit | `x/privacy/zk/povl_circuit.go` |
| ZK Transfer Handler | `x/privacy/keeper/msg_server_zk_transfer.go` |
| gnark dependency | go.mod |
| gnark-crypto dependency | go.mod |
| zksetup Makefile target | Makefile |
| zkprove Makefile target | Makefile |

---

## Feature Flag

```bash
# Start node with TFHE ENABLED (default: OFF)
zytheriond start --enable-tfhe

# Start node without TFHE (default):
zytheriond start
# -> tx/tfhe_submit transactions will fail with ErrTFHEDisabled
```

---

## RPC Endpoints

### tx/tfhe_submit
Submit a TFHE ciphertext to the network for distributed erasure-coded storage.

**Request:**
```json
{
  "sender": "zyth1abc...",
  "ciphertext": "<base64 ~21KB FheUint32 bytes>"
}
```

**Response:**
```json
{
  "commitment_hash": "<base64 32 bytes SHA-256(ciphertext)>",
  "total_shards": 16,
  "merkle_root": "<base64 32 bytes — Merkle root of shard hashes>"
}
```

**Gas charges:** 1,500 gas per KB of ciphertext (v0.4, was 1,000).  
**Quota:** Each address may hold at most **1 active commitment** at a time (v0.4).

### query/tfhe_result
Reconstruct and retrieve a ciphertext from the P2P shard network.

**Request:**
```json
{
  "commitment_hash": "<base64 32 bytes>"
}
```

**Response:**
```json
{
  "commitment_hash": "<base64 32 bytes>",
  "result_ciphertext": "<base64 ~21KB bytes>",
  "reconstructed_from": 10
}
```

---

## Build & Test

```bash
# 1. Compile tfhe_c Rust static library (first time: 5-15 mins)
make build-tfhe-rs

# 2. Build Go binary
make build

# 3. Run tests
make test-pqc       # Dilithium5 tests (fast, ~5s)
make test-erasure   # Reed-Solomon tests (fast, ~1s) — now tests 12+4 parameters
make test-tfhe      # TFHE CGo tests (slow, ~5-10 mins)
```

---

## Directory Structure (v0.4)

```
zytherion/
├── app/
│   ├── app.go               # Cosmos SDK app — register modules
│   ├── crypto_startup.go    # Dilithium5 integrity self-test
│   └── greenbft/            # GreenBFT adaptive commit
├── x/privacy/
│   ├── pqc/
│   │   ├── signature.go     # Dilithium5
│   │   ├── lwr_hash.go      # Ring-LWR hashing
│   │   └── povl.go          # Sequential VDF (Proof of Verifiable Lattices)
│   ├── tfhe/                # TFHE subsystem
│   │   ├── engine.go        # CGo wrapper for tfhe-rs
│   │   ├── engine_stub.go   # Pure-Go stub fallback
│   │   ├── erasure.go       # Reed-Solomon 12+4=16 (v0.4)
│   │   ├── merkle.go        # Binary Merkle tree over shards (NEW v0.4)
│   │   ├── shard_store.go   # Disk-based shard storage + Shard{Signature,MerkleProof}
│   │   ├── shard_distributor.go  # P2P shard server — auth + rate limit + POST handler
│   │   └── tfhe_c/          # Rust library FFI crate
│   ├── keeper/
│   │   ├── keeper.go        # Store + quota helpers (GetTFHEQuota/Incr/Decr)
│   │   ├── msg_server_tfhe_submit.go  # Handles MsgTFHESubmit (quota, Merkle, gas 1500/KB)
│   │   └── query_tfhe_result.go       # Handles QueryTFHEResult
│   ├── types/
│   │   ├── errors.go        # TFHE errors + ErrTFHEQuotaExceeded (1205), ErrShardAuthFailed (1206)
│   │   ├── keys.go          # KV store prefixes + TFHEQuotaKeyPrefix
│   │   └── msg_tfhe_submit.go  # MsgTFHESubmit types
│   └── zk/                  # DELETED (v0.3)
├── cmd/zytheriond/cmd/
│   └── root.go              # CLI setup, version print override
├── config.yml               # Ignite config
├── go.mod                   # Removed gnark, added reedsolomon
└── Makefile                 # Build config
```

---

## CLI & Transaction Guide (v0.4)

This section explains how to run transactions in the terminal using the `zytheriond` CLI.

### 1. Standard Transaction (Send ZYTC Alice to Bob)

Ensure keys for Alice and Bob are added to the local keyring:
```bash
# Add Alice & Bob keys
zytheriond keys add alice --keyring-backend test
zytheriond keys add bob --keyring-backend test
```

To send a standard transaction (e.g., `1000zytc` from Alice to Bob):
```bash
# Use Bob's address (e.g. zythe1abc...) as the recipient
zytheriond tx bank send alice <bob_address> 1000zytc \
  --chain-id zytherion \
  --keyring-backend test \
  --fees 200zytc \
  -y
```

---

### 2. TFHE Transaction (Submit Encrypted Ciphertext)

> [!WARNING]
> If you specify the `--chain-id` flag, you must provide its value (`zytherion`). Leaving the flag empty at the end of the command will result in: `Error: flag needs an argument: --chain-id`. Alternatively, you can omit the flag entirely as it defaults to `"zytherion"`.

> [!IMPORTANT]
> **v0.4 Quota rule:** Each address may submit at most **1 active TFHE commitment** at a time. A second `tfhe-submit` from the same address will return `ErrTFHEQuotaExceeded`. A `revoke-commitment` command will be available in v0.4.1 to release the quota slot.

To submit a TFHE ciphertext file (`ct1.bin`):
```bash
# Option A: With full chain-id flag (gas >= 500000 recommended; 1500 gas/KB in v0.4)
zytheriond tx privacy tfhe-submit \
  --ciphertext ct1.bin \
  --from alice \
  --chain-id zytherion \
  --gas 600000 \
  --keyring-backend test \
  -y

# Option B: Omit chain-id (uses default "zytherion")
zytheriond tx privacy tfhe-submit \
  --ciphertext ct1.bin \
  --from alice \
  --gas 600000 \
  --keyring-backend test \
  -y
```

---

### 3. Homomorphic Addition Process Flow

Homomorphic operations (such as `Add` or `Multiply`) are performed **automatically on the Validator Node (Go & Rust backend)** when processing transactions, not typed as interactive terminal commands by users.

The life cycle is illustrated below:

```
[ Alice (Terminal) ] 
       │
       ├─► 1. Encrypt value 10 locally -> ct1.bin
       ├─► 2. Encrypt value 20 locally -> ct2.bin
       │
       ├─► 3. Submit both files via CLI:
       │      zytheriond tx privacy tfhe-submit --ciphertext ct1.bin ...
       │      zytheriond tx privacy tfhe-submit --ciphertext ct2.bin ...
       ▼
[ Node Validator (State Machine) ]
       │
       ├─► 4. Receives ct1 and ct2.
       ├─► 5. Automatically runs homomorphic addition in backend:
       │      ctSum = tfhe.AddUint32(serverKey, ct1, ct2)
       ├─► 6. Stores encrypted ctSum on-chain (without knowing the plaintext is 30).
       ▼
[ Alice (Terminal) ]
       │
       ├─► 7. Query the encrypted result using the commitment hash:
       │      zytheriond query privacy commitment <alice_address>
       │
       └─► 8. Decrypt the retrieved result offline using Alice's private ClientKey.
              Output: 30
```

---

---

### 4. Shard Integrity Verification (v0.4)

Each shard received by a peer now carries a Merkle proof. The peer node verifies it automatically before storing:

```
Shard Server (POST /shard):
  1. Receive JSON: { commitment_hex, index, data_hex, merkle_root_hex, merkle_proof_hex }
  2. Decode shard data and Merkle proof.
  3. Verify: VerifyProof(root, index, shardData, proof)  → 403 Forbidden on failure
  4. Store shard to local disk on success.  → 201 Created
```

To query the on-chain Merkle root for a commitment:
```bash
zytheriond query privacy commitment <alice_address>
# Shows commitment_hex and the new merkle_root field
```

---

## v0.5.1 Module Integrations
The new modules added in v0.5 / v0.5.1 provide standard DeFi, contract utility, and ecosystem tooling:
- **Price Oracle (`x/oracle`)**: Median TWAP feeds via validator consensus price submissions.
- **IBC Collateral (`x/ibc-collateral`)**: ICS-20 middleware locking collateral tokens in a vault.
- **Stablecoin (`x/stablecoin`)**: Pegged `ZYTD` stablecoin backed by locked IBC collateral.
- **CosmWasm**: Permissioned contract uploading governed by on-chain authority.

---

*Founder: **Rayhan Aziel Abbrar** | Version: 0.5.1 | 2026*
