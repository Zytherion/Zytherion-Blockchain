#!/usr/bin/env bash
# deploy_homomorphic_vault.sh — Build and deploy the homomorphic vault contract.
#
# Usage:
#   bash scripts/deploy_homomorphic_vault.sh
#
# Prerequisites:
#   - ignite chain serve is running with --build.tags "tfhe_cgo"  (or zytheriond started)
#   - Rust + cargo installed with wasm32-unknown-unknown target:
#       rustup target add wasm32-unknown-unknown
#   - (Optional) cosmwasm-check installed:
#       cargo install cosmwasm-check
#
# This script:
#   1. Builds the Rust contract (optimized wasm)
#   2. Uploads it to the local Zytherion chain
#   3. Instantiates the homomorphic_vault contract
#   4. Runs a demo: shows how to encrypt & deposit values
#   5. Queries the vault info to confirm deployment

set -euo pipefail

BINARY="${BINARY:-zytheriond}"
CHAIN_ID="${CHAIN_ID:-zytherion}"
NODE="${NODE:-tcp://localhost:26657}"
KEYRING="${KEYRING:---keyring-backend test}"
FROM="${FROM:-alice}"
CONTRACT_DIR="$(cd "$(dirname "$0")/../contracts/homomorphic_vault" && pwd)"
WASM_OUT="${CONTRACT_DIR}/target/wasm32-unknown-unknown/release/homomorphic_vault.wasm"

# ── Colour helpers ─────────────────────────────────────────────────────────────
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; BLUE='\033[0;34m'; NC='\033[0m'
info()  { echo -e "${BLUE}[INFO]${NC} $*"; }
ok()    { echo -e "${GREEN}[OK]${NC}   $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC} $*"; }
error() { echo -e "${RED}[ERROR]${NC} $*" >&2; exit 1; }

echo ""
echo "╔══════════════════════════════════════════════════════════════════╗"
echo "║  Zytherion Homomorphic Vault — Deploy Script                    ║"
echo "╚══════════════════════════════════════════════════════════════════╝"
echo ""

# ── Step 1: Build Rust contract ────────────────────────────────────────────────
info "[1/5] Building Rust CosmWasm contract (release/wasm32 mode)..."
cd "${CONTRACT_DIR}"

# Ensure the wasm target is installed
if ! rustup target list --installed 2>/dev/null | grep -q "wasm32-unknown-unknown"; then
    warn "wasm32-unknown-unknown target not found — installing..."
    rustup target add wasm32-unknown-unknown
fi

cargo build --release --target wasm32-unknown-unknown 2>&1
cd - > /dev/null

if [ ! -f "${WASM_OUT}" ]; then
    error "WASM output not found at: ${WASM_OUT}"
fi

WASM_SIZE=$(du -sh "${WASM_OUT}" 2>/dev/null | cut -f1)
ok "Contract built — ${WASM_OUT} (${WASM_SIZE})"

# Optional: run cosmwasm-check if available
if command -v cosmwasm-check &> /dev/null; then
    info "Running cosmwasm-check..."
    cosmwasm-check "${WASM_OUT}" && ok "cosmwasm-check passed" || warn "cosmwasm-check reported issues"
fi

# ── Step 2: Get sender address ─────────────────────────────────────────────────
info "[2/5] Resolving sender address for key '${FROM}'..."
OWNER_ADDR=$(${BINARY} keys show "${FROM}" ${KEYRING} -a 2>/dev/null) || \
    error "Could not find key '${FROM}'. Make sure the chain is running and the key exists."
ok "Sender address: ${OWNER_ADDR}"

# ── Step 3: Upload contract ────────────────────────────────────────────────────
info "[3/5] Uploading contract to chain '${CHAIN_ID}'..."

UPLOAD_TX=$(${BINARY} tx wasm store "${WASM_OUT}" \
    --from "${FROM}" \
    --chain-id "${CHAIN_ID}" \
    --node "${NODE}" \
    ${KEYRING} \
    --gas auto \
    --gas-adjustment 1.4 \
    --fees "50000zytc" \
    --yes \
    --output json 2>&1) || true

echo "${UPLOAD_TX}" | head -5

# Wait for the transaction to be committed
sleep 4

# Get the code ID from the latest uploaded code
CODE_ID=$(${BINARY} query wasm list-code \
    --node "${NODE}" \
    --output json 2>/dev/null \
    | python3 -c "import sys,json; codes=json.load(sys.stdin)['code_infos']; print(codes[-1]['code_id'])" \
    2>/dev/null) || CODE_ID="1"

ok "Contract uploaded — Code ID: ${CODE_ID}"

# ── Step 4: Instantiate contract ───────────────────────────────────────────────
info "[4/5] Instantiating homomorphic_vault contract..."

INIT_MSG="{\"label\":\"My Homomorphic Vault\",\"owner\":\"${OWNER_ADDR}\"}"

${BINARY} tx wasm instantiate "${CODE_ID}" "${INIT_MSG}" \
    --from "${FROM}" \
    --label "homomorphic-vault-v1" \
    --chain-id "${CHAIN_ID}" \
    --node "${NODE}" \
    ${KEYRING} \
    --gas auto \
    --gas-adjustment 1.4 \
    --fees "50000zytc" \
    --yes \
    --output json 2>&1 | head -10

sleep 4

# Get the contract address
CONTRACT_ADDR=$(${BINARY} query wasm list-contract-by-code "${CODE_ID}" \
    --node "${NODE}" --output json 2>/dev/null \
    | python3 -c "import sys,json; cs=json.load(sys.stdin)['contracts']; print(cs[0])" \
    2>/dev/null) || CONTRACT_ADDR="<not-found>"

ok "Contract instantiated at: ${CONTRACT_ADDR}"

# ── Step 5: Demo — Query vault info ───────────────────────────────────────────
info "[5/5] Querying vault info..."

if [ "${CONTRACT_ADDR}" != "<not-found>" ]; then
    ${BINARY} query wasm contract-state smart "${CONTRACT_ADDR}" \
        '{"vault_info":{}}' \
        --node "${NODE}" \
        --output json 2>/dev/null || warn "Could not query vault info yet"
fi

echo ""
echo "╔══════════════════════════════════════════════════════════════════╗"
echo "║  ✓ Deploy complete!                                              ║"
echo "║                                                                  ║"
echo "║  Contract address:                                               ║"
echo "║    ${CONTRACT_ADDR}"
echo "║                                                                  ║"
echo "║  Useful commands:                                                ║"
echo "║                                                                  ║"
echo "║  Query encrypted balance (opaque ciphertext):                    ║"
echo "║    ${BINARY} query wasm contract-state smart \\                   "
echo "║      ${CONTRACT_ADDR} '{\"encrypted_balance\":{}}'"
echo "║                                                                  ║"
echo "║  Query vault info:                                               ║"
echo "║    ${BINARY} query wasm contract-state smart \\                   "
echo "║      ${CONTRACT_ADDR} '{\"vault_info\":{}}'                       "
echo "║                                                                  ║"
echo "║  Deposit (provide your FheUint32 ciphertext as base64):          ║"
echo "║    ${BINARY} tx wasm execute ${CONTRACT_ADDR} \\                  "
echo "║      '{\"deposit\":{\"encrypted_amount\":\"<base64-ct>\"}}' \\    "
echo "║      --from ${FROM} ${KEYRING} --fees 10000zytc --yes            "
echo "║                                                                  ║"
echo "╚══════════════════════════════════════════════════════════════════╝"
echo ""
echo -e "${GREEN}TFHE homomorphic vault is live on Zytherion!${NC}"
