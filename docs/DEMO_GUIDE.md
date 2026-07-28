# Zytherion Demo Guide

**Quantum Account → Homomorphic Smart Contract — End-to-End**

> Pastikan chain sudah berjalan:
> ```bash
> cd ~/zytherion
> ignite chain serve --build.tags "tfhe_cgo" --reset-once
> ```

---

## 1. Buat Akun Quantum (Dilithium5)

```bash
~/go/bin/zytheriond keys add myaccount --key-type dilithium5
```

Verifikasi pubkey bertipe post-quantum:

```bash
~/go/bin/zytheriond keys show myaccount --keyring-backend test --output json
```

Output yang diharapkan di field `pubkey`:

```
"@type": "/zytherion.crypto.dilithium5.PubKey"
```

---

## 2. Dapatkan Token dari Genesis Account

Cek address akun baru:

```bash
~/go/bin/zytheriond keys show myaccount --keyring-backend test -a
```

Transfer 10 juta `zytc` dari genesis account `alice`:

```bash
~/go/bin/zytheriond tx bank send alice \
  $(~/go/bin/zytheriond keys show myaccount --keyring-backend test -a) \
  10000000zytc \
  --chain-id zytherion \
  --node tcp://localhost:26657 \
  --keyring-backend test \
  --fees 50000zytc \
  -y
```

Cek saldo setelah ~3 detik:

```bash
~/go/bin/zytheriond query bank balance \
  $(~/go/bin/zytheriond keys show myaccount --keyring-backend test -a) \
  zytc \
  --node tcp://localhost:26657
```

---

## 3. Build Smart Contract (Rust → WASM)

```bash
rustup target add wasm32-unknown-unknown

cd ~/zytherion/contracts/homomorphic_vault
cargo build --release --target wasm32-unknown-unknown

ls -lh target/wasm32-unknown-unknown/release/homomorphic_vault.wasm
cd ~/zytherion
```

---

## 4. Upload Contract ke Chain

```bash
~/go/bin/zytheriond tx wasm store \
  contracts/homomorphic_vault/target/wasm32-unknown-unknown/release/homomorphic_vault.wasm \
  --from myaccount \
  --chain-id zytherion \
  --node tcp://localhost:26657 \
  --keyring-backend test \
  --fees 50000zytc \
  --gas auto \
  --gas-adjustment 1.5 \
  -y
```

Cek Code ID yang ter-upload:

```bash
~/go/bin/zytheriond query wasm list-code \
  --node tcp://localhost:26657 \
  --output json
```

Catat `code_id` dari output (biasanya `1` kalau ini contract pertama).

---

## 5. Instantiate homomorphic\_vault

Ganti `zyth1...` dengan address `myaccount` kamu:

```bash
~/go/bin/zytheriond tx wasm instantiate 1 \
  '{"label":"My Quantum Vault","owner":"zyth1...GANTI_DENGAN_ALAMAT_KAMU"}' \
  --from myaccount \
  --label "homomorphic-vault-v1" \
  --chain-id zytherion \
  --node tcp://localhost:26657 \
  --keyring-backend test \
  --fees 50000zytc \
  --gas auto \
  --gas-adjustment 1.5 \
  -y
```

Cek contract address:

```bash
~/go/bin/zytheriond query wasm list-contract-by-code 1 \
  --node tcp://localhost:26657 \
  --output json
```

Catat `contracts[0]` — itulah alamat contract kamu.

---

## 6. Query Vault Info

Ganti `zyth1CONTRACT...` dengan contract address dari step 5:

```bash
~/go/bin/zytheriond query wasm contract-state smart \
  zyth1CONTRACT... \
  '{"vault_info":{}}' \
  --node tcp://localhost:26657 \
  --output json
```

Output yang diharapkan:

```json
{
  "data": {
    "label": "My Quantum Vault",
    "owner": "zyth1...",
    "deposit_count": 0,
    "transfer_count": 0
  }
}
```

---

## 7. Generate Ciphertext TFHE

Buat file helper untuk encrypt nilai:

```bash
cat > /tmp/tfhe_encrypt.go << 'EOF'
//go:build ignore

package main

import (
    "encoding/base64"
    "fmt"
    "os"
    "strconv"

    tfhe "zytherion/x/privacy/tfhe"
)

func main() {
    value := uint32(42)
    if len(os.Args) > 1 {
        v, _ := strconv.ParseUint(os.Args[1], 10, 32)
        value = uint32(v)
    }
    ck, _, err := tfhe.EnsureNodeKeys(os.Getenv("HOME"))
    if err != nil {
        fmt.Fprintf(os.Stderr, "key error: %v\n", err)
        os.Exit(1)
    }
    ct, err := tfhe.EncryptUint32(ck, value)
    if err != nil {
        fmt.Fprintf(os.Stderr, "encrypt error: %v\n", err)
        os.Exit(1)
    }
    fmt.Printf("value=%d size=%d bytes\n", value, len(ct))
    fmt.Println(base64.StdEncoding.EncodeToString(ct))
}
EOF
```

Encrypt nilai `42`:

```bash
cd ~/zytherion
go run -tags tfhe_cgo /tmp/tfhe_encrypt.go 42
```

> ⏳ Pertama kali bisa butuh **10–60 detik** untuk generate TFHE keys.
> Keys disimpan di `~/.zytherion_tfhe_client.key` — selanjutnya instan.

