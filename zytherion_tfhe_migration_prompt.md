# Zytherion Blockchain — Migrasi FHE: BFV → TFHE

Kamu adalah AI engineering agent yang bertugas mengganti skema Homomorphic
Encryption (HE) di Zytherion Blockchain dari BFV (Lattigo v4) menjadi TFHE
(Zama TFHE-rs via CGo binding).

Tujuan migrasi ini:
- Turunkan ukuran ciphertext dari ~376KB per tx menjadi ~1–5KB
- Sesuaikan implementasi dengan klaim whitepaper yang sudah menyebut TFHE
- Naikkan estimasi TPS dari ~1–3 TPS menjadi ~500–2,000 TPS
- Aktifkan kompresi ciphertext TFHE-rs v0.7 (hingga 1,900x lebih kecil)

Repo: https://github.com/Zytherion/Zytherion-Blockchain
Bahasa utama: Go
Dependency baru: Zama TFHE-rs (Rust) via CGo binding

---

## Konteks file yang akan dimodifikasi

Baca semua file ini sebelum mulai:

- `x/privacy/fhe/fhe.go` — implementasi BFV saat ini dengan Lattigo v4,
  INI yang akan diganti total
- `x/privacy/keeper/keeper.go` — inject fheCtx, method EncryptAmount,
  HomomorphicAdd, DecryptBalance
- `x/privacy/keeper/msg_server.go` — handler MsgEncryptedTransfer
- `x/privacy/fhe/fhe_test.go` — unit test yang harus tetap pass setelah migrasi
  (nama test sama, behavior sama, implementasi dalam diganti)
- `go.mod` — perlu tambah dependency CGo binding
- `app/app.go` — inisialisasi fhe.NewContext() di startup

---

## Arsitektur target setelah migrasi

```
x/privacy/fhe/
├── fhe.go          ← interface Go yang sama seperti sekarang (tidak berubah dari luar)
├── tfhe_binding.go ← CGo wrapper ke TFHE-rs
├── tfhe_compress.go← logika kompresi/dekompresi ciphertext
└── fhe_test.go     ← test yang sama, behavior sama
```

Interface publik `fhe.go` TIDAK BOLEH berubah dari perspektif caller.
`keeper.go` dan `msg_server.go` tidak perlu tahu apakah di bawahnya BFV atau TFHE.
Semua perubahan dibungkus di dalam package `fhe`.

---

## TASK 1 — Setup TFHE-rs sebagai library Rust yang di-expose via C ABI

### Latar belakang
TFHE-rs adalah library Rust dari Zama. Go tidak bisa import Rust secara langsung,
jadi kita perlu layer C di antaranya (C ABI). Caranya:
1. Buat Rust crate kecil (`tfhe-cgo/`) yang wrap TFHE-rs dan export fungsi via `#[no_mangle] extern "C"`
2. Build crate itu menjadi static library (`.a` file)
3. Dari Go, gunakan `import "C"` untuk load fungsi dari `.a` tersebut

### Yang harus dilakukan

**A. Buat Rust crate wrapper di `tfhe-cgo/`**

Buat folder `tfhe-cgo/` di root repo dengan struktur:
```
tfhe-cgo/
├── Cargo.toml
├── Cargo.lock
└── src/
    └── lib.rs
```

Isi `Cargo.toml`:
- `[lib]` dengan `crate-type = ["staticlib"]`
- Dependency: `tfhe = { version = "0.7", features = ["integer", "x86_64-unix"] }`
  — gunakan feature `x86_64-unix` untuk Linux/Mac, atau `x86_64` saja untuk Windows
- Dependency: `bincode = "1"` untuk serialisasi key dan ciphertext

Fungsi yang harus di-export dari `src/lib.rs` via `extern "C"`:

