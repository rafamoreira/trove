package vault

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rafamoreira/trove/internal/config"
)

func TestResolveSelectorAndListWarnings(t *testing.T) {
	base := t.TempDir()
	v := New(&config.Config{
		VaultPath: base,
		FilePath:  filepath.Join(t.TempDir(), "config.toml"),
	})
	v.Now = func() time.Time { return time.Date(2026, 3, 14, 12, 0, 0, 0, time.UTC) }

	if _, err := v.CreateSnippet("python", "retry", []byte("print('hi')\n"), "python retry", []string{"Python"}); err != nil {
		t.Fatal(err)
	}
	if _, err := v.CreateSnippet("go", "retry", []byte("package retry\n"), "go retry", []string{"Go"}); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(base, "shell"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, "shell", "orphan.toml"), []byte("description = 'orphan'\ncreated = 2026-03-14T12:00:00Z\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, warnings, err := v.Resolve("python/retry"); err != nil {
		t.Fatal(err)
	} else if len(warnings) != 1 || warnings[0].Code != "orphan_metadata" {
		t.Fatalf("warnings = %#v, want orphan metadata warning", warnings)
	}

	if _, _, err := v.Resolve("retry"); err == nil {
		t.Fatal("expected ambiguous bare selector error")
	}
}

func TestSyncPushesEvenWhenNothingNewToCommit(t *testing.T) {
	if !GitAvailable() {
		t.Skip("git not available")
	}

	// Create a bare repo to act as the remote.
	remoteDir := t.TempDir()
	remotePath := filepath.Join(remoteDir, "remote.git")
	cmd := exec.Command("git", "init", "--bare", remotePath)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init --bare: %v\n%s", err, out)
	}
	// Force the bare remote's HEAD to "main" so git log works after we push.
	cmd = exec.Command("git", "--git-dir="+remotePath, "symbolic-ref", "HEAD", "refs/heads/main")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("set remote HEAD: %v\n%s", err, out)
	}
	remoteURL := "file://" + filepath.Join(remoteDir, "remote.git")

	// Init vault and add the remote.
	vaultBase := t.TempDir()
	cfg := &config.Config{
		VaultPath:  vaultBase,
		FilePath:   filepath.Join(t.TempDir(), "config.toml"),
		GitRemote:  "origin",
		GitBranch:  "main",
		AutoPush:   true,
	}
	v := New(cfg)
	v.Now = func() time.Time { return time.Date(2026, 3, 14, 12, 0, 0, 0, time.UTC) }

	if _, err := v.Init(remoteURL); err != nil {
		t.Fatal(err)
	}

	// Force initial branch to "main" regardless of the system default.
	if _, err := v.git("symbolic-ref", "HEAD", "refs/heads/main"); err != nil {
		t.Fatal(err)
	}

	// Configure git identity so commits work in a clean environment.
	for _, kv := range [][2]string{
		{"user.email", "test@example.com"},
		{"user.name", "Test"},
	} {
		if _, err := v.git("config", kv[0], kv[1]); err != nil {
			t.Fatalf("git config %s: %v", kv[0], err)
		}
	}

	// Commit the initial files that Init creates so they don't pollute SyncNow.
	if err := v.GitAddAll(); err != nil {
		t.Fatal(err)
	}
	if _, err := v.GitCommit("init"); err != nil {
		t.Fatal(err)
	}

	// Create a snippet and commit it locally (no push).
	snippet, err := v.CreateSnippet("go", "hello", []byte("package main\n"), "hello", nil)
	if err != nil {
		t.Fatal(err)
	}
	if warnings := v.CommitSnippet("add hello", snippet.Path, snippet.MetaPath); len(warnings) != 0 {
		t.Fatalf("CommitSnippet warnings: %v", warnings)
	}

	// SyncNow should find nothing new to commit but still push.
	committed, pushed, warnings, err := v.SyncNow()
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Fatalf("SyncNow warnings: %v", warnings)
	}
	if committed {
		t.Error("committed should be false — nothing new to stage")
	}
	if !pushed {
		t.Error("pushed should be true — unpushed local commit exists")
	}

	// Verify the remote received the commit.
	out, err := exec.Command("git", "--git-dir="+filepath.Join(remoteDir, "remote.git"), "log", "--oneline").Output()
	if err != nil {
		t.Fatalf("git log on remote: %v", err)
	}
	if !strings.Contains(string(out), "add hello") {
		t.Errorf("remote log does not contain expected commit; got:\n%s", out)
	}
}

func TestNextGeneratedNameUsesTrackedCountAndSkipsExistingName(t *testing.T) {
	base := t.TempDir()
	v := New(&config.Config{
		VaultPath: base,
		FilePath:  filepath.Join(t.TempDir(), "config.toml"),
	})
	v.Now = func() time.Time { return time.Date(2026, 3, 14, 12, 30, 0, 0, time.UTC) }

	names := []struct {
		language string
		name     string
	}{
		{language: "python", name: "retry"},
		{language: "go", name: "retry"},
		{language: "python", name: "trove_3"},
		{language: "shell", name: "cleanup"},
		{language: "python", name: "trove_5"},
	}
	for _, item := range names {
		if _, err := v.CreateSnippet(item.language, item.name, []byte("body\n"), "", nil); err != nil {
			t.Fatal(err)
		}
	}

	snippet, warnings, err := v.Resolve("shell/cleanup")
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %#v", warnings)
	}
	if err := v.DeleteSnippet(snippet); err != nil {
		t.Fatal(err)
	}

	got, err := v.NextGeneratedName("python")
	if err != nil {
		t.Fatal(err)
	}
	if got != "trove_6" {
		t.Fatalf("NextGeneratedName(python) = %q, want trove_6", got)
	}
}
