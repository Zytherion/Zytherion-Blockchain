.PHONY: build build-tfhe-rs build-no-tfhe tfhe-rs test test-tfhe test-pqc test-erasure lint install version check-tfhe-headers

# ═══════════════════════════════════════════════════════════════════════════════
# TFHE_CGO_REQUIRED — Zytherion enforces TFHE as ALWAYS ON.
# All production builds MUST use: make build
# Direct `go build ./...` without -tags tfhe_cgo will be rejected at runtime.
# ═══════════════════════════════════════════════════════════════════════════════

TFHE_CGO_BRIDGE_HEADER := x/privacy/tfhe/cgo_bridge.h
TFHE_STATIC_LIB        := x/privacy/tfhe/tfhe_c/target/release/libtfhe_c.a
BUILD_FLAGS            := -tags tfhe_cgo
LDFLAGS                := -X main.version=$(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

# ── Check CGo headers are present ──────────────────────────────────────────────

## check-tfhe-headers: Verify CGo bridge header + compiled Rust library exist.
check-tfhe-headers:
	@echo "[CHECK] Verifying TFHE CGo build prerequisites..."
	@if [ ! -f "$(TFHE_CGO_BRIDGE_HEADER)" ]; then \
		echo ""; \
		echo "╔══════════════════════════════════════════════════════════════╗"; \
		echo "║  ERROR: CGo bridge header not found                          ║"; \
		echo "║  Missing: $(TFHE_CGO_BRIDGE_HEADER)             ║"; \
		echo "║  This file must exist for TFHE CGo compilation.             ║"; \
		echo "╚══════════════════════════════════════════════════════════════╝"; \
		exit 1; \
	fi
	@if [ ! -f "$(TFHE_STATIC_LIB)" ]; then \
		echo ""; \
		echo "╔══════════════════════════════════════════════════════════════╗"; \
		echo "║  ERROR: TFHE static library not compiled                     ║"; \
		echo "║  Missing: $(TFHE_STATIC_LIB)  ║"; \
		echo "║  Run: make build-tfhe-rs                                     ║"; \
		echo "╚══════════════════════════════════════════════════════════════╝"; \
		exit 1; \
	fi
	@echo "[OK]  cgo_bridge.h found: $(TFHE_CGO_BRIDGE_HEADER)"
	@echo "[OK]  libtfhe_c.a found:  $(TFHE_STATIC_LIB)"
	@echo "[OK]  CGo prerequisites satisfied — proceeding with build."

# ── Main build targets ─────────────────────────────────────────────────────────

## build: Build all packages with TFHE CGo support ENFORCED.
##        Checks for CGo headers and Rust library BEFORE compilation.
##        This is the ONLY sanctioned production build target.
build: check-tfhe-headers
	@echo "[BUILD] Compiling Zytherion with TFHE CGo support (-tags tfhe_cgo)..."
	go build $(BUILD_FLAGS) -ldflags="$(LDFLAGS)" ./...
	@echo "[OK]  Build complete. TFHE is active."

## build-tfhe-rs: Compile the tfhe_c Rust static library (required for CGo TFHE engine).
##                First build takes 5-15 minutes (tfhe-rs is large).
##                Subsequent builds are incremental and fast.
build-tfhe-rs:
	@echo "=== Building tfhe_c Rust static library ==="
	@echo "    First build may take 5-15 minutes..."
	cd x/privacy/tfhe/tfhe_c && cargo build --release
	@echo "=== tfhe_c build complete ==="
	@ls -lh x/privacy/tfhe/tfhe_c/target/release/libtfhe_c.a

## build-tfhe-debug: Debug build of the Rust library (faster, less optimised).
build-tfhe-debug:
	cd x/privacy/tfhe/tfhe_c && cargo build

## install: Install zytheriond to GOPATH/bin with TFHE enforcement.
install: check-tfhe-headers
	@echo "[INSTALL] Installing zytheriond with -tags tfhe_cgo..."
	go install $(BUILD_FLAGS) -ldflags="$(LDFLAGS)" ./cmd/zytheriond
	@echo "[OK]  zytheriond installed."

## build-no-tfhe: Build WITHOUT TFHE (for unit testing non-TFHE code only).
##                WARNING: The resulting binary will PANIC on startup.
##                This target exists only for running unit tests that do not
##                import the tfhe package (e.g. CI lint, type checks).
build-no-tfhe:
	@echo "[WARN] Building WITHOUT tfhe_cgo — resulting binary WILL PANIC on startup."
	@echo "[WARN] Use this target only for isolated unit tests that skip tfhe package."
	go build ./...

# ── Testing ───────────────────────────────────────────────────────────────────

## test: Run all Go tests (excludes TFHE tests which require Rust lib).
test:
	go test ./x/privacy/pqc/... ./x/privacy/tfhe/... -run "^(TestErasure|TestDilithium)" -v -timeout=60s

## test-pqc: Run Dilithium5 signature tests.
test-pqc:
	go test ./x/privacy/pqc/... -v -timeout=60s

## test-erasure: Run Reed-Solomon erasure coding tests (pure Go, no Rust needed).
test-erasure:
	go test ./x/privacy/tfhe/... -run "^TestErasure" -v -timeout=60s

## test-tfhe: Run full TFHE engine tests (requires Rust library compiled first).
##            WARNING: Key generation takes 30-120 seconds per test.
test-tfhe:
	@echo "=== Running TFHE engine tests (CGo, requires libtfhe_c.a) ==="
	go test ./x/privacy/tfhe/... -run "^TestTFHE" -v -timeout=600s

## test-all: Run all tests including slow TFHE tests.
test-all: test-pqc test-erasure test-tfhe
	go test ./app/... ./x/privacy/keeper/... -v -timeout=120s

# ── Code quality ──────────────────────────────────────────────────────────────

## lint: Run Go linter.
lint:
	golangci-lint run ./...

# ── Version info ──────────────────────────────────────────────────────────────

## version: Print the current node version info.
version:
	@echo "Zytherion Blockchain and Cryptocurrency"
	@echo "Version: 0.5.0"
	@echo "Founder: Rayhan Aziel Abbrar"
	@echo "Signature: Dilithium5 (ML-DSA Level 5, ~256-bit PQ)"
	@echo "Hashing:   LWR (Ring-LWR / SHAKE-256)"
	@echo "Consensus: GreenBFT + PoVL VDF"
	@echo "TFHE:      tfhe-rs (Zama) via CGo"
	@echo "Stablecoin: ZYTD (Multi-Collateral)"
	@echo "IBC:       ICS-20 Collateral Vault"
	@echo "Oracle:    Median TWAP Price Feed"
	@echo "CosmWasm:  Permissioned Smart Contracts"

# ── v0.5 module tests ─────────────────────────────────────────────────────────

## test-oracle: Run x/oracle unit tests.
test-oracle:
	go test ./x/oracle/... -v -count=1

## test-stablecoin: Run x/stablecoin unit tests.
test-stablecoin:
	go test ./x/stablecoin/... -v -count=1

## test-ibccollateral: Run x/ibc-collateral unit tests.
test-ibccollateral:
	go test ./x/ibc-collateral/... -v -count=1

## test-v05: Run all v0.5 module tests.
test-v05: test-oracle test-stablecoin test-ibccollateral
	@echo "✅ All v0.5 module tests passed"