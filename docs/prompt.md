# Zytherion Blockchain v0.7 — Panduan Perintah CLI & Modul (Quantum-Only Stack)

**Project:** Zytherion Blockchain and Cryptocurrency  
**Version:** 0.7.0 (QuantumBFT + ML-KEM)  
**Chain ID:** `zytherion`  
**Founder:** Rayhan Aziel Abbrar  
**Repository:** https://github.com/Zytherion/Zytherion-Blockchain  

---

> ⚡ **RULE UTAMA (V0.7 MANDATE): QUANTUM-ONLY CRYPTOGRAPHY**
> 
> Seluruh sistem Zytherion v0.7 beroperasi di bawah aturan **Quantum-Only**:
> 1. **Tanda Tangan Digital (Signing):** Wajib **Dilithium5 (ML-DSA Level 5)**. Algoritma klasik (secp256k1/ed25519) dilarang untuk transaksi/konsensus utama.
> 2. **Key Encapsulation / Key Exchange (KEM):** Wajib **CRYSTALS-Kyber1024 (ML-KEM-1024)**.
> 3. **Transport P2P:** Wajib **Hybrid Kyber1024 + X25519 SecretConnection**.
> 4. **Privasi & Enkripsi Data:** Wajib **TFHE (FheUint32)**.

---

## 🔑 Spesifikasi & Ukuran Kunci Post-Quantum (v0.7)

Ukuran kunci di Zytherion jauh lebih besar dibanding algoritma klasik karena menggunakan matematika *lattice-based*:

| Algoritma | Jenis / Fungsi | Public Key Size | Private Key Size | Subcommand CLI |
|---|---|---|---|---|
| **Dilithium5 (ML-DSA-87)** | Digital Signature (Blok & Akun) | **2,592 bytes** | **4,858 bytes** | `zytheriond keys add <nama>` *(Default v0.7)* |
| **Kyber1024 (ML-KEM-1024)** | Key Encapsulation (Enkripsi File & KEM) | **1,568 bytes** | **3,168 bytes** | `zytheriond keys kyber keygen` |
| **TFHE (FheUint32)** | Homomorphic Encryption (Kalkulasi Rahasia) | **112 MB** *(ServerKey)* | **22.6 KB** *(ClientKey)* | `zytheriond keys tfhe keygen` |

---

## 1. Manajemen Akun & Transaksi Dasar ZYTC

### A. Memeriksa Saldo Akun (`q bank balances`)
**Fungsi:** Melihat jumlah koin ZYTC atau token lainnya yang dimiliki oleh alamat akun Alice.
```bash
zytheriond q bank balances zyth1qfd89spzuajhdpwwhtu9k2rtetwcff2g2ncq8g \
  --node tcp://localhost:26657
```

### B. Memeriksa Detail Akun & Sequence Number (`q auth account`)
**Fungsi:** Melihat `account_number` dan `sequence` (nonce transaksi terkini) untuk mencegah kesalahan transaksi berurutan.
```bash
zytheriond q auth account zyth1qfd89spzuajhdpwwhtu9k2rtetwcff2g2ncq8g \
  --node tcp://localhost:26657
```

### C. Mengirim Koin ZYTC (`tx bank send`)
**Fungsi:** Mentransfer koin ZYTC dari akun Alice (`zyth1qfd89...`) ke Bob (`zyth19x06f...`) secara publik.
```bash
zytheriond tx bank send alice zyth19x06fdff9e04xdyknpzpugzwjklqu2q3uuu2cv 10000zytc \
  --chain-id zytherion \
  --gas auto \
  --gas-adjustment 1.5 \
  --fees 5000zytc \
  --broadcast-mode sync \
  --node tcp://localhost:26657 \
  -y
```

### D. Memeriksa Status Transaksi (`q tx`)
**Fungsi:** Melihat status eksekusi transaksi di dalam blok menggunakan TX Hash riil.
```bash
zytheriond q tx C6589111915CBBC6C1EF77559F734A1D75D327C8D27838671E5F2DE77555147C \
  --node tcp://localhost:26657
```

---

## 2. Confidential Stablecoin ZYTD

Modul `x/stablecoin` mengelola mata uang stabil **ZYTD** (Zytherion Dollar) dengan opsi transaksi terenkripsi (**Confidential Transfer**) menggunakan skema TFHE (Fully Homomorphic Encryption).

