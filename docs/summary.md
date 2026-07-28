# Zytherion Blockchain and Cryptocurrency — Full Technical Summary

**Version:** 0.6.0 | **Token:** ZYTC | **Chain ID:** `zytherion`  
**Founder:** Rayhan Aziel Abbrar  
**Repository:** https://github.com/Zytherion/Zytherion-Blockchain  
**Stack:** Cosmos SDK v0.47 · CometBFT v0.37 (QuantumBFT) · Go · Rust (TFHE via CGo)

---

## Abstract

**Zytherion** adalah Layer-1 blockchain generasi berikutnya yang dibangun di atas Cosmos SDK dan CometBFT, menggabungkan kriptografi post-quantum, enkripsi homomorfik penuh (FHE), stablecoin multi-kolateral, dan smart contract dalam satu arsitektur terintegrasi.

Proyek ini dimulai sebagai eksplorasi kriptografi terapan dan berkembang menjadi protokol blockchain lengkap dengan tujuh fase evolusi:
- **v0.1–v0.2**: Fondasi post-quantum (LWR hashing, PoVL VDF, Dilithium)
- **v0.3**: Homomorphic encryption + Erasure Coding (penggantian ZK-SNARK)
- **v0.4**: Pengerasan shard storage (Merkle integrity, auth P2P, quota)
- **v0.5**: Ekosistem DeFi (Oracle, IBC Collateral, Stablecoin, CosmWasm)
- **v0.5.1**: API Layer + Worker Pool concurrency
- **v0.5.2**: Security patch — dua kerentanan kritis dipatch sebelum mainnet
- **v0.5.3**: TFHE Always-On + API Status Endpoint
- **v0.6**: QuantumBFT — Post-quantum consensus signing (Dilithium5) menggantikan Ed25519 CometBFT

---

## Full Version Changelog

### v0.6 — QuantumBFT Consensus Engine (2026)

**Menggantikan Ed25519 validator consensus signing dengan Dilithium5 (ML-DSA-87, NIST Category 5).**

| Perubahan | Status | Detail |
|---|---|---|
| QuantumBFT Engine | 🚀 **MAJOR** | Custom `PrivValidator` (`QuantumFilePV`) berbasis Dilithium5 menggantikan Ed25519 FilePV CometBFT |
| `PubKey` & `PrivKey` | 🆕 **NEW** | `quantumbft.PubKey` (2592B) & `quantumbft.PrivKey` (4864B) mengimplementasikan interface `cmtcrypto` |
| `quantum_validator_key.json` | 🆕 **NEW** | Key file format baru menyimpan pasangan kunci Dilithium5 validator |
| `quantum_validator_state.json` | 🆕 **NEW** | Double-sign protection state tracker khusus QuantumBFT |
| CLI `quantumbft` | 🆕 **NEW** | Subcommands `zytheriond quantumbft (init/show/validate)` untuk manajemen kunci validator |
| Startup Detection | 🆕 **NEW** | Node otomatis mendeteksi `quantum_validator_key.json` saat startup dan mengaktifkan QuantumBFT |

---

### v0.5.3 — TFHE Always-On & API Status Endpoint (2026)

**TFHE tidak lagi membutuhkan flag khusus — subsystem selalu aktif.**

| Perubahan | Status | Detail |
|---|---|---|
| TFHE Always-On | 🔥 **BREAKING** | Hapus flag `--enable-tfhe`. TFHE aktif secara default tanpa flag apapun |
| REST API TFHE Status | 🆕 **NEW** | `GET /zytherion/privacy/v1/tfhe/status` — endpoint real-time status TFHE |
| `TFHEStatusHTTPHandler` | 🆕 **NEW** | HTTP handler di `x/privacy/keeper/query_tfhe_status.go` |
| Startup Banner | ✏️ **UPDATED** | Log startup kini menampilkan versi `v0.5.3` dan TFHE sebagai `ALWAYS ENABLED` |
| `ErrTFHEDisabled` (types) | ❌ **REMOVED** | Error runtime ini dihapus karena TFHE tidak bisa dinonaktifkan lagi |
| `addModuleInitFlags` | ✏️ **UPDATED** | Flag `enable-tfhe` dihapus dari definisi CLI |
| `NewKeeper` signature | ✏️ **UPDATED** | Parameter `enableTFHE bool` dihapus — shard store selalu diinisialisasi |
| `IsTFHEEnabled()` | ✏️ **UPDATED** | Selalu return `true` |

---

### v0.5.2 — Security Patch (2026)

**Dua CVE kritis ditemukan dan dipatch sebelum mainnet.**

| Perubahan | Status | Detail |
|---|---|---|
| CVE-ZYTH-001: Underflow Protection | 🔒 **PATCHED** | `PublicCreditLimit` guard mencegah SubUint32 wrap-around fraud |
| CVE-ZYTH-002: User-Held Decryption Keys | 🔒 **PATCHED** | `CompressedPublicKey` per-user, node operator tidak bisa decrypt balance user |
| Built-in TFHE Client CLI | 🆕 **NEW** | Subcommands `zytheriond keys tfhe (keygen/encrypt/decrypt)` terintegrasi langsung |
| `ZYTDAccountState` struct | 🆕 **NEW** | Menggantikan `EncryptedBalance`, menambah `PublicCreditLimit` field |
| `MsgRegisterUserTFHEPublicKey` | 🆕 **NEW** | User mendaftarkan TFHE public key mereka on-chain sebelum minting |
| `tfhe_encrypt_u32_pk` (Rust FFI) | 🆕 **NEW** | Enkripsi via CompressedPublicKey tanpa memerlukan ClientKey node |
| `EncryptWithPublicKey` (Go) | 🆕 **NEW** | Go wrapper untuk enkripsi menggunakan public key user |
| User TFHE Public Key Registry | 🆕 **NEW** | KV store: `tfhe_pubkey/<address>` → CompressedPublicKey bytes |

---

### v0.5.1 — Ecosystem Tooling & Concurrent TFHE (2026)

**Upgrade arsitektur performa + dokumentasi API lengkap.**

| Perubahan | Status | Detail |
|---|---|---|
| TFHE Worker Pool | 🆕 **NEW** | OS-thread-pinned pool (`max(1, NumCPU-2)`), hapus global mutex `tfheMu` |
| `EnsureNodeKeys` singleton | 🆕 **NEW** | Singleton load/generate TFHE key pair, thread-safe, disk-persistent |
| `SubUint32` + `tfhe_sub_u32` | 🆕 **NEW** | Homomorphic subtraction (Rust FFI + Go wrapper) |
| State Rent (`StateRentParams`) | 🆕 **NEW** | Ekonomi penyimpanan: 1 uzytc/byte/block, 1KB free tier, 14400 block grace period |
| Confidential ZYTD (MVP) | 🆕 **NEW** | Balance ZYTD dienkripsi sebagai FheUint32, mint dan transfer via TFHE |
| API_WALLET.md | 🆕 **NEW** | Referensi lengkap REST/RPC untuk wallet developer |
| API_EXPLORER.md | 🆕 **NEW** | Referensi endpoint block explorer, CosmWasm, privacy events |
| API_FAUCET.md | 🆕 **NEW** | Panduan faucet, Dilithium5 signing, rate limiting |
| DEMO_GUIDE.md | 🆕 **NEW** | Walkthrough end-to-end: account → homomorphic smart contract |

