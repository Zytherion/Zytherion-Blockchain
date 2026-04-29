.PHONY: build-tfhe build test lint

## build-tfhe: Build the TFHE-rs CGo static library (requires Rust toolchain).
build-tfhe:
	@echo "Building TFHE-rs static library..."
	@chmod +x tfhe-cgo/build.sh
	@bash tfhe-cgo/build.sh

## build: Build all Go packages (requires libtfhe_cgo.a â€” run build-tfhe first).
build: build-tfhe
	go build ./...

## test: Run all Go tests.
test:
	go test ./...

## lint: Run Go linter.
lint:
	golangci-lint run ./...