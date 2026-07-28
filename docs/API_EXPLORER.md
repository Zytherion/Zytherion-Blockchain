# Zytherion Block Explorer API Reference

> Base URL REST: `http://localhost:1317`  
> Base URL RPC: `http://localhost:26657`  
> WebSocket: `ws://localhost:26657/websocket`

---

## Blocks

### Latest Block
```
GET http://localhost:26657/block
GET /cosmos/base/tendermint/v1beta1/blocks/latest
```

**Response (RPC):**
```json
{
  "result": {
    "block": {
      "header": {
        "height": "12345",
        "time": "2026-07-19T10:00:00Z",
        "chain_id": "zytherion",
        "proposer_address": "...",
        "num_txs": "3"
      },
      "data": {
        "txs": ["<base64_tx>", "..."]
      }
    }
  }
}
```

### Block by Height
```
GET http://localhost:26657/block?height={height}
GET /cosmos/base/tendermint/v1beta1/blocks/{height}
```

### Block Range (untuk halaman blok)
```
GET http://localhost:26657/blockchain?minHeight={min}&maxHeight={max}
```
Max range: 20 blok per request.

### Block Results (events per blok)
```
GET http://localhost:26657/block_results?height={height}
```
Berisi semua events yang diproduksi per transaksi dalam blok tersebut.

---

## Transactions

### Transaction by Hash
```
GET /cosmos/tx/v1beta1/txs/{hash}
GET http://localhost:26657/tx?hash=0x{hex_hash}
```

**Response:**
```json
{
  "tx": { ... },
  "tx_response": {
    "height": "12345",
    "txhash": "ABC123...",
    "code": 0,
    "raw_log": "...",
    "gas_wanted": "200000",
    "gas_used": "145000",
    "timestamp": "2026-07-19T10:00:00Z",
    "events": [
      {
        "type": "transfer",
        "attributes": [
          { "key": "sender", "value": "zyth1..." },
          { "key": "recipient", "value": "zyth1..." },
          { "key": "amount", "value": "1000000zytc" }
        ]
      }
    ]
  }
}
```
> `code: 0` = sukses. Selain 0 = gagal.

### Search Transactions
```
GET /cosmos/tx/v1beta1/txs?events={event_query}&order_by=ORDER_BY_DESC&limit={n}&offset={n}
```

**Event query examples:**
```
# Semua tx dari address
transfer.sender='zyth1...'

# Semua tx ke address
transfer.recipient='zyth1...'

# Tx di block tertentu
tx.height=12345

# Transaksi ZYTD mint
message.action='/zytherion.stablecoin.MsgMintZYTD'

# Deposit ke vault (CosmWasm)
wasm.action='deposit'

# Green BFT events
green_bft.mode='busy'

# Rent collection events
rent_collected.commitment_key exists

# Homomorphic transfer events
wasm._contract_address='{contract_addr}' AND wasm.action='confidential_transfer_zytd'
```

### CometBFT TX Search
```
GET http://localhost:26657/tx_search?query="{query}"&per_page=20&page=1&order_by="desc"
```

---

## Validators

### List All Validators
```
GET /cosmos/staking/v1beta1/validators?status=BOND_STATUS_BONDED&pagination.limit=100
```

**Response:**
```json
{
  "validators": [
    {
      "operator_address": "zythvaloper1...",
      "consensus_pubkey": { "@type": "...", "key": "..." },
      "status": "BOND_STATUS_BONDED",
      "tokens": "1000000000",
      "delegator_shares": "1000000000.000000000000000000",
      "description": {
        "moniker": "Validator Node 1",
        "website": "https://...",
        "details": "..."
      },
      "commission": {
        "commission_rates": {
          "rate": "0.100000000000000000",
          "max_rate": "0.200000000000000000",
          "max_change_rate": "0.010000000000000000"
        }
      }
    }
  ]
}
```

### Validator by Address
```
GET /cosmos/staking/v1beta1/validators/{validator_address}
```

### Validator Uptime / Signing Info
```
GET /cosmos/slashing/v1beta1/signing_infos
GET /cosmos/slashing/v1beta1/signing_infos/{cons_address}
```

### CometBFT Validators (consensus layer)
```
GET http://localhost:26657/validators?height={height}&per_page=100
```

### Green BFT Metrics (Zytherion-specific)
Tersedia via ABCI events di setiap block result:
```
GET http://localhost:26657/block_results?height={height}
```
Cari event `green_bft` di `end_block_events`:
```json
{
  "type": "green_bft",
  "attributes": [
    { "key": "recommended_timeout_ms", "value": "800" },
    { "key": "block_tx_count", "value": "15" },
    { "key": "mode", "value": "busy" }
  ]
}
```

---

## Network Stats

### Node Status
```
GET http://localhost:26657/status
```

**Fields penting:**
```json
{
  "result": {
    "node_info": {
      "network": "zytherion",
      "moniker": "mynode",
      "version": "0.38.x"
    },
    "sync_info": {
      "latest_block_height": "12345",
      "latest_block_time": "2026-07-19T10:00:00Z",
      "catching_up": false
    },
    "validator_info": {
      "address": "...",
      "voting_power": "1000000"
    }
  }
}
```

### Network Info (peers)
```
GET http://localhost:26657/net_info
```

### Consensus State
```
GET http://localhost:26657/consensus_state
```

### ABCI Info
```
GET http://localhost:26657/abci_info
```

---

## Token Supply & Economics

### Total Supply
```
GET /cosmos/bank/v1beta1/supply
```

