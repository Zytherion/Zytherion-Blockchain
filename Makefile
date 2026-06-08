.PHONY: build build-tfhe-rs tfhe-rs test test-tfhe test-pqc test-erasure lint install version

# ── Main build targets ────────────────────────────────────────────────────────

## build: Build all Go packages (requires tfhe_c Rust library compiled first).
##        Run `make build-tfhe-rs` before this if building with TFHE CGo support.
build:
	go build ./...

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

## install: Install zytheriond binary to $GOPATH/bin.
install:
	go install ./cmd/zytheriond

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
	@echo "Version: 0.3.0"
	@echo "Founder: Rayhan Aziel Abbrar"
	@echo "Signature: Dilithium5 (ML-DSA Level 5, ~256-bit PQ)"
	@echo "Hashing:   LWR (Ring-LWR / SHAKE-256)"
	@echo "Consensus: GreenBFT + PoVL VDF"
	@echo "TFHE:      tfhe-rs (Zama) via CGo"
	@echo "ZK-SNARK:  REMOVED (v0.3)"