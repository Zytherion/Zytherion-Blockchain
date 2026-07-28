# Zytherion Faucet API Reference

> Ini adalah referensi API untuk membangun faucet (dispenser token gratis)  
> untuk testnet Zytherion.

---

## Cara Kerja Faucet

```
User request (address)
       ↓
[1] Validasi address (bech32 prefix "zyth")
[2] Cek saldo address — tolak jika sudah ada banyak token
[3] Rate limiting — satu address, satu IP, max 1x per 24 jam
[4] Broadcast tx bank send dari faucet wallet
[5] Return tx hash ke user
```

---

## API yang Digunakan Faucet

### 1. Validasi & Cek Saldo Address

**Cek apakah address valid dan sudah ada on-chain:**
```
GET http://localhost:1317/cosmos/auth/v1beta1/accounts/{address}
```

- Status 200 → akun sudah ada
- Status 404 / error → akun baru, belum pernah menerima token

**Cek saldo ZYTC:**
```
GET http://localhost:1317/cosmos/bank/v1beta1/balances/{address}/by_denom?denom=zytc
```

**Response:**
```json
{
  "balance": {
    "denom": "zytc",
    "amount": "0"
  }
}
```

**Contoh aturan faucet:**
- Tolak jika saldo sudah `> 5000000 zytc` (lebih dari 5 ZYTC)
- Tolak jika address sudah pernah dapat dalam 24 jam terakhir (simpan di DB faucet)

---

### 2. Cek Status Node (sebelum broadcast)

```
GET http://localhost:26657/status
```

Pastikan:
- `result.sync_info.catching_up == false` → node synced
- `result.node_info.network == "zytherion"` → chain ID benar

---

### 3. Dapatkan Account Number & Sequence (untuk sign tx)

```
GET http://localhost:1317/cosmos/auth/v1beta1/accounts/{faucet_address}
```

**Response fields yang diperlukan:**
```json
{
  "account": {
    "account_number": "1",
    "sequence": "42"
  }
}
```

> Setiap transaksi harus menggunakan `sequence` yang benar. Tambah +1 setelah setiap tx berhasil.

---

### 4. Simulate Gas

Sebelum broadcast, estimate gas yang dibutuhkan:

```
POST http://localhost:1317/cosmos/tx/v1beta1/simulate
Content-Type: application/json
```

**Body:**
```json
{
  "tx_bytes": "<base64_unsigned_or_signed_tx_bytes>"
}
```

**Response:**
```json
{
  "gas_info": {
    "gas_wanted": "200000",
    "gas_used": "87432"
  }
}
```

Gunakan `gas_used * 1.3` sebagai `gas_limit` yang aman.

---

### 5. Broadcast Transaksi (Drip Token)

```
POST http://localhost:1317/cosmos/tx/v1beta1/txs
Content-Type: application/json
```

**Body:**
```json
{
  "tx_bytes": "<base64_signed_tx>",
  "mode": "BROADCAST_MODE_SYNC"
}
```

**Response sukses:**
```json
{
  "tx_response": {
    "code": 0,
    "txhash": "ABC123DEF456...",
    "raw_log": "[]"
  }
}
```

**Response gagal:**
```json
{
  "tx_response": {
    "code": 5,
    "raw_log": "insufficient funds"
  }
}
```

> `code: 0` = sukses. `code != 0` = gagal — jangan simpan ke log "sudah klaim".

---

### 6. Konfirmasi Transaksi

Setelah broadcast, poll hingga tx masuk blok:

```
GET http://localhost:1317/cosmos/tx/v1beta1/txs/{txhash}
```

Atau via CometBFT RPC:
```
GET http://localhost:26657/tx?hash=0x{txhash}
```

Poll setiap 2–3 detik, timeout setelah 30 detik.

---

### 7. Verifikasi Saldo Setelah Drip

```
GET http://localhost:1317/cosmos/bank/v1beta1/balances/{recipient_address}/by_denom?denom=zytc
```

---

## Struktur Transaksi Bank Send

Faucet harus build dan sign transaksi ini dengan private key faucet wallet:

```json
{
  "body": {
    "messages": [
      {
        "@type": "/cosmos.bank.v1beta1.MsgSend",
        "from_address": "zyth1FAUCET_ADDRESS...",
        "to_address": "zyth1USER_ADDRESS...",
        "amount": [
          {
            "denom": "zytc",
            "amount": "1000000"
          }
        ]
      }
    ],
    "memo": "Zytherion Testnet Faucet",
    "timeout_height": "0"
  },
  "auth_info": {
    "signer_infos": [
      {
        "public_key": {
          "@type": "/zytherion.crypto.dilithium5.PubKey",
          "key": "<base64_pubkey>"
        },
        "mode_info": {
          "single": {
            "mode": "SIGN_MODE_DIRECT"
          }
        },
        "sequence": "42"
      }
    ],
    "fee": {
      "amount": [{ "denom": "zytc", "amount": "50000" }],
      "gas_limit": "200000"
    }
  },
  "signatures": ["<base64_dilithium5_signature>"]
}
```

