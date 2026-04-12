<div align="center">

<img src="https://zytherion.pages.dev/logo_zythc.png" alt="Zytherion Logo" width="120" />

# ZYTHERION

### *The Quantum-Resistant, Privacy-Preserving, Eco-Friendly Blockchain*

[![Built on Cosmos SDK](https://img.shields.io/badge/Cosmos%20SDK-v0.47-7C3AED?style=for-the-badge&logo=cosmos&logoColor=white)](https://github.com/cosmos/cosmos-sdk)
[![PQC: Dilithium3](https://img.shields.io/badge/PQC-Dilithium3-06B6D4?style=for-the-badge&logoColor=white)](https://pq-crystals.org/dilithium/)
[![HE: FHE Ready](https://img.shields.io/badge/HE-FHE%20Ready-10B981?style=for-the-badge&logoColor=white)](#homomorphic-encryption)
[![Consensus: Green BFT](https://img.shields.io/badge/Consensus-Green%20BFT-22C55E?style=for-the-badge&logoColor=white)](#green-bft)
[![Token: ZYTC](https://img.shields.io/badge/Token-ZYTC-F59E0B?style=for-the-badge&logoColor=white)](https://zytherion.pages.dev)
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache%202.0-6366F1?style=for-the-badge)](LICENSE)

<br/>

> **Zytherion** is a next-generation Layer 1 blockchain engineered from the ground up for the post-quantum era.
> It combines **Post-Quantum Cryptography (PQC)**, **Homomorphic Encryption (HE)**, and an energy-efficient
> **Green Byzantine Fault Tolerant (Green BFT)** consensus — creating a chain that is simultaneously
> *quantum-safe*, *privacy-preserving*, and *sustainable*.

<br/>

**[Website](https://zytherion.pages.dev) · [Docs](#documentation) · [Quick Start](#quick-start) · [Community](#community)**

</div>

---

## About

Zytherion is **founded by Rayhan Aziel Abbrar (Zhao Han)**, who designed the chain's core architecture with a focus on long-term cryptographic resilience, on-chain privacy, and environmental sustainability.

The project was built from a conviction that existing blockchains are structurally vulnerable — both to the coming era of quantum computing and to the growing demand for computation that does not expose sensitive user data. Zytherion is the answer to both.

---

## Why Zytherion?

The three greatest existential threats to modern blockchains are:

| Threat | Impact | Zytherion's Answer |
|--------|--------|-------------------|
| **Quantum Computers** | Break ECDSA and RSA signatures | **PQC with Dilithium3 + SHA3-256** |
| **On-chain Privacy Leaks** | Sensitive computation is exposed in plaintext | **Homomorphic Encryption (FHE)** |
| **Energy Waste** | PoW and idle validators burn power unnecessarily | **Green BFT adaptive consensus** |

Zytherion tackles all three simultaneously — without sacrificing decentralization or speed.

---

## Core Technology Pillars

### Post-Quantum Cryptography (PQC)

Zytherion replaces classical ECDSA-based cryptography with **NIST-standardized, lattice-based** algorithms
that are provably secure against quantum adversaries — even against Shor's algorithm running on a
large-scale quantum computer.

```
Block Hashing   ->  SHA3-256    (Keccak, collision-resistant, quantum-hardened)
TX Signing      ->  Dilithium3  (CRYSTALS-Dilithium, NIST PQC Round 3 winner)
Coin Type       ->  2823        (Custom derivation path, BIP-44 compatible)
```

- **Dilithium3 Signatures** — 1312-byte public keys, 2420-byte signatures; immune to Grover's and Shor's attacks
- **SHA3-256 Block Hashing** — each block header is hashed using Keccak-f[1600], providing 128-bit quantum security
- **SIMD-Optimized Verification** — PQC signature verification is accelerated via CPU vector instructions,
  dramatically reducing the energy cost per block

> **Result:** Zytherion validators and wallets are fully secure against data harvested today under "harvest now, decrypt later" attacks.

---

### Homomorphic Encryption (HE)

Homomorphic Encryption allows computation *directly on encrypted data* — meaning a validator
can process a transaction and compute a result **without ever seeing the plaintext**.

```
Input (plaintext)  ->  Encrypt  ->  [Ciphertext]  ->  Compute  ->  [Encrypted Result]
                                                                            |
                                                                   Decrypt -> Output
```

**How Zytherion uses HE:**

- **FHE Computation Layer** — Idle validator nodes run Fully Homomorphic Encryption (FHE) computations
  on encrypted payloads, contributing to network privacy without leaking sensitive state
- **Privacy-Preserving Modules** — The `x/privacy` module leverages HE to allow confidential state transitions:
  balances, votes, and analytics can be computed without exposing user data
- **Mock FHE Attestation** — Validators attest that HE computations were performed correctly,
  earning *Green Badges* for verified private compute contributions

> **Result:** Zytherion is one of the few L1 chains where on-chain computation can be privacy-preserving by design.

---

### Green BFT Consensus

Standard BFT consensus wastes energy when the network is idle. Zytherion's **Green BFT** is
an adaptive consensus mechanism that dynamically adjusts resource usage based on real-time network load.

**Key Green BFT Features**

| Feature | Description |
|---------|-------------|
| **Adaptive Block Times** | Block interval scales with TX load — slow when idle, fast when busy |
| **Performance-to-Power Slashing** | High-latency or power-hungry validators face proportional slashing |
| **Green Badge System** | Validators that demonstrate low power profiles earn on-chain reputation badges |
| **Carbon Savings Query** | `zytheriond q greenbft carbon-savings` reports real-time energy efficiency metrics |
| **SIMD PQC Acceleration** | Vector-optimized Dilithium3 verification cuts per-block energy usage significantly |

```mermaid
graph LR
    A[Low TX Load] -->|Adaptive| B[Extend Block Time]
    B --> C[Reduce Validator CPU]
    C --> D[Lower Energy Use]
    D --> E[Green Badge Awarded]
    E --> F[Higher Validator Score]
```

> **Result:** Zytherion validators consume significantly less energy during off-peak hours, making the network
> more sustainable without sacrificing security or decentralization.

---

## Architecture Overview

```
+-------------------------------------------------------------+
|                        ZYTHERION NODE                       |
|                                                             |
|  +--------------+  +--------------+  +------------------+  |
|  |  PQC Layer   |  |   HE Layer   |  |  Green BFT Layer |  |
|  |              |  |              |  |                  |  |
|  | Dilithium3   |  | FHE Compute  |  | Adaptive Blocks  |  |
|  | SHA3-256     |  | x/privacy    |  | Green Badge      |  |
|  | SIMD Verify  |  | HE Attest    |  | Carbon Savings   |  |
|  +--------------+  +--------------+  +------------------+  |
|                                                             |
|  +-----------------------------------------------------+   |
|  |              Cosmos SDK v0.47 Base                  |   |
|  |   Bank · Staking · Gov · IBC · Slashing · Mint     |   |
|  +-----------------------------------------------------+   |
|                                                             |
|  +-----------------------------------------------------+   |
|  |               CometBFT Consensus Engine             |   |
|  +-----------------------------------------------------+   |
+-------------------------------------------------------------+
```

---

## ZYTC Tokenomics

The native token of the Zytherion network is **ZYTC** (Zytherion Coin), with a **1 billion ZYTC** total supply.

| Allocation | Amount | Percentage | Purpose |
|------------|--------|------------|---------|
| Community / Ecosystem | 450,000,000 ZYTC | 45% | Public sale, ecosystem growth, airdrops |
| Staking Rewards | 250,000,000 ZYTC | 25% | Long-term validator and delegator incentives |
| Development Fund | 150,000,000 ZYTC | 15% | Protocol R&D, audits, tooling |
| Team and Founders | 50,000,000 ZYTC | 5% | Core contributors (subject to vesting) |
| Team Vesting | 50,000,000 ZYTC | 5% | Staged 4-year vesting schedule |
| Public Goods | 50,000,000 ZYTC | 5% | Open source, research grants |

- **Denom:** `zytc`
- **Coin Type:** `2823` (BIP-44 custom path)
- **Min Gas:** `0.001 zytc`
- **Faucet:** 10 ZYTC per request (from `community_pool`)

---

## Quick Start

### Prerequisites

- Go `>= 1.21`
- [Ignite CLI](https://docs.ignite.com/welcome/install) `v0.27.0`
- Git

### Build and Run a Local Node

```bash
# Clone the repository
git clone https://github.com/zhaomei/zytherion.git
cd zytherion

# Start the development chain (includes genesis accounts and faucet)
ignite chain serve
```

The node will be live at:

| Service | Endpoint |
|---------|----------|
| RPC | `http://localhost:26657` |
| REST API | `http://localhost:1317` |
| gRPC | `localhost:9090` |
| Faucet | `http://localhost:4500` |

### Send Your First Transaction

```bash
# Check balances
zytheriond q bank balances <your-address>

# Send ZYTC (signed with Dilithium3 under the hood)
zytheriond tx bank send alice bob 1000000zytc --chain-id zytherion --fees 1000zytc
```

### Query Green BFT Metrics

```bash
# Check carbon savings reported by the network
zytheriond q greenbft carbon-savings

# List validators with Green Badge status
zytheriond q greenbft validators --green-only
```

---

## Project Structure

```
zytherion/
├── app/                    # Core application wiring (app.go, AnteHandler, modules)
├── cmd/                    # CLI entrypoint (zytheriond)
├── x/
│   └── privacy/            # Homomorphic Encryption module (FHE compute + attestation)
├── proto/                  # Protobuf definitions for all modules
├── docs/                   # OpenAPI spec and developer documentation
├── config.yml              # Genesis accounts, validators, faucet configuration
└── readme.md               # This file
```

---

## Documentation

| Resource | Link |
|----------|------|
| Official Website | [zytherion.pages.dev](https://zytherion.pages.dev) |
| REST API (OpenAPI) | `http://localhost:1317/static/openapi.yml` |
| Cosmos SDK Docs | [docs.cosmos.network](https://docs.cosmos.network) |
| Dilithium3 Spec | [pq-crystals.org/dilithium](https://pq-crystals.org/dilithium/) |
| FHE Reference | [fhe.org](https://fhe.org) |
| Ignite CLI Docs | [docs.ignite.com](https://docs.ignite.com) |

---

## Roadmap

- [x] **Phase 1** — Cosmos SDK base chain scaffolding
- [x] **Phase 2** — PQC integration: SHA3-256 hashing + Dilithium3 signatures
- [x] **Phase 3** — Green BFT: adaptive block times, slashing, carbon metrics
- [x] **Phase 4** — Homomorphic Encryption module (`x/privacy`)
- [x] **Phase 5** — Electron dashboard (node controller + PQC wallet)
- [ ] **Phase 6** — Mainnet genesis launch + validator onboarding
- [ ] **Phase 7** — IBC connections, cross-chain PQC messaging
- [ ] **Phase 8** — Full FHE smart contract execution environment

---

## Contributing

Contributions are welcome. Here is how to get started:

1. **Fork** the repository and create a feature branch
2. **Implement** your change (follow Go best practices and Cosmos SDK conventions)
3. **Test** with `go test ./...` and `ignite chain serve`
4. **Submit** a Pull Request to the `main` branch with a clear description

Please open an issue before implementing large features so we can align on direction.

---

## Community

Stay connected with the Zytherion community:

- **Website:** [zytherion.pages.dev](https://zytherion.pages.dev)
- **GitHub:** [github.com/zhaomei/zytherion](https://github.com/zhaomei/zytherion)

---

## License

Zytherion is open-source software released under the **Apache License 2.0**.
See [LICENSE](LICENSE) for the full license text.

---

<div align="center">

*Zytherion — Quantum-Safe · Privacy-First · Sustainably Green*

Founded by **Rayhan Aziel Abbrar (Zhao Han)**

[![zytherion.pages.dev](https://img.shields.io/badge/zytherion.pages.dev-7C3AED?style=for-the-badge)](https://zytherion.pages.dev)

</div>