---

### v0.5.0 — Module Ecosystem Expansion (2026)

**Ekspansi menjadi ekosistem DeFi lengkap.**

| Perubahan | Status | Detail |
|---|---|---|
| Price Oracle (`x/oracle`) | 🆕 **NEW** | Validator-submitted price feed, median TWAP 30-block window |
| IBC Collateral Vault (`x/ibc-collateral`) | 🆕 **NEW** | ICS-20 middleware, vault module account, CollateralPosition KV |
| Multi-Collateral Stablecoin (`x/stablecoin`) | 🆕 **NEW** | Mint/burn/liquidate ZYTD, collateral ratio enforcement |
| CosmWasm Integration | 🆕 **NEW** | `wasmd` v0.45.0 + `wasmvm` v1.5.2, permissioned upload |
| TFHE CosmWasm Plugin | 🆕 **NEW** | Custom query plugin: kontrak bisa query ciphertext dan trigger eval |
| Homomorphic Smart Contract | 🆕 **NEW** | Kontrak Rust yang melakukan komputasi di atas data terenkripsi |

---

### v0.4.0 — Shard Security Hardening (2026)

**Pengerasan layer distribusi shard terhadap tampering dan availability attack.**

| Perubahan | Status | Detail |
|---|---|---|
| Merkle Tree Shard Integrity | 🆕 **NEW** | Binary SHA-256 Merkle tree atas 16 shard, root disimpan on-chain |
| Per-shard Merkle proof verification | 🆕 **NEW** | Receiver memverifikasi proof sebelum store ke disk |
| Authenticated P2P | 🆕 **NEW** | Bearer-token auth (`Authorization: Bearer <nodeID>`) wajib untuk POST shard |
| Rate Limiter | 🆕 **NEW** | Max 60 req/menit per IP di shard HTTP server |
| Per-Address TFHE Quota | 🆕 **NEW** | Max 1 active commitment per address, `ErrTFHEQuotaExceeded` (code 1205) |
| ReplicationFactor ditingkatkan | ✏️ **UPDATED** | 3 → 4 peer per shard untuk ketersediaan lebih tinggi |
| Reed-Solomon diubah | ✏️ **UPDATED** | Data:Parity = 10+6 → 12+4 (toleransi 4 node offline) |
| Gas charge diupdate | ✏️ **UPDATED** | 1,500 gas/KB ciphertext untuk `MsgTFHESubmit` |

---

### v0.3.0 — TFHE & Erasure Coding (2026)

**Penggantian ZK-SNARK dengan TFHE. Penambahan distribusi shard pertama.**

| Perubahan | Status | Detail |
|---|---|---|
| CRYSTALS-Dilithium5 | ✏️ **UPGRADED** | Dari Dilithium3 (Cat-3) → Dilithium5 (Cat-5, ~256-bit PQ) |
| ZK-SNARK Groth16/BN254 | ❌ **REMOVED** | Dihapus karena membutuhkan trusted setup; gnark deps dihapus total |
| TFHE Engine (`tfhe-rs`) | 🆕 **NEW** | FheUint32 via CGo static linking `libtfhe_c.a`, ~21 KB ciphertext |
| Erasure Coding | 🆕 **NEW** | Reed-Solomon 10+6=16 shard, toleransi 6 node offline |
| P2P Shard Distribution | 🆕 **NEW** | Local ShardStore + HTTP gossiping + on-demand reconstruction |
| `--enable-tfhe` flag | 🆕 **NEW** | Default OFF. Wajib aktifkan untuk fitur TFHE |
| `tx/tfhe-submit` RPC | 🆕 **NEW** | Submit dan distribusikan ciphertext TFHE ke jaringan |
| `query/tfhe-result` RPC | 🆕 **NEW** | Rekonstruksi on-demand ciphertext TFHE |

---

### v0.2.0 — GreenBFT & Dilithium3

**Penguatan konsensus dan pengenalan signature post-quantum pertama.**

| Perubahan | Status | Detail |
|---|---|---|
| GreenBFT Consensus | 🆕 **NEW** | Energy-efficient BFT consensus menggantikan vanilla Tendermint |
| CRYSTALS-Dilithium3 | 🆕 **NEW** | Signature post-quantum pertama, NIST Category 3 (~192-bit PQ) |
| PQC AnteDecorator | 🆕 **NEW** | Middleware yang memvalidasi Dilithium3 signature di setiap transaksi |
| ZK-SNARK Groth16/BN254 | ✅ **ACTIVE** | Pertama kali diintegrasikan (kemudian dihapus di v0.3) |

---

### v0.1.0 — Fondasi Post-Quantum (2025)

**Arsitektur dasar blockchain post-quantum.**

| Perubahan | Status | Detail |
|---|---|---|
| Ring-LWR Hashing | 🆕 **NEW** | Hash post-quantum berbasis Learning With Rounding, output 96 byte |
| PoVL Sequential VDF | 🆕 **NEW** | Proof of Verifiable Lattices, N=10 iterasi sequential per blok |
| ABCI 2.0 Hooks | 🆕 **NEW** | PrepareProposal + ProcessProposal untuk enforce VDF |
| Cosmos SDK bootstrap | 🆕 **NEW** | Chain ID `zytherion`, token `ZYTC`, modul `x/privacy` |

---

## Architecture Overview (v0.5.2)

