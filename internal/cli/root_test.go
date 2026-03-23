package cli_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rafamoreira/trove/internal/cli"
)

func TestCLIWorkflowAndJSONContract(t *testing.T) {
	workspace := t.TempDir()
	vaultPath := filepath.Join(workspace, "vault")
	configPath := filepath.Join(workspace, "config", "trove.toml")
	editorPath := writeEditorScript(t, workspace, "#!/bin/sh\nprintf 'print(\"hello\")\\n' > \"$1\"\n")

	t.Setenv("HOME", workspace)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(workspace, "xdg"))
	t.Setenv("TROVE_EDITOR", editorPath)

	now := func() time.Time { return time.Date(2026, 3, 14, 15, 9, 26, 0, time.UTC) }

	if _, stderr, err := runCLI(t, now, nil, "--config", configPath, "--vault", vaultPath, "init"); err != nil {
		t.Fatalf("init error: %v, stderr=%s", err, stderr)
	}

	if _, stderr, err := runCLI(t, now, nil, "--config", configPath, "--vault", vaultPath, "new", "retry.py", "--desc", "Retry helper", "--tags", "Python,Utils"); err != nil {
		t.Fatalf("new error: %v, stderr=%s", err, stderr)
	}

	stdout, stderr, err := runCLI(t, now, nil, "--json", "--config", configPath, "--vault", vaultPath, "show", "python/retry")
	if err != nil {
		t.Fatalf("show error: %v, stderr=%s", err, stderr)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatal(err)
	}
	if _, ok := payload["data"]; !ok {
		t.Fatalf("json payload missing data: %s", stdout)
	}
	if _, ok := payload["warnings"]; !ok {
		t.Fatalf("json payload missing warnings: %s", stdout)
	}

	data := payload["data"].(map[string]any)
	snippet := data["snippet"].(map[string]any)
	if snippet["id"] != "python/retry" {
		t.Fatalf("snippet id = %v, want python/retry", snippet["id"])
	}
	if snippet["path"] != "python/retry.py" {
		t.Fatalf("snippet path = %v, want python/retry.py", snippet["path"])
	}

	stdout, stderr, err = runCLI(t, now, nil, "--json", "--config", configPath, "--vault", vaultPath, "search", "HELLO")
	if err != nil {
		t.Fatalf("search error: %v, stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "python/retry") {
		t.Fatalf("search output missing snippet id: %s", stdout)
	}

	stdout, stderr, err = runCLI(t, now, nil, "--json", "--config", configPath, "--vault", vaultPath, "config", "--show")
	if err != nil {
		t.Fatalf("config show error: %v, stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "\"sources\"") {
		t.Fatalf("config output missing sources: %s", stdout)
	}
}

func TestCLIWarningsAndNoopEdit(t *testing.T) {
	workspace := t.TempDir()
	vaultPath := filepath.Join(workspace, "vault")
	configPath := filepath.Join(workspace, "trove.toml")
	editorPath := writeEditorScript(t, workspace, "#!/bin/sh\nif [ ! -s \"$1\" ]; then printf 'echo hi\\n' > \"$1\"; fi\n")
	noOpEditor := writeEditorScript(t, workspace, "#!/bin/sh\nexit 0\n")

	t.Setenv("HOME", workspace)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(workspace, "xdg"))
	t.Setenv("TROVE_EDITOR", editorPath)

	now := func() time.Time { return time.Date(2026, 3, 14, 16, 0, 0, 0, time.UTC) }

	if _, stderr, err := runCLI(t, now, nil, "--config", configPath, "--vault", vaultPath, "init"); err != nil {
		t.Fatalf("init error: %v, stderr=%s", err, stderr)
	}
	if _, stderr, err := runCLI(t, now, nil, "--config", configPath, "--vault", vaultPath, "new", "script.sh"); err != nil {
		t.Fatalf("new error: %v, stderr=%s", err, stderr)
	}

	t.Setenv("TROVE_EDITOR", noOpEditor)
	stdout, stderr, err := runCLI(t, now, nil, "--json", "--config", configPath, "--vault", vaultPath, "edit", "shell/script")
	if err != nil {
		t.Fatalf("edit error: %v, stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "\"changed\": false") {
		t.Fatalf("expected noop edit response, got %s", stdout)
	}

	if err := os.WriteFile(filepath.Join(vaultPath, "shell", "orphan.toml"), []byte("description = 'oops'\ncreated = 2026-03-14T16:00:00Z\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, err = runCLI(t, now, nil, "--json", "--config", configPath, "--vault", vaultPath, "list")
	if err != nil {
		t.Fatalf("list error: %v, stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "\"code\": \"orphan_metadata\"") {
		t.Fatalf("expected orphan warning in json, got %s", stdout)
	}
}

func TestNewWithLangGeneratesScratchStyleName(t *testing.T) {
	workspace := t.TempDir()
	vaultPath := filepath.Join(workspace, "vault")
	configPath := filepath.Join(workspace, "trove.toml")
	editorPath := writeEditorScript(t, workspace, "#!/bin/sh\nprintf 'print(\"hello\")\\n' > \"$1\"\n")

	t.Setenv("HOME", workspace)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(workspace, "xdg"))
	t.Setenv("TROVE_EDITOR", editorPath)

	now := func() time.Time { return time.Date(2026, 3, 14, 18, 0, 0, 0, time.UTC) }

	if _, stderr, err := runCLI(t, now, nil, "--config", configPath, "--vault", vaultPath, "init"); err != nil {
		t.Fatalf("init error: %v, stderr=%s", err, stderr)
	}

	stdout, stderr, err := runCLI(t, now, nil, "--config", configPath, "--vault", vaultPath, "new", "--lang", "python")
	if err != nil {
		t.Fatalf("new --lang error: %v, stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "python/trove_1") {
		t.Fatalf("expected generated snippet id, got %s", stdout)
	}
	if !strings.Contains(stdout, "python/trove_1.py") {
		t.Fatalf("expected generated snippet path, got %s", stdout)
	}

	if _, stderr, err := runCLI(t, now, nil, "--config", configPath, "--vault", vaultPath, "new", "script.sh"); err != nil {
		t.Fatalf("named new error: %v, stderr=%s", err, stderr)
	}

	stdout, stderr, err = runCLI(t, now, nil, "--config", configPath, "--vault", vaultPath, "new", "--lang", "python")
	if err != nil {
		t.Fatalf("second new --lang error: %v, stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "python/trove_3") {
		t.Fatalf("expected generated snippet to use tracked count, got %s", stdout)
	}
	if !strings.Contains(stdout, "python/trove_3.py") {
		t.Fatalf("expected generated snippet path, got %s", stdout)
	}
}

func TestAddWithLangFromEditorBufferGeneratesScratchStyleName(t *testing.T) {
	workspace := t.TempDir()
	vaultPath := filepath.Join(workspace, "vault")
	configPath := filepath.Join(workspace, "trove.toml")
	editorPath := writeEditorScript(t, workspace, "#!/bin/sh\nprintf 'print(\"hello\")\\n' > \"$1\"\n")

	t.Setenv("HOME", workspace)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(workspace, "xdg"))
	t.Setenv("TROVE_EDITOR", editorPath)

	now := func() time.Time { return time.Date(2026, 3, 14, 18, 15, 0, 0, time.UTC) }

	if _, stderr, err := runCLI(t, now, nil, "--config", configPath, "--vault", vaultPath, "init"); err != nil {
		t.Fatalf("init error: %v, stderr=%s", err, stderr)
	}

	stdout, stderr, err := runCLI(t, now, nil, "--config", configPath, "--vault", vaultPath, "add", "--lang", "python")
	if err != nil {
		t.Fatalf("add --lang error: %v, stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "python/trove_1") {
		t.Fatalf("expected generated snippet id, got %s", stdout)
	}
	if !strings.Contains(stdout, "python/trove_1.py") {
		t.Fatalf("expected generated snippet path, got %s", stdout)
	}
}

func TestCdLaunchesShellInVault(t *testing.T) {
	workspace := t.TempDir()
	vaultPath := filepath.Join(workspace, "vault")
	configPath := filepath.Join(workspace, "trove.toml")
	shellPath := writeEditorScript(t, workspace, "#!/bin/sh\npwd\n")

	if err := os.MkdirAll(vaultPath, 0o755); err != nil {
		t.Fatal(err)
	}

	t.Setenv("HOME", workspace)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(workspace, "xdg"))
	t.Setenv("SHELL", shellPath)

	now := func() time.Time { return time.Date(2026, 3, 14, 18, 30, 0, 0, time.UTC) }

	stdout, stderr, err := runCLI(t, now, nil, "--config", configPath, "--vault", vaultPath, "cd")
	if err != nil {
		t.Fatalf("cd error: %v, stderr=%s", err, stderr)
	}

	got, err := filepath.EvalSymlinks(strings.TrimSpace(stdout))
	if err != nil {
		t.Fatalf("eval symlinks(stdout): %v", err)
	}
	want, err := filepath.EvalSymlinks(vaultPath)
	if err != nil {
		t.Fatalf("eval symlinks(vaultPath): %v", err)
	}
	if got != want {
		t.Fatalf("cd launched in %q, want %q", got, want)
	}
}

func TestCdJSONReturnsVaultPath(t *testing.T) {
	workspace := t.TempDir()
	vaultPath := filepath.Join(workspace, "vault")
	configPath := filepath.Join(workspace, "trove.toml")

	t.Setenv("HOME", workspace)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(workspace, "xdg"))

	now := func() time.Time { return time.Date(2026, 3, 14, 18, 45, 0, 0, time.UTC) }

	stdout, stderr, err := runCLI(t, now, nil, "--json", "--config", configPath, "--vault", vaultPath, "cd")
	if err != nil {
		t.Fatalf("cd --json error: %v, stderr=%s", err, stderr)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatal(err)
	}

	data := payload["data"].(map[string]any)
	if data["vault_path"] != vaultPath {
		t.Fatalf("vault path = %v, want %s", data["vault_path"], vaultPath)
	}
}

func TestGitUnavailableWarning(t *testing.T) {
	workspace := t.TempDir()
	vaultPath := filepath.Join(workspace, "vault")
	configPath := filepath.Join(workspace, "trove.toml")

	t.Setenv("HOME", workspace)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(workspace, "xdg"))
	t.Setenv("PATH", filepath.Join(workspace, "missing"))

	now := func() time.Time { return time.Date(2026, 3, 14, 17, 0, 0, 0, time.UTC) }

	_, stderr, err := runCLI(t, now, nil, "--config", configPath, "--vault", vaultPath, "init")
	if err != nil {
		t.Fatalf("init error: %v, stderr=%s", err, stderr)
	}
	if !strings.Contains(stderr, "git is not available") {
		t.Fatalf("expected git warning, got %s", stderr)
	}
}

func TestZshCompletionUsesDescribedCompaddByDefault(t *testing.T) {
	workspace := t.TempDir()

	t.Setenv("HOME", workspace)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(workspace, "xdg"))

	now := func() time.Time { return time.Date(2026, 3, 14, 17, 30, 0, 0, time.UTC) }

	stdout, stderr, err := runCLI(t, now, nil, "completion", "zsh")
	if err != nil {
		t.Fatalf("completion zsh error: %v, stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "__complete ${words[2,-1]}") {
		t.Fatalf("expected zsh completion to request described completions, got %s", stdout)
	}
	if !strings.Contains(stdout, "__trove_compadd_described_completions") {
		t.Fatalf("expected zsh completion to use custom compadd helper, got %s", stdout)
	}
}

func TestZshCompletionNoDescriptionsFlag(t *testing.T) {
	workspace := t.TempDir()

	t.Setenv("HOME", workspace)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(workspace, "xdg"))

	now := func() time.Time { return time.Date(2026, 3, 14, 17, 45, 0, 0, time.UTC) }

	stdout, stderr, err := runCLI(t, now, nil, "completion", "zsh", "--no-descriptions")
	if err != nil {
		t.Fatalf("completion zsh --no-descriptions error: %v, stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "__completeNoDesc") {
		t.Fatalf("expected zsh completion to use __completeNoDesc, got %s", stdout)
	}
	if strings.Contains(stdout, "__trove_compadd_described_completions") {
		t.Fatalf("expected no-description zsh completion to omit custom compadd helper, got %s", stdout)
	}
}

func TestEditPublicFlag(t *testing.T) {
	workspace := t.TempDir()
	vaultPath := filepath.Join(workspace, "vault")
	configPath := filepath.Join(workspace, "trove.toml")
	editorPath := writeEditorScript(t, workspace, "#!/bin/sh\nif [ ! -s \"$1\" ]; then printf 'echo hi\\n' > \"$1\"; fi\n")

	t.Setenv("HOME", workspace)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(workspace, "xdg"))
	t.Setenv("TROVE_EDITOR", editorPath)

	now := func() time.Time { return time.Date(2026, 3, 14, 19, 0, 0, 0, time.UTC) }

	if _, stderr, err := runCLI(t, now, nil, "--config", configPath, "--vault", vaultPath, "init"); err != nil {
		t.Fatalf("init error: %v, stderr=%s", err, stderr)
	}
	if _, stderr, err := runCLI(t, now, nil, "--config", configPath, "--vault", vaultPath, "new", "script.sh"); err != nil {
		t.Fatalf("new error: %v, stderr=%s", err, stderr)
	}

	// Mark as public via edit --public.
	if _, stderr, err := runCLI(t, now, nil, "--config", configPath, "--vault", vaultPath, "edit", "shell/script", "--public"); err != nil {
		t.Fatalf("edit --public error: %v, stderr=%s", err, stderr)
	}

	// Verify the sidecar now contains public = true.
	tomlBytes, err := os.ReadFile(filepath.Join(vaultPath, "shell", "script.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(tomlBytes), "public = true") {
		t.Fatalf("expected public = true in sidecar, got:\n%s", string(tomlBytes))
	}

	// Verify JSON output includes public field.
	stdout, stderr, err := runCLI(t, now, nil, "--json", "--config", configPath, "--vault", vaultPath, "show", "shell/script")
	if err != nil {
		t.Fatalf("show error: %v, stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "\"public\": true") {
		t.Fatalf("expected public: true in JSON output, got:\n%s", stdout)
	}
}

func TestNewAndAddWithPublicFlag(t *testing.T) {
	workspace := t.TempDir()
	vaultPath := filepath.Join(workspace, "vault")
	configPath := filepath.Join(workspace, "trove.toml")
	editorPath := writeEditorScript(t, workspace, "#!/bin/sh\nprintf 'echo hello\\n' > \"$1\"\n")

	t.Setenv("HOME", workspace)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(workspace, "xdg"))
	t.Setenv("TROVE_EDITOR", editorPath)

	now := func() time.Time { return time.Date(2026, 3, 14, 19, 15, 0, 0, time.UTC) }

	if _, stderr, err := runCLI(t, now, nil, "--config", configPath, "--vault", vaultPath, "init"); err != nil {
		t.Fatalf("init error: %v, stderr=%s", err, stderr)
	}

	// new --public
	if _, stderr, err := runCLI(t, now, nil, "--config", configPath, "--vault", vaultPath, "new", "pub.sh", "--public"); err != nil {
		t.Fatalf("new --public error: %v, stderr=%s", err, stderr)
	}
	tomlBytes, err := os.ReadFile(filepath.Join(vaultPath, "shell", "pub.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(tomlBytes), "public = true") {
		t.Fatalf("expected public = true after new --public, got:\n%s", string(tomlBytes))
	}

	// add --public with file
	srcFile := filepath.Join(workspace, "source.py")
	if err := os.WriteFile(srcFile, []byte("print('hi')\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, stderr, err := runCLI(t, now, nil, "--config", configPath, "--vault", vaultPath, "add", srcFile, "--public"); err != nil {
		t.Fatalf("add --public error: %v, stderr=%s", err, stderr)
	}
	tomlBytes, err = os.ReadFile(filepath.Join(vaultPath, "python", "source.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(tomlBytes), "public = true") {
		t.Fatalf("expected public = true after add --public, got:\n%s", string(tomlBytes))
	}
}

func TestPublishCommand(t *testing.T) {
	workspace := t.TempDir()
	vaultPath := filepath.Join(workspace, "vault")
	configPath := filepath.Join(workspace, "trove.toml")
	outputDir := filepath.Join(workspace, "site")
	editorPath := writeEditorScript(t, workspace, "#!/bin/sh\nprintf 'echo hello\\n' > \"$1\"\n")

	t.Setenv("HOME", workspace)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(workspace, "xdg"))
	t.Setenv("TROVE_EDITOR", editorPath)

	now := func() time.Time { return time.Date(2026, 3, 14, 19, 30, 0, 0, time.UTC) }

	if _, stderr, err := runCLI(t, now, nil, "--config", configPath, "--vault", vaultPath, "init"); err != nil {
		t.Fatalf("init error: %v, stderr=%s", err, stderr)
	}

	// Create a public snippet and a private one.
	if _, stderr, err := runCLI(t, now, nil, "--config", configPath, "--vault", vaultPath, "new", "pub.sh", "--public"); err != nil {
		t.Fatalf("new --public error: %v, stderr=%s", err, stderr)
	}
	if _, stderr, err := runCLI(t, now, nil, "--config", configPath, "--vault", vaultPath, "new", "priv.sh"); err != nil {
		t.Fatalf("new error: %v, stderr=%s", err, stderr)
	}

	// Publish.
	stdout, stderr, err := runCLI(t, now, nil, "--json", "--config", configPath, "--vault", vaultPath, "publish", "--output", outputDir)
	if err != nil {
		t.Fatalf("publish error: %v, stderr=%s", err, stderr)
	}

	// Verify JSON output.
	if !strings.Contains(stdout, "\"snippet_count\": 1") {
		t.Fatalf("expected 1 public snippet in output, got:\n%s", stdout)
	}

	// Verify HTML files.
	if _, err := os.Stat(filepath.Join(outputDir, "index.html")); err != nil {
		t.Fatalf("index.html not created: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outputDir, "shell", "pub.html")); err != nil {
		t.Fatalf("pub.html not created: %v", err)
	}
	// Private snippet should NOT have a page.
	if _, err := os.Stat(filepath.Join(outputDir, "shell", "priv.html")); err == nil {
		t.Fatal("priv.html should not exist for private snippet")
	}
}

func TestPublishNoPublicSnippets(t *testing.T) {
	workspace := t.TempDir()
	vaultPath := filepath.Join(workspace, "vault")
	configPath := filepath.Join(workspace, "trove.toml")
	outputDir := filepath.Join(workspace, "site")
	editorPath := writeEditorScript(t, workspace, "#!/bin/sh\nprintf 'echo hello\\n' > \"$1\"\n")

	t.Setenv("HOME", workspace)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(workspace, "xdg"))
	t.Setenv("TROVE_EDITOR", editorPath)

	now := func() time.Time { return time.Date(2026, 3, 14, 19, 45, 0, 0, time.UTC) }

	if _, stderr, err := runCLI(t, now, nil, "--config", configPath, "--vault", vaultPath, "init"); err != nil {
		t.Fatalf("init error: %v, stderr=%s", err, stderr)
	}
	if _, stderr, err := runCLI(t, now, nil, "--config", configPath, "--vault", vaultPath, "new", "priv.sh"); err != nil {
		t.Fatalf("new error: %v, stderr=%s", err, stderr)
	}

	stdout, stderr, err := runCLI(t, now, nil, "--json", "--config", configPath, "--vault", vaultPath, "publish", "--output", outputDir)
	if err != nil {
		t.Fatalf("publish error: %v, stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "\"snippet_count\": 0") {
		t.Fatalf("expected 0 snippets, got:\n%s", stdout)
	}

	// Index should still be created.
	if _, err := os.Stat(filepath.Join(outputDir, "index.html")); err != nil {
		t.Fatalf("index.html not created: %v", err)
	}
}

func runCLI(t *testing.T, now func() time.Time, stdin *strings.Reader, args ...string) (string, string, error) {
	t.Helper()
	if stdin == nil {
		stdin = strings.NewReader("")
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := cli.Execute(args, cli.Options{
		Stdin:  stdin,
		Stdout: &stdout,
		Stderr: &stderr,
		Now:    now,
	})
	return stdout.String(), stderr.String(), err
}

func writeEditorScript(t *testing.T, dir string, body string) string {
	t.Helper()
	path := filepath.Join(dir, strings.ReplaceAll(t.Name(), "/", "_")+"_"+randomSuffix(t)+".sh")
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func randomSuffix(t *testing.T) string {
	t.Helper()
	return strings.ReplaceAll(time.Now().UTC().Format("150405.000000"), ".", "")
}