```
tfhe_generate_keys(out_client_key: *mut *mut u8, out_client_key_len: *mut usize,
                   out_server_key: *mut *mut u8, out_server_key_len: *mut usize) -> i32

tfhe_encrypt_u64(client_key_bytes: *const u8, client_key_len: usize,
                 value: u64,
                 out_ct: *mut *mut u8, out_ct_len: *mut usize) -> i32

tfhe_decrypt_u64(client_key_bytes: *const u8, client_key_len: usize,
                 ct_bytes: *const u8, ct_len: usize,
                 out_value: *mut u64) -> i32

tfhe_add(server_key_bytes: *const u8, server_key_len: usize,
         ct_a_bytes: *const u8, ct_a_len: usize,
         ct_b_bytes: *const u8, ct_b_len: usize,
         out_ct: *mut *mut u8, out_ct_len: *mut usize) -> i32

tfhe_sub(server_key_bytes: *const u8, server_key_len: usize,
         ct_a_bytes: *const u8, ct_a_len: usize,
         ct_b_bytes: *const u8, ct_b_len: usize,
         out_ct: *mut *mut u8, out_ct_len: *mut usize) -> i32

tfhe_compress(server_key_bytes: *const u8, server_key_len: usize,
              ct_bytes: *const u8, ct_len: usize,
              out_ct: *mut *mut u8, out_ct_len: *mut usize) -> i32

tfhe_decompress(server_key_bytes: *const u8, server_key_len: usize,
                compressed_bytes: *const u8, compressed_len: usize,
                out_ct: *mut *mut u8, out_ct_len: *mut usize) -> i32

tfhe_free_bytes(ptr: *mut u8, len: usize)
```

Setiap fungsi:
- Return `0` jika sukses, `-1` jika error
- Alokasi output via `Box::into_raw(Box::new(bytes))` agar aman di-free dari Go
- Semua serialisasi key dan ciphertext menggunakan `bincode::serialize`
- Wrap setiap fungsi dalam `std::panic::catch_unwind` untuk cegah panic cross FFI boundary

Tipe TFHE yang dipakai: `FheUint64` dari `tfhe::prelude::*` dengan config
`ConfigBuilder::default().build()`. Ini memberikan security level 128-bit
dengan parameter default TFHE-rs yang sudah dioptimasi.

**B. Build script**

Buat `tfhe-cgo/build.sh`:
```bash
#!/bin/bash
cd "$(dirname "$0")"
cargo build --release
cp target/release/libtfhe_cgo.a ../x/privacy/fhe/lib/
```

Buat `tfhe-cgo/build_headers.sh` yang generate header C dari fungsi-fungsi
di atas menggunakan `cbindgen`:
- Install: `cargo install cbindgen`
- Run: `cbindgen --config cbindgen.toml --crate tfhe-cgo --output ../x/privacy/fhe/lib/tfhe_cgo.h`

Output akhir yang diharapkan:
```
x/privacy/fhe/lib/
├── libtfhe_cgo.a   ← static library hasil build Rust
└── tfhe_cgo.h      ← C header file
```

**C. Tambahkan ke Makefile atau script build**

Tambahkan target `make build-tfhe` di Makefile root yang menjalankan
`tfhe-cgo/build.sh` sebelum `go build`. Dokumentasikan di README bahwa
Rust toolchain (rustup + cargo) harus terinstall sebelum build.

---

## TASK 2 — Buat CGo binding di Go (`tfhe_binding.go`)

### Yang harus dilakukan

Buat file `x/privacy/fhe/tfhe_binding.go` dengan package `fhe`.

Di bagian atas file, tambahkan CGo directives:
```go
// #cgo LDFLAGS: -L${SRCDIR}/lib -ltfhe_cgo -ldl -lm
// #cgo CFLAGS: -I${SRCDIR}/lib
// #include "tfhe_cgo.h"
// #include <stdlib.h>
import "C"
```

Implementasikan fungsi-fungsi internal Go berikut yang menjadi wrapper tipis
ke fungsi C di atas. Semua fungsi ini bersifat `package-private` (huruf kecil),
hanya dipakai oleh `fhe.go`:

