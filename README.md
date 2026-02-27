# NmodCleaner

[![Go Version](https://img.shields.io/badge/go-1.25+-blue)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

**NmodCleaner** is a blazing-fast, memory-efficient CLI tool written in Go. It finds bloated,
unoptimized, or corrupt `node_modules` directories across your projects, lets you pick which ones
to clean, and automatically reinstalls dependencies using [pnpm](https://pnpm.io) — migrating
your workspaces to a shared global store in the process.

Originally rewritten in Go to solve `heap out of memory` errors encountered when scanning massive
filesystems with JavaScript/Node.js.

---

## Features

- **Memory-safe scanning** — uses `filepath.WalkDir` to traverse large directory trees without
  hoarding RAM; gracefully skips permission-denied paths
- **Concurrent analysis** — goroutines calculate `node_modules` sizes in parallel
- **Interactive selection** — beautiful multi-select checklist (powered by
  [charmbracelet/huh](https://github.com/charmbracelet/huh)) sorted by disk usage
- **Dry-run mode** — preview deletions and space savings before touching anything
- **pnpm-aware** — automatically skips directories already managed by pnpm
- **Parallel cleanup** — deletes and reinstalls all selected projects concurrently

---

## Prerequisites

- [Go](https://go.dev/dl/) >= 1.25
- [pnpm](https://pnpm.io/installation) installed and available in your `PATH`

---

## Installation

**Using `go install`:**

```bash
go install github.com/Mahmoud-s-Khedr/nmod-cleaner@latest
```

**Build from source:**

```bash
git clone https://github.com/Mahmoud-s-Khedr/nmod-cleaner.git
cd nmod-cleaner
go build -o nmod-cleaner .
sudo mv nmod-cleaner /usr/local/bin/
```

---

## Usage

Navigate to any directory that contains multiple Node.js projects and run:

```bash
nmod-cleaner
```

Or point it at a specific path:

```bash
nmod-cleaner --path /path/to/your/projects
```

### Dry Run

Preview what would be removed and how much space you would reclaim — without deleting anything:

```bash
nmod-cleaner --dry-run
```

### Options

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--path` | `-p` | current directory | Directory to scan for `node_modules` |
| `--dry-run` | `-d` | `false` | Simulate execution, print space savings |
| `--help` | `-h` | | Display help |

---

## How It Works

1. **Scan** — recursively walks the target directory for `node_modules` folders, skipping hidden
   directories, build outputs (`dist/`, `build/`), and system paths
2. **Filter** — removes any `node_modules` already managed by pnpm (detected via
   `.modules.yaml` or a `pnpm-lock.yaml` in the parent project)
3. **Analyze** — calculates disk usage for each candidate concurrently
4. **Select** — presents an interactive checklist sorted by size (largest first), with all
   entries pre-selected
5. **Clean** — deletes selected `node_modules` directories and runs `pnpm install` in each
   project in parallel; reports a per-run summary of successes and failures

---

## Known Limitations

- Reinstallation requires `pnpm`; projects using `npm` or `yarn` only will be cleaned but
  require manual reinstallation
- Tested on Linux and macOS; Windows support is untested

---

## Contributing

Contributions are welcome! Please open an issue first to discuss significant changes.

1. Fork the repository
2. Create a feature branch (`git checkout -b feat/my-feature`)
3. Commit your changes
4. Push and open a Pull Request

---

## License

[MIT](LICENSE)
