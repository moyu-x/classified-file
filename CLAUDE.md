# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Classified File is a Go CLI tool for file classification and deduplication. It scans directories, computes xxHash checksums, stores results in SQLite, and can delete or relocate duplicate files. A secondary `classify` command sorts files by MIME type into categorized subdirectories.

## Build & Development Commands

Uses [go-task](https://taskfile.dev) (`task`) as the task runner:

```bash
task build          # Compile to build/classified-file
task build-all      # Cross-compile for darwin/linux/windows (amd64+arm64)
task test           # Run core module tests (scanner, hasher, database)
task test-all       # Run all tests verbose
task test-race      # Run tests with race detector
task test-coverage  # Generate HTML coverage report
task vet            # go vet ./...
task lint           # golangci-lint run (requires golangci-lint installed)
task fmt            # go fmt ./...
task dev            # build + test
task ci             # fmt-check + vet + test
task clean          # Remove build/ and coverage files
```

To run a single test: `go test ./pkg/hasher/... -run TestCalculateHash -v`

## Architecture

```
main.go                    → Entry point, calls cmd.Execute()
cmd/                       → Cobra CLI commands (root, classify, dedup, init)
internal/                  → Private app logic (not importable externally)
  app/classify.go          → Orchestrates the classify workflow
  app/dedup.go             → Orchestrates the dedup workflow
  types.go                 → Shared types: ProcessStats, FileRecord, OperationMode
  constants.go             → Default paths and buffer sizes
pkg/                       → Reusable public packages
  classifier/              → File classification by MIME type (uses h2non/filetype)
  deduplicator/            → Duplicate detection, delete/move logic, signal handling
  database/                → SQLite via GORM (glebarez/sqlite, pure Go), in-memory hash cache
  hasher/                  → xxHash (cespare/xxhash/v2) file hashing
  scanner/                 → Directory walking and file counting
  progress/                → Resume/checkpoint support via per-directory progress files
  config/                  → Viper-based YAML config (~/.classified-file/config.yaml)
  logger/                  → Zerolog with console writer, optional file output
```

### Key Data Flow

**Dedup pipeline:** `cmd/dedup.go` → `app.RunDedup()` → creates `Database` + `Deduplicator` → `scanner.FileWalker` counts then walks files → `hasher.CalculateHash()` per file → `Database.Exists()` checks hash (memory cache first, then SQLite) → new files get `Database.Insert()`, duplicates get deleted or moved.

**Classify pipeline:** `cmd/classify.go` → `app.RunClassify()` → `classifier.Classify()` → walks source dirs → `filetype.Match()` detects MIME type → files are copied into `dest/<category>/part_NNNN/` with auto-renaming on collision.

### Resume/Checkpoint

The dedup command supports `--resume` (skip already-scanned files) and `--reset` (clear progress). Progress is tracked per-directory via `.classified-file-progress.txt` files managed by `pkg/progress`. On SIGINT, the deduplicator flushes progress to allow resuming later.

## Conventions

- Module path: `github.com/moyu-x/classified-file`
- Go version: 1.25.1
- Database: SQLite with WAL mode, single connection (`MaxOpenConns=1`), hash lookup uses an in-memory `map[string]bool` cache
- Config location: `~/.classified-file/config.yaml`, database at `~/.classified-file/hashes.db`
- All logging goes through `pkg/logger` (zerolog); call `logger.Get()` to access the singleton
- Test files are co-located with source (e.g., `pkg/hasher/hasher_test.go`)
- Tests use temporary directories for file I/O operations