### A. Mint ZYTD (`tx stablecoin mint-zytd`)
**Fungsi:** Mencetak `1,000 ZYTD` baru dengan menjaminkan `2,000 uzytc`.
```bash
zytheriond tx stablecoin mint-zytd 1000000000zytd uzytc 2000000000uzytc \
  --from alice \
  --chain-id zytherion \
  --fees 5000zytc \
  --broadcast-mode sync \
  --node tcp://localhost:26657 \
  -y
```

### B. Burn ZYTD (`tx stablecoin burn-zytd`)
**Fungsi:** Membakar `500 ZYTD` untuk menarik kembali kolateral yang dijaminkan.
```bash
zytheriond tx stablecoin burn-zytd 500000000zytd \
  --from alice \
  --chain-id zytherion \
  --fees 5000zytc \
  --broadcast-mode sync \
  --node tcp://localhost:26657 \
  -y
```

### C. Confidential Transfer ZYTD (`tx stablecoin confidential-transfer`)
**Fungsi:** Mentransfer ZYTD terenkripsi dari Alice ke Bob di mana publik hanya melihat string Hex terenkripsi tanpa mengetahui nominal transaksi.
```bash
zytheriond tx stablecoin confidential-transfer zyth19x06fdff9e04xdyknpzpugzwjklqu2q3uuu2cv \
  02000000a4f891b2c5e3d710984aef01c385617a2b90efd4310a887c99e120f4b7a192837465019283f3e2d1c0b9a8f7e6d5c4b3a291807f6e5d4c3b2a1908 \
  01000000bf7120a3c9e8d7f6e5d4c3b2a1908f7e6d5c4b3a291807f6e5d4c3b2a1908f7e6d5c4b3a291807f6e5d4c3b2a1908f7e6d5c4b3a291807f6e5d4c3b2 \
  --from alice \
  --chain-id zytherion \
  --fees 5000zytc \
  --broadcast-mode sync \
  --node tcp://localhost:26657 \
  -y
```

---

#### 💡 PENJELASAN DATA RIIL PARAMETER TERENKRIPSI

1. **Ciphertext Base64 (`AAAA...base64...BBBB`)**:
   * **Asal Data:** Dihasilkan secara lokal di perangkat pengguna menggunakan subcommand bawaan `zytheriond`:
     ```bash
     zytheriond keys tfhe encrypt 5000
     ```
     ⚠️ **Tidak perlu source code, tidak perlu tfhe-tool terpisah** — cukup binary `zytheriond` saja.
   * **Fungsi:** Mengubah nominal angka `5,000 ZYTD` menjadi ciphertext Base64 terenkripsi TFHE FheUint32. Validator menambahkan ciphertext ini ke saldo penerima tanpa pernah tahu nilainya.

2. **ZK-Proof Hex (`01000000bf7120a3...`)**:
   * **Asal Data:** Dihasilkan secara lokal di perangkat Alice oleh generator Zero-Knowledge Range Proof (Bulletproofs) sebelum dikirim.
   * **Fungsi:** Membuktikan secara sah ke validator bahwa:
     - Nominal terenkripsi bernilai **Positif ($\ge 0$)** (mencegah manipulasi angka minus).
     - Nominal terenkripsi **tidak melebihi total saldo ZYTD Alice**.
     - *Validasi ini sah 100% tanpa membocorkan isi saldo Alice.*

---

### D. Memeriksa Saldo ZYTD Terenkripsi (`q stablecoin confidential-balance`)
**Fungsi:** Membaca ciphertext saldo ZYTD terenkripsi milik Alice dari state blockchain.
```bash
zytheriond q stablecoin confidential-balance zyth1qfd89spzuajhdpwwhtu9k2rtetwcff2g2ncq8g \
  --node tcp://localhost:26657
```

---

## 3. Fitur Privasi & Penggunaan TFHE (Native Modul & Smart Contract)

### 3.1. Penggunaan TFHE NATIVE (Tanpa Smart Contract)

Metode ini menggunakan subcommand **bawaan `zytheriond`** langsung — cukup dengan binary `zytheriond`, tidak perlu source code, tidak perlu tool terpisah.