```
┌──────────────────────────────────────────────────────────────────┐
│                    Client Layer                                  │
│   CLI zytheriond | REST :1317 | RPC :26657 | gRPC :9090         │
│   CosmWasm Contracts (Rust/Wasm) | TFHE Query Plugin            │
└──────────────────────────────┬───────────────────────────────────┘
                               │ ABCI 2.0
┌──────────────────────────────▼───────────────────────────────────┐
│                  CometBFT (GreenBFT)                            │
│          PrepareProposal → PoVL VDF Compute                     │
│          ProcessProposal → PoVL VDF Verify                      │
└──────────────────────────────┬───────────────────────────────────┘
                               │
┌──────────────────────────────▼───────────────────────────────────┐
│                Cosmos SDK Application Core                      │
│                                                                  │
│  ┌──────────────┐  ┌────────────┐  ┌────────────┐  ┌─────────┐  │
│  │  x/privacy   │  │ x/oracle   │  │x/stablecoin│  │ x/wasm  │  │
│  │  Dilithium5  │  │ Price TWAP │  │ ZYTD Mint  │  │CosmWasm │  │
│  │  LWR Hash    │  │ 30-block   │  │ Burn/Liq   │  │v0.45.0  │  │
│  │  PoVL VDF    │  │ Median     │  │ Conf.ZYTD  │  │         │  │
│  │  TFHE Engine │  └────────────┘  └────────────┘  └─────────┘  │
│  │  Worker Pool │  ┌────────────────────────────┐               │
│  │  State Rent  │  │   x/ibc-collateral         │               │
│  │  User PK Reg │  │   ICS-20 Middleware        │               │
│  └──────────────┘  │   Vault Module Account     │               │
│                    └────────────────────────────┘               │
│                                                                  │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │            TFHE Shard Storage Layer (v0.4+)               │  │
│  │  RS(12+4) → Merkle Tree → 16 Shards → Auth P2P → Disk    │  │
│  │  ReplicationFactor=4 | Quota/Address | Rate Limiter       │  │
│  └────────────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────────┘
                               │
┌──────────────────────────────▼───────────────────────────────────┐
│              Rust Library Layer (CGo Static Linking)            │
│   libtfhe_c.a: encrypt_u32 | encrypt_u32_pk | add_u32          │
│               sub_u32 | mul_scalar_u32 | decrypt_u32           │
│               keygen | ciphertext_max_bytes                     │
└──────────────────────────────────────────────────────────────────┘
```

---

## 1. Post-Quantum Cryptography Stack

### 1.1 CRYSTALS-Dilithium5 (ML-DSA Level 5)

Signature post-quantum utama Zytherion. Di-upgrade dari Dilithium3 di v0.3 sebagai breaking change pre-mainnet.

| Metrik | Dilithium3 (v0.2) | Dilithium5 (v0.3+) | FIPS 204 (ML-DSA-87) |
|---|---|---|---|
| NIST Level | Category 3 | **Category 5** | ML-DSA-87 |
| Keamanan PQ | ~192-bit | **~256-bit** | ~256-bit |
| Public Key | 1952 bytes | **2592 bytes** | 2592 bytes |
| Private Key | 4000 bytes | **4864 bytes** | 4864 bytes |
| Signature | 3293 bytes | **4595 bytes** | 4595 bytes |
| Library | cloudflare/circl (mode3) | **cloudflare/circl (mode5)** | — |

**Contoh generate key:**
```bash
zytheriond keys add zhaomei --key-type dilithium5
# Output: address: zyth1w30wreu6gqm0nd962wtsez9c9p7dwhqdyn9agu
#         pubkey: {"@type":"/zytherion.crypto.dilithium5.PubKey","key":"..."}
```

### 1.2 Ring-LWR Hashing

Hash post-quantum berbasis Learning With Rounding (LWR). Digunakan untuk block integrity dan sebagai basis PoVL VDF.

**Formula:**
$$H_n = \text{SHA3-256}(\text{LWR}(data_n) \| H_{n-1})$$

**Parameter:**
- Ring dimension: $n = 256$
- Modulus: $q = 3329$ (Kyber-compatible)
- Rounding modulus: $p = 256$
- Output size: 96 bytes (32-byte seed + 64-byte vector $b$)

### 1.3 Proof of Verifiable Lattices (PoVL)

Sequential VDF berbasis Ring-LWR. Setiap blok membutuhkan $N = 10$ iterasi hash berurutan (tidak bisa di-parallelize). Divalidasi di ABCI `ProcessProposal` sebelum voting konsensus.

```
[ Block Proposer ]
  └─► PrepareProposal:
        1. Compute LWR Hash atas data blok
        2. Compute PoVL: jalankan N=10 iterasi berurutan
        3. Pack VDF proof ke dalam proposal
[ Validator Nodes ]
  └─► ProcessProposal:
        1. Ekstrak VDF proof
        2. Verifikasi — jika valid → ACCEPT; jika tidak → REJECT
```

---

## 2. TFHE Homomorphic Encryption Engine

### 2.1 Library & FFI

Library: Zama's `tfhe-rs` via CGo static linking (`libtfhe_c.a`).  
Build tag: `tfhe_cgo`. Tanpa tag ini, `engine_stub.go` dipakai (semua operasi return `ErrTFHEDisabled`).

```go
// engine.go
/*
#cgo CFLAGS:  -I${SRCDIR}
#cgo LDFLAGS: -L${SRCDIR}/tfhe_c/target/release -ltfhe_c -lm -ldl -lpthread
#include "cgo_bridge.h"
*/
import "C"
```

### 2.2 Exposed Operations

| Fungsi Go | Fungsi Rust FFI | Keterangan |
|---|---|---|
| `GenerateKeys()` | `tfhe_keygen` | Generate ClientKey + ServerKey pair (~10–60 detik) |
| `EncryptUint32(ck, val)` | `tfhe_encrypt_u32` | Enkripsi dengan ClientKey node (~21 KB ciphertext) |
| `EncryptWithPublicKey(pk, val)` | `tfhe_encrypt_u32_pk` | **NEW v0.5.2**: enkripsi dengan CompressedPublicKey user |
| `EncryptWithServerKey(sk, val)` | *(menggunakan node CK)* | Enkripsi untuk operand TFHE evaluasi |
| `AddUint32(sk, ct1, ct2)` | `tfhe_add_u32` | Homomorphic addition: Enc(a) + Enc(b) = Enc(a+b) |
| `SubUint32(sk, ct1, ct2)` | `tfhe_sub_u32` | **NEW v0.5.1**: Homomorphic subtraction: Enc(a) - Enc(b) = Enc(a-b) |
| `MultiplyScalarUint32(sk, ct, s)` | `tfhe_mul_scalar_u32` | Scalar multiply: Enc(a) × s = Enc(a×s) |
| `DecryptUint32(ck, ct)` | `tfhe_decrypt_u32` | Dekripsi dengan ClientKey |
| `EnsureNodeKeys(home)` | — | **NEW v0.5.1**: Load atau generate node keys dari disk |

### 2.3 Worker Pool (v0.5.1 — Penghapusan Global Mutex)

**Problem lama (v0.5.0):** Satu `sync.Mutex` global (`tfheMu`) menserialisasikan semua panggilan CGo → bottleneck fatal untuk throughput tinggi → validator starvation → missed blocks.

**Insight kunci:** `set_server_key()` di `tfhe-rs` menggunakan thread-local storage. Beberapa goroutine yang masing-masing dikunci ke OS thread (`runtime.LockOSThread()`) dapat memanggil TFHE secara simultan tanpa konflik.

**Formula pool size:** `max(1, runtime.NumCPU() - 2)` — selalu sisakan 2 core untuk CometBFT + P2P.

```
Global Mutex (v0.5.0)          Worker Pool (v0.5.1+)
────────────────────           ──────────────────────────────
Tx A → [LOCK] → eval           Tx A → Worker-1 (OS Thread 1) → eval
Tx B → [WAIT]                  Tx B → Worker-2 (OS Thread 2) → eval (concurrent)
Tx C → [WAIT]                  Tx C → Worker-3 (OS Thread 3) → eval (concurrent)
```

