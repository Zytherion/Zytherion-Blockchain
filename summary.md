# Zytherion Blockchain — Whitepaper Summary

**Versi:** 1.0.0 | **Token:** ZYTC | **Chain ID:** `zytherion`

---

## Abstrak

**Zytherion** adalah blockchain Layer-1 generasi baru yang dibangun di atas Cosmos SDK dan CometBFT, yang menggabungkan empat teknologi kriptografi terdepan dalam satu arsitektur terintegrasi: *Post-Quantum Cryptography* berbasis kisi lattice, *Proof of Verifiable Lattices* sebagai jam jaringan tahan manipulasi, *Zero-Knowledge Proofs* berbasis Groth16 untuk privasi transaksi, dan *Green BFT Consensus* yang efisien energi.

Zytherion dirancang untuk menghadapi ancaman komputasi kuantum masa depan sekaligus menjaga finalitas konsensus yang cepat, privasi pengguna yang absolut, dan keamanan kriptografi jangka panjang.

---

## 1. Latar Belakang & Motivasi

Blockchain generasi pertama seperti Bitcoin dan Ethereum menggunakan SHA-256 dan ECDSA yang terbukti rentan terhadap algoritma kuantum Shor dan Grover. Seiring perkembangan komputasi kuantum, kebutuhan untuk mempersiapkan infrastruktur blockchain yang *quantum-resistant* semakin mendesak.

Di sisi lain, mekanisme privasi yang ada (seperti TFHE/FHE) memiliki overhead komputasi dan storage yang sangat besar, membuatnya tidak praktis untuk penggunaan skala jaringan. Zytherion hadir sebagai solusi yang menyelaraskan keamanan kuantum, efisiensi, dan privasi dalam satu jaringan.

---

## 2. Arsitektur Sistem

```
┌──────────────────────────────────────────────────────┐
│                   Aplikasi (DApp)                    │
│             REST 1317 | RPC 26657 | gRPC 9090        │
└────────────────────────┬─────────────────────────────┘
                         │ ABCI 2.0
┌────────────────────────▼─────────────────────────────┐
│               CometBFT (Green BFT)                   │
│         PrepareProposal / ProcessProposal            │
│           PoVL Sentinel Validation Layer             │
└────────────────────────┬─────────────────────────────┘
                         │
┌────────────────────────▼─────────────────────────────┐
│             Cosmos SDK Application Core              │
│  ┌──────────────┐  ┌──────────────┐  ┌────────────┐  │
│  │ Bank Module  │  │Staking Module│  │  Gov Module│  │
│  └──────────────┘  └──────────────┘  └────────────┘  │
│  ┌─────────────────────────────────────────────────┐  │
│  │              Privacy Module (x/privacy)         │  │
│  │  ┌────────────┐  ┌──────────┐  ┌─────────────┐  │  │
│  │  │  LWR Hash  │  │   PoVL   │  │ ZK Verifier │  │  │
│  │  │  (PQC)     │  │  (VDF)   │  │ (Groth16)   │  │  │
│  │  └────────────┘  └──────────┘  └─────────────┘  │  │
│  └─────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────┘
```

---

## 3. Empat Pilar Teknologi

### 3.1 Post-Quantum Cryptography — Deterministic LWR Hashing

**Masalah:** SHA-256 dan skema hashing konvensional akan rentan terhadap algoritma Grover yang berjalan di komputer kuantum, yang dapat memangkas keamanan secara efektif menjadi setengahnya.

**Solusi Zytherion:** Implementasi fungsi hash satu arah berbasis **Learning With Rounding (LWR)** di atas *Ring* $R_q = \mathbb{Z}_q[X]/(X^n+1)$.

**Parameter:**
| Parameter | Nilai | Keterangan |
|---|---|---|
| Dimensi Ring `n` | 256 | Koefisien polinomial |
| Modulus utama `q` | 3329 | Prima, kompatibel Kyber |
| Modulus rounding `p` | 256 | Output 1 byte/koefisien |
| Ukuran output | 96 byte | 32B seed + 64B vektor b |

**Konstruksi:**

$$b = \left\lfloor \frac{p}{q} \cdot (A \cdot s) \right\rfloor \mod p$$

Implementasi menggunakan **integer arithmetic murni** tanpa floating point (`(val * p) / q`) sehingga menjamin hasil bit-identik di semua arsitektur CPU. Output final:

$$H_n = \text{SHA3-256}(\text{LWR}(data_n) \| H_{n-1})$$

**Jaminan Keamanan:**
- Deterministik 100% — eliminasi total non-determinisme LWE
- Tahan serangan Grover karena domain kisi (lattice)
- `LastResultsHash` selalu konsisten di semua validator node

---

### 3.2 Proof of Verifiable Lattices (PoVL)

**Masalah:** Tanpa mekanisme penundaan waktu, proposer blok jahat dapat memanipulasi *timestamp* atau menyerang jaringan dengan proposal blok palsu secara cepat.

**Solusi Zytherion:** PoVL bertindak sebagai **Sequential Verifiable Delay Function (VDF)** berbasis LWR. Setiap blok memuat bukti komputasi sekuensial yang tidak bisa dipercepat secara paralel.

**Konstruksi Satu Langkah:**

$$\text{state}_n = \text{SHA3-256}(\text{LWRHash}(\text{state}_{n-1}) \| \text{state}_{n-1})$$

**Rantai N Langkah:**

$$\text{PoVLRoot} = \text{state}_N = f^N(\text{state}_0)$$

**Alur di Konsensus ABCI 2.0:**
1. **PrepareProposal:** Node proposer menghitung `PoVLRoot` dari N langkah sekuensial.
2. **ProcessProposal:** Semua validator memverifikasi `PoVLRoot` sebelum memberikan vote. Blok yang tidak memiliki PoVL valid akan langsung **ditolak (REJECT)**.

