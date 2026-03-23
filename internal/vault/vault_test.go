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

func TestCreateAndResolvePromptSnippet(t *testing.T) {
	base := t.TempDir()
	v := New(&config.Config{
		VaultPath: base,
		FilePath:  filepath.Join(t.TempDir(), "config.toml"),
	})
	v.Now = func() time.Time { return time.Date(2026, 3, 14, 12, 0, 0, 0, time.UTC) }

	body := []byte("Create a new Python web app using FastAPI with...\n")
	snippet, err := v.CreateSnippet("prompt", "new_python_app", body, "scaffold a python app", []string{"python", "scaffold"})
	if err != nil {
		t.Fatal(err)
	}

	// Verify file is stored with .prompt extension
	expectedPath := filepath.Join(base, "prompt", "new_python_app.prompt")
	if snippet.Path != expectedPath {
		t.Fatalf("snippet.Path = %q, want %q", snippet.Path, expectedPath)
	}
	if _, err := os.Stat(expectedPath); err != nil {
		t.Fatalf("prompt file does not exist: %v", err)
	}

	// Verify we can resolve it back
	resolved, warnings, err := v.Resolve("prompt/new_python_app")
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %v", warnings)
	}
	if resolved.Language != "prompt" {
		t.Fatalf("resolved.Language = %q, want prompt", resolved.Language)
	}
	if resolved.Name != "new_python_app" {
		t.Fatalf("resolved.Name = %q, want new_python_app", resolved.Name)
	}
}

func TestPublicFieldRoundTrip(t *testing.T) {
	base := t.TempDir()
	v := New(&config.Config{
		VaultPath: base,
		FilePath:  filepath.Join(t.TempDir(), "config.toml"),
	})
	v.Now = func() time.Time { return time.Date(2026, 3, 14, 12, 0, 0, 0, time.UTC) }

	snippet, err := v.CreateSnippet("go", "hello", []byte("package main\n"), "hello", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Default should be false.
	if snippet.Public {
		t.Fatal("expected Public to default to false")
	}

	// Set public, save, and reload.
	snippet.Public = true
	if err := snippet.SaveMeta(); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadSnippet(base, snippet.Path)
	if err != nil {
		t.Fatal(err)
	}
	if !loaded.Public {
		t.Fatal("expected Public to be true after round-trip")
	}
}

func TestPublicFieldDefaultsFalseForExistingSidecar(t *testing.T) {
	base := t.TempDir()
	langDir := filepath.Join(base, "go")
	if err := os.MkdirAll(langDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Write a sidecar without the public field (simulates pre-existing snippet).
	if err := os.WriteFile(filepath.Join(langDir, "old.toml"), []byte("description = 'old snippet'\ncreated = 2026-03-14T12:00:00Z\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(langDir, "old.go"), []byte("package old\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	snippet, err := LoadSnippet(base, filepath.Join(langDir, "old.go"))
	if err != nil {
		t.Fatal(err)
	}
	if snippet.Public {
		t.Fatal("expected Public to default to false for existing sidecar without field")
	}
}

func TestUpdateSnippetPublicField(t *testing.T) {
	base := t.TempDir()
	v := New(&config.Config{
		VaultPath: base,
		FilePath:  filepath.Join(t.TempDir(), "config.toml"),
	})
	v.Now = func() time.Time { return time.Date(2026, 3, 14, 12, 0, 0, 0, time.UTC) }

	snippet, err := v.CreateSnippet("go", "hello", []byte("package main\n"), "hello", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Update with public=true.
	pub := true
	if err := v.UpdateSnippet(snippet, nil, nil, &pub); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadSnippet(base, snippet.Path)
	if err != nil {
		t.Fatal(err)
	}
	if !loaded.Public {
		t.Fatal("expected Public to be true after UpdateSnippet")
	}

	// Update with nil public should not change it.
	if err := v.UpdateSnippet(snippet, nil, nil, nil); err != nil {
		t.Fatal(err)
	}
	loaded, err = LoadSnippet(base, snippet.Path)
	if err != nil {
		t.Fatal(err)
	}
	if !loaded.Public {
		t.Fatal("expected Public to remain true when nil passed")
	}
}

func TestListFilterByPublic(t *testing.T) {
	base := t.TempDir()
	v := New(&config.Config{
		VaultPath: base,
		FilePath:  filepath.Join(t.TempDir(), "config.toml"),
	})
	v.Now = func() time.Time { return time.Date(2026, 3, 14, 12, 0, 0, 0, time.UTC) }

	s1, err := v.CreateSnippet("go", "public_one", []byte("package main\n"), "public", nil)
	if err != nil {
		t.Fatal(err)
	}
	s1.Public = true
	if err := s1.SaveMeta(); err != nil {
		t.Fatal(err)
	}

	if _, err := v.CreateSnippet("go", "private_one", []byte("package main\n"), "private", nil); err != nil {
		t.Fatal(err)
	}

	// Filter for public only.
	pub := true
	items, _, err := v.List(ListOptions{Public: &pub})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Name != "public_one" {
		t.Fatalf("expected 1 public snippet, got %d", len(items))
	}

	// Filter for private only.
	priv := false
	items, _, err = v.List(ListOptions{Public: &priv})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Name != "private_one" {
		t.Fatalf("expected 1 private snippet, got %d", len(items))
	}

	// No filter returns all.
	items, _, err = v.List(ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 snippets, got %d", len(items))
	}
}

func TestListIgnoresSiteDirectory(t *testing.T) {
	base := t.TempDir()
	v := New(&config.Config{
		VaultPath: base,
		FilePath:  filepath.Join(t.TempDir(), "config.toml"),
	})
	v.Now = func() time.Time { return time.Date(2026, 3, 14, 12, 0, 0, 0, time.UTC) }

	if _, err := v.CreateSnippet("go", "hello", []byte("package main\n"), "hello", nil); err != nil {
		t.Fatal(err)
	}

	// Simulate a _site directory with HTML files.
	siteDir := filepath.Join(base, "_site")
	if err := os.MkdirAll(filepath.Join(siteDir, "go"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(siteDir, "index.html"), []byte("<html></html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(siteDir, "go", "hello.html"), []byte("<html></html>"), 0o644); err != nil {
		t.Fatal(err)
	}

	items, warnings, err := v.List(ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 snippet, got %d", len(items))
	}
	for _, w := range warnings {
		if strings.Contains(w.Path, "_site") {
			t.Fatalf("unexpected warning about _site: %v", w)
		}
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
