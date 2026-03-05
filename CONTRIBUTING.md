# Contributing to NmodCleaner

Thanks for your interest in contributing! Contributions of any kind are welcome — bug fixes, new features, documentation improvements, or even just reporting issues.

---

## Getting Started

### Prerequisites

- [Go](https://go.dev/dl/) >= 1.22
- [pnpm](https://pnpm.io/installation) (needed to test the reinstall step)
- `git`

### Setting Up the Dev Environment

```bash
# 1. Fork the repo on GitHub, then clone your fork
git clone https://github.com/<your-username>/node_cleaner.git
cd node_cleaner

# 2. Install dependencies
go mod download

# 3. Build and run locally
go build -o nmod-cleaner .
./nmod-cleaner --help

# 4. Run with a test path (dry-run is safe)
./nmod-cleaner --path /path/to/your/projects --dry-run
```

---

## Project Structure

```
node_cleaner/
├── main.go                   # Entry point
├── cmd/
│   └── root.go               # CLI orchestration (scan → filter → analyze → select → clean)
└── internal/
    ├── scanner/              # Filesystem traversal and pnpm detection
    ├── analyzer/             # Concurrent disk-usage calculation
    ├── cleaner/              # node_modules deletion
    ├── installer/            # pnpm install execution
    ├── ui/                   # TUI prompt and terminal styles
    ├── config/               # (planned) .nmodcleanerrc skip-list loader
    └── history/              # (planned) append-only run history and cumulative stats
```

The core pipeline is defined in `cmd/root.go`. Each package in `internal/` is deliberately small and single-responsibility, making it easy to extend or swap components.

---

## Making Changes

1. **Open an issue first** for significant changes — this avoids duplicate work and lets us discuss the approach
2. Create a feature branch from `main`:
   ```bash
   git checkout -b feat/my-feature
   ```
3. Make your changes, following the existing code style
4. Verify your changes work end-to-end with a dry run:
   ```bash
   go build -o nmod-cleaner . && ./nmod-cleaner --dry-run
   ```
5. Run `go vet ./...` to catch any static issues
6. Commit with a clear, descriptive message:
   ```bash
   git commit -m "feat: add yarn support for reinstallation"
   ```
7. Push your branch and open a Pull Request against `main`

---

## Roadmap Contributions

The following features are planned and available for contribution. Each is scoped to a small set of files so they are good isolated tasks.

| Feature | Key files | Complexity |
|---|---|---|
| **npm / yarn support** | `internal/installer/installer.go`, `internal/scanner/scanner.go`, `cmd/root.go` | Medium |
| **Progress bars** | `internal/ui/prompt.go`, `cmd/root.go` | Medium |
| **Windows support** | `internal/scanner/scanner.go`, `internal/installer/installer.go` | Low |
| **Config file** | `internal/config/config.go` *(new)*, `cmd/root.go` | Low |
| **Stats history** | `internal/history/history.go` *(new)*, `cmd/root.go` | Low |

Before starting, open an issue so we can assign it to you and avoid duplicate work.

---

## Code Style

- Follow standard Go conventions (`gofmt`, `go vet`)
- Keep packages small and focused on a single responsibility
- Add comments to exported functions
- Prefer explicit error handling over panics

---

## Reporting Bugs

Please open a GitHub Issue with:
- Your OS and Go version
- The exact command you ran
- The full error output or unexpected behavior

---

## License

By contributing, you agree that your contributions will be licensed under the [MIT License](LICENSE).