```go
func tfheGenerateKeys() (clientKeyBytes []byte, serverKeyBytes []byte, err error)
func tfheEncrypt(clientKey []byte, value uint64) ([]byte, error)
func tfheDecrypt(clientKey []byte, ct []byte) (uint64, error)
func tfheAdd(serverKey []byte, ctA []byte, ctB []byte) ([]byte, error)
func tfheSub(serverKey []byte, ctA []byte, ctB []byte) ([]byte, error)
func tfheCompress(serverKey []byte, ct []byte) ([]byte, error)
func tfheDecompress(serverKey []byte, compressed []byte) ([]byte, error)
```

Pattern yang harus diikuti untuk setiap fungsi yang menerima output pointer:
1. Deklarasi `var outPtr *C.uint8_t` dan `var outLen C.size_t`
2. Panggil fungsi C dengan `&outPtr` dan `&outLen`
3. Cek return code, jika `-1` return error
4. Copy bytes dari C memory ke Go slice: `C.GoBytes(unsafe.Pointer(outPtr), C.int(outLen))`
5. Free C memory: `C.tfhe_free_bytes(outPtr, outLen)`
6. Return Go slice

Jangan pernah menyimpan pointer C di luar scope fungsi. Semua C memory
harus di-free sebelum fungsi return.

---

## TASK 3 — Implementasi kompresi ciphertext (`tfhe_compress.go`)

### Latar belakang
TFHE-rs v0.7 mendukung post-computation compression yang bisa mengurangi
ukuran ciphertext hingga 1,900x. Ini adalah fitur kunci untuk menurunkan
ukuran tx dari ~20–50KB (TFHE raw) menjadi ~1–5KB.

### Yang harus dilakukan

Buat file `x/privacy/fhe/tfhe_compress.go` dengan package `fhe`.

Implementasikan dua fungsi publik:

```go
// CompressCiphertext mengompresi TFHE ciphertext untuk penyimpanan di KVStore
// atau pengiriman via transaksi. Output jauh lebih kecil dari raw ciphertext.
// Panggil ini sebelum SetEncryptedBalance di keeper.
func (c *Context) CompressCiphertext(ct []byte) ([]byte, error)

// DecompressCiphertext mendekompresi ciphertext sebelum operasi homomorphic.
// Panggil ini setelah GetEncryptedBalance di keeper, sebelum Add/Sub.
func (c *Context) DecompressCiphertext(compressed []byte) ([]byte, error)
```

Kedua fungsi ini hanya mendelegasikan ke `tfheCompress` dan `tfheDecompress`
dari `tfhe_binding.go` dengan server key dari `c.serverKey`.

Tambahkan juga konstanta dokumentasi:

```go
// CiphertextCompressionRatio adalah perkiraan rasio kompresi untuk FheUint64.
// Nilai aktual bervariasi tergantung konten, tapi secara umum 100x–1900x.
const CiphertextCompressionRatio = 1000
```

---

## TASK 4 — Ganti implementasi `fhe.go` dari BFV ke TFHE

### Yang harus dilakukan

Hapus seluruh isi `fhe.go` yang saat ini berisi implementasi BFV dengan
Lattigo v4, dan ganti dengan implementasi TFHE. Interface publik TIDAK BOLEH
berubah — caller (`keeper.go`, `msg_server.go`) tidak perlu modifikasi.

**A. Struct `Context` yang baru**

```go
type Context struct {
    clientKey []byte  // serialized TFHE ClientKey — RAHASIA, jangan log
    serverKey []byte  // serialized TFHE ServerKey — bisa dibagi ke validator
}
```

Hapus semua field Lattigo: `params`, `encoder`, `evaluator`, `keygen`,
`sk`, `pk`, `encryptor`, `decryptor`.

**B. `NewContext()` yang baru**

Panggil `tfheGenerateKeys()` dari `tfhe_binding.go`. Simpan hasilnya di struct.
Error handling: return `(*Context, error)` — signature tidak berubah dari
perspektif `app.go`.