**Konfigurasi Default:** N = 10 langkah per blok.

---

### 3.3 Zero-Knowledge Privacy (Groth16 / BN254)

**Masalah:** Skema FHE/TFHE memiliki *ciphertext* berukuran ~10-30KB per nilai, overhead komputasi yang sangat besar, dan tidak dapat diverifikasi secara efisien oleh semua node.

**Solusi Zytherion:** Migrasi penuh ke **ZK-SNARKs** menggunakan algoritma Groth16 di atas kurva eliptik BN254.

**Spesifikasi:**
| Komponen | Detail |
|---|---|
| Proving System | Groth16 |
| Elliptic Curve | BN254 (alt-bn128) |
| ZK Library | GNARK (Go) |
| Proof Size | ~128 byte |
| Verifikasi | O(1) — konstan |

**Alur Transaksi Privat:**
1. Pengguna menggunakan `zkprove` tool (off-chain) untuk generate proof atas nilai transaksi.
2. Proof dikirim ke chain via `tx privacy init-commitment`.
3. Node memverifikasi proof menggunakan *Verifying Key* (VK) yang sudah di-commit ke repository.

**Fail-Fast Security:** Node Zytherion akan **menolak menyala (panic)** jika Verifying Key tidak ditemukan atau rusak. Ini mencegah node berjalan dalam kondisi keamanan yang dikompromikan.

---

### 3.4 Green BFT Consensus

**Landasan:** CometBFT (Tendermint) BFT dengan finalitas deterministik dan instan.

**Peningkatan Zytherion:**

**a) PQC SIMD AnteDecorator**
- Transaksi masuk dievaluasi oleh `PQCAnteDecorator` sebelum handler standar.
- Menggunakan instruksi CPU SIMD untuk verifikasi kriptografi PQC berkecepatan tinggi.
- Transaksi spam atau tidak valid di-reject di lapisan terluar tanpa membuang gas validator.

**b) Adaptive Timeout (Green BFT)**
- Timeout blok disesuaikan secara dinamis berdasarkan jumlah transaksi rata-rata dalam periode terakhir.
- Blok kosong tidak memakan waktu round penuh, menghemat energi dan bandwidth jaringan.

**c) Deterministik DeliverTx**
- PoVL Sentinel diintersi langsung di level `DeliverTx` override sebelum masuk ke pipeline normal Cosmos SDK.
- Menjamin `LastResultsHash` identik di semua node, menghilangkan penyebab utama *consensus mismatch*.

---

## 4. Tokenomik (ZYTC)

**Total Supply:** 1.000.000.000 ZYTC (1 Miliar)

| Alokasi | Jumlah | Persentase | Keterangan |
|---|---|---|---|
| Community Pool / Public Sale | 450.000.000 | 45% | Ekosistem & adopsi |
| Staking Rewards | 250.000.000 | 25% | Insentif validator |
| Development Fund | 150.000.000 | 15% | Pengembangan protokol |
| Team & Founders | 100.000.000 | 10% | Vesting jangka panjang |
| Public Goods Funding | 50.000.000 | 5% | Hibah komunitas |

**Mekanisme Staking:**
- Validator harus melakukan *self-delegation* minimum.
- Reward staking didistribusikan setiap blok via `x/distribution`.
- Validator yang tidak aktif atau double-sign akan di-slash dan di-jail.

---

## 5. Konsensus & Keamanan

**Tipe Konsensus:** BFT (Byzantine Fault Tolerant)  
**Toleransi Kegagalan:** Aman selama **< 1/3** validator jahat atau offline  
**Finalitas:** Instan (tidak ada reorganisasi rantai)

**Lapisan Keamanan Berlapis:**

```
Lapisan 1: PQC (Ring-LWR)         → Keamanan kriptografi kuantum
Lapisan 2: PoVL (VDF)             → Penundaan waktu anti-manipulasi
Lapisan 3: ZK-SNARKs (Groth16)   → Privasi transaksi
Lapisan 4: Green BFT (CometBFT)  → Finalitas konsensus
Lapisan 5: Fail-Fast ZK VK       → Integritas startup node
```

---

## 6. Antarmuka & Integrasi

Zytherion menyediakan tiga jalur integrasi untuk aplikasi pihak ketiga:

| Interface | Port | Protokol | Rekomendasi Penggunaan |
|---|---|---|---|
| Tendermint RPC | 26657 | HTTP/WebSocket | Monitoring, block explorer |
| REST API / LCD | 1317 | HTTP/JSON | Frontend DApp (React, Vue) |
| gRPC | 9090 | Protobuf | Backend performansi tinggi |

---

## 7. Roadmap

| Fase | Target | Fitur |
|---|---|---|
| **Phase 1** ✅ | Selesai | LWR-SHA3 Hashing, PoVL VDF, ZK Groth16 Verifier |
| **Phase 2** 🔄 | Q3 2026 | Multi-node testnet, PoVL ZK Proof produksi penuh |
| **Phase 3** 📅 | Q4 2026 | IBC Inter-chain Privacy, Dilithium3 validator signing |
| **Phase 4** 📅 | 2027 | Mainnet launch, full quantum-resistant signature scheme |

---

## 8. Kesimpulan

Zytherion adalah implementasi nyata blockchain *post-quantum* yang menggabungkan teori kriptografi terdepan ke dalam sistem konsensus yang berfungsi. Dengan pilar LWR, PoVL, ZK-SNARKs, dan Green BFT yang saling melengkapi, Zytherion menawarkan jaminan keamanan jangka panjang yang tidak dimiliki blockchain konvensional, sambil tetap mempertahankan performa, efisiensi energi, dan kemudahan integrasi bagi pengembang aplikasi.
