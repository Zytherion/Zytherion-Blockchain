<div align="center">

<img src="https://zytherion.pages.dev/logo_zythc.png" alt="Zytherion Logo" width="120" />

# Zytherion Blockchain

**Next-Generation Post-Quantum Privacy Infrastructure**

[![License](https://img.shields.io/badge/License-MIT-blue.svg?style=flat-square)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square&logo=go)](https://golang.org/)
[![gnark](https://img.shields.io/badge/gnark-Groth16%20%2F%20BN254-FF6B6B?style=flat-square)](https://github.com/ConsenSys/gnark)
[![Website](https://img.shields.io/badge/Website-zytherion.pages.dev-4CAF50?style=flat-square)](https://zytherion.pages.dev/)

*Zero-Knowledge Privacy. Quantum-Resistant Integrity. Green Consensus.*

</div>

---

## 🌐 Overview

**Zytherion** is a privacy-first Layer-1 blockchain that integrates four cutting-edge cryptographic pillars into a single unified architecture. It is engineered to protect user confidentiality and maintain absolute integrity against both classical and quantum-computing threats.

The network utilizes a highly optimized execution environment (built on Cosmos SDK) to achieve fast finality while preserving absolute user privacy.

| Pillar | Technology | Purpose |
|--------|-----------|---------|
| 🔐 **Zero-Knowledge Proofs** | Groth16 / BN254 via gnark (Go) | Private transaction verification (~128-byte proofs) |
| 🛡️ **Post-Quantum Cryptography** | Ring-LWR Deterministic Hashing | Quantum-resistant block integrity |
| ⏱️ **Proof of Verifiable Lattices** | PoVL — Sequential VDF on LWR | Anti-manipulation network clock |
| 🌿 **Green BFT Consensus** | CometBFT + ABCI 2.0 | Energy-efficient Byzantine fault tolerance |

> [!NOTE]
> **🔐 ZK Privacy** — Users generate off-chain Groth16 proofs over transaction values. The chain stores only the ~128-byte proof + public commitment — no plaintext ever touches the chain.
>
> **🛡️ PQC** — Every block carries a Ring-LWR–based hash sentinel, making block integrity resistant to Grover's algorithm on quantum computers.
>
> **⏱️ PoVL** — A Sequential Verifiable Delay Function built on LWR. Proposers must compute it; all validators verify it. Invalid PoVL proofs cause immediate block rejection.
>
> **🌿 Green BFT** — ABCI 2.0 `PrepareProposal` / `ProcessProposal` hooks enforce all cryptographic layers while keeping the network energy-efficient.

---

## 🏗️ Architecture

```
zytherion/
├── x/privacy/                  # Cosmos SDK privacy module
│   ├── keeper/
│   │   ├── keeper.go           # Store operations & ZK commitment state
│   │   └── msg_server_init_commitment.go  # MsgInitCommitment — ZK proof verify + store
│   │
│   ├── zk/
│   │   └── keys.go             # Load & validate Groth16 Verifying Key at startup
│   │
│   ├── types/
│   │   └── keys.go             # Store key prefixes
│   │
│   └── ante/
│       └── pqc_ante.go         # PQCAnteDecorator — SIMD-accelerated PQC tx screening
│
├── app/
│   ├── lwr_proposal.go         # ABCI 2.0 PrepareProposal / ProcessProposal
│   │                           # — PoVL computation, LWR-SHA3 block hashing
│   └── app.go                  # Application wiring
│
├── zkprove/                    # Off-chain Groth16 prover tool (Go binary)
│   └── main.go                 # Generates proof + public inputs for MsgInitCommitment
│
├── zksetup/                    # Trusted setup — generates proving & verifying keys
│   └── main.go
│
├── zk_artifacts/               # Committed ZK artifacts
│   ├── commitment.vk           # Groth16 Verifying Key (committed to repo)
│   └── commitment.pk           # Proving Key (large, for prover use only)
│
└── Makefile                    # build, test, zk-setup, zk-prove targets
```

---

## 🔬 Core Features

### 🔐 Zero-Knowledge Privacy (Groth16 / BN254)

Users prove knowledge of a private value without revealing it on-chain. The `MsgInitCommitment` transaction accepts a Groth16 proof and public commitment; the node verifies the proof and stores only the commitment.

**ZK Transaction Flow:**

```
1. [Off-chain] User runs `zkprove` tool with secret value → generates proof + public inputs
2. [Client]    User submits MsgInitCommitment { proof_bytes, public_inputs }
3. [Node]      PQCAnteDecorator screens the tx (SIMD PQC verification)
4. [Node]      ZK Verifier loads commitment.vk → verifies Groth16 proof
5. [Node]      If valid → stores commitment in KVStore
6. [Node]      If invalid → tx rejected, no state change
```

| Property | Value |
|----------|-------|
| Proving System | Groth16 |
| Elliptic Curve | BN254 (alt-bn128) |
| ZK Library | gnark (Go) |
| Proof Size | ~128 bytes |
| Verification | O(1) — constant time |

> [!IMPORTANT]
> **Fail-Fast Security:** Zytherion nodes will **panic on startup** if `commitment.vk` is missing or corrupted. This prevents any node from running in a security-compromised state.

### 🛡️ Ring-LWR Post-Quantum Block Hashing

Each block carries a deterministic LWR-based hash sentinel injected at the ABCI layer, anchoring block integrity against quantum adversaries.

**Construction:**

$$b = \left\lfloor \frac{p}{q} \cdot (A \cdot s) \right\rfloor \mod p$$

$$H_n = \text{SHA3-256}(\text{LWR}(data_n) \| H_{n-1})$$

| Parameter | Value | Notes |
|-----------|-------|-------|
| Ring dimension `n` | 256 | Polynomial coefficients |
| Prime modulus `q` | 3329 | Kyber-compatible |
| Rounding modulus `p` | 256 | 1 byte/coefficient output |
| Output size | 96 bytes | 32B seed + 64B vector b |

Implementation uses **pure integer arithmetic** (no floating-point) — guaranteeing bit-identical results across all CPU architectures. This ensures `LastResultsHash` is always consistent across all validator nodes.

### ⏱️ Proof of Verifiable Lattices (PoVL)

PoVL acts as a **Sequential Verifiable Delay Function (VDF)** built on LWR. Each block carries a sequential computation proof that cannot be parallelized or skipped.

**Chain Construction (N steps):**

$$\text{state}_n = \text{SHA3-256}(\text{LWRHash}(\text{state}_{n-1}) \| \text{state}_{n-1})$$

$$\text{PoVLRoot} = \text{state}_N = f^N(\text{state}_0)$$

**ABCI 2.0 Integration:**
1. **PrepareProposal:** Block proposer computes `PoVLRoot` over N sequential steps.
2. **ProcessProposal:** All validators verify `PoVLRoot` before casting votes. Blocks without a valid PoVL are **immediately rejected (REJECT)**.

Default configuration: **N = 10 steps per block**.

### 🌿 Green BFT Consensus

| Enhancement | Description |
|------------|-------------|
| **PQC SIMD AnteDecorator** | Screens all incoming transactions with SIMD-accelerated PQC verification before any gas is spent |
| **Adaptive Timeout** | Block timeout is dynamically adjusted based on recent tx volume — empty blocks do not consume a full round |
| **Deterministic DeliverTx** | PoVL sentinel is injected at the `DeliverTx` override layer, guaranteeing identical `LastResultsHash` across all nodes |

---

## 🚀 Getting Started

### Prerequisites

| Tool | Version | Install |
|------|---------|---------|
| Go | ≥ 1.21 | [golang.org/dl](https://golang.org/dl/) |
| Ignite CLI | latest | [docs.ignite.com](https://docs.ignite.com/welcome/install) |
| GCC / glibc | system | Required for CGo (gnark uses it internally) |

> ⚠️ **Linux recommended** — tested on `x86_64-linux`. macOS may work with minor adjustments. Windows requires WSL2.

---

### 1. Clone the Repository

```bash
git clone https://github.com/Zytherion/Zytherion-Blockchain.git
cd Zytherion-Blockchain
```

### 2. Generate ZK Trusted Setup (one-time)

> ⏱️ This generates the Groth16 proving and verifying keys. **Only required if `zk_artifacts/` is not already committed.**

```bash
make zk-setup
# outputs: zk_artifacts/commitment.pk and zk_artifacts/commitment.vk
```

### 3. Build Go Packages

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
# Full test suite
make test

# Run without ZK integration (fast unit tests)
go test ./...
```

---

## 💡 Usage — Private Commitment CLI

### Step 1: Generate a ZK Proof (off-chain)

```bash
# Generate a Groth16 proof for a secret value (e.g., 1000)
go run ./zkprove --secret 1000 --pk zk_artifacts/commitment.pk --out proof.json
```

This produces `proof.json` containing the proof bytes and public inputs.

### Step 2: Submit the Commitment On-chain

```bash
zytherion tx privacy init-commitment \
  --proof-file proof.json \
  --from alice \
  --chain-id zytherion \
  --yes
```

> The node receives only the ~128-byte Groth16 proof and the public commitment hash. The secret value (`1000`) **never leaves the client machine**.

---

## 🔑 Key Concepts

### ZK Artifact Security Model

| Artifact | Size | Committed to Repo | Purpose |
|----------|------|-------------------|---------|
| `commitment.vk` | ~1 KB | ✅ Yes | Node startup verification — must be present |
| `commitment.pk` | ~10–100 MB | ❌ No (gitignored) | Off-chain proof generation by users |

The **Verifying Key (VK) is committed to the repository** so all nodes use the same trusted setup. The Proving Key is large and only needed by users who want to generate proofs.

### Layered Security Stack

```
Layer 1: Ring-LWR (PQC)          → Quantum-resistant block integrity
Layer 2: PoVL (VDF)              → Sequential anti-manipulation clock
Layer 3: ZK-SNARKs (Groth16)    → Transaction privacy
Layer 4: Green BFT (CometBFT)   → Consensus finality
Layer 5: Fail-Fast ZK VK        → Node startup integrity guard
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
| **Chain Denom** | `uzytc` |
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

### 🔒 Vesting Schedule

| Allocation | Cliff | Vesting Duration | TGE Unlock |
|------------|-------|-----------------|------------|
| Team & Founders | 12 months | 48 months (linear) | 0% |
| Strategic Partners | 6 months | 24 months (linear) | 5% |
| Development Fund | None | 36 months (milestone) | 10% |
| Public Sale | None | 12 months (linear) | 20% |
| Staking Rewards | None | Emitted over ~10 years | — |
| Community Pool | None | DAO-governed | 0% |

---

### ⚙️ Token Utility

| Use Case | Description |
|----------|-------------|
| **Staking** | Stake ZYTC to run or delegate to validators; earn staking rewards |
| **Gas Fees** | Pay transaction fees in `uzytc` for all on-chain operations including `MsgInitCommitment` |
| **Governance** | Vote on protocol upgrades, parameter changes, and treasury spending |
| **ZK Pool Access** | Future: stake ZYTC to participate in anonymous ZK liquidity pools |
| **Key Escrow** | Future: deposit ZYTC as collateral for threshold commitment management |

---

### 📈 Emission Schedule

Staking rewards follow a **halving model** adjusted to a 5-second block time:

| Year | Annual Emission | Cumulative Circulating |
|------|----------------|------------------------|
| Year 1 | 50,000,000 ZYTC | ~380,000,000 ZYTC |
| Year 2 | 40,000,000 ZYTC | ~470,000,000 ZYTC |
| Year 3 | 30,000,000 ZYTC | ~550,000,000 ZYTC |
| Year 4 | 25,000,000 ZYTC | ~625,000,000 ZYTC |
| Year 5+ | Decreasing (halving every 4 years) | → 1,000,000,000 ZYTC |

> [!IMPORTANT]
> All tokenomics parameters are subject to on-chain governance votes before mainnet launch. The DAO can adjust emission rates, vesting schedules, and treasury allocations.

---

## 🗺️ Roadmap

- [x] Ring-LWR Deterministic Block Hashing (PQC sentinel)
- [x] Proof of Verifiable Lattices (PoVL — Sequential VDF)
- [x] ZK-SNARK Groth16 / BN254 verifier integration (gnark)
- [x] `MsgInitCommitment` — on-chain ZK proof verification & commitment storage
- [x] Off-chain `zkprove` and `zksetup` tools
- [x] Fail-Fast VK startup guard
- [x] PQC SIMD AnteDecorator
- [x] ABCI 2.0 `PrepareProposal` / `ProcessProposal` hooks
- [x] Deterministic `LastResultsHash` across all validator nodes
- [ ] Multi-node testnet with full PoVL ZK proof in production
- [ ] ZYTC Token Generation Event (TGE)
- [ ] IBC Inter-chain Privacy Bridge
- [ ] Dilithium3 validator signing scheme
- [ ] On-chain governance module with ZYTC voting
- [ ] Mainnet launch — full quantum-resistant signature scheme
- [ ] Web dashboard integration with [zytherion.pages.dev](https://zytherion.pages.dev/)

---

## 📦 Module Dependencies

```toml
# Go (go.mod)
github.com/cosmos/cosmos-sdk         v0.47.x
github.com/cometbft/cometbft         v0.37.x
github.com/consensys/gnark           latest   # Groth16 / BN254 ZK proving system
github.com/consensys/gnark-crypto    latest   # BN254 elliptic curve primitives
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