---

## 3. Reed-Solomon Erasure Coding & P2P Shard Storage

### 3.1 Pipeline

```
Ciphertext (~21 KB)
      │
      ▼
Reed-Solomon Encode: 12 data + 4 parity = 16 shards (~1.75 KB each)
      │
      ▼
Build Merkle Tree (SHA-256) atas semua 16 shard hashes
      │
      ▼
Store seluruh 16 shard di local disk
      │
      ▼
Untuk setiap shard i: kirim ke (ReplicationFactor-1)=3 random peers
POST /shard { data_hex, merkle_root_hex, merkle_proof_hex }
                │
                ▼ (Di peer node)
           1. Verify Bearer token auth
           2. Rate check (≤60 req/min per IP)
           3. VerifyMerkleProof(root, i, data, proof)
           4. Store ke disk jika valid
```

**Rekonstruksi:** Minimum 12 dari 16 shard. Toleransi 4 node offline sekaligus.

### 3.2 Merkle Tree Integrity (v0.4)

```
       Root (SHA-256)
      /               \
   H(0-7)           H(8-15)
   /     \           /     \
 H(0-3) H(4-7)  H(8-11) H(12-15)
  / \    / \     / \      / \
 H0 H1  H2 H3  H8 H9  H12 H13 ...

Leaf Hᵢ = SHA-256(shard[i].Data)
Depth   = log₂(16) = 4
Proof   = 4 × 32 = 128 bytes per shard
```

Root Merkle disimpan on-chain di `TFHEShardMeta`. Receiver memverifikasi proof sebelum menulis ke disk — mencegah shard tampering/injection.

### 3.3 On-Chain Metadata

```json
KV: tfhe_meta/<commitmentHash> → {
    "commitment_hash": "<hex>",
    "original_len":    21504,
    "merkle_root":     "<hex 32 bytes>",
    "proposer_pubkey": "<hex dilithium5 pubkey>",
    "shard_node_map":  {
        "0": ["node1", "node7", "node12", "node15"],
        "1": ["node2", "node8", "node13", "node4"],
        ...
    }
}
```

---

## 4. DeFi Module Ecosystem (v0.5.0)

### 4.1 Price Oracle (`x/oracle`)

Validator-submitted price feed dengan median TWAP untuk mencegah flash loan manipulation.

**Whitelisted denoms:** `ZYTC`, `axlUSDC`, `mUSDT`, `ATOM`, `wBTC`, `wETH`

**Default parameters:**
```go
OracleParams{
    TwapWindowBlocks:  30,    // ~3 menit (6s/block)
    MinSubmissions:    2,     // minimum 2 validator submit
    MaxPriceAge:       5,     // tolak price > 5 block lama
    WhitelistedDenoms: [...], // daftar di atas
}
```

**TWAP computation:**
1. Kumpulkan semua `PriceEntry` untuk denom dalam window
2. Sort by PriceUSD, ambil median
3. Simpan ke `KV: twap/<denom>`
4. EndBlocker: prune entry lebih tua dari `window + maxAge`

**CLI:**
```bash
zytheriond tx oracle submit-price ATOM 8.50 --from validator1
zytheriond query oracle twap ATOM
```

### 4.2 IBC Collateral Vault (`x/ibc-collateral`)

ICS-20 middleware yang mengunci aset collateral incoming dari IBC.

```
IBC Transfer Packet (ATOM dari Cosmos Hub)
      │
      ▼ OnRecvPacket (ICS-20 Middleware)
Is token in whitelist?
      ├─► YES: Lock to vault module account "ibc_collateral_vault"
      │         Create CollateralPosition{Depositor, Denom, Amount}
      └─► NO:  Pass-through ke penerima normal
```

**CLI:**
```bash
# Lihat posisi collateral
zytheriond query ibc-collateral position <address> <denom>
zytheriond query ibc-collateral max-mintable <address> <ibc-denom>
```

### 4.3 Multi-Collateral Stablecoin (`x/stablecoin`)

Stablecoin `ZYTD` yang di-peg ke $1, di-back oleh IBC collateral.

**Formula mint:**
$$\text{max\_mintable} = \frac{\text{collateral\_amount} \times \text{TWAP\_price}}{\text{min\_collateral\_ratio}}$$

**Lifecycle:**
```
Lock IBC collateral (via IBC transfer)
      │
      ▼
MsgMintZYTD(amount, ibc_denom)
  ├─► Fetch TWAP price dari x/oracle
  ├─► Verify amount ≤ max_mintable
  └─► Mint ZYTD to recipient

MsgBurnZYTD(amount, ibc_denom)
  ├─► Burn ZYTD dari sender
  └─► Release proportional collateral

MsgLiquidate(borrower, ibc_denom) [jika ratio < threshold]
  ├─► Liquidator bayar ZYTD debt
  └─► Liquidator terima collateral - protocol fee
```

**CLI:**
```bash
zytheriond tx stablecoin mint 1000000uzytd --collateral axlUSDC --from alice
zytheriond tx stablecoin burn 1000000uzytd --collateral axlUSDC --from alice
zytheriond query stablecoin collateral-ratio alice axlUSDC
```

### 4.4 CosmWasm Integration

**wasmd** v0.45.0 + **wasmvm** v1.5.2. Mode: permissioned (hanya akun ter-authorize atau governance yang bisa upload bytecode).

**TFHE Query Plugin:** Kontrak CosmWasm dapat memanggil evaluasi TFHE melalui custom query:
```json
{
  "tfhe_eval": {
    "op": "add",
    "ct1": "<base64 ciphertext>",
    "ct2": "<base64 ciphertext>"
  }
}
```

**Contoh kontrak homomorphic (Rust/CosmWasm):**
```rust
// Kontrak menyimpan Enc(balance) dan bisa menambah tanpa decrypt
pub fn execute_deposit(
    deps: DepsMut,
    env: Env,
    info: MessageInfo,
    encrypted_amount: Binary,
) -> Result<Response, ContractError> {
    let current = ENCRYPTED_BALANCE.load(deps.storage)?;
    // Trigger homomorphic add via TFHE plugin
    let new_balance = tfhe_add(deps.querier, current, encrypted_amount)?;
    ENCRYPTED_BALANCE.save(deps.storage, &new_balance)?;
    Ok(Response::new().add_attribute("action", "deposit"))
}
```

---

## 5. Confidential ZYTD (Partial Privacy)

### 5.1 Model Privasi (v0.5.1 → v0.5.2)