Tambahkan log warning (bukan panic) jika key generation memakan waktu > 5 detik,
karena TFHE key generation memang lebih lambat dari BFV (~1–3 detik normal).

**C. `Encrypt(value uint64)` yang baru**

Panggil `tfheEncrypt(c.clientKey, value)`.
Return `([]byte, error)` langsung — tidak perlu `*rlwe.Ciphertext` lagi.
Sebelum return, panggil `c.CompressCiphertext(ct)` agar bytes yang keluar
sudah terkompresi dan siap disimpan ke KVStore.

**D. `Decrypt(ct []byte)` yang baru**

Terima `[]byte` (bukan `*rlwe.Ciphertext`).
Pertama dekompresi: `raw, err := c.DecompressCiphertext(ct)`.
Kemudian panggil `tfheDecrypt(c.clientKey, raw)`.

**E. `AddCiphertexts(a, b []byte)` yang baru**

Terima dua `[]byte` (sudah terkompresi, dari KVStore).
Dekompresi keduanya terlebih dahulu.
Panggil `tfheAdd(c.serverKey, rawA, rawB)`.
Kompresi hasilnya sebelum return.
Return `([]byte, error)`.

**F. `SubCiphertexts(a, b []byte)` yang baru — TAMBAHAN**

Ini method baru yang tidak ada di BFV. Diperlukan untuk mengurangi balance
sender saat transfer.
Dekompresi, panggil `tfheSub`, kompresi, return.

**G. Hapus method yang tidak relevan**

Hapus `PlaintextModulus()`, `Params()`, `SecretKey()`, `PublicKey()`,
`MulCiphertexts()` (opsional: keep MulCiphertexts tapi implementasi via TFHE).
Hapus semua import Lattigo dari `go.mod` setelah semua referensi hilang.

**H. Update komentar package**

Update komentar di header `fhe.go` untuk merefleksikan bahwa implementasi
sekarang menggunakan TFHE (Torus FHE) via Zama TFHE-rs, bukan BFV Lattigo.
Sebutkan:
- Skema: TFHE / Torus FHE
- Library: Zama TFHE-rs v0.7 via CGo
- Security: 128-bit default TFHE-rs parameter
- Plaintext type: FheUint64 (unsigned 64-bit integer)
- Kompresi: aktif by default, ~1,000x rasio untuk FheUint64

---

## TASK 5 — Update keeper.go untuk signature baru

### Latar belakang
Setelah migrasi, tipe ciphertext berubah dari `*rlwe.Ciphertext` menjadi
`[]byte` yang sudah terkompresi. Keeper sudah menyimpan `[]byte`, jadi
perubahan minimal — tapi method-method FHE perlu diupdate.

### Yang harus dilakukan

**A. Update `EncryptAmount`**

Signature tetap: `func (k Keeper) EncryptAmount(amount uint64) ([]byte, error)`
Implementasi: panggil `k.fheCtx.Encrypt(amount)` — ini sekarang return
compressed bytes langsung, tidak perlu marshal manual lagi.
Hapus semua kode marshal BFV (`MarshalBinary`, `rlwe.NewCiphertext`, dsb).

**B. Update `HomomorphicAdd`**

Ambil bytes dari store (sudah compressed), panggil `k.fheCtx.AddCiphertexts(a, b)`,
simpan result (sudah compressed) kembali ke store.
Tidak perlu dekompresi manual — `AddCiphertexts` di `fhe.go` sudah handle itu
secara internal.

**C. Tambahkan `HomomorphicSub`**

Method baru yang dibutuhkan oleh `msg_server.go` untuk kurangi balance sender:

```go
func (k Keeper) HomomorphicSub(ctx sdk.Context,
    addrA sdk.AccAddress,  // balance yang dikurangi
    ctB []byte,            // jumlah yang dikurangkan (compressed ciphertext)
) error
```

Ambil ciphertext addrA dari store, panggil `k.fheCtx.SubCiphertexts(ctA, ctB)`,
simpan hasilnya kembali ke store addrA.

