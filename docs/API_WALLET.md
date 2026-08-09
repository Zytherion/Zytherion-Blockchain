# Zytherion Wallet API Reference

> Base URL REST: `http://localhost:1317`  
> Base URL RPC: `http://localhost:26657`  
> Chain ID: `zytherion`  
> Native Denom: `zytc` (1 ZYTC = 1,000,000 uzytc)  
> Stablecoin: `zytd`

---

## Account Management

### Get Account Info
Ambil sequence number dan account number (wajib untuk sign transaksi).

```
GET /cosmos/auth/v1beta1/accounts/{address}
```

**Response:**
```json
{
  "account": {
    "@type": "/cosmos.auth.v1beta1.BaseAccount",
    "address": "zyth1...",
    "pub_key": {
      "@type": "/zytherion.crypto.dilithium5.PubKey",
      "key": "..."
    },
    "account_number": "1",
    "sequence": "5"
  }
}
```

> ⚠️ Field `pub_key` bertipe **Dilithium5**, bukan secp256k1. Wallet harus support deserialisasi custom type ini.

---

### Check if Account Exists
```
GET /cosmos/auth/v1beta1/accounts/{address}
```
Jika 404 → akun belum pernah menerima token (belum exist on-chain).

---

## Balance

### Get All Balances
```
GET /cosmos/bank/v1beta1/balances/{address}
```

**Response:**
```json
{
  "balances": [
    { "denom": "zytc", "amount": "10000000" },
    { "denom": "zytd", "amount": "500000" }
  ]
}
```

### Get Balance by Denom
```
GET /cosmos/bank/v1beta1/balances/{address}/by_denom?denom=zytc
```

### Get Total Supply
```
GET /cosmos/bank/v1beta1/supply
GET /cosmos/bank/v1beta1/supply/by_denom?denom=zytc
```

---

## Encrypted ZYTD Balance (Privacy Feature)

ZYTD balance bisa disimpan sebagai **FheUint32 ciphertext** on-chain.  
Query ini diakses melalui CosmWasm smart contract (homomorphic_vault).

### Query Encrypted Balance
```
GET /cosmwasm/wasm/v1/contract/{vault_address}/smart/{base64_query}
```

Dengan query JSON (di-base64):
```json
{"encrypted_balance":{}}
```

**Response:**
```json
{
  "data": {
    "encrypted_balance": "<base64_fheuint32_ciphertext>",
    "deposit_count": 3,
    "has_balance": true
  }
}
```

> 💡 `encrypted_balance` adalah ciphertext FheUint32 — hanya pemegang TFHE client key yang bisa decrypt ke plaintext.

### TFHE Custom Queries (via CosmWasm contract)
```
GET /cosmwasm/wasm/v1/contract/{contract_address}/smart/{base64_query}
```

Query JSON yang didukung:
```json
{ "homomorphic_add": { "ct1": "<base64>", "ct2": "<base64>" } }
```

---

## Send Transaction

### Broadcast Transaction
```
POST /cosmos/tx/v1beta1/txs
```

**Body:**
```json
{
  "tx_bytes": "<base64_signed_tx>",
  "mode": "BROADCAST_MODE_SYNC"
}
```

Modes:
- `BROADCAST_MODE_SYNC` — tunggu masuk mempool, langsung return tx hash
- `BROADCAST_MODE_ASYNC` — tidak tunggu
- `BROADCAST_MODE_BLOCK` — tunggu sampai committed (lambat, hindari di production)

### Simulate Gas (sebelum broadcast)
```
POST /cosmos/tx/v1beta1/simulate
```
Body: sama seperti broadcast. Response berisi `gas_used`.

---

## Transfer Token

Transaksi bank send dibuat client-side lalu di-sign dengan Dilithium5 private key.

**Tx body JSON:**
```json
{
  "@type": "/cosmos.bank.v1beta1.MsgSend",
  "from_address": "zyth1...",
  "to_address": "zyth1...",
  "amount": [{ "denom": "zytc", "amount": "1000000" }]
}
```

---

## ZYTD Stablecoin

### Mint ZYTD (Lock Collateral)
Transaksi on-chain. Collateral yang didukung: `zytc`, `axlUSDC`, `ATOM`, `wBTC`, `wETH` (via IBC).

```json
{
  "@type": "/zytherion.stablecoin.MsgMintZYTD",
  "minter": "zyth1...",
  "collateral": { "denom": "zytc", "amount": "5000000" },
  "mint_amount": { "denom": "zytd", "amount": "1000000" }
}
```

### Burn ZYTD (Redeem Collateral)
```json
{
  "@type": "/zytherion.stablecoin.MsgBurnZYTD",
  "burner": "zyth1...",
  "burn_amount": { "denom": "zytd", "amount": "1000000" }
}
```

### Query Mint Record (Posisi Kolateral)
```
GET /zytherion/stablecoin/v1/mint_record/{address}/{ibc_denom}
```
Returns informasi posisi kolateral: jumlah kolateral terkunci, ZYTD yang dicetak, liquidation threshold.

### Query Collateral Ratio (Live)
```
GET /zytherion/stablecoin/v1/collateral_ratio/{address}/{ibc_denom}
```
**Response:**
```json
{ "ratio": "1.850000000000000000" }
```
> Jika `ratio < 1.5` → posisi bisa dilikuidasi. Wallet harus warning user.

### Query Total ZYTD Supply
```
GET /zytherion/stablecoin/v1/total_supply
```

### Query Max Mintable
Berapa ZYTD maksimum yang bisa dicetak dari kolateral tertentu:
```
GET /zytherion/stablecoin/v1/max_mintable?ibc_denom=zytc&collateral_amount=5000000
```