| Komponen | v0.5.1 | v0.5.2 |
|---|---|---|
| Balance ZYTD | 🔒 Private (FheUint32 ciphertext) | 🔒 Private |
| Transfer amount | 🔒 Private (ciphertext) | 👁️ Public (underflow guard) |
| Kolateral | 👁️ Public (liquidator perlu lihat) | 👁️ Public |
| Siapa yang encrypt balance | Node (ClientKey) | **User (CompressedPublicKey)** |
| Node bisa decrypt? | ✅ Ya | ❌ Tidak |

**Alasan kolateral public:** Bot likuidator harus bisa monitor posisi unhealthy secara real-time untuk menjaga peg $1 ZYTD. Tanpa visibilitas ini, peg akan collapse.

### 5.2 Storage Model (v0.5.2)

```go
// KV: enc_zytd_v2/<address> → ZYTDAccountState (JSON)
type ZYTDAccountState struct {
    EncryptedBalance  []byte `json:"encrypted_balance"`    // FheUint32 ct (private)
    PublicCreditLimit uint64 `json:"public_credit_limit"` // max outflow guard (public)
    LastUpdatedBlock  int64  `json:"last_updated_block"`   // untuk state rent
    SizeBytes         int64  `json:"size_bytes"`            // cache len(EncryptedBalance)
}

// KV: tfhe_pubkey/<address> → raw CompressedPublicKey bytes
```

### 5.3 CVE-ZYTH-001: SubUint32 Underflow (CRITICAL — PATCHED di v0.5.2)

**Attack vector:**
```
Attacker (balance=0) → ConfidentialTransferZYTD(encAmount=Enc(1_000_000))
Chain eksekusi: Enc(0) - Enc(1_000_000) = Enc(2^32 - 1_000_000) = Enc(~4.3 billion ZYTD)
Attacker kaya seketika. Validator tidak bisa deteksi (ciphertext opaque).
```

**Fix:** `PublicCreditLimit` plaintext per akun.

```go
// CRITICAL CHECK — prevents exploit:
if plaintextAmount > senderState.PublicCreditLimit {
    return fmt.Errorf("transfer %d exceeds credit limit %d", plaintextAmount, creditLimit)
}
// Invariant: PublicCreditLimit ≤ true_plaintext_balance (ALWAYS)
```

### 5.4 CVE-ZYTH-002: Node-Held Decryption Keys (HIGH — PATCHED di v0.5.2)

**Attack vector:**
```
Semua balance dienkripsi dengan node ClientKey.
Operator node → DecryptUint32(nodeClientKey, anyUserBalance) → semua balance terbaca.
```

**Fix: Two-Key Model**
```
User generates (offline):  ClientKey   ← RAHASIA, tidak pernah keluar dari device
                           PublicKey   ← aman dibagikan, upload ke chain

On-chain:                  PublicKey[user] ← registered via MsgRegisterUserTFHEPublicKey
                           Enc_PK[user](balance) ← hanya user yg bisa decrypt

Validator capability:      Dapat evaluate (Add/Sub via ServerKey) ✅
                           TIDAK dapat decrypt ❌
```

**Sisa risiko (acknowledged):** Path `SubUint32` masih menggunakan node ClientKey untuk membuat ciphertext operand intermediate. Eliminasi penuh membutuhkan threshold re-encryption → direncanakan v0.8.

### 5.5 State Rent (v0.5.1)

Mencegah state bloat dari ciphertext ~21 KB yang tidak pernah dihapus.

```go
type StateRentParams struct {
    RentRatePerBytePerBlock int64  // default: 1 uzytc/byte/block
    MaxFreeSizeBytes        int64  // default: 1024 bytes (free tier)
    GracePeriodBlocks       int64  // default: 14400 (~1 hari)
}

// Kalkulasi biaya:
// 21 KB × 1 uzytc/byte/block × 14,400 block/hari = 302,400 uzytc/hari = 0.3024 ZYTC/hari
// ≈ $0.03/hari pada $0.10/ZYTC
```

Events:
- `rent_collected` — sewa berhasil ditarik
- `rent_default` — owner gagal bayar, mulai grace period
- `commitment_evicted` — data dihapus (event dikirim SEBELUM hapus, untuk archival node)

---

## 6. API & RPC Reference

### 6.1 TFHE Operations

| Endpoint | Metode | Deskripsi |
|---|---|---|
| `/zytherion/privacy/v1/commitment/{hash}` | GET | Query commitment metadata |
| `/zytherion/privacy/v1/result/{hash}` | GET | Rekonstruksi dan ambil ciphertext |
| `tx/tfhe-submit` | MSG | Submit ciphertext TFHE ke jaringan |

### 6.2 Oracle

| Endpoint | Metode | Deskripsi |
|---|---|---|
| `/zytherion/oracle/v1/price/{denom}` | GET | Latest price entry |
| `/zytherion/oracle/v1/twap/{denom}` | GET | TWAP price |
| `/zytherion/oracle/v1/prices/{denom}` | GET | Semua price history dalam window |
| `tx/submit-price` | MSG | Validator submit harga |

### 6.3 Stablecoin

| Endpoint | Metode | Deskripsi |
|---|---|---|
| `/zytherion/stablecoin/v1/mint_record/{address}/{denom}` | GET | Mint records per address |
| `/zytherion/stablecoin/v1/collateral_ratio/{address}/{denom}` | GET | Current collateral ratio |
| `/zytherion/stablecoin/v1/max_mintable` | GET | Kalkulasi max ZYTD yang bisa di-mint |
| `tx/mint-zytd` | MSG | Mint ZYTD |
| `tx/burn-zytd` | MSG | Burn ZYTD dan release collateral |
| `tx/liquidate` | MSG | Trigger likuidasi posisi unhealthy |

### 6.4 CosmWasm

| Endpoint | Metode | Deskripsi |
|---|---|---|
| `/cosmwasm/wasm/v1/code` | GET | List uploaded contract codes |
| `/cosmwasm/wasm/v1/contract/{address}` | GET | Contract info |
| `/cosmwasm/wasm/v1/contract/{address}/smart` | POST | Smart query |
| `tx/store-code` | MSG | Upload bytecode (permissioned) |
| `tx/instantiate-contract` | MSG | Instantiate contract |
| `tx/execute-contract` | MSG | Execute contract method |

---

## 7. Tokenomics (ZYTC)

**Total Supply:** 1,000,000,000 ZYTC (1 Miliar)

| Alokasi | Jumlah | % | Tujuan |
|---|---|---|---|
| Community Pool / Public Sale | 450,000,000 | 45% | Ekosistem, dApp incentive, adopsi |
| Staking Rewards | 250,000,000 | 25% | Validator dan delegator emission |
| Development Fund | 150,000,000 | 15% | Protocol development |
| Team & Founders | 100,000,000 | 10% | Long-term vesting |
| Public Goods Funding | 50,000,000 | 5% | Community grants |

**ZYTD (Stablecoin):**
- Peg: $1 USD
- Backing: Multi-collateral IBC assets (axlUSDC, ATOM, wBTC, wETH, dll)
- Minting: Overcollateralized (minimum ratio enforced on-chain)
- Burning: Returns proportional collateral

