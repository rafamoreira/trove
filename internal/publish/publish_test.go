package publish

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rafamoreira/trove/internal/vault"
)

func createTestSnippet(t *testing.T, dir, lang, name, body, desc string, public bool) *vault.Snippet {
	t.Helper()
	langDir := filepath.Join(dir, lang)
	if err := os.MkdirAll(langDir, 0o755); err != nil {
		t.Fatal(err)
	}
	ext := ".txt"
	switch lang {
	case "go":
		ext = ".go"
	case "python":
		ext = ".py"
	case "shell":
		ext = ".sh"
	}
	codePath := filepath.Join(langDir, name+ext)
	if err := os.WriteFile(codePath, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	snippet := &vault.Snippet{
		ID:          lang + "/" + name,
		Name:        name,
		Language:    lang,
		Path:        codePath,
		MetaPath:    filepath.Join(langDir, name+".toml"),
		Description: desc,
		Public:      public,
		Created:     time.Date(2026, 3, 14, 12, 0, 0, 0, time.UTC),
	}
	if err := snippet.SaveMeta(); err != nil {
		t.Fatal(err)
	}
	return snippet
}

func TestGenerateEmptySnippets(t *testing.T) {
	outputDir := filepath.Join(t.TempDir(), "site")
	result, err := Generate(nil, outputDir)
	if err != nil {
		t.Fatal(err)
	}
	if result["snippet_count"] != 0 {
		t.Fatalf("expected 0 snippets, got %v", result["snippet_count"])
	}

	indexBytes, err := os.ReadFile(filepath.Join(outputDir, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	index := string(indexBytes)
	if !strings.Contains(index, "<h1>") {
		t.Fatal("expected <h1> in index.html")
	}
}

func TestGenerateCreatesIndexAndSnippetPages(t *testing.T) {
	vaultDir := t.TempDir()
	outputDir := filepath.Join(t.TempDir(), "site")

	s1 := createTestSnippet(t, vaultDir, "go", "hello", "package main\n", "hello world", true)
	s2 := createTestSnippet(t, vaultDir, "python", "retry", "print('hi')\n", "retry helper", true)

	result, err := Generate([]*vault.Snippet{s1, s2}, outputDir)
	if err != nil {
		t.Fatal(err)
	}
	if result["snippet_count"] != 2 {
		t.Fatalf("expected 2 snippets, got %v", result["snippet_count"])
	}

	// Check index.html has links.
	indexBytes, err := os.ReadFile(filepath.Join(outputDir, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	index := string(indexBytes)
	if !strings.Contains(index, "go/hello.html") {
		t.Fatalf("index missing link to go/hello.html:\n%s", index)
	}
	if !strings.Contains(index, "python/retry.html") {
		t.Fatalf("index missing link to python/retry.html:\n%s", index)
	}

	// Check snippet pages exist and have <pre> content.
	goPage, err := os.ReadFile(filepath.Join(outputDir, "go", "hello.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(goPage), "<pre>package main") {
		t.Fatalf("go snippet page missing <pre> body:\n%s", string(goPage))
	}

	pyPage, err := os.ReadFile(filepath.Join(outputDir, "python", "retry.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(pyPage), "<pre>print(&#39;hi&#39;)") {
		t.Fatalf("python snippet page missing <pre> body:\n%s", string(pyPage))
	}
}

func TestGenerateHTMLEscapesBody(t *testing.T) {
	vaultDir := t.TempDir()
	outputDir := filepath.Join(t.TempDir(), "site")

	s := createTestSnippet(t, vaultDir, "go", "xss", "<script>alert('xss')</script>\n", "xss test", true)

	if _, err := Generate([]*vault.Snippet{s}, outputDir); err != nil {
		t.Fatal(err)
	}

	page, err := os.ReadFile(filepath.Join(outputDir, "go", "xss.html"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(page)
	if strings.Contains(content, "<script>") {
		t.Fatalf("expected HTML-escaped script tag, got raw script:\n%s", content)
	}
	if !strings.Contains(content, "&lt;script&gt;") {
		t.Fatalf("expected &lt;script&gt; in escaped output:\n%s", content)
	}
}

func TestGenerateGroupsByLanguage(t *testing.T) {
	vaultDir := t.TempDir()
	outputDir := filepath.Join(t.TempDir(), "site")

	s1 := createTestSnippet(t, vaultDir, "go", "a", "package a\n", "", true)
	s2 := createTestSnippet(t, vaultDir, "go", "b", "package b\n", "", true)
	s3 := createTestSnippet(t, vaultDir, "python", "c", "pass\n", "", true)

	if _, err := Generate([]*vault.Snippet{s1, s2, s3}, outputDir); err != nil {
		t.Fatal(err)
	}

	indexBytes, err := os.ReadFile(filepath.Join(outputDir, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	index := string(indexBytes)

	// Check that language headings appear.
	if !strings.Contains(index, ">go<") {
		t.Fatalf("expected go heading in index:\n%s", index)
	}
	if !strings.Contains(index, ">python<") {
		t.Fatalf("expected python heading in index:\n%s", index)
	}
}
