# Zytherion Blockchain and Cryptocurrency — Whitepaper Summary

**Version:** 0.4.0 | **Token:** ZYTC | **Chain ID:** `zytherion`  
**Founder:** Rayhan Aziel Abbrar  
**Repository:** https://github.com/Zytherion/Zytherion-Blockchain

---

## Changelog

### v0.4.0 — Security Hardening (TFHE Shard System)

| Change | Status | Details |
|---|---|---|
| Merkle Tree over Shards | 🆕 **NEW** | Binary SHA-256 Merkle tree; root stored on-chain; per-shard proof verification |
| ReplicationFactor | 🔄 **3 → 4** | Each shard now replicated to 4 nodes (was 3) |
| DataShards / ParityShards | 🔄 **10+6 → 12+4** | Higher data shards; reconstruction needs 12/16 shards |
| TFHE Quota | 🆕 **NEW** | Max 1 active commitment per address; `ErrTFHEQuotaExceeded` on overflow |
| Gas per KB | 🔄 **1000 → 1500** | Covers Merkle tree computation overhead |
| Shard server auth | 🆕 **NEW** | Bearer token required on all `/shard` endpoints |
| Rate limiter | 🆕 **NEW** | 60 req/min per IP; returns HTTP 429 on exceed |
| POST /shard + Merkle verify | 🆕 **NEW** | Peers verify Merkle proof before accepting shard |
| `ErrTFHEQuotaExceeded` (1205) | 🆕 **NEW** | Error code for quota violation |
| `ErrShardAuthFailed` (1206) | 🆕 **NEW** | Error code for invalid signature / Merkle proof |
| `TFHEQuotaKeyPrefix` | 🆕 **NEW** | KV prefix for per-address quota counter |

### v0.3.0 — TFHE & Erasure Coding

| Change | Status | Details |
|---|---|---|
| CRYSTALS-Dilithium5 | ✅ **ACTIVE** | Upgraded from Dilithium3. NIST Category 5 security (~256-bit PQ) |
| ZK-SNARK (Groth16/BN254) | ❌ **REMOVED** | Removed all gnark dependencies, ZK circuits, and verifiers |
| TFHE Engine (tfhe-rs) | ✅ **NEW** | FheUint32 via CGo static linking, ~21 KB ciphertext |
| Erasure Coding | ✅ **NEW** | Reed-Solomon 10+6=16 shards, tolerates up to 6 missing nodes |
| P2P Shard Distribution | ✅ **NEW** | Local ShardStore + HTTP P2P gossiping & reconstruction |
| `--enable-tfhe` flag | ✅ **NEW** | Default: OFF. Startup flag required to enable TFHE features |
| `tx/tfhe-submit` RPC | ✅ **NEW** | Broadcast and distribute TFHE ciphertext to the network |
| `query/tfhe-result` RPC | ✅ **NEW** | On-demand reconstruction of TFHE ciphertext |

---

## Abstract

**Zytherion** is a next-generation privacy-first Layer-1 blockchain built on the Cosmos SDK and CometBFT, combining state-of-the-art cryptographic primitives into a single integrated architecture:

- **Post-Quantum Signatures**: Dilithium5 (ML-DSA Level 5, NIST Cat-5, ~256-bit PQ security).
- **Post-Quantum Hashing**: Ring-LWR (Learning With Rounding) for block integrity.
- **Consensus**: GreenBFT + PoVL sequential VDF (Proof of Verifiable Lattices) for anti-timestamp manipulation.
- **Homomorphic Encryption**: Fully Homomorphic Encryption (TFHE) via Zama's `tfhe-rs` for computing on encrypted data.
- **Distributed Storage**: Reed-Solomon erasure coding (12+4=16 shards) distributed across peer nodes (**v0.4**).
- **Shard Integrity**: Binary Merkle Tree (SHA-256) over all 16 shards; root stored on-chain; per-shard Merkle proofs verified by receivers (**v0.4 new**).

Zytherion v0.3 eliminated the dependency on ZK-SNARKs (Groth16/BN254) which required a risky one-time trusted setup, replacing it with TFHE to provide a stronger, setup-free security model. v0.4 hardens the distributed shard storage layer against tampering and availability attacks.

---

## 1. System Architecture (v0.4)

```
┌──────────────────────────────────────────────────────────────┐
│                    Application (DApp / CLI)                  │
│         REST :1317 | RPC :26657 | gRPC :9090                 │
│         tx/tfhe-submit | query/tfhe-result                   │
└─────────────────────────────┬────────────────────────────────┘
                              │ ABCI 2.0
┌─────────────────────────────▼────────────────────────────────┐
│                 CometBFT (Green BFT)                         │
│           PrepareProposal / ProcessProposal                  │
│            PoVL Sentinel Validation Layer                    │
└─────────────────────────────┬────────────────────────────────┘
                              │
┌─────────────────────────────▼────────────────────────────────┐
│               Cosmos SDK Application Core                    │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │              Privacy Module (x/privacy)                 │ │
│  │  ┌───────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐  │ │
│  │  │ LWR Hash  │ │   PoVL   │ │Dilithium5│ │  TFHE    │  │ │
│  │  │ (PQC)     │ │  (VDF)   │ │(ML-DSA5) │ │ (tfhe-rs)│  │ │
│  │  └───────────┘ └──────────┘ └──────────┘ └──────────┘  │ │
│  └─────────────────────────────────────────────────────────┘ │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │     TFHE Shard Storage & Security (x/privacy/tfhe/) v0.4 │ │
│  │  RS(12+4) → Merkle Tree → 16 shards → Auth P2P → disk  │ │
│  │  ReplicationFactor=4, Quota per address, Rate Limiter    │ │
│  └─────────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────┘
```

