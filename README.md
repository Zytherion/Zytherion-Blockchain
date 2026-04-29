<div align="center">

<img src="https://img.shields.io/badge/Zytherion-Blockchain-6C63FF?style=for-the-badge&logoColor=white" alt="Zytherion Blockchain"/>

# ⛓️ Zytherion Blockchain

**A Post-Quantum Privacy Blockchain built on Cosmos SDK**

[![License](https://img.shields.io/badge/License-MIT-blue.svg?style=flat-square)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square&logo=go)](https://golang.org/)
[![Rust Edition](https://img.shields.io/badge/Rust-2021%20Edition-orange?style=flat-square&logo=rust)](https://www.rust-lang.org/)
[![Cosmos SDK](https://img.shields.io/badge/Cosmos%20SDK-v0.47-5C4EE5?style=flat-square)](https://docs.cosmos.network/)
[![TFHE-rs](https://img.shields.io/badge/TFHE--rs-v0.7-FF6B6B?style=flat-square)](https://github.com/zama-ai/tfhe-rs)
[![Website](https://img.shields.io/badge/Website-zytherion.pages.dev-4CAF50?style=flat-square)](https://zytherion.pages.dev/)

*Encrypted transfers. Homomorphic balances. Quantum-resistant future.*

</div>

---

## 🌐 Overview

**Zytherion** is a privacy-first blockchain built on the **Cosmos SDK**, combining three cutting-edge cryptographic pillars to deliver a chain where account balances remain encrypted end-to-end — even from validators.

| Pillar | Technology | Purpose |
|--------|-----------|---------|
| 🔐 **Fully Homomorphic Encryption** | TFHE-rs v0.7 (`FheUint64`) via CGo FFI | Encrypted balance arithmetic on-chain |
| 🛡️ **Post-Quantum Cryptography** | LWE-SHA3-Hybrid Block Hashing | Quantum-resistant block integrity |
| 🌿 **Green BFT Consensus** | CometBFT + ABCI 2.0 | Energy-efficient Byzantine fault tolerance |

> [!NOTE]
> **🔐 FHE** — Validators process `EncryptedTransfer` transactions without ever seeing plaintext amounts. The chain state stores only TFHE ciphertexts.
>
> **🛡️ PQC** — Every block carries an LWE-SHA3-Hybrid sentinel injected at the ABCI layer, making block hashes resistant to quantum adversaries.
>
> **🌿 Green BFT** — CometBFT consensus is wired with ABCI 2.0 `PrepareProposal` / `ProcessProposal` hooks, keeping the chain both secure and energy-efficient.

---

## 🏗️ Architecture

```
zytherion/
├── tfhe-cgo/               # Rust FFI crate — TFHE-rs static library
│   ├── src/lib.rs          # C-ABI exports: generate_keys, encrypt, decrypt, add, sub, compress
│   ├── Cargo.toml          # tfhe v0.7, bincode, staticlib target
│   └── build.sh            # Cargo build → libtfhe_cgo.a
│
├── x/privacy/              # Cosmos SDK privacy module
│   ├── fhe/                # Go FHE layer (wraps CGo bindings)
│   │   ├── fhe.go          # Context, Encrypt/Decrypt, AddCiphertexts, SubCiphertexts
│   │   ├── tfhe_binding.go # Raw CGo call wrappers
│   │   ├── tfhe_compress.go# Compress/decompress ciphertext helpers
│   │   ├── fhe_mock.go     # Mock implementation for unit tests (build tag: notfhe)
│   │   └── fhe_test.go     # Roundtrip and arithmetic tests
│   │
│   ├── keeper/             # Module keeper
│   │   ├── keeper.go       # Store operations, HomomorphicAdd/Sub, EncryptAmount
│   │   └── msg_server_encrypted_transfer.go  # MsgEncryptedTransfer handler
│   │
│   └── client/cli/         # CLI commands
│       └── tx.go           # CmdEncryptedTransfer (accepts plaintext amount, encrypts on-the-fly)
│
└── Makefile                # build-tfhe, build, test, lint targets
```

---

## 🔬 Core Features

### 🔒 Homomorphic Encrypted Transfers

The `EncryptedTransfer` message lets users transfer funds without revealing the amount to anyone — including validators and full nodes.

**Transfer flow (all operations in ciphertext space):**

```
1. Validate sender & recipient addresses
2. Reject sender if no encrypted balance exists
3. sender_balance   = Enc(sender_bal)   − Enc(amount)    ← HomomorphicSub
4. recipient_balance = Enc(recv_bal)    + Enc(amount)    ← HomomorphicAdd
5. Emit EventTypeEncryptedTransfer (no amounts in event)
```

No plaintext is ever computed server-side. Validators learn only: *"a transfer occurred from A to B."*

### 🦀 TFHE-rs via CGo FFI

The `tfhe-cgo` crate compiles to a **static C library** (`libtfhe_cgo.a`) exposing these functions:

| C Function | Description |
|------------|-------------|
| `tfhe_generate_keys` | Generate client + server key pair (bincode-serialized) |
| `tfhe_encrypt_u64` | Encrypt a `u64` → raw ciphertext (~30–50 KB) |
| `tfhe_decrypt_u64` | Decrypt raw ciphertext → `u64` |
| `tfhe_add` | Homomorphic addition of two raw ciphertexts |
| `tfhe_sub` | Homomorphic subtraction of two raw ciphertexts |
| `tfhe_compress` | Compress raw ciphertext → compact form (~1–5 KB) |
| `tfhe_decompress` | Decompress compact ciphertext → operable form |
| `tfhe_free_bytes` | Free heap-allocated output buffers |

The Go `fhe.Context` wraps these bindings and implements a **compress-on-store** strategy:
- All ciphertexts persisted in the KVStore are **compressed** (~1–5 KB vs ~30–50 KB raw)
- Arithmetic ops decompress → compute → recompress transparently

### 🌐 LWE-SHA3-Hybrid Block Hashing (PQC)

Each block is anchored with an LWE-based post-quantum hash sentinel injected via **ABCI 2.0** (`PrepareProposal` / `ProcessProposal`). The audit hash is persisted in the PrivacyKeeper store and rolled into `AppHash`, providing quantum-resistant block integrity.

---

## 🚀 Getting Started

### Prerequisites

| Tool | Version | Install |
|------|---------|---------|
| Go | ≥ 1.21 | [golang.org/dl](https://golang.org/dl/) |
| Rust + Cargo | stable | `curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs \| sh` |
| Ignite CLI | latest | [docs.ignite.com](https://docs.ignite.com/welcome/install) |
| GCC / glibc | system | Required for CGo linking |

> ⚠️ **Linux only** — the Rust crate targets `x86_64-unix` TFHE-rs features. macOS and Windows are not supported without modification.

---

### 1. Clone the Repository

```bash
git clone https://github.com/Zytherion/Zytherion-Blockchain.git
cd Zytherion-Blockchain
```

### 2. Build the TFHE-rs Static Library

> ⏱️ This step compiles Rust with `lto = true` and `opt-level = 3`. **Expect 30–90 minutes** on first run.

```bash
make build-tfhe
```

This runs `cargo build --release` inside `tfhe-cgo/` and copies the resulting `libtfhe_cgo.a` to `x/privacy/fhe/lib/`.

### 3. Build All Go Packages

```bash
make build
# equivalent to: go build ./...
```

### 4. Run the Chain (Development Mode)

```bash
ignite chain serve
```

### 5. Run Tests

```bash
# Full test suite (requires libtfhe_cgo.a)
make test

# Fast unit tests without CGo (uses mock FHE)
go test -tags notfhe ./...
```

---

## 💡 Usage — Encrypted Transfer CLI

The CLI accepts a **plaintext amount** and encrypts it locally before submitting the transaction:

```bash
# Send 1000 tokens from alice to bob (amount is encrypted client-side)
zytherion tx privacy encrypted-transfer cosmos1bob... 1000 \
  --from alice \
  --chain-id zytherion \
  --yes
```

> The node receives only the FHE ciphertext — the plaintext `1000` never leaves the client machine.

---

## 🔑 Key Concepts

### Compressed Ciphertext Storage

| State | Size |
|-------|------|
| Raw `FheUint64` ciphertext | ~30–50 KB |
| Compressed `FheUint64` ciphertext | ~1–5 KB |
| On-chain KVStore entry | **~1–5 KB** ✅ |

Compression uses TFHE-rs's built-in `compress()` / `decompress()` API, reducing storage overhead by **~10×** with zero security loss.

### Mock Build Tag

For CI pipelines or machines without a Rust toolchain, build with `-tags notfhe`:

```bash
go test -tags notfhe ./...
```

This swaps in `fhe_mock.go` and `tfhe_binding_mock.go`, which implement plaintext arithmetic behind the same `fhe.Context` interface — no `libtfhe_cgo.a` required.

---

## 💰 Tokenomics

### Token Overview

| Property | Value |
|----------|-------|
| **Token Name** | Zytherion |
| **Ticker** | `ZYT` |
| **Total Supply** | 1,000,000,000 ZYT (1 Billion) |
| **Decimals** | 6 (1 ZYT = 1,000,000 uzyt) |
| **Chain Denom** | `uzyt` |
| **Consensus** | Green BFT (CometBFT) |
| **Block Time** | ~5 seconds |

---

### 📊 Supply Distribution

| Allocation | % | Amount (ZYT) | Purpose |
|------------|---|-------------|---------|
| 🌱 **Ecosystem & Grants** | 30% | 300,000,000 | Developer grants, dApp incentives, hackathons |
| 🔐 **Privacy Staking Rewards** | 25% | 250,000,000 | Validator & delegator staking emissions |
| 👥 **Team & Contributors** | 15% | 150,000,000 | Core team, subject to 4-year vesting |
| 🏦 **Treasury & Reserve** | 15% | 150,000,000 | Protocol-controlled reserve, DAO-governed |
| 🚀 **Public Sale / TGE** | 10% | 100,000,000 | Community token generation event |
| 🤝 **Strategic Partners** | 5% | 50,000,000 | Early backers and strategic investors |

```
  Ecosystem & Grants  ████████████ 30%
  Staking Rewards     ██████████   25%
  Team                ██████       15%
  Treasury            ██████       15%
  Public Sale         ████         10%
  Strategic Partners  ██            5%
```

---

### 🔒 Vesting Schedule

| Allocation | Cliff | Vesting Duration | TGE Unlock |
|------------|-------|-----------------|------------|
| Team & Contributors | 12 months | 48 months (linear) | 0% |
| Strategic Partners | 6 months | 24 months (linear) | 5% |
| Ecosystem & Grants | None | 36 months (milestone) | 10% |
| Public Sale | None | 12 months (linear) | 20% |
| Staking Rewards | None | Emitted over ~10 years | — |
| Treasury | None | DAO-governed | 0% |

---

### ⚙️ Token Utility

| Use Case | Description |
|----------|-------------|
| **Staking** | Stake ZYT to run or delegate to validators; earn staking rewards |
| **Gas Fees** | Pay transaction fees in `uzyt` for all on-chain operations including `EncryptedTransfer` |
| **Governance** | Vote on protocol upgrades, parameter changes, and treasury spending |
| **Privacy Pool Access** | Future: stake ZYT to participate in anonymous liquidity pools |
| **FHE Key Escrow** | Future: deposit ZYT as collateral for threshold key management services |

---

### 📈 Emission Schedule

Staking rewards follow a **halving model** similar to Bitcoin, adjusted to a 5-second block time:

| Year | Annual Emission | Cumulative Circulating |
|------|----------------|------------------------|
| Year 1 | 50,000,000 ZYT | ~380,000,000 ZYT |
| Year 2 | 40,000,000 ZYT | ~470,000,000 ZYT |
| Year 3 | 30,000,000 ZYT | ~550,000,000 ZYT |
| Year 4 | 25,000,000 ZYT | ~625,000,000 ZYT |
| Year 5+ | Decreasing (halving every 4 years) | → 1,000,000,000 ZYT |

> [!IMPORTANT]
> All tokenomics parameters are subject to on-chain governance votes before mainnet launch. The DAO can adjust emission rates, vesting schedules, and treasury allocations.

---

## 🗺️ Roadmap

- [x] TFHE-rs CGo FFI (key generation, encrypt, decrypt, add, sub)
- [x] Compress-on-store strategy for KVStore efficiency
- [x] `EncryptedTransfer` MsgServer with homomorphic balance updates
- [x] ABCI 2.0 LWE-SHA3 block hashing (PQC sentinel)
- [x] CLI: plaintext amount → on-the-fly encryption → TX submission
- [ ] Persistent key management (keystore for `fhe.Context` keys)
- [ ] Query endpoints for encrypted balance (returns ciphertext bytes)
- [ ] ZYT Token Generation Event (TGE)
- [ ] Cross-chain IBC privacy bridge
- [ ] Threshold decryption for authorized auditors
- [ ] On-chain governance module with ZYT voting
- [ ] Web dashboard integration with [zytherion.pages.dev](https://zytherion.pages.dev/)

---

## 📦 Module Dependencies

```toml
# Rust (tfhe-cgo/Cargo.toml)
tfhe    = { version = "0.7", features = ["integer", "x86_64-unix"] }
bincode = "1"

# Go (x/privacy)
github.com/cosmos/cosmos-sdk  v0.47.x
github.com/cometbft/cometbft  v0.37.x
```

---

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feat/your-feature`
3. Commit your changes: `git commit -m "feat: add your feature"`
4. Push to the branch: `git push origin feat/your-feature`
5. Open a Pull Request

Please make sure all tests pass (`make test`) before submitting.

---

## 📄 License

This project is licensed under the **MIT License** — see the [LICENSE](LICENSE) file for details.

---

<div align="center">

Built with ❤️ by the [Zytherion Team](https://zytherion.pages.dev/)

**[🌐 Website](https://zytherion.pages.dev/) · [📦 GitHub](https://github.com/Zytherion/Zytherion-Blockchain)**

</div>