**D. Update `DecryptBalance`**

Signature tetap: `func (k Keeper) DecryptBalance(ctx sdk.Context, addr sdk.AccAddress) (uint64, error)`
Implementasi: ambil compressed bytes dari store, panggil `k.fheCtx.Decrypt(compressed)`.
Tidak perlu dekompresi manual — `Decrypt` di `fhe.go` sudah handle itu.

---

## TASK 6 — Update msg_server.go untuk pakai SubCiphertexts

### Yang harus dilakukan

Di handler `EncryptedTransfer`, perbarui langkah 7 (yang sebelumnya
menggunakan evaluator BFV):

Ganti dari:
- Deserialize `*rlwe.Ciphertext` menggunakan `UnmarshalBinary`
- Panggil evaluator BFV `SubNew`

Menjadi:
- Bytes dari store sudah siap dipakai langsung (compressed TFHE ciphertext)
- Panggil `k.HomomorphicSub(ctx, senderAddr, encryptedAmountBytes)` untuk
  kurangi balance sender
- Panggil `k.HomomorphicAdd(ctx, receiverAddr, receiverAddr, encryptedAmountBytes)`
  — atau buat helper `AddToBalance(addr, amountCt)` di keeper yang lebih idiomatis

Hapus semua import `rlwe` dan `lattigo` dari `msg_server.go`.

---

## TASK 7 — Update unit test `fhe_test.go`

### Yang harus dilakukan

Test yang sudah ada harus tetap pass. Perubahan yang perlu dilakukan:

**A. Update import**

Hapus import Lattigo (`github.com/tuneinsight/lattigo/v4/...`).
Tidak ada import baru yang perlu ditambahkan karena interface `fhe.go` sama.

**B. Update tipe parameter**

Semua fungsi yang sebelumnya menerima `*rlwe.Ciphertext` sekarang menerima
`[]byte`. Update pemanggilan test accordingly.

**C. Tambahkan test kompresi**

```go
// TestCompressDecompress verifies bahwa compress → decompress menghasilkan
// ciphertext yang fungsional (bisa decrypt ke nilai yang sama).
func TestCompressDecompress(t *testing.T) {
    ctx, err := fhe.NewContext()
    require.NoError(t, err)

    original := uint64(999_888_777)
    compressed, err := ctx.Encrypt(original) // sudah compressed by default
    require.NoError(t, err)

    // Decrypt dari compressed — fhe.Decrypt handle dekompresi internal
    decrypted, err := ctx.Decrypt(compressed)
    require.NoError(t, err)
    require.Equal(t, original, decrypted)

    // Verifikasi ukuran jauh lebih kecil dari raw BFV
    require.Less(t, len(compressed), 10_000, // kurang dari 10KB
        "compressed ciphertext harus jauh lebih kecil dari BFV 376KB")
}
```

**D. Tambahkan test SubCiphertexts**

```go
// TestHomomorphicSub verifies bahwa pengurangan homomorphic menghasilkan
// selisih yang benar setelah decrypt.
func TestHomomorphicSub(t *testing.T) {
    ctx, err := fhe.NewContext()
    require.NoError(t, err)

    a := uint64(1_000_000)
    b := uint64(250_000)

    ctA, err := ctx.Encrypt(a)
    require.NoError(t, err)
    ctB, err := ctx.Encrypt(b)
    require.NoError(t, err)

    ctResult, err := ctx.SubCiphertexts(ctA, ctB)
    require.NoError(t, err)

    result, err := ctx.Decrypt(ctResult)
    require.NoError(t, err)
    require.Equal(t, a-b, result, "homomorphic sub harus sama dengan plaintext sub")
}
```

---

## TASK 8 — Update go.mod dan dokumentasi

### Yang harus dilakukan

**A. Hapus Lattigo dari go.mod**

