#!/bin/bash
# build.sh — Build the tfhe_c Rust static library for CGo linking.
#
# Usage:
#   ./x/privacy/tfhe/tfhe_c/build.sh          # Release build (default)
#   ./x/privacy/tfhe/tfhe_c/build.sh --debug  # Debug build (faster, less optimised)
#
# After a successful build, the static library is at:
#   x/privacy/tfhe/tfhe_c/target/release/libtfhe_c.a
#
# Prerequisites:
#   - Rust toolchain: https://rustup.rs  (rustup install stable)
#   - For Linux cross-compilation targets, install the appropriate target:
#       rustup target add x86_64-unknown-linux-gnu
#   - On Windows: use WSL2 with the above, or install MSVC-based Rust toolchain

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

PROFILE="${1:-release}"
if [ "$PROFILE" = "--debug" ]; then
    PROFILE="debug"
    BUILD_FLAG=""
else
    BUILD_FLAG="--release"
fi

echo "=== Building tfhe_c Rust library (profile: $PROFILE) ==="
echo "    This may take 5-15 minutes on first build (tfhe-rs compiles FHE circuits)"
echo ""

# Check Rust toolchain.
if ! command -v cargo &> /dev/null; then
    echo "ERROR: cargo not found. Install Rust from https://rustup.rs"
    exit 1
fi

cargo_version=$(cargo --version)
echo "Using: $cargo_version"

# Build.
cargo build $BUILD_FLAG

LIB_PATH="$SCRIPT_DIR/target/$PROFILE/libtfhe_c.a"
if [ -f "$LIB_PATH" ]; then
    lib_size=$(du -sh "$LIB_PATH" | cut -f1)
    echo ""
    echo "=== Build SUCCESS ==="
    echo "    Library: $LIB_PATH"
    echo "    Size:    $lib_size"
    echo ""
    echo "You can now run: go build ./..."
else
    echo "ERROR: Expected library not found at $LIB_PATH"
    exit 1
fi
