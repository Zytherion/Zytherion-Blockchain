<div align="center">

<img src="https://zytherion.pages.dev/logo_zythc.png" alt="Zytherion Logo" width="120" />

# Zytherion Blockchain

**Next-Generation Post-Quantum Privacy Infrastructure**

[![License](https://img.shields.io/badge/License-MIT-blue.svg?style=flat-square)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?style=flat-square&logo=go)](https://golang.org/)
[![TFHE-rs](https://img.shields.io/badge/TFHE--rs-Zama%20v0.6-blueviolet?style=flat-square)](https://github.com/zama-ai/tfhe-rs)
[![Version](https://img.shields.io/badge/Version-v0.7.0-orange?style=flat-square)](docs/summary.md)
[![Website](https://img.shields.io/badge/Website-zytherion.pages.dev-4CAF50?style=flat-square)](https://zytherion.pages.dev/)

*Fully Homomorphic Privacy. Quantum-Resistant Integrity. QuantumBFT Consensus.*

</div>

---

## 🌐 Overview

**Zytherion** is a post-quantum, privacy-first Layer-1 blockchain built on Cosmos SDK and QuantumBFT (CometBFT). It integrates five cutting-edge cryptographic pillars into a unified architecture to protect user confidentiality and maintain total integrity against both classical and quantum-computing threats.

| Pillar | Technology | Purpose |
|--------|-----------|---------|
| 🔐 **Fully Homomorphic Encryption** | TFHE via `tfhe-rs` (Zama) / CGo | Private computation on encrypted data (~21 KB to 512 KB ciphertexts) |
| 🛡️ **Post-Quantum Signatures** | CRYSTALS-Dilithium5 (ML-DSA Level 5) | Quantum-resistant validator signing (FIPS 204) & transaction signatures |
| 🔑 **Post-Quantum KEM** | CRYSTALS-Kyber1024 (ML-KEM Level 5) | Quantum-resistant P2P SecretConnection & file encryption (FIPS 203) |
| 💽 **Shard Integrity & Auth** | Binary Merkle Tree + Dilithium5 Signatures | On-chain Merkle root & per-shard ML-DSA-87 authenticity verification |
| 💾 **Zero-Config Distributed Sharding** | Reed-Solomon (12+4=16) + Hash Placement | Automatic validator auto-discovery via `StakingKeeper`; nodes store only assigned shards |
| 🛡️ **Post-Quantum Hashing** | Ring-LWR Deterministic Hashing | Quantum-resistant block integrity |
| ⏱️ **Proof of Verifiable Lattices** | PoVL — Sequential VDF on LWR | Anti-manipulation network clock |
| 🌿 **QuantumBFT Consensus** | CometBFT + ABCI 2.0 + Dilithium5 PV | Energy-efficient Byzantine fault tolerance with PQC validator consensus |
| 📊 **Price Oracle** | `x/oracle` Module | Validator-driven Median TWAP price feeds |
| 📦 **IBC Collateral Vault** | `x/ibc-collateral` Module | ICS-20 token middleware and vault locks |
| 🪙 **Multi-Collateral Stablecoin** | `x/stablecoin` Module | Mint/burn/liquidate ZYTD pegged stablecoin |
| 📜 **Smart Contracts** | CosmWasm (`wasmd` v0.45.0) | WebAssembly smart contracts with `TFHECustomQuery` plugin |

> [!NOTE]
> **🔐 TFHE Privacy** — Users encrypt values offline into `FheUint32` ciphertexts. The chain stores, verifies, and evaluates calculations directly on these ciphertexts without decrypting them. TFHE is always-on with thread-pinned worker pool concurrency.
>
> **🛡️ PQC Signatures & KEM** — Validator key pairs use CRYSTALS-Dilithium5 (ML-DSA Level 5) providing NIST Category-5 post-quantum security (~256-bit PQ). P2P handshakes use hybrid CRYSTALS-Kyber1024 (ML-KEM Level 5) + X25519.
>
> **💾 Zero-Config Distributed Sharding** — A 21 KB TFHE ciphertext is split using Reed-Solomon coding into 12 data and 4 parity shards. Shards are distributed deterministically across active validators auto-discovered from the chain state (`StakingKeeper`). Each node stores only its assigned shard subset.
>
> **🔒 Strict Protobuf Security Compliance** — All modules use formal Protobuf `.proto` definitions compiled via `protoc-gen-gogo`. Full strict unmarshaling enforcement (`unknownproto`) is active across all chain transactions.

---

## 🏗️ Architecture

```
zytherion/
├── x/privacy/                  # Cosmos SDK privacy module
│   ├── keeper/
│   │   ├── keeper.go           # Store ops, TFHE meta/results, quota helpers
│   │   ├── msg_server_tfhe_submit.go  # Validator auto-discovery & sharding
│   │   └── query_tfhe_result.go       # Reconstructs ciphertexts on-demand (CLI & REST)
│   │
│   ├── tfhe/                   # Fully Homomorphic Encryption engine
│   │   ├── engine.go           # Go wrapper linking Rust static lib via CGo
│   │   ├── worker_pool.go      # OS-thread-pinned worker pool concurrency
│   │   ├── erasure.go          # Reed-Solomon sharding (12+4=16)
│   │   ├── merkle.go           # Binary Merkle tree over shards
│   │   ├── shard_store.go      # Disk storage + Shard{Signature,MerkleProof}
│   │   ├── shard_distributor.go  # Deterministic Hash-Based Shard Placement
│   │   └── tfhe_c/             # Rust FFI wrapper crate for tfhe-rs
│   │
│   └── pqc/                    # Post-quantum cryptography primitives
│       ├── signature.go        # Dilithium5 (ML-DSA Level 5) keygen/sign/verify
│       ├── lwr_hash.go         # Ring-LWR hashing
│       └── povl.go             # Sequential VDF (Proof of Verifiable Lattices)
│
├── x/oracle/                   # Price Oracle module (TWAP 30-block window)
├── x/stablecoin/               # ZYTD Stablecoin minting/burning/liquidation
├── x/ibc-collateral/           # ICS-20 middleware & vault position management
├── proto/zytherion/            # Official Protobuf schemas (tx.proto, query.proto)
├── quantumbft/                 # QuantumBFT consensus engine (Dilithium5 PV)
│
├── app/
│   ├── lwr_proposal.go         # ABCI 2.0 PrepareProposal / ProcessProposal
│   ├── crypto_startup.go       # Post-quantum cryptographic integrity startup self-test
│   └── app.go                  # Application wiring & module registration
│
└── Makefile                    # build, build-tfhe-rs, test, install targets
```

---

## 🚀 Quickstart Guide

### 1. Prerequisites

| Tool | Version | Install |
|------|---------|---------|
| Go | ≥ 1.23 | [golang.org/dl](https://golang.org/dl/) |
| Rust / Cargo | stable | [rustup.rs](https://rustup.rs/) |
| GCC / glibc | system | Required for CGo static linking |

### 2. Build Rust TFHE Library & Go Binary

```bash
git clone https://github.com/Zytherion/Zytherion-Blockchain.git
cd Zytherion-Blockchain

# 1. Compile Rust static library
make build-tfhe-rs

# 2. Build & Install binary with CGo TFHE enabled
CGO_ENABLED=1 go install -tags tfhe_cgo ./cmd/zytheriond
```

### 3. Run Node

```bash
zytheriond start
```

---

## 💡 Usage — Commands & REST API

### 🔒 1. TFHE Privacy (Submit & Query Result)

```bash
# Generate 20 KB ciphertext payload
head -c 20480 /dev/urandom > data1.bin

# Submit ciphertext to chain
zytheriond tx privacy tfhe-submit \
  --ciphertext data1.bin \
  --from alice \
  --chain-id zytherion \
  --keyring-backend test \
  --gas 500000 \
  -y

# Query result via CLI
zytheriond query privacy tfhe-result --commitment <COMMITMENT_HASH_HEX>

# Query result via REST HTTP API (curl)
curl http://localhost:1317/zytherion/privacy/v1/tfhe/result/<COMMITMENT_HASH_HEX>
```

### 🔮 2. Oracle Price Feed & Stablecoin Minting

```bash
# 1. Submit Oracle Price Feed
zytheriond tx oracle submit-price uzytc 1.00 \
  --from alice --chain-id zytherion --keyring-backend test --gas 500000 -y

# 2. Mint ZYTD Stablecoin
zytheriond tx stablecoin mint-zytd \
  --collateral-denom uzytc \
  --collateral-amount 2000000000 \
  --zytd-amount 1000000000 \
  --expiration-block-height 1000 \
  --from alice --chain-id zytherion --keyring-backend test \
  --fees 5000zytc --gas 500000 -y
```

### ⚡ 3. Staking & Instant Redelegation

```bash
# Redelegate voting power between validators instantly without unbonding
zytheriond tx staking redelegate \
  zythvaloper1ska696lz9h44g6gysrp0tg5j7lvqd65qzwckt6 \
  zythvaloper1j3gndh7jruxwkzt2tcxaytvpvm9gr7xuz2hls2 \
  70000000000000zytc \
  --from alice --chain-id zytherion --keyring-backend test \
  --fees 5000zytc --gas 1000000 -y
```

---

## 💰 Tokenomics (ZYTC)

| Property | Value |
|----------|-------|
| **Token Name** | Zytherion |
| **Ticker** | `ZYTC` |
| **Total Supply** | 1,000,000,000 ZYTC (1 Billion) |
| **Decimals** | 6 (1 ZYTC = 1,000,000 uzytc) |
| **Consensus** | QuantumBFT (CometBFT) |
| **Block Time** | ~5 seconds |

### 📊 Supply Distribution

| Allocation | % | Amount (ZYTC) | Purpose |
|------------|---|--------------|---------|
| 🌱 **Community Pool / Public Sale** | 45% | 450,000,000 | Ecosystem, dApp incentives, adoption |
| 🔐 **Staking Rewards** | 25% | 250,000,000 | Validator & delegator staking emissions |
| 🏗️ **Development Fund** | 15% | 150,000,000 | Protocol development |
| 👥 **Team & Founders** | 10% | 100,000,000 | Long-term vesting |
| 🌍 **Public Goods Funding** | 5% | 50,000,000 | Community grants |

---

## 📄 License

This project is licensed under the **MIT License** — see the [LICENSE](LICENSE) file for details.

---

<div align="center">

Built with ❤️ by the [Zytherion Team](https://zytherion.pages.dev/)

**[🌐 Website](https://zytherion.pages.dev/) · [📦 GitHub](https://github.com/Zytherion/Zytherion-Blockchain) · [📑 Docs](docs/summary.md)**

</div>
