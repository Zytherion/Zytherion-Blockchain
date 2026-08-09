# `x/privacy/tfhe/cosmwasm` — TFHE × CosmWasm Query Plugin

This package bridges the Zytherion TFHE subsystem (Fully Homomorphic Encryption via [tfhe-rs](https://github.com/zama-ai/tfhe-rs)) into CosmWasm smart contracts using the **custom query** mechanism.

## Overview

CosmWasm contracts can call into the host chain via `QueryRequest::Custom`. This package registers a handler that interprets TFHE-shaped JSON queries, routes them to the node's TFHE cryptographic engine, and returns JSON-encoded responses.

## Supported Operations

| Query Field     | Description                                              | Response Type             |
|-----------------|----------------------------------------------------------|---------------------------|
| `tfhe_encrypt`  | Encrypt a `uint32` plaintext with the node's client key  | `TFHECiphertextResponse`  |
| `tfhe_decrypt`  | Decrypt a ciphertext (demo/test only!)                   | `TFHEPlaintextResponse`   |
| `tfhe_add`      | Homomorphically add two `FheUint32` ciphertexts          | `TFHECiphertextResponse`  |
| `tfhe_mul_scalar` | Multiply a ciphertext by a plaintext scalar             | `TFHECiphertextResponse`  |
| `tfhe_verify`   | Check if a ciphertext blob is structurally valid         | `TFHEVerifyResponse`      |

## Build Modes

| Build Tag    | Behaviour                                                       |
|--------------|-----------------------------------------------------------------|
| (default)    | Stub — all queries return `ErrTFHEDisabled` gracefully           |
| `tfhe_cgo`   | Real — CGo bridge to Rust `libtfhe_c.a` static library          |

```bash
# Enable TFHE:
go build -tags tfhe_cgo ./...
```

## Wiring into the App

The plugin is registered in `app/app.go` via `wasmkeeper.WithQueryPlugins`:

```go
tfhePlugin := tcosmwasm.NewTFHEQueryPlugin()
wasmOpts := []wasmkeeper.Option{
    wasmkeeper.WithQueryPlugins(&wasmkeeper.QueryPlugins{
        Custom: tfhePlugin,
    }),
}
app.WasmKeeper = wasmkeeper.NewKeeper(..., wasmOpts...)
```

## Contract Usage (Rust)

```rust
// In your CosmWasm contract msg.rs:
#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
#[serde(rename_all = "snake_case")]
pub enum TFHECustomQuery {
    TfheEncrypt { value: u32 },
    TfheAdd { ct1: Binary, ct2: Binary },
    TfheMulScalar { ciphertext: Binary, scalar: u32 },
    TfheDecrypt { ciphertext: Binary },
    TfheVerify  { ciphertext: Binary },
}

// In contract.rs:
let q = TFHECustomQuery::TfheEncrypt { value: 42 };
let req: QueryRequest<TFHECustomQuery> = QueryRequest::Custom(q);
let resp: TFHECiphertextResponse = deps.querier.query(&req)?;
// resp.ciphertext holds the FheUint32 encrypted value — completely opaque!
```

## Key Management

On first use, the plugin generates a fresh TFHE `ClientKey` + `ServerKey` pair and persists them to the user's home directory:

```
~/.zytherion_tfhe_client.key  (mode 0600 — private)
~/.zytherion_tfhe_server.key  (mode 0600)
```

On subsequent node restarts the keys are loaded from disk automatically.

> ⚠️ **Security Note**: The `ClientKey` is sensitive. Guard it like a private key. Only the `ServerKey` is needed for homomorphic evaluation (add, mul_scalar).
