# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Build
go build -o bin/trove ./cmd/trove

# Run tests
go test ./...

# Run a single test
go test ./internal/vault/ -run TestFunctionName

# Run with verbose output
go test ./... -v
```

## Architecture

Trove is a local-first snippet vault CLI built in Go. Snippets are stored as files in a language-organized directory tree, with TOML sidecar files for metadata.

### Layered structure

```
cmd/trove/main.go
    ↓
internal/cli/        ← Cobra commands, I/O, output rendering
    ↓
internal/vault/      ← Core domain: CRUD, language detection, git integration
    ↓
internal/config/     ← Config loading (file → env → flags precedence)
internal/search/     ← Full-text search across snippet body/metadata
internal/editor/     ← External editor invocation
```

### Vault layout on disk

```
~/.local/share/trove/vault/
    go/
        my_snippet.go       ← snippet body
        my_snippet.toml     ← metadata sidecar (description, tags, created)
    python/
        ...
```

Snippets are addressed as `lang/name` (e.g., `go/my_snippet`). Language aliases and extension mappings live in `internal/vault/language.go`.

### Key types

- `internal/cli/root.go` — all 13 commands (`new`, `add`, `edit`, `show`, `list`, `search`, `rm`, `sync`, `status`, `config`, `cd`, `completion`); uses a `runtime` struct holding config, vault, and I/O streams
- `internal/vault/vault.go` — `CreateSnippet`, `List`, `Resolve`, `DeleteSnippet`, `UpdateSnippet`
- `internal/vault/snippet.go` — `Snippet` struct; TOML metadata serialization
- `internal/vault/git.go` — thin wrappers around `exec.Command("git", ...)` for init, add, commit, push, pull, status
- `internal/config/config.go` — XDG-aware config at `~/.config/trove/config.toml`; fields: `vault_path`, `editor`, `git_remote`, `git_branch`, `auto_sync`, `auto_push`, `sync_debounce_seconds`

### Output

Commands support `--json` for machine-readable output alongside human-readable table format.