### Supply by Denom
```
GET /cosmos/bank/v1beta1/supply/by_denom?denom=zytc
GET /cosmos/bank/v1beta1/supply/by_denom?denom=zytd
```

### Staking Pool (Bonded vs Unbonded)
```
GET /cosmos/staking/v1beta1/pool
```

### Inflation & Mint Params
```
GET /cosmos/mint/v1beta1/inflation
GET /cosmos/mint/v1beta1/annual_provisions
GET /cosmos/mint/v1beta1/params
```

---

## Accounts & Richlist

### All Accounts (pagination)
```
GET /cosmos/auth/v1beta1/accounts?pagination.limit=100&pagination.offset=0
```

### Account Info
```
GET /cosmos/auth/v1beta1/accounts/{address}
```

---

## Staking & Delegation

### All Delegations to Validator
```
GET /cosmos/staking/v1beta1/validators/{validator_address}/delegations?pagination.limit=100
```

### Delegation by Delegator
```
GET /cosmos/staking/v1beta1/delegations/{delegator_address}
```

### Unbonding Delegations
```
GET /cosmos/staking/v1beta1/delegators/{delegator_address}/unbonding_delegations
```

---

## Governance

### All Proposals
```
GET /cosmos/gov/v1beta1/proposals?pagination.limit=50
```

### Proposal by ID
```
GET /cosmos/gov/v1beta1/proposals/{proposal_id}
```

### Votes for Proposal
```
GET /cosmos/gov/v1beta1/proposals/{proposal_id}/votes?pagination.limit=100
```

### Tally Result
```
GET /cosmos/gov/v1beta1/proposals/{proposal_id}/tally
```

---

## IBC

### IBC Channels
```
GET /ibc/core/channel/v1/channels
```

### IBC Connections
```
GET /ibc/core/connection/v1/connections
```

### IBC Transfer Denom Traces
```
GET /ibc/apps/transfer/v1/denom_traces
```

---

## Oracle Prices (Zytherion-specific)

Oracle TWAP diakses melalui keeper query atau event parsing.

Denom yang didukung: `ZYTC`, `axlUSDC`, `mUSDT`, `ATOM`, `wBTC`, `wETH`

Search events untuk price submissions:
```
GET /cosmos/tx/v1beta1/txs?events=submit_price.denom='ZYTC'&order_by=ORDER_BY_DESC&limit=10
```

TWAP event attributes:
```
submit_price.denom
submit_price.price
submit_price.submitter
```

---

## CosmWasm Contracts

### List All Contracts by Code
```
GET /cosmwasm/wasm/v1/code/{code_id}/contracts
```

### Contract Info
```
GET /cosmwasm/wasm/v1/contract/{address}
```

### Contract History
```
GET /cosmwasm/wasm/v1/contract/{address}/history
```

### Contract State (raw KV)
```
GET /cosmwasm/wasm/v1/contract/{address}/state?pagination.limit=100
```

### Contract Smart Query
```
GET /cosmwasm/wasm/v1/contract/{address}/smart/{base64_query}
```

Contoh query vault info:
```bash
# base64 dari '{"vault_info":{}}'
echo -n '{"vault_info":{}}' | base64
# → eyJ2YXVsdF9pbmZvIjp7fX0=

GET /cosmwasm/wasm/v1/contract/zyth1.../smart/eyJ2YXVsdF9pbmZvIjp7fX0=
```

### List All Codes (uploaded WASMs)
```
GET /cosmwasm/wasm/v1/code?pagination.limit=50
```

---

## Privacy Module Events (Zytherion-specific)

Search TFHE-related events untuk explorer:
```
# Commitment submitted
wasm.action='deposit'

# Rent collected
rent_collected.commitment_key exists

# Data entered grace period (akan dihapus)
rent_default.commitment_key exists

# Data dievict (sudah dihapus, emit sebelum prune)
commitment_evicted.commitment_key exists
```

---

## WebSocket Real-time Streaming

```
ws://localhost:26657/websocket
```

**Subscribe new blocks:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "subscribe",
  "params": { "query": "tm.event='NewBlock'" }
}
```

**Subscribe all transactions:**
```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "subscribe",
  "params": { "query": "tm.event='Tx'" }
}
```

**Subscribe txs ke address tertentu:**
```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "subscribe",
  "params": { "query": "tm.event='Tx' AND transfer.recipient='zyth1...'" }
}
```

**Unsubscribe:**
```json
{ "jsonrpc": "2.0", "id": 99, "method": "unsubscribe_all", "params": {} }
```

---

## Pagination

Semua endpoint list mendukung pagination:
```
?pagination.limit=20
?pagination.offset=0
?pagination.key=<next_key_base64>  ← lebih efisien untuk cursor-based
?pagination.reverse=true
?pagination.count_total=true       ← tambahkan total count di response
```

---

## Decoding Transaksi

Transaksi di blok tersimpan dalam base64-encoded protobuf.  
Decode via:
```
POST /cosmos/tx/v1beta1/decode
Body: { "tx_bytes": "<base64_tx>" }
```

---

## Konfigurasi Chain

| Parameter | Nilai |
|-----------|-------|
| Chain ID | `zytherion` |
| Bech32 Prefix Account | `zyth` |
| Bech32 Prefix Validator | `zythvaloper` |
| Bech32 Prefix Consensus | `zythvalcons` |
| Native Denom | `zytc` |
| Stablecoin Denom | `zytd` |
| Block Time Target | ~6 detik |
| Max Validators | 100 |
| Signing Algorithm | Dilithium5 (ML-DSA) |
