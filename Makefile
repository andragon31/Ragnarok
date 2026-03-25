# Ragnarok Ecosystem Makefile

.PHONY: all build build-fenrir build-hati build-skoll build-tyr test clean install

# Go commands
GO := go
GOFLAGS := -ldflags="-s -w"

# Directories
FENRIR_DIR := fenrir
HATI_DIR := hati
SKOLL_DIR := skoll
TYR_DIR := tyr
INSTALLER_DIR := installer

# Binaries output
BIN_DIR := bin
FENRIR_BIN := $(BIN_DIR)/fenrir.exe
HATI_BIN := $(BIN_DIR)/hati.exe
SKOLL_BIN := $(BIN_DIR)/skoll.exe
TYR_BIN := $(BIN_DIR)/tyr.exe
INSTALLER_BIN := $(BIN_DIR)/rag.exe

all: build

build: build-fenrir build-hati build-skoll build-tyr build-installer
	@echo "Build complete!"

build-fenrir:
	@echo "Building Fenrir..."
	cd $(FENRIR_DIR) && $(GO) build $(GOFLAGS) -o ../$(BIN_DIR)/fenrir.exe ./cmd/fenrir
	@echo "✓ Fenrir built"

build-hati:
	@echo "Building Hati..."
	cd $(HATI_DIR) && $(GO) build $(GOFLAGS) -o ../$(BIN_DIR)/hati.exe ./cmd/hati
	@echo "✓ Hati built"

build-skoll:
	@echo "Building Skoll..."
	cd $(SKOLL_DIR) && $(GO) build $(GOFLAGS) -o ../$(BIN_DIR)/skoll.exe ./cmd/skoll
	@echo "✓ Skoll built"

build-tyr:
	@echo "Building Tyr..."
	cd $(TYR_DIR) && $(GO) build $(GOFLAGS) -o ../$(BIN_DIR)/tyr.exe ./cmd/tyr
	@echo "✓ Tyr built"

build-installer:
	@echo "Building Installer..."
	cd $(INSTALLER_DIR) && $(GO) build $(GOFLAGS) -o ../$(BIN_DIR)/rag.exe ./cmd/rag
	@echo "✓ Installer built"

test:
	@echo "Running tests..."
	cd $(FENRIR_DIR) && $(GO) test ./...
	cd $(HATI_DIR) && $(GO) test ./...
	cd $(SKOLL_DIR) && $(GO) test ./...
	cd $(TYR_DIR) && $(GO) test ./...

test-fenrir:
	cd $(FENRIR_DIR) && $(GO) test ./...

test-hati:
	cd $(HATI_DIR) && $(GO) test ./...

test-skoll:
	cd $(SKOLL_DIR) && $(GO) test ./...

test-tyr:
	cd $(TYR_DIR) && $(GO) test ./...

clean:
	@echo "Cleaning..."
	rm -rf $(BIN_DIR)
	@echo "Clean complete!"

install: build
	@echo "Installing to ~/.local/bin..."
	mkdir -p ~/.local/bin
	cp $(FENRIR_BIN) ~/.local/bin/ 2>/dev/null || true
	cp $(HATI_BIN) ~/.local/bin/ 2>/dev/null || true
	cp $(SKOLL_BIN) ~/.local/bin/ 2>/dev/null || true
	cp $(TYR_BIN) ~/.local/bin/ 2>/dev/null || true
	cp $(INSTALLER_BIN) ~/.local/bin/ 2>/dev/null || true
	@echo "Installation complete!"

deps:
	cd $(FENRIR_DIR) && $(GO) mod tidy
	cd $(HATI_DIR) && $(GO) mod tidy
	cd $(SKOLL_DIR) && $(GO) mod tidy
	cd $(TYR_DIR) && $(GO) mod tidy
	cd $(INSTALLER_DIR) && $(GO) mod tidy

lint:
	cd $(FENRIR_DIR) && $(GO) vet ./...
	cd $(HATI_DIR) && $(GO) vet ./...
	cd $(SKOLL_DIR) && $(GO) vet ./...
	cd $(TYR_DIR) && $(GO) vet ./...

help:
	@echo "Ragnarok Ecosystem Makefile"
	@echo ""
	@echo "Targets:"
	@echo "  all             Build all plugins (default)"
	@echo "  build           Build all plugins"
	@echo "  build-fenrir   Build Fenrir plugin"
	@echo "  build-hati     Build Hati plugin"
	@echo "  build-skoll    Build Skoll plugin"
	@echo "  build-tyr      Build Tyr plugin"
	@echo "  build-installer Build Ragnarok installer"
	@echo "  test           Run all tests"
	@echo "  clean          Clean build artifacts"
	@echo "  install        Build and install to ~/.local/bin"
	@echo "  deps           Download dependencies"
	@echo "  lint           Run linters"
