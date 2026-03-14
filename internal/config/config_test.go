package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadPrecedenceAndSources(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "cfg"))

	configPath := filepath.Join(t.TempDir(), "trove.toml")
	if err := os.WriteFile(configPath, []byte(`
vault_path = "/from-file"
editor = "nano"
git_remote = "upstream"
git_branch = "develop"
auto_sync = false
sync_debounce_seconds = 11
auto_push = false
`), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("TROVE_VAULT_PATH", "/from-env")
	t.Setenv("TROVE_EDITOR", "hx")

	branch := "feature"
	cfg, err := Load(configPath, Overrides{
		GitBranch: &branch,
	})
	if err != nil {
		t.Fatal(err)
	}

	if cfg.VaultPath != "/from-env" {
		t.Fatalf("vault path = %q, want /from-env", cfg.VaultPath)
	}
	if cfg.Editor != "hx" {
		t.Fatalf("editor = %q, want hx", cfg.Editor)
	}
	if cfg.GitRemote != "upstream" {
		t.Fatalf("git remote = %q, want upstream", cfg.GitRemote)
	}
	if cfg.GitBranch != "feature" {
		t.Fatalf("git branch = %q, want feature", cfg.GitBranch)
	}
	if cfg.Sources[FieldVaultPath] != SourceEnv {
		t.Fatalf("vault source = %q, want %q", cfg.Sources[FieldVaultPath], SourceEnv)
	}
	if cfg.Sources[FieldEditor] != SourceEnv {
		t.Fatalf("editor source = %q, want %q", cfg.Sources[FieldEditor], SourceEnv)
	}
	if cfg.Sources[FieldGitRemote] != SourceFile {
		t.Fatalf("git_remote source = %q, want %q", cfg.Sources[FieldGitRemote], SourceFile)
	}
	if cfg.Sources[FieldGitBranch] != SourceFlag {
		t.Fatalf("git_branch source = %q, want %q", cfg.Sources[FieldGitBranch], SourceFlag)
	}
}

func TestExpandPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	got, err := ExpandPath("~/vault")
	if err != nil {
		t.Fatal(err)
	}

	want := filepath.Join(home, "vault")
	if got != want {
		t.Fatalf("ExpandPath = %q, want %q", got, want)
	}
}
