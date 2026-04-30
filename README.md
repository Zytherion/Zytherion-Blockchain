<div align="center">
  <img src="https://zytherion.pages.dev/logo_zythc.png" alt="Zytherion Logo" width="180"/>

  <h1>Zytherion Blockchain</h1>

  <p><strong>A quantum-resistant, privacy-preserving Layer 1 blockchain</strong><br/>
  built on Cosmos SDK with Torus FHE, Dilithium3 PQC signing, and Green BFT consensus.</p>

  <p>
    <img src="https://img.shields.io/badge/Cosmos_SDK-v0.47-5664D2?style=flat-square&logo=cosmos" alt="Cosmos SDK"/>
    <img src="https://img.shields.io/badge/CometBFT-v0.37-blue?style=flat-square" alt="CometBFT"/>
    <img src="https://img.shields.io/badge/FHE-TFHE_(Torus_LWE)-brightgreen?style=flat-square" alt="TFHE"/>
    <img src="https://img.shields.io/badge/PQC-Dilithium3-orange?style=flat-square" alt="Dilithium3"/>
    <img src="https://img.shields.io/badge/Token-ZYTC-gold?style=flat-square" alt="ZYTC"/>
    <img src="https://img.shields.io/badge/License-MIT-lightgrey?style=flat-square" alt="License"/>
  </p>

  <p>
    <a href="https://zytherion.pages.dev/Zytherion_White_Paper.pdf">📄 Whitepaper</a> ·
    <a href="#getting-started">🚀 Quick Start</a> ·
    <a href="#architecture">🏗 Architecture</a> ·
    <a href="#tokenomics">💰 Tokenomics</a>
  </p>
</div>

---

## Overview

Zytherion is a Layer 1 blockchain that integrates three cutting-edge technologies into a single coherent protocol:

- **Post-Quantum Cryptography (PQC)** — transaction signing with CRYSTALS-Dilithium3 (NIST PQC standard), SHA3-256 block hash commitments
- **Threshold Fully Homomorphic Encryption (TFHE)** — encrypted on-chain balances using LWE-based FHE; validators compute on ciphertext without ever learning plaintext amounts
- **Green BFT** — adaptive commit timeout that reduces validator energy consumption during low-traffic periods by extending block intervals automatically

> **Live demo:** An encrypted transfer of 250,000 ZYTC produces a ~21KB TFHE ciphertext on-chain — down from 376KB with the prior BFV scheme — with no plaintext amount visible in any event or log.

---

## Features

| Feature | Description |
|---|---|
| 🔐 **TFHE Encrypted Balances** | Deposit ZYTC into a privacy pool; balances stored as TFHE ciphertexts. Validators perform homomorphic addition/subtraction without decrypting. |
| 🛡 **Dilithium3 PQC Signing** | All transactions signed with CRYSTALS-Dilithium3, resistant to Shor's algorithm attacks from quantum computers. |
| ⚡ **SHA3-256 PQC Ante** | Every transaction receives a SHA3-256 domain-separated fingerprint at the ante layer for quantum-hardened integrity. |
| 🌿 **Green BFT** | Rolling 10-block window tracks mempool load. Idle chain → 5s commit timeout. Busy chain → 1s commit timeout. Saves validator CPU automatically. |
| 🏦 **Privacy Module** | `MsgDeposit` and `MsgEncryptedTransfer` messages for entering and moving within the encrypted balance pool. |
| ⛓ **Cosmos SDK v0.47** | IBC-compatible, modular architecture. Staking, governance, and minting all denominated in ZYTC. |

---

## Architecture

```
┌─────────────────────────────────────────────────────┐
│                   Zytherion Node                    │
│                                                     │
│  ┌─────────────┐   ┌──────────────────────────┐    │
│  │  AnteChain  │   │       ABCI App            │    │
│  │             │   │                          │    │
│  │ PQCante     │   │  BeginBlock              │    │
│  │ (SHA3-256   │   │  └─ Record validator     │    │
│  │  tx hash)   │   │     latency (Green BFT)  │    │
│  │     ↓       │   │                          │    │
│  │ BaseAnte    │   │  EndBlock                │    │
│  │ (standard   │   │  └─ Drain tx count       │    │
│  │  Cosmos)    │   │  └─ Compute adaptive     │    │
│  │     ↓       │   │     timeout              │    │
│  │ AddTx()     │   │  └─ Emit ABCI event      │    │
│  │ (Green BFT  │   │     recommended_ms       │    │
│  │  counter)   │   │                          │    │
│  └─────────────┘   └──────────────────────────┘    │
│                                                     │
│  ┌──────────────────────────────────────────────┐   │
│  │              x/privacy Module                │   │
│  │                                              │   │
│  │  MsgDeposit → lock ZYTC → Encrypt(amount)   │   │
│  │              → store TFHE ciphertext         │   │
│  │                                              │   │
│  │  MsgEncryptedTransfer                        │   │
│  │  → HomomorphicSub(sender, amount)            │   │
│  │  → HomomorphicAdd(receiver, amount)          │   │
│  │  → no plaintext ever logged or emitted       │   │
│  │                                              │   │
│  │  KVStore: addr → compressed TFHE ciphertext  │   │
│  └──────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────┘
```

