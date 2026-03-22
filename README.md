# Stanza CLI

[![CI](https://github.com/stanza-go/cli/actions/workflows/ci.yml/badge.svg)](https://github.com/stanza-go/cli/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

CLI tool for managing [Stanza](https://github.com/stanza-go/standalone) applications. Operates on the data directory — backup, restore, inspect.

## Commands

| Command | Description |
|---------|-------------|
| `stanza export` | Export the data directory as a zip archive |
| `stanza import <file>` | Restore from a zip archive |
| `stanza backup` | Create a consistent SQLite backup (VACUUM INTO) |
| `stanza status` | Show application status and data directory info |
| `stanza db` | SQLite database info and stats |
| `stanza logs` | View and tail structured logs |

## Install

Requires Go 1.26.1+ and CGo.

```bash
go install github.com/stanza-go/cli@latest
```

Or build from source:

```bash
git clone https://github.com/stanza-go/cli.git
cd cli
CGO_ENABLED=1 go build -o stanza .
```

## Usage

```bash
# Export data directory to a zip
stanza export

# Import from backup
stanza import backup.zip

# Create a consistent database backup
stanza backup

# Check application status
stanza status

# View recent logs
stanza logs

# Tail logs in real-time
stanza logs -f
```

The CLI reads the data directory from `DATA_DIR` environment variable, falling back to `~/.stanza/`.

## License

MIT