---

## 2. Four Pillars of Technology (v0.3)

### 2.1 Post-Quantum Signatures — Dilithium5 (ML-DSA Level 5)

Upgraded from Dilithium3 to Dilithium5 as a breaking change pre-mainnet.

| Metric | Dilithium3 (v0.2) | Dilithium5 (v0.4) | FIPS 204 Standard |
|---|---|---|---|
| NIST Level | Category 3 | **Category 5** | ML-DSA-87 |
| PQ Security | ~192-bit | **~256-bit** | ~256-bit |
| Public Key Size | 1952 bytes | **2592 bytes** | 2592 bytes |
| Private Key Size | 4000 bytes | **4864 bytes** | 4864 bytes |
| Signature Size | 3293 bytes | **4595 bytes** | 4595 bytes |

**Key Generation Instructions:**
```bash
# Generate a new Dilithium5 validator key
zytheriond keys add my-validator --keyring-backend test

# Verify public key size
zytheriond keys show my-validator -p
# Output: pubkey bytes must be exactly 2592 bytes
```

### 2.2 Post-Quantum Hashing — Ring-LWR (Ring Learning With Rounding)

**Unchanged from v0.2.** Ring-LWR uses deterministic integer arithmetic over the ring $R_q = \mathbb{Z}_q[X]/(X^n + 1)$:

$$H_n = \text{SHA3-256}(\text{LWR}(data_n) \| H_{n-1})$$

Parameters: Ring dimension $n = 256$, prime modulus $q = 3329$ (Kyber-compatible), rounding modulus $p = 256$, output size $= 96$ bytes (32-byte seed + 64-byte vector $b$).

### 2.3 TFHE Homomorphic Encryption

Replaces the removed ZK-SNARK (Groth16/BN254) subsystem.

**Library:** Zama's `tfhe-rs` via CGo static linking (`x/privacy/tfhe/engine.go`).

**Exposed Operations:**
- **Encrypt**: `EncryptUint32(clientKey, value) -> ciphertext` (~21 KB).
- **Add**: `AddUint32(serverKey, ct1, ct2) -> ciphertext` (homomorphic addition).
- **Multiply**: `MultiplyScalarUint32(serverKey, ct, scalar) -> ciphertext` (homomorphic scalar multiplication).
- **Decrypt**: `DecryptUint32(clientKey, ciphertext) -> value` (requires private client key).

### 2.4 Proof of Verifiable Lattices (PoVL)

Sequential VDF built on Ring-LWR. Each block requires $N$ sequential hashing iterations (default: $N = 10$). Validated in the ABCI `ProcessProposal` phase prior to consensus voting.

---

## 3. TFHE Erasure Coding & P2P Storage

### Ciphertext Storage Architecture

```
                       Ciphertext (~21 KB)
                               │
               ┌───────────────▼──────────────┐
               │   Reed-Solomon Erasure Coding │
               │   DataShards=12, Parity=4     │
               └───────────────┬──────────────┘
                               │
               ┌───────────────▼──────────────┐
               │         16 Shards            │
               │   (12 data + 4 parity)       │
               │   ~1.75 KB each              │
               └───────────────┬──────────────┘
                               │
          ┌───────────────┼───────────────────┐
          ▼               ▼                   ▼
     Node 1          Node 2              Node N
    shards [0,3,6]  shards [1,4,7]     shards [2,5,8...]
     (4 shards)     (4 shards)        (4 shards)
                                      ReplicationFactor=4
```

**Reconstruction:** Minimum of **12 out of 16 shards** required for reconstruction.  
**Redundancy:** Tolerates up to **4 missing/offline nodes** without data loss.

### On-Chain Metadata Schema

```json
KV Store: tfhe_meta/<commitmentHash> → JSON {
    "commitment_hash": "<hex>",
    "original_len": 21504,
    "merkle_root": "<hex 32 bytes>",
    "proposer_pubkey": "<hex dilithium5 pubkey>",
    "shard_node_map": {
        "0": ["node1", "node7", "node12", "node15"],
        "1": ["node2", "node8", "node13", "node4"],
        ...
    }
}
```

---

## 4. API & RPC (v0.3)

### tx/tfhe-submit