### Technology Stack

| Layer | Technology |
|---|---|
| Consensus | CometBFT v0.37 + Green BFT adaptive timeout |
| Application | Cosmos SDK v0.47 |
| FHE Scheme | TFHE (Threshold FHE, LWE-based) via Lattigo v4 BFV → TFHE migration |
| PQC Signing | CRYSTALS-Dilithium3 (NIST PQC Round 3 winner) |
| PQC Hashing | SHA3-256 (128-bit Grover-resistant) |
| Chain ID | `zytherion` |
| Token | `zytc` (micro-denom) |

---

## Getting Started

### Prerequisites

- **Go** 1.21+
- **Ignite CLI** v0.27+ — `curl https://get.ignite.com/cli | bash`
- **Git**

```bash
git clone https://github.com/Zytherion/Zytherion-Blockchain
cd Zytherion-Blockchain
```

### Run a Local Node

```bash
ignite chain serve
```

This command scaffolds genesis, builds the binary, and starts a single-validator devnet. The node exposes:
- RPC: `http://localhost:26657`
- API: `http://localhost:1317`
- gRPC: `localhost:9090`

### Build Only

```bash
ignite chain build
```

Binary is placed at `$(go env GOPATH)/bin/zytheriond`.

---

## Usage

### 1. Deposit — Enter the Privacy Pool

Lock plaintext ZYTC into an encrypted balance. The amount is encrypted client-side and stored as a TFHE ciphertext on-chain.

```bash
# First attempt — will fail with out of gas (FHE is compute-heavy)
zytheriond tx privacy deposit 1000000zytc --from alice -y

# Correct — always set explicit gas limit for FHE transactions
zytheriond tx privacy deposit 1000000zytc --from alice -y --gas 1000000
```

> **Note:** FHE transactions require significantly more gas than standard transfers.
> Always use `--gas 1000000` for deposits and `--gas 2500000` (or `--gas auto --gas-adjustment 1.5`) for encrypted transfers.

### 2. Encrypted Transfer — Move Funds Without Revealing Amount

```bash
zytheriond tx privacy encrypted-transfer <RECIPIENT_ADDRESS> <AMOUNT> \
  --from alice -y \
  --gas auto --gas-adjustment 1.5
```

**Example:**
```bash
zytheriond tx privacy encrypted-transfer \
  zyth17w9ztef6evu7lvuwgqyv8u8uvyksnlrzn2rpy5 \
  250000 \
  --from alice -y \
  --gas auto --gas-adjustment 1.5
```

Expected output:
```
Initialising TFHE context and encrypting amount (this may take a moment)...
Amount 250000 encrypted (21645 bytes). Broadcasting transaction...
gas estimate: 2424820
code: 0
txhash: 1CE17949...
```

The on-chain event reveals only sender and recipient — never the amount:
```yaml
type: encrypted_transfer
attributes:
  - key: sender
    value: zyth17nea9...
  - key: recipient
    value: zyth17w9z...
```

### 3. Query Encrypted Balance

```bash
zytheriond q privacy show-encrypted-balance <ADDRESS>
```

Returns the raw TFHE ciphertext hex and its size in bytes. The ciphertext is only decryptable by the holder of the corresponding secret key.

### 4. Query a Transaction

```bash
zytheriond q tx <TXHASH>
```

### 5. Faucet (Devnet)

The devnet faucet is funded from `community_pool` and dispenses 10 ZYTC per request via Ignite's built-in faucet on port `4500`.

---

## Gas Reference

| Transaction Type | Approximate Gas |
|---|---|
| Standard ZYTC transfer | ~50,000 – 80,000 |
| `MsgDeposit` (FHE encrypt) | ~700,000 – 800,000 |
| `MsgEncryptedTransfer` (FHE add/sub) | ~1,500,000 – 1,700,000 |

FHE gas overhead is inherent to homomorphic computation. Use `--gas auto --gas-adjustment 1.5` to let the node estimate automatically.

