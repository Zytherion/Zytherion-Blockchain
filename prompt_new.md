# Zytherion Blockchain — Verified Command Guide (`PROMPT_NEW.MD`)

Panduan ringkas ini berisi perintah-perintah CLI dan REST API Zytherion Blockchain yang **100% teruji dan berfungsi**.

---

## 🔒 1. TFHE PRIVACY (SUBMIT & QUERY RESULT)

### Langkah 1: Buat file ciphertext (20 KB)
```bash
head -c 20480 /dev/urandom > data1.bin
```

### Langkah 2: Submit TFHE Ciphertext
```bash
zytheriond tx privacy tfhe-submit \
  --ciphertext data1.bin \
  --from alice \
  --chain-id zytherion \
  --keyring-backend test \
  --gas 500000 \
  -y
```

### Langkah 3: Cek Tx hash untuk mengambil commitment hash
```bash
zytheriond q tx <TX_HASH>
```

### Langkah 4: Query TFHE Result

#### Option A: Menggunakan CLI
```bash
zytheriond query privacy tfhe-result --commitment <COMMITMENT_HASH_HEX>
```

#### Option B: Menggunakan REST HTTP Endpoint (curl)
```bash
curl http://localhost:1317/zytherion/privacy/v1/tfhe/result/<COMMITMENT_HASH_HEX>
```

---

## 🔮 2. ORACLE PRICE FEED

Set harga oracle untuk denom collateral `uzytc`:
```bash
zytheriond tx oracle submit-price uzytc 1.00 \
  --from alice \
  --chain-id zytherion \
  --keyring-backend test \
  --gas 500000 \
  -y
```

---

## 🪙 3. STABLECOIN (MINTING ZYTD)

Minting token ZYTD dengan gas limit 500.000:
```bash
zytheriond tx stablecoin mint-zytd \
  --collateral-denom uzytc \
  --collateral-amount 2000000000 \
  --zytd-amount 1000000000 \
  --expiration-block-height 1000 \
  --from alice \
  --chain-id zytherion \
  --keyring-backend test \
  --fees 5000zytc \
  --gas 500000 \
  -y
```

---

## ⚡ 4. MANAJEMEN AKUN, SEND BALANCE, STAKING & REDELEGATION

### A. Membuat Akun Wallet Baru (Node 2)
```bash
zytheriond keys add node2_key --keyring-backend test
```

### B. Kirim Saldo (Transfer Token dari Alice ke Akun Baru)
```bash
zytheriond tx bank send alice <NODE2_WALLET_ADDRESS> 600000000000zytc \
  --chain-id zytherion \
  --keyring-backend test \
  --fees 5000zytc \
  --gas 500000 \
  -y
```

### C. Buat Validator di Node 2
```bash
zytheriond tx staking create-validator \
  --amount=500000000000zytc \
  --pubkey=$(zytheriond tendermint show-validator) \
  --moniker="node2-validator" \
  --chain-id=zytherion \
  --commission-rate="0.10" \
  --commission-max-rate="0.20" \
  --commission-max-change-rate="0.01" \
  --min-self-delegation="1" \
  --from=rayhan \
  --keyring-backend=test \
  --fees=5000zytc \
  --gas=1000000 \
  --node tcp://localhost:3724 \
  -y
```

### D. Instant Redelegate Voting Power ke Node 2
```bash
zytheriond tx staking redelegate \
  zythvaloper1ska696lz9h44g6gysrp0tg5j7lvqd65qzwckt6 \
  zythvaloper1j3gndh7jruxwkzt2tcxaytvpvm9gr7xuz2hls2 \
  70000000000000zytc \
  --from alice \
  --chain-id zytherion \
  --keyring-backend test \
  --fees 5000zytc \
  --gas 1000000 \
  -y
```

### E. Cek Status Voting Power
```bash
zytheriond query staking validators --output json | jq '.validators[] | {moniker: .description.moniker, tokens: .tokens}'
```