### Liquidate (untuk bot likuidasi, bukan user biasa)
```json
{
  "@type": "/zytherion.stablecoin.MsgLiquidate",
  "liquidator": "zyth1...",
  "debtor": "zyth1...",
  "collateral_denom": "zytc"
}
```

---

## Staking

### Get Delegations
```
GET /cosmos/staking/v1beta1/delegations/{delegator_address}
```

### Delegate
```json
{
  "@type": "/cosmos.staking.v1beta1.MsgDelegate",
  "delegator_address": "zyth1...",
  "validator_address": "zythvaloper1...",
  "amount": { "denom": "zytc", "amount": "1000000" }
}
```

### Get Pending Rewards
```
GET /cosmos/distribution/v1beta1/delegators/{address}/rewards
```

### Claim Rewards
```json
{
  "@type": "/cosmos.distribution.v1beta1.MsgWithdrawDelegatorReward",
  "delegator_address": "zyth1...",
  "validator_address": "zythvaloper1..."
}
```

---

## Oracle Price (TWAP)

Denom yang didukung: `ZYTC`, `axlUSDC`, `mUSDT`, `ATOM`, `wBTC`, `wETH`

### Query Latest Price
```
GET /zytherion/oracle/v1/price/{denom}
```

### Query TWAP
```
GET /zytherion/oracle/v1/twap/{denom}
```

**Response:**
```json
{
  "denom": "ZYTC",
  "twap": "0.100000000000000000",
  "window_start": 12300,
  "window_end": 12330,
  "num_samples": 25
}
```

### Mencari Harga via TX Events
```
GET /cosmos/tx/v1beta1/txs?events=submit_price.denom%3D%27ZYTC%27&order_by=ORDER_BY_DESC&limit=10
```

---

## Transaction History

### Get Transaction by Hash
```
GET /cosmos/tx/v1beta1/txs/{hash}
```

### Search Transactions by Address
```
GET /cosmos/tx/v1beta1/txs?events=transfer.sender%3D%27{address}%27&order_by=ORDER_BY_DESC&limit=20
GET /cosmos/tx/v1beta1/txs?events=transfer.recipient%3D%27{address}%27&order_by=ORDER_BY_DESC&limit=20
```

### CometBFT TX Search (alternatif)
```
GET http://localhost:26657/tx_search?query="transfer.recipient='{address}'"&per_page=20&page=1&order_by="desc"
```

---

## IBC Transfer

### Send via IBC
```json
{
  "@type": "/ibc.applications.transfer.v1.MsgTransfer",
  "source_port": "transfer",
  "source_channel": "channel-0",
  "token": { "denom": "zytc", "amount": "1000000" },
  "sender": "zyth1...",
  "receiver": "cosmos1...",
  "timeout_timestamp": 1800000000000000000
}
```

---

## CosmWasm Smart Contract

### Query Contract State
```
GET /cosmwasm/wasm/v1/contract/{address}/smart/{base64_encoded_query}
```

### Execute Contract (via broadcast)
```json
{
  "@type": "/cosmwasm.wasm.v1.MsgExecuteContract",
  "sender": "zyth1...",
  "contract": "zyth1CONTRACT...",
  "msg": "<base64_encoded_execute_msg>",
  "funds": []
}
```

---

## Governance

### List Proposals
```
GET /cosmos/gov/v1beta1/proposals?proposal_status=PROPOSAL_STATUS_VOTING_PERIOD
```

### Vote
```json
{
  "@type": "/cosmos.gov.v1beta1.MsgVote",
  "proposal_id": "1",
  "voter": "zyth1...",
  "option": "VOTE_OPTION_YES"
}
```

---

## Node Status

### Check Node Sync Status
```
GET http://localhost:26657/status
```

**Response fields yang penting:**
- `result.sync_info.catching_up` — `false` = synced
- `result.sync_info.latest_block_height`
- `result.node_info.network` — harus `zytherion`

---

## Key Signing Requirements

| Komponen | Spesifikasi |
|----------|------------|
| Algorithm | **Dilithium5** (ML-DSA Level 5) |
| Public Key Size | 2,592 bytes |
| Signature Size | 4,595 bytes |
| Key Type String | `dilithium5` |
| Proto Type URL | `/zytherion.crypto.dilithium5.PubKey` |
| HD Derivation | HKDF-SHA256 (custom, bukan BIP44) |
| Bech32 Prefix | `zyth` |

> ⚠️ **Wallet developer:** Tidak bisa pakai library Cosmos SDK standar untuk signing! Perlu implementasi Dilithium5 terpisah. Referensi: `crypto/dilithium5/` di repo.

---

## WebSocket Streaming

Subscribe ke events real-time:
```
ws://localhost:26657/websocket
```

Subscribe ke semua transaksi:
```json
{ "jsonrpc": "2.0", "method": "subscribe", "id": 1,
  "params": { "query": "tm.event='Tx'" } }
```

Subscribe ke transaksi address tertentu:
```json
{ "jsonrpc": "2.0", "method": "subscribe", "id": 1,
  "params": { "query": "tm.event='Tx' AND transfer.recipient='zyth1...'" } }
```

---

## Fee Estimation

| Operasi | Gas Estimasi | Fee (zytc) |
|---------|-------------|-----------|
| Bank Send | ~100,000 | 50,000 uzytc |
| Mint ZYTD | ~200,000 | 100,000 uzytc |
| TFHE Deposit | ~500,000 | 250,000 uzytc |
| Delegate | ~150,000 | 75,000 uzytc |
| CosmWasm Execute | ~300,000+ | variable |

> Selalu gunakan `/simulate` sebelum broadcast untuk gas estimate akurat.
