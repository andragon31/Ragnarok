# Contributing to Ragnarok

Thank you for your interest in contributing to Ragnarok.

## Development Setup

### Prerequisites

- Go 1.22+
- Git
- PowerShell (for Windows installers)
- Bash (for Linux/macOS installers)

### Initial Setup

```bash
# Clone the repository
git clone https://github.com/andragon31/Ragnarok
cd Ragnarok

# Install dependencies
go mod tidy

# Build the project
go build -o bin/rag ./cmd/rag

# Run tests
go test ./...
```

### Project Structure

```
Ragnarok/
├── cmd/rag/           # CLI entrypoint
├── internal/
│   ├── hati/          # Planning module
│   ├── fenrir/        # Memory module
│   ├── skoll/         # Orchestration module
│   ├── tyr/           # Quality module
│   └── mcp/           # Unified MCP server
├── .github/
│   └── workflows/     # CI/CD pipelines
└── *.ps1, *.sh       # Installers
```

## Making Changes

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/my-feature`
3. Make your changes
4. Run tests: `go test -race ./...`
5. Run linting: `go vet ./...`
6. Commit your changes
7. Push to your fork
8. Open a Pull Request

## Code Style

- Follow Go conventions (run `go fmt` before committing)
- All MCP handlers follow the signature: `func (h *Handler) HandleXxx(ctx context.Context, args map[string]any) (map[string]any, error)`
- IDs are generated with `generateID()` (thread-safe)
- All databases use `PRAGMA foreign_keys = ON` and `PRAGMA journal_mode = WAL`

## Testing

Tests are located in `internal/*/database/` directories:

```bash
# Run all tests with race detector
go test -race ./...

# Run tests for a specific module
go test -race ./internal/hati/database/...
```

## Release Process

Releases are automated via GitHub Actions using GoReleaser:

1. Update version in relevant files if needed
2. Create a tag: `git tag v.x.x.x`
3. Push tags: `git push --tags`
4. GitHub Actions will build, test, and release automatically

## Reporting Issues

Please report issues via GitHub Issues with:
- Clear description of the problem
- Steps to reproduce
- Expected vs actual behavior
- Go version and OS

## Questions

For questions, open a GitHub Discussion or reach out via the project repository.