---

## 8. Dependencies (v0.5.2)

### Go Dependencies

| Library | Tujuan | Versi |
|---|---|---|
| `github.com/cosmos/cosmos-sdk` | Blockchain framework | v0.47.x |
| `github.com/cometbft/cometbft` | Consensus engine | v0.37.x |
| `github.com/CosmWasm/wasmd` | Smart contract engine | v0.45.0 |
| `github.com/CosmWasm/wasmvm` | Wasm VM | v1.5.2 |
| `github.com/cloudflare/circl` (mode5) | Dilithium5 signatures | v1.6.3 |
| `github.com/klauspost/reedsolomon` | Erasure coding | v1.12.1 |
| `cosmossdk.io/errors` | Error codes | v1.x |

### Rust Dependencies (tfhe_c)

| Library | Tujuan | Versi |
|---|---|---|
| `tfhe` (Zama) | TFHE FheUint32 engine | v0.6 |
| `bincode` | FFI serialization | v1.3 |

### Dihapus

| Library | Alasan Dihapus |
|---|---|
| `github.com/consensys/gnark` | ZK-SNARK membutuhkan trusted setup; diganti TFHE |
| `github.com/consensys/gnark-crypto` | Dependency gnark |

---

## 9. Build Instructions

```bash
# 1. Prasyarat
#    - Go 1.21+
#    - Rust toolchain (rustup)
#    - gcc, make

# 2. Compile Rust static library (pertama kali: 5–15 menit)
cd x/privacy/tfhe/tfhe_c
cargo build --release
cd ../../../..

# 3. Build Go binary dengan TFHE enabled
go build -tags tfhe_cgo -o zytheriond ./cmd/zytheriond

# 4. Atau tanpa TFHE (stub mode, untuk testing tanpa CGo):
go build -o zytheriond ./cmd/zytheriond

# 5. Ignite (development server)
ignite chain serve --build.tags "tfhe_cgo" --reset-once

# 6. Verifikasi
zytheriond version
# Zytherion Blockchain v0.5.2

# 7. Test suite
make test-pqc      # Dilithium5 + LWR tests (~5s)
make test-erasure  # Reed-Solomon tests (~1s)
make test-tfhe     # TFHE CGo engine (~5–10 menit)
make test-v05      # Oracle + stablecoin + CosmWasm (~60s)
```

**Start node:**
```bash
# TFHE enabled:
zytheriond start --enable-tfhe

# Default (TFHE disabled):
zytheriond start
```

---

## 10. File Structure (v0.5.2)

```
zytherion/
├── app/
│   ├── app.go                        # Application wiring (semua modul)
│   └── crypto_startup.go             # Dilithium5 self-test saat startup
│
├── x/
│   ├── privacy/                      # Modul inti kriptografi
│   │   ├── keeper/
│   │   │   ├── keeper.go             # KVStore helpers, worker pool init
│   │   │   ├── msg_server_tfhe_submit.go  # MsgTFHESubmit handler
│   │   │   ├── query_commitment.go   # MiMC commitment query
│   │   │   └── state_rent.go         # CollectRent, CheckAndEvict
│   │   ├── pqc/
│   │   │   ├── signature.go          # Dilithium5 wrappers
│   │   │   ├── lwr_hash.go           # Ring-LWR hash
│   │   │   └── povl.go               # PoVL sequential VDF
│   │   ├── tfhe/
│   │   │   ├── tfhe_c/src/lib.rs     # Rust FFI (encrypt, add, sub, keygen, pk_encrypt)
│   │   │   ├── cgo_bridge.h          # C header declarations
│   │   │   ├── engine.go             # Go CGo bridge (build: tfhe_cgo)
│   │   │   ├── engine_stub.go        # Stub (build: !tfhe_cgo)
│   │   │   ├── worker_pool.go        # OS-thread-pinned worker pool
│   │   │   ├── worker_pool_stub.go   # Stub
│   │   │   ├── node_keys.go          # EnsureNodeKeys singleton
│   │   │   ├── node_keys_stub.go     # Stub
│   │   │   ├── erasure.go            # Reed-Solomon 12+4=16
│   │   │   ├── merkle.go             # Binary SHA-256 Merkle tree
│   │   │   ├── shard_store.go        # Disk shard storage
│   │   │   └── shard_distributor.go  # P2P HTTP server + auth + rate limiter
│   │   └── types/
│   │       ├── state_rent.go         # StateRentParams, RentDue
│   │       ├── errors.go             # Error codes (1200–1299)
│   │       └── keys.go               # KV key prefixes
│   │
│   ├── oracle/                       # x/oracle — Price feed
│   │   ├── keeper/keeper.go          # SetPrice, GetTWAP, ComputeTWAP
│   │   ├── keeper/msg_server.go      # SubmitPrice handler
│   │   └── types/                    # PriceEntry, TWAPData, OracleParams
│   │
│   ├── ibc-collateral/               # x/ibc-collateral — Vault
│   │   ├── keeper/keeper.go          # LockCollateral, ReleaseCollateral
│   │   └── middleware/               # ICS-20 OnRecvPacket intercept
│   │
│   └── stablecoin/                   # x/stablecoin — ZYTD
│       ├── keeper/
│       │   ├── keeper.go             # Mint, Burn, Liquidate
│       │   ├── confidential_transfer.go  # ZYTDAccountState, CVE-001/002 fix
│       │   └── state_rent.go         # Stablecoin rent hooks
│       └── types/                    # MsgMintZYTD, MsgBurnZYTD, etc.
│
├── docs/
│   ├── summary.md                    # Dokumen ini
│   ├── prompt.md                     # Architecture prompt untuk LLM tools
│   ├── code_architecture.md          # Detail teknis kode
│   ├── zytherion_v05 update.md       # Implementation prompt v0.5 – v0.5.2
│   ├── DEMO_GUIDE.md                 # End-to-end demo walkthrough
│   ├── API_WALLET.md                 # Wallet API reference
│   ├── API_EXPLORER.md               # Block explorer API reference
│   └── API_FAUCET.md                 # Faucet API reference
│
└── contracts/
    └── homomorphic_counter/          # Contoh CosmWasm contract TFHE
        └── src/lib.rs
```

---

## 11. Roadmap