> ⚠️ **PENTING:** Semua perintah TFHE di bawah ini menggunakan `zytheriond keys tfhe ...`.
> Hanya butuh 1 binary: `zytheriond` yang sudah dikompilasi dengan TFHE static linked.

#### A. Generate Kunci TFHE Pribadi (Satu Kali)
**Fungsi:** Membuat `client.key` (rahasia, disimpan lokal) dan `server.key` (diunggah ke jaringan untuk komputasi terenkripsi). Wajib dilakukan sekali sebelum pakai TFHE.
```bash
zytheriond keys tfhe keygen
```
*Output:*
```
Keys generated and saved successfully!
  Client Key (secret): /home/zhaohan/.zytherion/tfhe/client.key
  Server Key (public): /home/zhaohan/.zytherion/tfhe/server.key
```
⚠️ **`client.key` adalah kunci rahasia — jangan pernah dibagikan ke siapapun!**

#### B. Enkripsi Nominal Angka (Misalnya 5000)
**Fungsi:** Mengubah angka `5000` menjadi Base64 ciphertext TFHE. Output inilah yang dikirim ke blockchain sebagai `ENCRYPTED_AMOUNT`.
```bash
zytheriond keys tfhe encrypt 5000
# Atau dengan path client.key eksplisit:
zytheriond keys tfhe encrypt 5000 ~/.zytherion/tfhe/client.key
```
*Output Base64 ciphertext (salin seluruh string ini):*
```
AgAAAKT4kbLF49cQmErvAcOFYXorlO/UMQqIfJnhIPSn...
```

#### C. Mendaftarkan ServerKey TFHE ke Blockchain (`tx privacy register-server-key`)
**Fungsi:** Mempublikasikan `server.key` ke state `x/privacy` agar validator dapat melakukan penjumlahan/perkalian terenkripsi atas nama akun pengguna.
```bash
# Ambil isi server.key dalam format Base64 dulu:
base64 ~/.zytherion/tfhe/server.key

# Lalu daftarkan:
zytheriond tx privacy register-server-key $(base64 ~/.zytherion/tfhe/server.key) \
  --from alice \
  --chain-id zytherion \
  --fees 5000zytc \
  --broadcast-mode sync \
  --node tcp://localhost:26657 \
  -y
```

#### D. Kirim Transfer Terenkripsi (`tx privacy confidential-send`)
**Fungsi:** Transfer koin terenkripsi langsung via modul `x/privacy`. Publik hanya melihat Base64 acak, tidak tahu nominalnya.
```bash
# Step 1: Encrypt dulu nominalnya
ENCRYPTED=$(zytheriond keys tfhe encrypt 5000)

# Step 2: Kirim transaksi
zytheriond tx privacy confidential-send \
  zyth19x06fdff9e04xdyknpzpugzwjklqu2q3uuu2cv \
  "$ENCRYPTED" \
  --from alice \
  --chain-id zytherion \
  --fees 5000zytc \
  --broadcast-mode sync \
  --node tcp://localhost:26657 \
  -y
```

#### E. Penjumlahan Homomorfik On-Chain ($C_3 = C_1 + C_2$) (`tx privacy execute-homomorphic-add`)
**Fungsi:** Meminta validator menjumlahkan dua ciphertext di blockchain tanpa mendekripsi nilainya. Validator hanya tahu hasilnya adalah $C_3$ tanpa tahu nilai $C_1$ atau $C_2$.
```bash
CT1=$(zytheriond keys tfhe encrypt 1000)
CT2=$(zytheriond keys tfhe encrypt 4000)

zytheriond tx privacy execute-homomorphic-add \
  "$CT1" \
  "$CT2" \
  --from alice \
  --chain-id zytherion \
  --fees 5000zytc \
  --broadcast-mode sync \
  --node tcp://localhost:26657 \
  -y
```

#### F. Dekripsi Ciphertext Hasil dari Blockchain
**Fungsi:** Mengunduh ciphertext dari blockchain lalu mendekripsinya secara lokal menggunakan `client.key` milik sendiri. Hanya pemilik `client.key` yang bisa melihat angka aslinya.
```bash
# Query ciphertext saldo dari chain
CT_RESULT=$(zytheriond q privacy confidential-balance zyth1qfd89spzuajhdpwwhtu9k2rtetwcff2g2ncq8g \
  --node tcp://localhost:26657 --output json | jq -r '.ciphertext')

# Decrypt secara lokal
zytheriond keys tfhe decrypt "$CT_RESULT"
# Atau dengan path client.key eksplisit:
zytheriond keys tfhe decrypt "$CT_RESULT" ~/.zytherion/tfhe/client.key
```
*Output:*
```
Decrypted value: 5000
```
*(Hanya pemilik `client.key` yang bisa melihat angka 5000 ini)*

