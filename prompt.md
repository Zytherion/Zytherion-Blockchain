# Zytherion — Panduan Lengkap CLI & API

> **Chain ID:** `zytherion` | **Token:** `zytc` | **Binary:** `zytheriond`

---

## Daftar Isi
1. [Setup & Instalasi](#1-setup--instalasi)
2. [Menjalankan Node](#2-menjalankan-node)
3. [Konfigurasi CLI](#3-konfigurasi-cli)
4. [Manajemen Akun](#4-manajemen-akun)
5. [Transaksi Token](#5-transaksi-token)
6. [Query Data Blockchain](#6-query-data-blockchain)
7. [Staking & Delegasi](#7-staking--delegasi)
8. [Distribusi & Reward](#8-distribusi--reward)
9. [Validator Lifecycle](#9-validator-lifecycle)
10. [Governance](#10-governance)
11. [Privacy Module (ZK)](#11-privacy-module-zk)
12. [Faucet Development](#12-faucet-development)
13. [ZK Tooling Off-chain](#13-zk-tooling-off-chain)
14. [API & Endpoints](#14-api--endpoints)
15. [Monitoring & Debug](#15-monitoring--debug)

---

## 1. Setup & Instalasi

```bash
# Build dari source
make build

# Install ke PATH
go install ./cmd/zytheriond

# Init konfigurasi node (sekali saja)
zytheriond init <moniker> --chain-id zytherion
```

---

## 2. Menjalankan Node

```bash
# Node validator
zytheriond start

# Mode development (Ignite)
ignite chain serve

# Reset data lokal (hati-hati!)
zytheriond tendermint unsafe-reset-all
```

---

## 3. Konfigurasi CLI

```bash
zytheriond config chain-id zytherion
zytheriond config keyring-backend os
zytheriond config node tcp://localhost:26657
zytheriond config broadcast-mode sync

# Lihat konfigurasi aktif
zytheriond config
```

---

## 4. Manajemen Akun

```bash
# Daftar semua akun lokal
zytheriond keys list

# Buat wallet baru (simpan mnemonic!)
zytheriond keys add <nama_wallet>

# Pulihkan dari mnemonic
zytheriond keys add <nama_wallet> --recover

# Tampilkan alamat wallet
zytheriond keys show <nama_wallet> -a

# Tampilkan alamat valoper
zytheriond keys show <nama_wallet> --bech val -a

# Hapus akun
zytheriond keys delete <nama_wallet>
```

---

## 5. Transaksi Token

```bash
# Kirim koin
zytheriond tx bank send <wallet_pengirim> <alamat_penerima> 1000zytc \
  --fees 200zytc -y

# Simulasi tanpa broadcast (dry-run)
zytheriond tx bank send <wallet> <alamat_penerima> 1000zytc \
  --fees 200zytc --dry-run
```

---

## 6. Query Data Blockchain

```bash
# Saldo wallet
zytheriond query bank balances <alamat>

# Total supply
zytheriond query bank total

# Detail transaksi
zytheriond query tx <tx_hash>

# Lihat blok terbaru
zytheriond query block

# Blok pada height tertentu
zytheriond query block <height>

# Status node (dengan jq)
zytheriond status 2>&1 | jq .SyncInfo
```

---

## 7. Staking & Delegasi

```bash
# Daftar semua validator
zytheriond query staking validators

# Detail validator
zytheriond query staking validator <valoper_address>

# Delegasi ke validator
zytheriond tx staking delegate <valoper_address> 500000zytc \
  --from <nama_wallet> --fees 200zytc -y

# Pindah delegasi antar validator
zytheriond tx staking redelegate <valoper_lama> <valoper_baru> 100000zytc \
  --from <nama_wallet> --fees 200zytc -y

# Cabut delegasi (unbond)
zytheriond tx staking unbond <valoper_address> 10000zytc \
  --from <nama_wallet> --fees 200zytc -y

# Lihat delegasi wallet
zytheriond query staking delegations <alamat>

# Lihat yang sedang unbonding
zytheriond query staking unbonding-delegations <alamat>
```

---

## 8. Distribusi & Reward

```bash
# Lihat total reward
zytheriond query distribution rewards <alamat>

# Ambil semua reward
zytheriond tx distribution withdraw-all-rewards \
  --from <nama_wallet> --fees 200zytc -y

# Ambil reward dari validator tertentu
zytheriond tx distribution withdraw-rewards <valoper_address> \
  --from <nama_wallet> --fees 200zytc -y
```

---

## 9. Validator Lifecycle

```bash
# Buat validator baru
zytheriond tx staking create-validator \
  --amount=1000000000000zytc \
  --pubkey=$(zytheriond tendermint show-validator) \
  --moniker="<nama_validator>" \
  --chain-id=zytherion \
  --commission-rate="0.10" \
  --commission-max-rate="0.20" \
  --commission-max-change-rate="0.01" \
  --min-self-delegation="1" \
  --from=<nama_wallet> --fees 200zytc -y

# Edit info validator
zytheriond tx staking edit-validator \
  --moniker="<nama_baru>" \
  --website="https://zytherion.com" \
  --from=<nama_wallet> --fees 200zytc -y

# Unjail validator yang kena slashing
zytheriond tx slashing unjail \
  --from <nama_wallet> --fees 200zytc -y

# Cek signing info (apakah aktif/jail)
zytheriond query slashing signing-info $(zytheriond tendermint show-validator)
```

---

## 10. Governance

```bash
# Semua proposal
zytheriond query gov proposals

# Detail proposal
zytheriond query gov proposal 1

# Hasil vote
zytheriond query gov votes 1

# Submit proposal
zytheriond tx gov submit-proposal \
  --title "Upgrade PQC Parameters" \
  --description "Meningkatkan parameter ring-LWR" \
  --type "Text" \
  --deposit 100000000zytc \
  --from <nama_wallet> --fees 200zytc -y

# Deposit ke proposal
zytheriond tx gov deposit 1 50000000zytc \
  --from <nama_wallet> --fees 200zytc -y

# Vote (yes/no/no_with_veto/abstain)
zytheriond tx gov vote 1 yes \
  --from <nama_wallet> --fees 200zytc -y
```

---

## 11. Privacy Module (ZK)

```bash
# Submit ZK Commitment (setelah generate proof off-chain)
zytheriond tx privacy init-commitment <commitment_hex> \
  --from <nama_wallet> --fees 200zytc -y
```

> ZK Proof (Groth16/BN254) dikomputasi *off-chain* terlebih dahulu. Lihat bagian [ZK Tooling](#13-zk-tooling-off-chain).

---

## 12. Faucet Development

Saat development lokal via Ignite, faucet aktif otomatis:
```bash
curl -X POST "http://localhost:4500" \
  -H "Content-Type: application/json" \
  -d '{"address": "<alamat_wallet>"}'
```
Default: **100,000,000 zytc (~10 ZYTC)** per request.

---

## 13. ZK Tooling Off-chain

```bash
# Step 1: Trusted Setup (hanya sekali)
make zksetup
# Output: keys/verifying_key.bin + keys/proving_key.bin

# WAJIB commit verifying_key.bin ke repo:
git add keys/verifying_key.bin && git commit -m "zk: add trusted setup VK"

# Step 2: Generate ZK Proof untuk transaksi
make zkprove AMOUNT=1000000
# Output: proof.json

# Step 3: Submit proof ke chain
zytheriond tx privacy init-commitment $(cat proof.json | jq -r .commitment) \
  --from <nama_wallet> --fees 200zytc -y
```

> ⚠️ Node akan **panic dan menolak menyala** jika `keys/verifying_key.bin` tidak ditemukan — ini fitur keamanan `fail-fast`.

---

## 14. API & Endpoints

### A. Tendermint RPC — Port `26657`

| Endpoint | Deskripsi |
|---|---|
| `GET /status` | Status node, sync info |
| `GET /block?height=<N>` | Data blok pada height N |
| `GET /tx?hash=<HASH>` | Detail transaksi |
| `GET /net_info` | Info peer yang terhubung |
| `POST /broadcast_tx_sync` | Submit TX (tunggu CheckTx) |
| `POST /broadcast_tx_commit` | Submit TX (tunggu Commit) |

```bash
curl http://localhost:26657/status
curl http://localhost:26657/block?height=1
curl -X POST "http://localhost:26657/broadcast_tx_sync" \
  -H "Content-Type: application/json" \
  -d '{"tx": "<base64_tx>"}'
```

---

### B. REST API / LCD — Port `1317`

| Endpoint | Deskripsi |
|---|---|
| `GET /cosmos/bank/v1beta1/balances/{address}` | Saldo wallet |
| `GET /cosmos/bank/v1beta1/supply` | Total supply |
| `GET /cosmos/staking/v1beta1/validators` | Daftar validator |
| `GET /cosmos/staking/v1beta1/delegations/{address}` | Delegasi wallet |
| `GET /cosmos/distribution/v1beta1/delegators/{address}/rewards` | Reward |
| `GET /cosmos/gov/v1beta1/proposals` | Semua proposal |
| `GET /cosmos/tx/v1beta1/txs/{hash}` | Detail TX |
| `POST /cosmos/tx/v1beta1/txs` | Submit TX |

```bash
curl http://localhost:1317/cosmos/bank/v1beta1/balances/<alamat>
curl http://localhost:1317/cosmos/staking/v1beta1/validators

# Submit TX via REST
curl -X POST http://localhost:1317/cosmos/tx/v1beta1/txs \
  -H "Content-Type: application/json" \
  -d '{"tx_bytes": "<base64_tx>", "mode": "BROADCAST_MODE_SYNC"}'

# Swagger UI:
# http://localhost:1317/swagger/
```

---

### C. gRPC — Port `9090`

```bash
# Cek saldo via grpcurl
grpcurl -plaintext \
  -d '{"address": "<alamat>"}' \
  localhost:9090 \
  cosmos.bank.v1beta1.Query/AllBalances
```

**Aktifkan di `~/.zytherion/config/app.toml`:**
```toml
[api]
enable = true
address = "tcp://0.0.0.0:1317"

[grpc]
enable = true
address = "0.0.0.0:9090"
```

---

## 15. Monitoring & Debug

```bash
# Log real-time
journalctl -u zytheriond -f --output cat

# Cek height blok saat ini
curl -s http://localhost:26657/status | jq .result.sync_info.latest_block_height

# Apakah node sudah sinkron? (false = sudah sinkron)
curl -s http://localhost:26657/status | jq .result.sync_info.catching_up

# Peer yang terhubung
curl -s http://localhost:26657/net_info | jq .result.peers[].node_info.id

# Profiling (pprof)
# http://localhost:6060/debug/pprof/

# Jalankan semua unit test
make test

# Linter
make lint
```
