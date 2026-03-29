# Ragnarok Ecosystem Makefile

VERSION  := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS  := -ldflags="-s -w -X main.version=$(VERSION)"
BIN_DIR  := bin

ifeq ($(OS),Windows_NT)
    BINARY := $(BIN_DIR)/rag.exe
    INSTALL_DIR := $(LOCALAPPDATA)\Ragnarok
else
    BINARY := $(BIN_DIR)/rag
    INSTALL_DIR := $(HOME)/.local/bin
endif

.PHONY: all build test clean install lint release help

all: build

build:
	@echo "Building Ragnarok $(VERSION)..."
	@mkdir -p $(BIN_DIR)
	go build $(LDFLAGS) -o $(BINARY) ./cmd/rag
	@echo "Built: $(BINARY)"

build-all:
	@echo "Cross-compiling for all platforms..."
	@mkdir -p $(BIN_DIR)
	GOOS=linux   GOARCH=amd64  go build $(LDFLAGS) -o $(BIN_DIR)/rag_linux_amd64   ./cmd/rag
	GOOS=linux   GOARCH=arm64  go build $(LDFLAGS) -o $(BIN_DIR)/rag_linux_arm64   ./cmd/rag
	GOOS=darwin  GOARCH=amd64  go build $(LDFLAGS) -o $(BIN_DIR)/rag_darwin_amd64  ./cmd/rag
	GOOS=darwin  GOARCH=arm64  go build $(LDFLAGS) -o $(BIN_DIR)/rag_darwin_arm64  ./cmd/rag
	GOOS=windows GOARCH=amd64  go build $(LDFLAGS) -o $(BIN_DIR)/rag_windows_amd64.exe ./cmd/rag

test:
	go test ./...

lint:
	go vet ./...

clean:
	rm -rf $(BIN_DIR)

install: build
	@echo "Installing to $(INSTALL_DIR)..."
	@mkdir -p $(INSTALL_DIR)
	cp $(BINARY) $(INSTALL_DIR)/
	@echo "Installed"

deps:
	go mod tidy

release:
	@echo "Creating release $(VERSION)..."
	goreleaser release --clean

help:
	@echo "Ragnarok $(VERSION) - AI Governance & Autonomous Development Ecosystem"
	@echo ""
	@echo "Targets:"
	@echo "  build       Build for current platform"
	@echo "  build-all   Cross-compile for all platforms"
	@echo "  test        Run all tests"
	@echo "  lint        Run go vet"
	@echo "  clean       Remove build artifacts"
	@echo "  install     Build and install locally"
	@echo "  deps        Download dependencies"
	@echo "  release     Create GoReleaser release"
