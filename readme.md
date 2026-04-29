<div align="center">

<img src="https://zytherion.pages.dev/logo_zythc.png" alt="Zytherion Logo" width="120" />

# Zytherion Blockchain

**A next-generation, privacy-first blockchain built on Cosmos SDK — secured by Fully Homomorphic Encryption, Torus FHE, and LWE-based post-quantum cryptography.**

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Built with Cosmos SDK](https://img.shields.io/badge/Cosmos%20SDK-v0.47-purple)](https://docs.cosmos.network/)
[![TFHE-rs](https://img.shields.io/badge/TFHE--rs-v0.7-orange)](https://github.com/zama-ai/tfhe-rs)
[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://go.dev/)
[![Rust](https://img.shields.io/badge/Rust-Stable-orange?logo=rust)](https://www.rust-lang.org/)
[![Website](https://img.shields.io/badge/Website-zytherion.pages.dev-green)](https://zytherion.pages.dev/)

</div>

---

## Overview

Zytherion is a custom Cosmos SDK-based blockchain that enforces **cryptographic privacy at the protocol level**. Unlike conventional blockchains where all state is publicly observable, Zytherion stores and processes balances as **encrypted ciphertexts** — validators can verify correctness of transactions *without ever learning the plaintext amounts*.

**Core innovations:**

| Pillar | Technology | Purpose |
|--------|-----------|---------|
| Fully Homomorphic Encryption | [TFHE-rs v0.7](https://github.com/zama-ai/tfhe-rs) via CGo FFI | Encrypted balance arithmetic on-chain |
| Torus FHE / LWE Cryptography | TFHE FheUint64 scheme | Post-quantum-resistant ciphertext operations |
| Homomorphic Transfers | Add/Sub in ciphertext space | Transfer value without revealing amount |
| LWE Block Hashing | ABCI 2.0 PrepareProposal | LWE-SHA3 hybrid sentinel per block |
| TFHE Compression | In-library compression via TFHE-rs | 30-50 KB to 1-5 KB ciphertext storage |
| Green BFT Consensus | CometBFT | Energy-efficient Byzantine fault-tolerant consensus |

---

## Key Features

### Encrypted Balance Privacy
Every account's balance is stored on-chain as a **compressed TFHE ciphertext**. No validator, block explorer, or observer can determine a wallet's balance from chain state alone.

```
On-chain state:  [ Enc(balance_A) ]  [ Enc(balance_B) ]
Observers see:   [ 1-5 KB blob   ]  [ 1-5 KB blob   ]
                      |                   |
               Only the key holder can decrypt
```

### Homomorphic Transfers
The `EncryptedTransfer` message allows value transfer without revealing amounts:

```
newSenderBal    = Enc(senderBal)    - Enc(amount)   (all in ciphertext space)
newReceiverBal  = Enc(receiverBal)  + Enc(amount)
```

Validators only learn: *"a transfer happened from address A to address B."*

### TFHE Ciphertext Compression
Raw TFHE ciphertexts are large (~30-50 KB per value). Zytherion applies **in-library compression** at the point of encryption, reducing on-chain storage to **~1-5 KB per ciphertext** — making FHE-based storage economically viable.

```
Encrypt(value) -> raw ciphertext (~30-50 KB)
                         |
                    compress()
                         |
           compressed ciphertext (~1-5 KB)  <- stored in KVStore
```

### LWE Block Hashing (ABCI 2.0)
Every block is anchored with an **LWE-SHA3 hybrid sentinel** injected as the first transaction via `PrepareProposal`. The sentinel is independently verified in `ProcessProposal` and the audit hash is persisted in the `PrivacyKeeper` store, anchoring it in the `AppHash`.

---

## Architecture

```
zytherion/
|-- Makefile                          # Build targets (build-tfhe, build, test, lint)
|-- tfhe-cgo/                         # Rust FFI crate - TFHE-rs static library
|   |-- Cargo.toml                    # tfhe = { version = "0.7", features = ["integer", "x86_64-unix"] }
|   |-- build.sh                      # Compiles libtfhe_cgo.a
|   `-- src/
|       `-- lib.rs                    # C-exported FFI: generate_keys, encrypt, decrypt, add, sub, compress, decompress
`-- x/
    `-- privacy/                      # Privacy Cosmos SDK module
        |-- client/
        |   `-- cli/
        |       `-- tx_encrypted_transfer.go  # CLI: submit EncryptedTransfer with plaintext amount
        |-- fhe/
        |   |-- fhe.go                # FHE Context: NewContext, Encrypt, Decrypt, AddCiphertexts, SubCiphertexts
        |   |-- tfhe_binding.go       # CGo bridge to libtfhe_cgo.a (real build)
        |   |-- tfhe_binding_mock.go  # Mock binding (notfhe build tag - no CGo needed)
        |   |-- tfhe_compress.go      # compressCiphertext / decompressCiphertext wrappers
        |   |-- tfhe_compress_mock.go # Mock compression
        |   |-- fhe_mock.go           # Mock FHE context for testing
        |   `-- fhe_test.go           # Integration tests
        `-- keeper/
            |-- keeper.go             # HomomorphicAdd, HomomorphicSub, EncryptAmount, DecryptBalance
            `-- msg_server_encrypted_transfer.go  # EncryptedTransfer message handler
```

---

## Technical Stack

### Rust FFI Layer (tfhe-cgo)

The core cryptographic engine is a **Rust static library** compiled from [Zama's TFHE-rs](https://github.com/zama-ai/tfhe-rs) and linked into the Go node via CGo:

```toml
# Cargo.toml
[dependencies]
tfhe    = { version = "0.7", features = ["integer", "x86_64-unix"] }
bincode = "1"

[lib]
crate-type = ["staticlib"]   # -> libtfhe_cgo.a

[profile.release]
opt-level      = 3
lto            = true
codegen-units  = 1
```

**Exported C functions:**

| Function | Description |
|----------|-------------|
| `tfhe_generate_keys` | Generate client/server key pair (serialised with bincode) |
| `tfhe_encrypt_u64` | Encrypt u64 value to raw ciphertext |
| `tfhe_decrypt_u64` | Decrypt raw ciphertext to u64 |
| `tfhe_add` | Homomorphic addition of two ciphertexts |
| `tfhe_sub` | Homomorphic subtraction of two ciphertexts |
| `tfhe_compress` | Compress a raw ciphertext to compact form |
| `tfhe_decompress` | Decompress compact ciphertext to operable form |
| `tfhe_free_bytes` | Free memory allocated by the library |

### Go FHE Layer (x/privacy/fhe)

The `fhe.Context` struct wraps the CGo calls and provides a clean Go API:

```go
ctx, _ := fhe.NewContext()           // Generate fresh key pair (~2-10 sec)

ct, _  := ctx.Encrypt(1000)          // Returns compressed ciphertext (~1-5 KB)
val, _ := ctx.Decrypt(ct)            // val == 1000

sum, _ := ctx.AddCiphertexts(a, b)   // Homomorphic addition
dif, _ := ctx.SubCiphertexts(a, b)   // Homomorphic subtraction
```

All operations are **concurrency-safe** (mutex-guarded for arithmetic ops).

### Privacy Keeper (x/privacy/keeper)

The keeper bridges the FHE layer with Cosmos SDK KVStore:

```go
// Encrypt and store
k.EncryptAmount(1000)                           // -> compressed TFHE bytes

// Homomorphic balance update (no plaintext ever leaves the keeper)
k.HomomorphicAdd(ctx, recipientAddr, ctAmount)  // receiverBal += amount
k.HomomorphicSub(ctx, senderAddr, ctAmount)     // senderBal   -= amount

// Decrypt (only the key holder can do this)
k.DecryptBalance(ctx, addr)                     // -> uint64
```

Encrypted balances are stored under `types.EncryptedBalanceKey(addr)` in the module's KVStore. **The keeper never decrypts balances during normal operation.**

---

## Getting Started

### Prerequisites

| Requirement | Version | Notes |
|------------|---------|-------|
| Go | >= 1.21 | `go version` |
| Rust + Cargo | Stable | `rustup toolchain install stable` |
| Ignite CLI | Latest | `curl https://get.ignite.com/cli! | bash` |
| GCC / Clang | System default | Required for CGo linking |

### 1. Clone the Repository

```bash
git clone https://github.com/Zytherion/Zytherion-Blockchain.git
cd Zytherion-Blockchain
```

### 2. Build the TFHE Static Library

This compiles the Rust FFI crate into `libtfhe_cgo.a`. **Run this before any Go build.**

```bash
make build-tfhe
```

> First build takes 2-5 minutes — Rust must compile TFHE-rs from source.

### 3. Build the Node

```bash
make build
# or
go build ./...
```

### 4. Run the Chain (Development)

```bash
ignite chain serve
```

### 5. Run Tests

```bash
# Full test suite (requires libtfhe_cgo.a)
make test

# Tests without CGo (uses mock FHE)
go test -tags notfhe ./...
```

---

## CLI Usage

### Send an Encrypted Transfer

The CLI accepts a **plaintext amount** and performs on-the-fly FHE encryption internally:

```bash
zytherion tx privacy encrypted-transfer [recipient] [amount] \
  --from [sender-key] \
  --chain-id zytherion \
  --gas auto
```

The amount is encrypted client-side using the node's FHE context before being submitted as `MsgEncryptedTransfer`. Validators and observers only see the encrypted ciphertext bytes.

### Query Encrypted Balance

```bash
zytherion query privacy encrypted-balance [address]
```

Returns the raw compressed ciphertext bytes. Decryption requires the private client key.

---

## Security Model

### Privacy Guarantees

| What validators know | What validators do NOT know |
|---------------------|---------------------------|
| Transfer occurred (sender -> recipient) | Transfer amount |
| Sender had a balance | Sender's balance |
| Recipient received funds | Recipient's balance |
| Ciphertext size (~1-5 KB) | Any plaintext value |

### Key Management

- **Client key** (`clientKeyBytes`): Secret key — held only by the account owner. MUST NOT be logged, emitted in events, or stored in KVStore.
- **Server key** (`serverKeyBytes`): Evaluation key — used by the node to perform homomorphic operations. Can be shared with the TFHE-rs computation layer.
- **Key lifetime**: Keys are currently ephemeral per-process. For production, persist key blobs and reconstruct via `fhe.NewContextFromKeys(ck, sk)`.

### Build Tags

```bash
# Real TFHE (requires Rust + libtfhe_cgo.a):
go build ./...

# Mock FHE (CI, testing, no CGo):
go build -tags notfhe ./...
```

---

## Development

### Makefile Reference

```bash
make build-tfhe   # Compile Rust -> libtfhe_cgo.a (run once)
make build        # build-tfhe + go build ./...
make test         # go test ./...
make lint         # golangci-lint run ./...
```

### Adding a New FHE Operation

1. Add the C FFI function in `tfhe-cgo/src/lib.rs` with `#[no_mangle] pub extern "C"`.
2. Declare the CGo import in `x/privacy/fhe/tfhe_binding.go`.
3. Add the mock equivalent in `x/privacy/fhe/tfhe_binding_mock.go` (build tag `notfhe`).
4. Expose a high-level method on `fhe.Context` in `x/privacy/fhe/fhe.go`.
5. Run `make build-tfhe && make test`.

---

## Roadmap

- [x] TFHE-rs CGo FFI integration (FheUint64)
- [x] Compressed ciphertext storage (~1-5 KB on-chain)
- [x] Homomorphic encrypted transfers (Add/Sub in ciphertext space)
- [x] LWE-SHA3 hybrid block hashing via ABCI 2.0
- [x] CLI EncryptedTransfer with plaintext input
- [ ] Persistent key management (key derivation from mnemonic)
- [ ] Multi-validator FHE threshold decryption
- [ ] ZK proof integration for balance range proofs
- [ ] IBC-compatible encrypted cross-chain transfers
- [ ] TFHE FheUint32 migration for further ciphertext size reduction (~5 KB target)
- [ ] ZSTD compression at KVStore boundary

---

## Links

| Resource | URL |
|----------|-----|
| Official Website | [zytherion.pages.dev](https://zytherion.pages.dev/) |
| GitHub Repository | [github.com/Zytherion/Zytherion-Blockchain](https://github.com/Zytherion/Zytherion-Blockchain) |
| Cosmos SDK Docs | [docs.cosmos.network](https://docs.cosmos.network/) |
| TFHE-rs | [github.com/zama-ai/tfhe-rs](https://github.com/zama-ai/tfhe-rs) |
| CometBFT | [github.com/cometbft/cometbft](https://github.com/cometbft/cometbft) |

---

## License

This project is licensed under the **MIT License** — see the [LICENSE](LICENSE) file for details.

---

<div align="center">

*Built with love by the Zytherion Team · Powered by TFHE-rs, Cosmos SDK & CometBFT*

**[Visit zytherion.pages.dev](https://zytherion.pages.dev/)**

</div>
