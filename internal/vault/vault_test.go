package vault

import (
	"os"
	"path/filepath"
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