---

### 3.2. Homomorphic Smart Contract (CosmWasm × TFHE)

Smart Contract berbasis Rust/CosmWasm di Zytherion mengeksekusi komputasi homomorfik di dalam VM WASM via TFHE Custom Querier plugin.

#### A. Mengunggah Kode WASM (`tx wasm store`)
```bash
zytheriond tx wasm store contracts/homomorphic_vault.wasm \
  --from alice \
  --chain-id zytherion \
  --gas auto \
  --gas-adjustment 1.5 \
  --fees 10000zytc \
  --broadcast-mode sync \
  --node tcp://localhost:26657 \
  -y
```

#### B. Inisialisasi Smart Contract (`tx wasm instantiate`)
```bash
zytheriond tx wasm instantiate 1 '{"owner":"zyth1qfd89spzuajhdpwwhtu9k2rtetwcff2g2ncq8g"}' \
  --label "TFHE Vault" \
  --no-admin \
  --from alice \
  --chain-id zytherion \
  --fees 5000zytc \
  --broadcast-mode sync \
  --node tcp://localhost:26657 \
  -y
```

#### C. Deposito Terenkripsi ke Contract (`tx wasm execute`)
```bash
zytheriond tx wasm execute zyth14hj2tavq8fpesw2wpp4w2z0p5tw2c8ph6h6khn \
  '{"deposit_encrypted":{"ciphertext":"02000000a4f891b2c5e3d710984aef01c385617a2b90efd4310a887c99e120f4b7a192837465019283f3e2d1c0b9a8f7e6d5c4b3a291807f6e5d4c3b2a1908"}}' \
  --from alice \
  --chain-id zytherion \
  --fees 5000zytc \
  --broadcast-mode sync \
  --node tcp://localhost:26657 \
  -y
```

#### D. Memeriksa State Contract Terenkripsi (`q wasm contract-state smart`)
```bash
zytheriond q wasm contract-state smart zyth14hj2tavq8fpesw2wpp4w2z0p5tw2c8ph6h6khn \
  '{"get_encrypted_balance":{"user":"zyth1qfd89spzuajhdpwwhtu9k2rtetwcff2g2ncq8g"}}' \
  --node tcp://localhost:26657
```

---

## 4. IBC Collateral & Price Oracle

### A. Memeriksa Harga Oracle & TWAP (`q oracle price` / `twap`)
```bash
# Harga Terkini ZYTC
zytheriond q oracle price ZYTC --node tcp://localhost:26657

# TWAP (Time-Weighted Average Price) ZYTC
zytheriond q oracle twap ZYTC --node tcp://localhost:26657
```

### B. Mengunci Kolateral IBC (`tx ibccollateral lock-collateral`)
```bash
zytheriond tx ibccollateral lock-collateral ibc/ATOM 50000000 \
  --from alice \
  --chain-id zytherion \
  --fees 5000zytc \
  --broadcast-mode sync \
  --node tcp://localhost:26657 \
  -y
```

---

## 5. Staking & Voting Power

### A. Mengirim Delegasi Tambahan ke Laptop (`tx staking delegate`)
Meningkatkan voting power Laptop (`zythvaloper1qkacpnrz...`) menjadi ~68.7% agar dapat memproduksi blok secara mandiri saat PC offline.
```bash
zytheriond tx staking delegate zythvaloper1qkacpnrzqmejfprfpepjky800aheddmsdchlf5 60000000000000zytc \
  --from alice \
  --chain-id zytherion \
  --gas auto \
  --gas-adjustment 1.5 \
  --fees 5000zytc \
  --broadcast-mode sync \
  --node tcp://localhost:26657 \
  -y
```

### B. Memeriksa Seluruh Voting Power Validator (`q tendermint-validator-set`)
```bash
zytheriond q tendermint-validator-set --node tcp://localhost:26657
```
