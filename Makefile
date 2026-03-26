# Ragnarok Ecosystem Makefile

.PHONY: all build test clean install help

# Go commands
GO := go
GOFLAGS := -ldflags="-s -w"

# Binary output
BIN_DIR := bin
RAG_BIN := $(BIN_DIR)/rag.exe

all: build

build:
	@echo "Building Ragnarok Unified MCP Server..."
	mkdir -p $(BIN_DIR)
	$(GO) build $(GOFLAGS) -o $(RAG_BIN) ./cmd/rag
	@echo "✓ Ragnarok built"
	@echo ""
	@echo "  rag serve     Start MCP server (stdio mode)"
	@echo "  rag init      Initialize all plugins"
	@echo "  rag scan      Scan project and bootstrap"
	@echo "  rag stats     Show ecosystem health"

test:
	@echo "Running tests..."
	$(GO) test ./...

clean:
	@echo "Cleaning..."
	rm -rf $(BIN_DIR)
	@echo "✓ Clean complete"

install: build
	@echo "Installing to ~/.local/bin..."
	mkdir -p ~/.local/bin
	cp $(RAG_BIN) ~/.local/bin/
	@echo "✓ Installed to ~/.local/bin/rag.exe"

deps:
	$(GO) mod tidy

lint:
	$(GO) vet ./...

help:
	@echo "Ragnarok v1.1.1 - AI Governance & Memory Layer Ecosystem"
	@echo ""
	@echo "Targets:"
	@echo "  build     Build rag.exe (unified MCP server)"
	@echo "  test      Run all tests"
	@echo "  clean     Remove build artifacts"
	@echo "  install   Build and install to ~/.local/bin"
	@echo "  deps      Download dependencies"
	@echo "  lint      Run linters"
	@echo ""
	@echo "Usage:"
	@echo "  rag serve              Start unified MCP server"
	@echo "  rag init --project X   Initialize plugins for project X"
	@echo "  rag scan --path ./X    Scan project X"
	@echo "  rag stats --ecosystem  Show ecosystem health"