Setelah semua referensi Lattigo hilang dari kode Go, jalankan:
```bash
go mod tidy
```
Ini akan otomatis hapus `github.com/tuneinsight/lattigo/v4` dari `go.mod`
dan `go.sum`.

**B. Update README**

Di bagian "Getting Started" atau "Prerequisites", tambahkan:
```
Prerequisites:
- Go 1.21+
- Rust toolchain (rustup): https://rustup.rs
- Build TFHE-rs wrapper: make build-tfhe
```

**C. Update komentar di `fhe.go`**

Pastikan komentar package menyebut TFHE, bukan BFV. Sebutkan bahwa
ini adalah implementasi LWE-based (TFHE berbasis Torus-LWE), sehingga
konsisten dengan klaim whitepaper "LWE-based FHE".

**D. Update whitepaper reference (jika ada di kode)**

Jika ada string literal atau komentar yang menyebut "BFV" di codebase,
ganti dengan "TFHE (Torus FHE, LWE-based)".

---

## Urutan eksekusi

Kerjakan dalam urutan ini — ada dependency ketat antar task:

1. **Task 1** — Build Rust crate dulu. Tanpa `libtfhe_cgo.a`, Go tidak bisa compile.
   Verifikasi: `ls x/privacy/fhe/lib/libtfhe_cgo.a` harus ada.

2. **Task 2** — CGo binding. Verifikasi: buat file test kecil yang import
   package `fhe` dan panggil `tfheGenerateKeys()`, pastikan compile.

3. **Task 3** — Kompresi. Bisa dikerjakan paralel dengan Task 2 karena
   hanya tergantung pada `tfhe_binding.go`.

4. **Task 4** — Ganti `fhe.go`. Ini langkah terbesar. Kerjakan setelah
   Task 2 dan 3 selesai. Verifikasi: `go build ./x/privacy/fhe/...` pass.

5. **Task 5** — Update `keeper.go`. Kerjakan setelah Task 4 karena
   tergantung interface baru.

6. **Task 6** — Update `msg_server.go`. Kerjakan setelah Task 5.

7. **Task 7** — Update test. Kerjakan terakhir, setelah implementasi stabil.
   Verifikasi: `go test ./x/privacy/...` semua pass.

8. **Task 8** — Cleanup dan dokumentasi. Kerjakan paling akhir setelah
   semua test hijau.

---

## Constraint wajib

- Interface publik `fhe.go` tidak boleh berubah dari perspektif `keeper.go`
  dan `msg_server.go` — tujuannya zero-change untuk caller
- Semua C memory yang dialokasi di Rust HARUS di-free via `tfhe_free_bytes`
  sebelum Go function return — memory leak di CGo sangat sulit di-debug
- `clientKey` (secret key) TIDAK BOLEH pernah di-log, di-emit sebagai event,
  atau disimpan di KVStore — hanya `serverKey` yang boleh dibagikan
- Setelah migrasi, `go test ./x/privacy/fhe/...` harus pass semua
- Ukuran compressed ciphertext hasil `Encrypt()` harus < 10KB untuk
  nilai uint64 apapun — ini adalah acceptance criteria utama migrasi ini
- Jalankan `go build ./...` setelah setiap task selesai
- Jika CGo tidak bisa dipakai di environment tertentu (misalnya CI tanpa
  Rust toolchain), tambahkan build tag `//go:build !notfhe` dan sediakan
  fallback mock di file terpisah untuk keperluan CI

---

## Definisi "done"

Migrasi dianggap selesai jika semua kondisi ini terpenuhi:

1. `go build ./...` berhasil tanpa error atau warning
2. `go test ./x/privacy/...` semua pass termasuk test baru di Task 7
3. `ignite chain serve` berhasil start tanpa panic
4. Ukuran output `ctx.Encrypt(uint64(1_000_000))` kurang dari 10KB
5. Decrypt dari ciphertext terkompresi menghasilkan nilai yang sama
6. Tidak ada import `lattigo` tersisa di seluruh codebase Go
7. README sudah update dengan prerequisite Rust toolchain