| Fase | Version | Target | Status | Fitur Utama |
|---|---|---|---|---|
| Phase 1 | v0.1 | 2025 | ✅ Done | Ring-LWR Hashing, PoVL VDF, ABCI 2.0 |
| Phase 2 | v0.2 | 2025 | ✅ Done | GreenBFT, Dilithium3, ZK-SNARK (temp) |
| Phase 3 | v0.3 | 2026 | ✅ Done | Dilithium5, TFHE (tfhe-rs), Erasure Coding |
| Phase 4 | v0.4 | 2026 | ✅ Done | Merkle Integrity, Auth P2P, Quota, Rate Limiter |
| Phase 5a | v0.5.0 | 2026 | ✅ Done | Oracle, IBC Collateral, Stablecoin, CosmWasm |
| Phase 5b | v0.5.1 | 2026 | ✅ Done | Worker Pool, State Rent, Confidential ZYTD, API Docs |
| Phase 5c | v0.5.2 | 2026 | ✅ Done | **Security Patch: CVE-001 + CVE-002** |
| Phase 5d | v0.5.3 | 2026 | ✅ Done | **TFHE Always-On + API Status Endpoint** |
| Phase 6 | v0.6 | 2026 | ✅ Done | **QuantumBFT: Post-Quantum Consensus Signing (Dilithium5)** |
| Phase 7 | v0.7 | Q1 2027 | 📅 Planned | IBC Privacy Bridge, ZK infrastructure prep |
| Phase 8 | v0.8 | Q2 2027 | 📅 Planned | ZK Range Proofs, full amount privacy, threshold re-enc |
| Phase 9 | v1.0 | 2027 | 🎯 Target | Mainnet launch |

### v0.8 Full Privacy Target

```
v0.5.2 (sekarang):
  ✅ Underflow blocked via PublicCreditLimit
  ✅ Operator cannot decrypt user balances
  ⚠️  Transfer amounts public (privacy trade-off)
  ⚠️  Sub path intermediate still uses node key

v0.8 (planned):
  🎯 ZK range proof: prove balance ≥ amount without revealing either
  🎯 Transfer amounts kembali private
  🎯 Threshold re-encryption untuk Sub path isolation
  🎯 Full user-key isolation di semua code paths
```

---

## 12. Security Properties Summary

