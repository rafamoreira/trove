# trove

A local-first snippet vault. Store, search, and sync code snippets from your terminal.

## Install

```bash
go install github.com/rafamoreira/trove/cmd/trove@latest
```

Or build from source:

```bash
go build -o bin/trove ./cmd/trove
```

## Setup

```bash
trove init                          # initialize vault
trove init --remote <git-remote>    # initialize with git sync
```

## Usage

```bash
# Create a new snippet (opens editor)
trove new my_snippet --lang go
trove new my_snippet.go             # language inferred from extension

# Add an existing file
trove add ./script.sh

# Add from stdin or editor buffer
trove add --lang python --name my_script

# List and search
trove list
trove list --lang go --tag auth
trove search "http handler"

# View a snippet
trove show go/my_snippet
trove show go/my_snippet --meta     # include metadata header

# Edit a snippet
trove edit go/my_snippet
trove edit go/my_snippet --desc "new description" --tags "api,auth"

# Remove a snippet
trove rm go/my_snippet
trove rm go/my_snippet --force      # skip confirmation

# Sync
trove sync
trove status
```

All commands support `--json` for machine-readable output.

## Configuration

Config file: `~/.config/trove/config.toml` (respects `$XDG_CONFIG_HOME`)

```toml
vault_path = "~/.local/share/trove/vault"
editor     = "vi"
git_remote = "origin"
git_branch = "main"
auto_sync  = true
auto_push  = true
sync_debounce_seconds = 5
```

Config precedence: defaults → file → environment variables (`TROVE_VAULT_PATH`, `TROVE_EDITOR`, `TROVE_GIT_REMOTE`, `TROVE_GIT_BRANCH`, `TROVE_AUTO_SYNC`, `TROVE_AUTO_PUSH`, `TROVE_SYNC_DEBOUNCE_SECONDS`) → flags.

```bash
trove config           # show resolved config
trove config --edit    # open config in editor
```

## Shell completions

```bash
trove completion bash   # or zsh, fish, powershell
```

## Supported languages

Trove uses [go-enry](https://github.com/go-enry/go-enry) (a Go port of GitHub's [Linguist](https://github.com/github/linguist)) for language detection. Any language recognized by Linguist is supported — including aliases (e.g., `js` for JavaScript, `rs` for Rust) and file extension inference (e.g., `.py` → Python).