> ⚠️ **Faucet wallet harus pakai Dilithium5!**  
> Signing menggunakan algoritma Dilithium5 (ML-DSA), bukan secp256k1.  
> Referensi implementasi: `crypto/dilithium5/` di repo Zytherion.

---

## Stablecoin ZYTD Faucet (Opsional)

Untuk memberikan sejumlah kecil ZYTD agar user bisa langsung coba stablecoin:

**Alternatif 1:** Faucet mint ZYTD menggunakan kolateral miliknya:
```json
{
  "@type": "/zytherion.stablecoin.MsgMintZYTD",
  "minter": "zyth1FAUCET...",
  "collateral": { "denom": "zytc", "amount": "500000" },
  "mint_amount": { "denom": "zytd", "amount": "100000" }
}
```

**Alternatif 2:** Faucet kirim ZYTD langsung (jika sudah punya stok):
```json
{
  "@type": "/cosmos.bank.v1beta1.MsgSend",
  "from_address": "zyth1FAUCET...",
  "to_address": "zyth1USER...",
  "amount": [{ "denom": "zytd", "amount": "100000" }]
}
```

---

## Query: Riwayat Klaim (untuk Explorer Faucet)

Cari semua transaksi dari faucet wallet:
```
GET http://localhost:1317/cosmos/tx/v1beta1/txs?events=transfer.sender%3D%27{faucet_address}%27&order_by=ORDER_BY_DESC&limit=50
```

---

## Konfigurasi Faucet yang Disarankan

| Parameter | Nilai Disarankan |
|-----------|-----------------|
| Drip Amount | `1000000 zytc` (1 ZYTC) |
| ZYTD Drip | `100000 zytd` (0.1 ZYTD) |
| Rate Limit per Address | 1x per 24 jam |
| Rate Limit per IP | 3x per 24 jam |
| Max Saldo Penerima | `5000000 zytc` (5 ZYTC) |
| Gas Limit | `200000` |
| Fee | `50000 uzytc` |
| Signing Algorithm | **Dilithium5** |

---

## Checklist Implementasi Faucet

```
[ ] Buat faucet wallet dengan --key-type dilithium5
[ ] Fund faucet wallet dari genesis account
[ ] Implementasi validasi bech32 prefix "zyth"
[ ] Implementasi cek saldo penerima (tolak jika sudah kaya)
[ ] Implementasi rate limiting (Redis / SQLite)
[ ] Implementasi Dilithium5 signing
[ ] GET /auth/v1beta1/accounts/{faucet} → dapat sequence number
[ ] Build & sign tx MsgSend
[ ] POST /txs dengan BROADCAST_MODE_SYNC
[ ] Poll tx confirmation
[ ] Return txhash ke user
[ ] (Opsional) Simple UI dengan form input address
```

---

## Custom Privacy Module Endpoints

### Get ZK Commitment (untuk faucet yang butuh verifikasi)
```
GET http://localhost:1317/zytherion/privacy/v1/commitment/{address}
```

**Response:**
```json
{
  "address": "zyth1...",
  "commitment_hex": "a3f2b1...",
  "note": "Commitment is a MiMC hash. Verify off-chain with your blinding factor."
}
```

> Faucet bisa gunakan ini untuk memverifikasi apakah user sudah punya commitment aktif (sudah pakai privacy features).

---

## Monitoring Faucet

### Cek Saldo Faucet Wallet
```
GET http://localhost:1317/cosmos/bank/v1beta1/balances/{faucet_address}/by_denom?denom=zytc
```

Buat alert jika saldo faucet < threshold (misalnya < 100 ZYTC → kirim notifikasi ke admin).

### Cek Node Health
```
GET http://localhost:26657/health
```

Response: `{"jsonrpc":"2.0","result":{}}` = node sehat.

---

## Error Codes

| Code | Makna | Solusi |
|------|-------|--------|
| `5` | Insufficient funds (faucet kehabisan) | Top up faucet wallet |
| `4` | Unauthorized / signature invalid | Cek signing algorithm (harus Dilithium5) |
| `19` | Tx already in mempool | Naikkan sequence number atau tunggu |
| `32` | Wrong sequence | Re-fetch sequence dari `/auth/v1beta1/accounts/{faucet}` |
| `11` | Out of gas | Naikkan gas limit |
