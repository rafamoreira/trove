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
