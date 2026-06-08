<div align="center">

<img src="https://zytherion.pages.dev/logo_zythc.png" alt="Zytherion Logo" width="120" />

# Zytherion Blockchain

**Next-Generation Post-Quantum Privacy Infrastructure**

[![License](https://img.shields.io/badge/License-MIT-blue.svg?style=flat-square)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?style=flat-square&logo=go)](https://golang.org/)
[![TFHE-rs](https://img.shields.io/badge/TFHE--rs-Zama%20v0.6-blueviolet?style=flat-square)](https://github.com/zama-ai/tfhe-rs)
[![Website](https://img.shields.io/badge/Website-zytherion.pages.dev-4CAF50?style=flat-square)](https://zytherion.pages.dev/)

*Fully Homomorphic Privacy. Quantum-Resistant Integrity. Green Consensus.*

</div>

---

## 🌐 Overview

**Zytherion** is a privacy-first Layer-1 blockchain that integrates five cutting-edge cryptographic pillars into a single unified architecture. It is engineered to protect user confidentiality and maintain absolute integrity against both classical and quantum-computing threats.

The network utilizes a highly optimized execution environment (built on Cosmos SDK) to achieve fast finality while preserving absolute user privacy using Fully Homomorphic Encryption.

| Pillar | Technology | Purpose |
|--------|-----------|---------|
| 🔐 **Fully Homomorphic Encryption** | TFHE via `tfhe-rs` (Zama) / CGo | Private computation on encrypted data (~21 KB ciphertexts) |
| 🛡️ **Post-Quantum Signatures** | CRYSTALS-Dilithium5 (ML-DSA Level 5) | Quantum-resistant validator signing (FIPS 204) |
| 💽 **Shard Integrity** | Binary Merkle Tree (SHA-256) over 16 shards | Root on-chain; per-shard proof verification (v0.4 new) |
| 💾 **Distributed Storage** | Reed-Solomon Erasure Coding (12+4=16 shards, RF=4) | Fault-tolerant P2P ciphertext sharding (v0.4 updated) |
| 🛡️ **Post-Quantum Hashing** | Ring-LWR Deterministic Hashing | Quantum-resistant block integrity |
| ⏱️ **Proof of Verifiable Lattices** | PoVL — Sequential VDF on LWR | Anti-manipulation network clock |
| 🌿 **Green BFT Consensus** | CometBFT + ABCI 2.0 | Energy-efficient Byzantine fault tolerance |

> [!NOTE]
> **🔐 TFHE Privacy** — Users encrypt values offline into `FheUint32` ciphertexts. The chain stores, verifies, and evaluates calculations directly on these ciphertexts without decrypting them. ZK-SNARKs were completely removed in v0.3 to eliminate trusted setup risks.
>
> **🛡️ PQC Signatures** — Validator key pairs and block signatures use CRYSTALS-Dilithium5 (ML-DSA Level 5) providing NIST Category-5 post-quantum security (~256-bit PQ).
>
> **💽 Merkle Shard Integrity (v0.4)** — After splitting a ciphertext into 16 shards, a SHA-256 Merkle tree is built over the shard set. The root is stored on-chain. Each receiving peer verifies the shard's Merkle proof before accepting storage, preventing tampering.
>
> **💾 Erasure Coding & Storage (v0.4)** — A 21 KB TFHE ciphertext is split using Reed-Solomon coding into 12 data and 4 parity shards. Shards are distributed to nodes with a **4x** replication factor, allowing reconstruction if up to 4 nodes fail.
>
> **🛡️ Ring-LWR** — Every block carries a Ring-LWR–based hash sentinel, making block integrity resistant to Grover's algorithm on quantum computers.
>
> **⏱️ PoVL** — A Sequential Verifiable Delay Function built on LWR. Proposers must compute it; all validators verify it. Invalid PoVL proofs cause immediate block rejection.

---

## 🏗️ Architecture

```
zytherion/
├── x/privacy/                  # Cosmos SDK privacy module
│   ├── keeper/
│   │   ├── keeper.go           # Store ops, TFHE meta/results, quota helpers (v0.4)
│   │   ├── msg_server_tfhe_submit.go  # Quota check, Merkle build, gas 1500/KB (v0.4)
│   │   └── query_tfhe_result.go       # Reconstructs ciphertexts on-demand
│   │
│   ├── tfhe/                   # Fully Homomorphic Encryption engine
│   │   ├── engine.go           # Go wrapper linking Rust static lib via CGo
│   │   ├── engine_stub.go      # Pure-Go stub fallback when CGo is disabled
│   │   ├── erasure.go          # Reed-Solomon sharding (12+4=16) [v0.4]
│   │   ├── merkle.go           # Binary Merkle tree over shards [NEW v0.4]
│   │   ├── shard_store.go      # Local disk storage + Shard{Signature,MerkleProof} [v0.4]
│   │   ├── shard_distributor.go  # Auth middleware, rate limiter, POST+Merkle verify [v0.4]
│   │   └── tfhe_c/             # Rust FFI wrapper crate for tfhe-rs
│   │
│   ├── pqc/                    # Post-quantum cryptography primitives
│   │   ├── signature.go        # Dilithium5 (ML-DSA Level 5) keygen/sign/verify
│   │   ├── lwr_hash.go         # Ring-LWR hashing
│   │   └── povl.go             # Sequential VDF (Proof of Verifiable Lattices)
│   │
│   └── types/
│       ├── errors.go           # ErrTFHEQuotaExceeded (1205), ErrShardAuthFailed (1206) [v0.4]
│       ├── keys.go             # KV prefixes + TFHEQuotaKeyPrefix [v0.4]
│       └── msg_tfhe_submit.go  # MsgTFHESubmit message types
│
├── app/
│   ├── lwr_proposal.go         # ABCI 2.0 PrepareProposal / ProcessProposal
│   ├── crypto_startup.go       # Post-quantum cryptographic integrity startup self-test
│   └── app.go                  # Application wiring & module registration
│
├── cmd/zytheriond/cmd/
│   └── root.go                 # --enable-tfhe startup flag, CLI version override
│
└── Makefile                    # build, build-tfhe-rs, test, install targets
```

---

## 🔬 Core Features

### 🔐 Fully Homomorphic Encryption (TFHE)

TFHE allows the network to process encrypted inputs directly. Instead of zero-knowledge proofs that only prove knowledge, TFHE enables **general computations** on ciphertexts.

**TFHE Transaction & Storage Flow (v0.4):**
```
1. [Client]    User encrypts value locally (e.g., 42 -> ct1.bin) using ClientKey.
2. [Client]    User submits tx using `tfhe-submit --ciphertext ct1.bin`.
3. [Node]      Quota check: reject if sender already has 1 active commitment.
4. [Node]      Verify size (1 KB – 32 KB), compute SHA-256 commitment.
5. [Node]      Split ct1.bin into 16 shards (12 data + 4 parity) using Reed-Solomon.
6. [Node]      Build SHA-256 Merkle tree over 16 shards; store root on-chain.
7. [Node]      Distribute shards to peers (RF=4); peers verify Merkle proof before storing.
8. [Node]      Write metadata (commitment, merkle_root, shard_node_map) on-chain.
9. [Validator] Nodes perform homomorphic additions/multiplications in-memory using ServerKey.
```

| Property | Value |
|----------|-------|
| Library | `tfhe-rs` (Zama) |
| Integration | CGo static binding |
| Target Type | `FheUint32` (32-bit unsigned integers) |
| Ciphertext Size | ~21 KB |
| Erasure Scheme | Reed-Solomon 12+4=16 shards (**v0.4**) |
| Replication | 4x distribution factor (**v0.4**) |
| Shard Integrity | Merkle proof per shard (**v0.4 new**) |
| Quota | 1 active commitment per address (**v0.4 new**) |
| Gas | 1,500 gas/KB (**v0.4**, was 1,000) |

> [!IMPORTANT]
> **TFHE Startup Flag:** By default, TFHE is disabled. The node must be started with the `--enable-tfhe` flag for TFHE transactions (`tfhe-submit`) to succeed. If not set, transactions will fail with `ErrTFHEDisabled`.

### 🛡️ Dilithium5 Validator Signatures (ML-DSA Level 5)

All validator keys and signatures have been upgraded from Dilithium3 to CRYSTALS-Dilithium5, achieving NIST Category 5 security (~256-bit post-quantum security).

| Metric | Dilithium3 (v0.2) | Dilithium5 (v0.3) | FIPS 204 Standard |
|---|---|---|---|
| NIST Level | Category 3 | **Category 5** | ML-DSA-87 |
| PQ Security | ~192-bit | **~256-bit** | ~256-bit |
| Public Key Size | 1952 bytes | **2592 bytes** | 2592 bytes |
| Private Key Size | 4000 bytes | **4864 bytes** | 4864 bytes |
| Signature Size | 3293 bytes | **4595 bytes** | 4595 bytes |

---

## 🚀 Getting Started

### Prerequisites

| Tool | Version | Install |
|------|---------|---------|
| Go | ≥ 1.23 | [golang.org/dl](https://golang.org/dl/) |
| Rust / Cargo | stable | [rustup.rs](https://rustup.rs/) |
| Ignite CLI | latest | [docs.ignite.com](https://docs.ignite.com/welcome/install) |
| GCC / glibc | system | Required for CGo static linking |

---

### 1. Clone the Repository

```bash
git clone https://github.com/Zytherion/Zytherion-Blockchain.git
cd Zytherion-Blockchain
```

### 2. Build the Rust TFHE Library

Before building the Go code, compile the static FFI bridge library in Rust:

```bash
make build-tfhe-rs
# outputs compiled libtfhe_c.a static library
```

### 3. Install the Zytherion Node

```bash
# To install the binary directly to $GOPATH/bin:
make install
```

### 4. Run the Chain (P2P / Development Mode)

To start a development node with the TFHE subsystem enabled:

```bash
# Using Ignite CLI:
ignite chain serve --args "--enable-tfhe"

# Or running the binary directly:
zytheriond start --enable-tfhe
```

---

## 💡 Usage — Submitting TFHE Ciphertext

### Step 1: Create a Ciphertext File
Generate a 20 KB dummy ciphertext file (in production, this is produced offline using the client Go/Rust SDK):

```bash
head -c 20480 /dev/urandom > ct1.bin
```

### Step 2: Submit to Blockchain
Submit the ciphertext file. Note that a higher `--gas` limit (e.g., `500000`) is required to cover the transaction payload size:

```bash
zytheriond tx privacy tfhe-submit \
  --ciphertext ct1.bin \
  --from alice \
  --gas 500000 \
  --keyring-backend test \
  -y
```

### Step 3: Query the Registered Commitment
Check the blockchain state to verify that the commitment is registered under your address:

```bash
zytheriond query privacy commitment <alice_address>
```

---

## 💰 Tokenomics

### Token Overview

| Property | Value |
|----------|-------|
| **Token Name** | Zytherion |
| **Ticker** | `ZYTC` |
| **Total Supply** | 1,000,000,000 ZYTC (1 Billion) |
| **Decimals** | 6 (1 ZYTC = 1,000,000 uzytc) |
| **Consensus** | Green BFT (CometBFT) |
| **Block Time** | ~5 seconds |

---

### 📊 Supply Distribution

| Allocation | % | Amount (ZYTC) | Purpose |
|------------|---|--------------|---------|
| 🌱 **Community Pool / Public Sale** | 45% | 450,000,000 | Ecosystem, dApp incentives, adoption |
| 🔐 **Staking Rewards** | 25% | 250,000,000 | Validator & delegator staking emissions |
| 🏗️ **Development Fund** | 15% | 150,000,000 | Protocol development |
| 👥 **Team & Founders** | 10% | 100,000,000 | Long-term vesting |
| 🌍 **Public Goods Funding** | 5% | 50,000,000 | Community grants |

```
  Community Pool      ██████████████████ 45%
  Staking Rewards     ██████████         25%
  Development Fund    ██████             15%
  Team & Founders     ████               10%
  Public Goods        ██                  5%
```

---

## 🗺️ Roadmap (v0.4 Status)

- [x] CRYSTALS-Dilithium5 Validator Signing (Cat-5, ~256-bit PQ)
- [x] Ring-LWR Deterministic Block Hashing (PQC sentinel)
- [x] Proof of Verifiable Lattices (PoVL — Sequential VDF)
- [x] ZK-SNARK removal (all gnark, BN254 circuits deleted)
- [x] TFHE Engine Integration (tfhe-rs via CGo wrapper)
- [x] Reed-Solomon Erasure Coding (12+4=16 shards) **[v0.4 updated]**
- [x] P2P Shard Distribution Server & Store
- [x] MsgTFHESubmit transaction handler
- [x] On-demand ciphertext reconstruction API
- [x] Binary Merkle Tree over shards — root on-chain **[v0.4 new]**
- [x] Per-shard Merkle proof verification on POST /shard **[v0.4 new]**
- [x] TFHE submission quota (max 1 per address) **[v0.4 new]**
- [x] Shard server auth (Bearer token) + rate limiter **[v0.4 new]**
- [x] ReplicationFactor 3 → 4 **[v0.4 new]**
- [x] Full Dilithium5 shard signing (ProposerPubkey + Signature) **[v0.4.1 new]**
- [ ] Multi-node testnet deployment
- [ ] ZYTC Token Generation Event (TGE)
- [ ] Web dashboard integration with [zytherion.pages.dev](https://zytherion.pages.dev/)

---

## 📄 License

This project is licensed under the **MIT License** — see the [LICENSE](LICENSE) file for details.

---

<div align="center">

Built with ❤️ by the [Zytherion Team](https://zytherion.pages.dev/)

**[🌐 Website](https://zytherion.pages.dev/) · [📦 GitHub](https://github.com/Zytherion/Zytherion-Blockchain)**

</div>