| Ancaman | Perlindungan | Status |
|---|---|---|
| Quantum computer (Shor's algorithm) breaking ECDSA | Dilithium5 (ML-DSA Level 5, ~256-bit PQ) | ✅ Active |
| Block timestamp manipulation | PoVL sequential VDF (10 iterasi/blok) | ✅ Active |
| Block hash pre-image attack | Ring-LWR hash (post-quantum hardness) | ✅ Active |
| TFHE ciphertext availability (node offline) | Reed-Solomon 12+4=16, RF=4 | ✅ Active |
| TFHE shard tampering by malicious peer | Merkle proof verification before store | ✅ Active |
| Unauthorized shard injection | Bearer-token auth + rate limiter | ✅ Active |
| Storage DoS (too many TFHE commitments) | Per-address quota (max 1 active) | ✅ Active |
| TFHE state bloat (unpaid storage) | State Rent (0.3024 ZYTC/day/21KB) | ✅ Active |
| TFHE CGo concurrency race condition | OS-thread-pinned Worker Pool | ✅ Active (v0.5.1) |
| Stablecoin underflow attack (Enc(0)-Enc(x)) | PublicCreditLimit guard (CVE-ZYTH-001) | ✅ Patched (v0.5.2) |
| Validator decrypts user ZYTD balance | User-held CompressedPublicKey (CVE-ZYTH-002) | ✅ Patched (v0.5.2) |
| Oracle flash loan manipulation | Median TWAP (outliers di-reject) | ✅ Active |
| Undercollateralized ZYTD minting | Collateral ratio enforcement + liquidation | ✅ Active |
| Unauthorized CosmWasm code upload | Permissioned mode (governance only) | ✅ Active |

---

## 13. Conclusion

Zytherion telah berkembang dari sebuah eksperimen kriptografi post-quantum menjadi ekosistem blockchain lengkap yang mencakup:

1. **Kriptografi fondasi** yang sepenuhnya quantum-resistant (Dilithium5, LWR, PoVL)
2. **Privasi komputasional** via TFHE — blockchain pertama yang mengintegrasikan FHE ke dalam ekosistem Cosmos
3. **DeFi terdesentralisasi** — stablecoin multi-kolateral dengan oracle dan IBC
4. **Smart contract** dengan kemampuan homomorphic lewat CosmWasm TFHE plugin
5. **Security yang jujur** — kerentanan diakui, dipatch, dan didokumentasikan secara transparan sebelum mainnet

Dengan v0.5.2, Zytherion telah mempatch dua CVE kritis dan kini memiliki model privasi yang lebih kuat: validator dapat mengevaluasi operasi homomorphic tanpa pernah bisa membaca balance pengguna.

Target mainnet v1.0 di 2027 akan membawa ZK range proofs, full transfer privacy, dan IBC privacy bridge — menjadikan Zytherion sebagai blockchain Cosmos-ekosistem dengan privasi paling komprehensif yang pernah dibangun.

---

## 14. Cara Pakai TFHE (User Guide)

Panduan ini menjelaskan cara menggunakan fitur Fully Homomorphic Encryption (TFHE) di Zytherion menggunakan CLI resmi `zytheriond`. TFHE selalu aktif di v0.5.3+ — tidak perlu flag atau konfigurasi tambahan.

### A. Generate TFHE Key Pair

Langkah pertama: buat pasangan kunci TFHE di komputer lokal Anda. Kunci ini disimpan di `~/.zytherion/tfhe/`.

```bash
zytheriond keys tfhe keygen
# Output:
#   Client Key (secret): ~/.zytherion/tfhe/client.key  (~22 KB)
#   Server Key (public): ~/.zytherion/tfhe/server.key  (~107 MB)
```

> ⚠️ **Jaga kerahasiaan `client.key`!** Node validator hanya perlu `server.key`. Hanya pemegang `client.key` yang bisa mendekripsi saldo terenkripsi.

---

### B. Enkripsi Nilai secara Lokal

Enkripsi angka (misalnya `88`) menjadi ciphertext FheUint32 di komputer lokal:

```bash
# Enkripsi angka 88 → output base64 di terminal
zytheriond keys tfhe encrypt 88

# Simpan ke file untuk digunakan dalam transaksi
zytheriond keys tfhe encrypt 88 > ciphertext_b64.txt
openssl base64 -d -in ciphertext_b64.txt -out ciphertext.bin
```

---

### C. Kirim Ciphertext ke Blockchain (Confidential Submission)

Kirim file ciphertext terenkripsi ke jaringan Zytherion. Validator memecahnya menjadi 16 shard (12 data + 4 parity) dan menyimpannya secara terdistribusi:

```bash
zytheriond tx privacy tfhe-submit \
  --ciphertext ciphertext.bin \
  --from alice \
  --chain-id zytherion \
  --fees 50000zytc \
  --keyring-backend test -y
```

Output akan menyertakan **`commitment_hash`** (SHA-256 dari ciphertext). Simpan hash ini.

---

### D. Query Status TFHE dari Node (REST API)

Cek apakah TFHE aktif di node dan lihat statistik real-time:

```bash
curl http://localhost:1317/zytherion/privacy/v1/tfhe/status
```

Response JSON:
```json
{
  "enabled": true,
  "version": "tfhe-rs (FheUint32 / 32-bit Levelled FHE)",
  "erasure_coding": "12+4=16 shards (DataShards=12, ParityShards=4)",
  "replication_factor": 3,
  "node_id": "local-node",
  "active_commitments": 1,
  "shard_store_ready": true
}
```

---

### E. Ambil Kembali Ciphertext dari Blockchain

Siapa pun bisa mengambil ciphertext terenkripsi dari jaringan menggunakan commitment hash. Node akan merekonstruksi data dari shard-shard yang tersebar:

```bash
zytheriond query privacy tfhe-result \
  --commitment <commitment_hash_hex_dari_step_C> \
  --output json
```

Output mengandung `result_ciphertext` (base64). Simpan ke file:

```bash
# Ambil ciphertext dari jaringan dan decode ke file biner
zytheriond query privacy tfhe-result --commitment <hash> --output json \
  | jq -r '.result_ciphertext' \
  | openssl base64 -d -out retrieved.bin
```

---

### F. Dekripsi Ciphertext secara Lokal

Hanya pemegang `client.key` yang bisa membaca angka aslinya:

```bash
# Dekripsi dan tampilkan angka asli (88 dalam contoh ini)
zytheriond keys tfhe decrypt "$(base64 -w0 retrieved.bin)"
# Output:
#   Decrypted value: 88
```

---

### G. Enkripsi Saldo Akun (Confidential ZYTD)

Untuk menggunakan saldo terenkripsi ZYTD (Confidential Stablecoin):

```bash
# 1. Daftarkan public key Anda on-chain
zytheriond tx stablecoin register-tfhe-key \
  --pubkey-file ~/.zytherion/tfhe/server.key \
  --from alice --chain-id zytherion --fees 5000zytc -y

# 2. Mint ZYTD dengan saldo terenkripsi
#    (Validator mengenkripsi amount menggunakan public key Anda)
zytheriond tx stablecoin mint-confidential 1000000uusdc \
  --from alice --chain-id zytherion --fees 5000zytc -y

# 3. Transfer ZYTD secara homomorfik (Validator tidak tahu amount)
#    (Encrypt 100 ZYTD untuk dikirim ke bob)
ENC=$(zytheriond keys tfhe encrypt 100)
zytheriond tx stablecoin confidential-transfer bob "$ENC" 100 \
  --from alice --chain-id zytherion --fees 5000zytc -y

# 4. Lihat saldo terenkripsi Anda dari chain, decrypt secara lokal
zytheriond query stablecoin encrypted-balance alice \
  | jq -r '.encrypted_balance' \
  | xargs zytheriond keys tfhe decrypt
```

---

## 15. QuantumBFT Migration & Terminal Startup Prompt Comparison

Bagian ini membandingkan tampilan terminal log & startup prompt saat node dijalankan (misal via `ignite chain serve` atau `zytheriond start`) sebelum dan sesudah migrasi dari CometBFT Ed25519 ke QuantumBFT Dilithium5.

### A. Sebelum Migrasi (CometBFT Legacy Ed25519)

Sebelumnya, CometBFT menggunakan kunci `priv_validator_key.json` berbasis **Ed25519 (32-byte public key)** standar:

```text
INF ═══════════════════════════════════════════════════════════
INF   ⛛  ZYTHERION BLOCKCHAIN AND CRYPTOCURRENCY v0.5.3  ⛛
INF   ⛛  Founder: Rayhan Aziel Abbrar                               ⛛
INF   ⛛  CRYPTOGRAPHIC SUBSYSTEM STARTUP REPORT                     ⛛
INF ═══════════════════════════════════════════════════════════
INF   [✅ OK  ] Dilithium5 (ML-DSA Level 5) detail="sign/verify self-test PASSED" elapsed=12ms
INF   [✅ OK  ] LWR (Ring-LWR / SHAKE-256) detail="hash generation & avalanche check PASSED" elapsed=4ms
INF   [✅ OK  ] TFHE (tfhe-rs / FheUint32) detail="subsystem ALWAYS ENABLED — erasure coding: 10+6=16 shards" elapsed=0s
INF ═══════════════════════════════════════════════════════════
INF   ✅ ALL CRYPTO SUBSYSTEMS OPERATIONAL — node is READY
INF ═══════════════════════════════════════════════════════════
INF CometBFT node starting...
INF Loaded PrivValidator key file path=/home/user/.zytherion/config/priv_validator_key.json type=ed25519 pubkey=A1B2C3D4... (32 bytes)
```

---

### B. Sesudah Migrasi (QuantumBFT v0.6 — Dilithium5)

Di v0.6, saat `ignite chain serve` atau `zytheriond start` dijalankan, node secara otomatis mendeteksi/membuat kunci konsensus **Dilithium5 (2592-byte public key)** di `quantum_validator_key.json` dan menyuntikkan `QuantumFilePV`:

```text
INF ═══════════════════════════════════════════════════════════
INF   ⛛  ZYTHERION BLOCKCHAIN AND CRYPTOCURRENCY v0.6 (QuantumBFT)  ⛛
INF   ⛛  Founder: Rayhan Aziel Abbrar                               ⛛
INF   ⛛  CRYPTOGRAPHIC SUBSYSTEM STARTUP REPORT                     ⛛
INF ═══════════════════════════════════════════════════════════
INF   [✅ OK  ] Dilithium5 (ML-DSA Level 5) detail="sign/verify self-test PASSED" elapsed=11ms
INF   [✅ OK  ] LWR (Ring-LWR / SHAKE-256) detail="hash generation & avalanche check PASSED" elapsed=3ms
INF   [✅ OK  ] TFHE (tfhe-rs / FheUint32) detail="subsystem ALWAYS ENABLED — erasure coding: 10+6=16 shards" elapsed=0s
INF   [✅ OK  ] QuantumBFT (Dilithium5 consensus signing) detail="ACTIVE — quantum_validator_key.json found, Dilithium5 signing enabled" elapsed=0s
INF ═══════════════════════════════════════════════════════════
INF   ✅ ALL CRYPTO SUBSYSTEMS OPERATIONAL — node is READY
INF      Signature algorithm: Dilithium5 (ML-DSA-87, NIST Cat-5, ~256-bit PQ)
INF      Consensus Engine:    QuantumBFT (Dilithium5 validator signing)
INF ═══════════════════════════════════════════════════════════
INF QuantumBFT: Dilithium5 validator key loaded address=zytherionvalcons1... key_file=/home/user/.zytherion/config/quantum_validator_key.json algorithm="Dilithium5 (ML-DSA-87, NIST Category 5)"
INF QuantumBFT node active — validator signing votes and block proposals with Dilithium5
```

---

*Dokumen ini adalah ringkasan teknis komprehensif Zytherion v0.6 (QuantumBFT).*  
*Founder: **Rayhan Aziel Abbrar** | Terakhir diupdate: Juli 2026*

