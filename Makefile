.PHONY: build build-zk zksetup zkprove test lint

## build: Build all Go packages (pure Go, no CGo required).
build:
	go build ./...

## zksetup: Generate Groth16 proving key and verifying key (run once).
## Outputs: keys/verifying_key.bin, keys/proving_key.bin
zksetup:
	@echo "Running ZK trusted setup (Groth16/BN254)..."
	@mkdir -p keys
	go run ./tools/zksetup --out ./keys
	@echo ""
	@echo "Commit verifying_key.bin to the repository:"
	@echo "  git add keys/verifying_key.bin && git commit -m 'zk: add trusted setup VK'"

## zkprove: Generate a ZK proof for a given amount (off-chain, demo mode).
## Usage: make zkprove AMOUNT=1000000
zkprove:
	@echo "Generating ZK proof for amount=$(AMOUNT)..."
	go run ./tools/zkprove \
		--amount $(AMOUNT) \
		--pk keys/proving_key.bin \
		--out proof.json
	@echo "Proof written to proof.json"

## test: Run all Go tests.
test:
	go test ./...

## lint: Run Go linter.
lint:
	golangci-lint run ./...