```protobuf
message MsgTFHESubmit {
    string sender         = 1;  // bech32 address
    bytes  ciphertext     = 2;  // FheUint32 serialised, ~21 KB
}

message MsgTFHESubmitResponse {
    bytes  commitment_hash = 1;  // SHA-256 of ciphertext
    uint32 total_shards    = 2;  // always 16
}
```

**Gas charges:** 1,500 gas per KB of ciphertext (~31,500 gas for a standard 21 KB FheUint32) **[v0.4 updated]**.  
**Quota:** Each address may hold at most **1 active commitment** at a time. Returns `ErrTFHEQuotaExceeded` if exceeded.

### query/tfhe-result

```protobuf
message QueryTFHEResultRequest {
    bytes commitment_hash = 1;  // 32 bytes
}

message QueryTFHEResultResponse {
    bytes  commitment_hash     = 1;
    bytes  result_ciphertext   = 2;  // reconstructed ciphertext
    uint32 reconstructed_from  = 3;  // number of shards fetched (0 = cache hit)
}
```

---

## 5. Feature Flag --enable-tfhe

```bash
# Start node with TFHE ENABLED:
zytheriond start --enable-tfhe

# Default (TFHE DISABLED):
zytheriond start
# -> TFHE transactions rejected with ErrTFHEDisabled
```

When TFHE is enabled, the node initializes:
- Local shard directory: `~/.zytherion/tfhe_shards/`
- Shard server HTTP listener (port 26780)
- `MsgTFHESubmit` handlers and state storage
- `QueryTFHEResult` retrieval and reconstruction

---

## 6. Tokenomics (ZYTC)

**Total Supply:** 1,000,000,000 ZYTC (1 Billion)

| Allocation | Amount | % | Purpose |
|---|---|---|---|
| Community Pool / Public Sale | 450,000,000 | 45% | Ecosystem, dApp incentives, adoption |
| Staking Rewards | 250,000,000 | 25% | Validator and delegator staking emissions |
| Development Fund | 150,000,000 | 15% | Protocol development |
| Team & Founders | 100,000,000 | 10% | Long-term vesting |
| Public Goods Funding | 50,000,000 | 5% | Community grants |

---

## 7. Dependencies (v0.3)

| Library | Language | Purpose | Version |
|---|---|---|---|
| `github.com/klauspost/reedsolomon` | Go | Erasure coding pipeline | v1.12.1 |
| `tfhe` (Zama) | Rust | TFHE FheUint32 library | v0.6 |
| `bincode` | Rust | FFI serialization bridge | v1.3 |
| `github.com/cloudflare/circl` (mode5) | Go | CRYSTALS-Dilithium5 signatures | v1.6.3 |

**Removed:**
- `github.com/consensys/gnark` (ZK-SNARK removed)
- `github.com/consensys/gnark-crypto` (ZK-SNARK removed)

---

## 8. Build Instructions

```bash
# 1. Install Rust toolchain
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh

# 2. Compile tfhe_c Rust static library (takes 5-15 mins initially)
make build-tfhe-rs

# 3. Build Go binary
make build

# 4. Verify version output
zytheriond version
# Output:
# Zytherion Blockchain and Cryptocurrency
# Version: v0.3.0
# Founder: Rayhan Aziel Abbrar
# ...

# 5. Run tests
make test-pqc      # Dilithium5 and LWR hash tests (fast, ~5s)
make test-erasure  # Reed-Solomon tests (fast, ~1s)
make test-tfhe     # CGo TFHE engine tests (slow, ~5-10 mins)
```

---

## 9. Roadmap

| Phase | Target | Features |
|---|---|---|
| **Phase 1** ✅ | Completed | LWR Hashing, PoVL VDF, ZK Groth16 |
| **Phase 2** ✅ | Completed | GreenBFT, Dilithium3, PQC AnteDecorator |
| **Phase 3** ✅ | **v0.3 (2026)** | **Dilithium5, TFHE (tfhe-rs), Erasure Coding** |
| **Phase 4** ✅ | **v0.4 (2026)** | **Merkle Shard Integrity, Auth P2P, Quota, Rate Limiter** |
| **Phase 5** 📅 | Q4 2026 | Multi-node testnet deployment, TFHE bootstrapping |
| **Phase 6** 📅 | 2027 | IBC Privacy Bridge, Mainnet launch |

---

## 10. Conclusion

Zytherion v0.4 advances the distributed TFHE shard storage layer with enterprise-grade security:
1. **Merkle integrity** — every shard is verifiable against an on-chain root without downloading all 16 shards.
2. **Per-address quotas** — prevents storage abuse by limiting each account to 1 active commitment.
3. **Authenticated P2P** — Bearer-token auth and rate limiting block unauthorized shard injection.
4. **Higher replication** — ReplicationFactor=4 improves availability against node churn.

Combining Dilithium5, Ring-LWR, PoVL, TFHE, and now Merkle-verified sharded storage, Zytherion offers a fully quantum-resistant blockchain architecture with homomorphic privacy.

---

*This document is a technical summary of Zytherion v0.4. Founder: **Rayhan Aziel Abbrar**.*