---

## Tokenomics

**Total Supply: 1,000,000,000 ZYTC**

| Allocation | Amount | % | Notes |
|---|---|---|---|
| Community / Ecosystem | 450,000,000 ZYTC | 45% | Public sale, grants, liquidity |
| Staking Rewards | 250,000,000 ZYTC | 25% | Distributed to validators over time |
| Development Fund | 150,000,000 ZYTC | 15% | Zytherion Foundation |
| Team & Founders | 100,000,000 ZYTC | 10% | 4-year vesting |
| Public Goods | 50,000,000 ZYTC | 5% | Open-source & community funding |

- **Bond denom:** `zytc`
- **Mint denom:** `zytc`
- **Gov min deposit:** 100 ZYTC
- **Faucet:** 10 ZYTC per request (devnet)

---

## Module Overview

### `x/privacy`

The core privacy module implementing TFHE-based encrypted balances.

| Message | Description |
|---|---|
| `MsgDeposit` | Lock ZYTC from standard balance into encrypted balance pool |
| `MsgEncryptedTransfer` | Transfer between encrypted balances homomorphically |

| Query | Description |
|---|---|
| `show-encrypted-balance` | Returns TFHE ciphertext for an address |

**Privacy invariants enforced by the keeper:**
- Plaintext amounts are never written to KVStore
- Plaintext amounts are never emitted in ABCI events
- Plaintext amounts are never written to logs at `Info` level or above
- Decryption is only possible by the secret key holder (off-chain)

### `app/greenbft`

Green BFT adaptive timeout system.

| Component | Description |
|---|---|
| `AdaptiveTimeoutManager` | Tracks rolling 10-block tx average, recommends commit timeout |
| `PQCAnteDecorator` | Computes SHA3-256 fingerprint of each tx at ante stage |
| `BaseAnteDecorator` | Wraps standard Cosmos ante handler, increments tx counter |

Timeout logic:
- Rolling avg < 5 tx/block → recommend `5s` (idle)
- Rolling avg ≥ 5 tx/block → recommend `1s` (normal)

The recommended timeout is persisted to KVStore and emitted as ABCI event `green_bft.recommended_timeout_ms` every block.

---

## Security Notes

- **Quantum resistance:** Dilithium3 signing protects against Shor's algorithm. SHA3-256 hashing is Grover-resistant (128-bit quantum security).
- **FHE scheme:** TFHE is based on Torus-LWE, a post-quantum hardness assumption. Encrypted balances are semantically secure under LWE.
- **Secret key management:** The TFHE secret key is generated per-node at startup and never leaves the node. It is never stored on-chain, never emitted in events, and never logged.
- **Ciphertext size:** Each encrypted balance entry is approximately 21KB on-chain. This is an inherent property of FHE and is acknowledged in the [whitepaper](https://zytherion.pages.dev/Zytherion_White_Paper.pdf).

---

## Project Structure

```
Zytherion-Blockchain/
├── app/
│   ├── app.go                  # Main application wiring
│   └── greenbft/
│       ├── adaptive_commit.go  # Green BFT timeout manager
│       ├── adaptive_commit_test.go
│       ├── ante_pqc.go         # PQC SHA3-256 ante decorator
│       └── base_ante_decorator.go
├── x/
│   └── privacy/
│       ├── keeper/
│       │   ├── keeper.go       # TFHE-backed encrypted balance storage
│       │   └── msg_server.go   # MsgDeposit, MsgEncryptedTransfer handlers
│       ├── fhe/
│       │   └── fhe.go          # TFHE context: Encrypt, Decrypt, Add, Sub
│       ├── pqc/                # SHA3-256 PQC block hash
│       └── types/              # Message types, interfaces, store keys
├── config.yml                  # Genesis accounts and tokenomics
└── docs/
    └── static/openapi.yml
```

---

## Contributing

Contributions are welcome. Please open an issue before submitting a pull request for significant changes.

1. Fork the repository
2. Create a feature branch: `git checkout -b feat/your-feature`
3. Commit your changes: `git commit -m "feat: add your feature"`
4. Push and open a PR

Please ensure `go build ./...` and `go test ./...` pass before submitting.

---

## License

MIT License — see [LICENSE](LICENSE) for details.

---

<div align="center">
  <img src="https://zytherion.pages.dev/logo_zythc.png" alt="Zytherion" width="48"/>
  <br/>
  <sub>Built with Cosmos SDK · Secured by Post-Quantum Cryptography · Powered by Torus FHE</sub>
</div>
