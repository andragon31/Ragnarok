# Ragnarok Makefile

BINARY_NAME=rag
MOD_DIRECTORIES=./cmd/fenrir ./cmd/hati ./cmd/skoll ./cmd/tyr ./cmd/rag

.PHONY: all build build-all test clean doctor release help lint

all: build test

build:
	go build -v -o $(BINARY_NAME) ./cmd/rag/main.go

build-all:
	@echo "Building all modules..."
	go build -v -o fenrir ./cmd/fenrir/main.go
	go build -v -o hati ./cmd/hati/main.go
	go build -v -o skoll ./cmd/skoll/main.go
	go build -v -o tyr ./cmd/tyr/main.go
	go build -v -o rag ./cmd/rag/main.go

test:
	go test -v -race ./...

doctor: build
	./$(BINARY_NAME) mcp diagnose --verbose

clean:
	@echo "Cleaning binaries..."
	rm -f fenrir hati skoll tyr rag
	rm -f *.exe
	rm -rf dist/

release:
	goreleaser release --snapshot --clean

lint:
	golangci-lint run

help:
	@echo "Ragnarok v3 Build System"
	@echo "  build       - Build the unified 'rag' binary"
	@echo "  build-all   - Build all 5 standalone binaries"
	@echo "  test        - Run all tests with race detection"
	@echo "  doctor      - Run ecosystem diagnostics"
	@echo "  clean       - Remove all binaries and artifacts"
	@echo "  release     - Create a snapshot release with goreleaser"
	@echo "  lint        - Run golangci-lint"
