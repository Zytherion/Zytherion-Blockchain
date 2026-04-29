#!/bin/bash
set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"
export PATH="$HOME/.cargo/bin:$PATH"
echo "[build-tfhe] Building TFHE-rs static library (this may take 30-90 minutes)..."
cargo build --release
DEST="$SCRIPT_DIR/../x/privacy/fhe/lib"
mkdir -p "$DEST"
cp target/release/libtfhe_cgo.a "$DEST/libtfhe_cgo.a"
echo "[build-tfhe] Done â€” libtfhe_cgo.a copied to $DEST"