Contoh output:

```
value=42 size=18432 bytes
AAAABQAAAAAAAABm7+3P... (base64 panjang)
```

Salin baris base64 (baris kedua) untuk dipakai di step berikutnya.

---

## 8. Deposit Encrypted Amount

Ganti `BASE64_CIPHERTEXT_42` dengan output base64 dari step 7:

```bash
~/go/bin/zytheriond tx wasm execute zyth1CONTRACT... \
  '{"deposit":{"encrypted_amount":"BASE64_CIPHERTEXT_42","memo":"deposit pertamaku"}}' \
  --from myaccount \
  --chain-id zytherion \
  --node tcp://localhost:26657 \
  --keyring-backend test \
  --fees 50000zytc \
  --gas auto \
  --gas-adjustment 1.5 \
  -y
```

Query encrypted balance setelah ~3 detik:

```bash
~/go/bin/zytheriond query wasm contract-state smart \
  zyth1CONTRACT... \
  '{"encrypted_balance":{}}' \
  --node tcp://localhost:26657 \
  --output json
```

> 🔐 **Yang terjadi:** Validator menyimpan ciphertext terenkripsi di state.
> Mereka **tidak tahu** bahwa nilainya adalah `42`.

---

## 9. Homomorphic Add — Penjumlahan Tanpa Decrypt

Generate ciphertext kedua untuk nilai `100`:

```bash
cd ~/zytherion
go run -tags tfhe_cgo /tmp/tfhe_encrypt.go 100
```

Hitung `Enc(42) + Enc(100)` langsung di chain — hasilnya tetap terenkripsi:

```bash
~/go/bin/zytheriond query wasm contract-state smart \
  zyth1CONTRACT... \
  '{"homomorphic_add":{"ct1":"BASE64_CT_42","ct2":"BASE64_CT_100"}}' \
  --node tcp://localhost:26657 \
  --output json
```

Output adalah `Enc(142)` — ciphertext dari 42+100, **tanpa chain pernah melihat 42 atau 100**.

---

## 10. Decrypt untuk Verifikasi

Buat file helper decrypt:

```bash
cat > /tmp/tfhe_decrypt.go << 'EOF'
//go:build ignore

package main

import (
    "encoding/base64"
    "fmt"
    "os"

    tfhe "zytherion/x/privacy/tfhe"
)

func main() {
    if len(os.Args) < 2 {
        fmt.Fprintln(os.Stderr, "Usage: go run -tags tfhe_cgo tfhe_decrypt.go <base64_ct>")
        os.Exit(1)
    }
    ct, err := base64.StdEncoding.DecodeString(os.Args[1])
    if err != nil {
        fmt.Fprintf(os.Stderr, "invalid base64: %v\n", err)
        os.Exit(1)
    }
    ck, _, err := tfhe.EnsureNodeKeys(os.Getenv("HOME"))
    if err != nil {
        fmt.Fprintf(os.Stderr, "key error: %v\n", err)
        os.Exit(1)
    }
    val, err := tfhe.DecryptUint32(ck, ct)
    if err != nil {
        fmt.Fprintf(os.Stderr, "decrypt error: %v\n", err)
        os.Exit(1)
    }
    fmt.Printf("Decrypted: %d\n", val)
}
EOF
```

Decrypt hasil penjumlahan (ganti dengan base64 dari output step 9):

```bash
cd ~/zytherion
go run -tags tfhe_cgo /tmp/tfhe_decrypt.go BASE64_HASIL_ADD
# Output: Decrypted: 142  ✓
```

Decrypt balance vault:

```bash
cd ~/zytherion
go run -tags tfhe_cgo /tmp/tfhe_decrypt.go BASE64_ENCRYPTED_BALANCE
# Output: Decrypted: 42  ✓
```

---

## 11. Vault Info Akhir

```bash
~/go/bin/zytheriond query wasm contract-state smart \
  zyth1CONTRACT... \
  '{"vault_info":{}}' \
  --node tcp://localhost:26657 \
  --output json
```

```json
{
  "data": {
    "label": "My Quantum Vault",
    "owner": "zyth1...",
    "deposit_count": 1,
    "transfer_count": 0
  }
}
```

---

## Ringkasan

```
Kamu baru saja membuktikan:

  ✅ Akun Dilithium5 (post-quantum, tahan serangan komputer kuantum)
  ✅ Smart contract CosmWasm berjalan di chain Cosmos
  ✅ Nilai 42 disimpan on-chain dalam bentuk terenkripsi
  ✅ Validator melakukan 42+100=142 tanpa tahu angka-angkanya
  ✅ Hanya pemegang client key yang bisa decrypt hasilnya
```

---

## Troubleshooting

| Error | Solusi |
|-------|--------|
| `cannot find -ltfhe_c` | `cd ~/zytherion/x/privacy/tfhe/tfhe_c && cargo build --release` |
| `algorithm "dilithium5" is not supported` | Pastikan pakai `~/go/bin/zytheriond`, chain dijalankan dengan `--build.tags "tfhe_cgo"` |
| `out of gas` | Tambah flag `--gas-adjustment 2.0` |
| Helper script gagal | Pastikan ada `-tags tfhe_cgo` di command `go run` |
| Keys generate lama | Normal — hanya sekali, selanjutnya load dari disk |
