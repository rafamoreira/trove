package vault

import "testing"

func TestNormalizeLanguageAlias(t *testing.T) {
	got, err := NormalizeLanguage("js")
	if err != nil {
		t.Fatal(err)
	}
	if got != "javascript" {
		t.Fatalf("NormalizeLanguage(js) = %q, want javascript", got)
	}
}

func TestNormalizeLanguageFullName(t *testing.T) {
	got, err := NormalizeLanguage("Python")
	if err != nil {
		t.Fatal(err)
	}
	if got != "python" {
		t.Fatalf("NormalizeLanguage(Python) = %q, want python", got)
	}
}

func TestNormalizeLanguageExtensionFallback(t *testing.T) {
	// "py" is not a recognized alias in enry, but works via extension fallback
	got, err := NormalizeLanguage("py")
	if err != nil {
		t.Fatal(err)
	}
	if got != "python" {
		t.Fatalf("NormalizeLanguage(py) = %q, want python", got)
	}
}

func TestNormalizeLanguageDisambiguation(t *testing.T) {
	got, err := NormalizeLanguage("rs")
	if err != nil {
		t.Fatal(err)
	}
	if got != "rust" {
		t.Fatalf("NormalizeLanguage(rs) = %q, want rust", got)
	}
}

func TestNormalizeLanguagePlaintext(t *testing.T) {
	for _, input := range []string{"plaintext", "text", "txt"} {
		got, err := NormalizeLanguage(input)
		if err != nil {
			t.Fatalf("NormalizeLanguage(%q): %v", input, err)
		}
		if got != "plaintext" {
			t.Fatalf("NormalizeLanguage(%q) = %q, want plaintext", input, got)
		}
	}
}

func TestNormalizeLanguageNewLanguages(t *testing.T) {
	// These were unsupported with the old hardcoded list
	cases := map[string]string{
		"java":   "java",
		"c++":    "c++",
		"cpp":    "c++",
		"kotlin": "kotlin",
		"swift":  "swift",
	}
	for input, want := range cases {
		got, err := NormalizeLanguage(input)
		if err != nil {
			t.Fatalf("NormalizeLanguage(%q): %v", input, err)
		}
		if got != want {
			t.Fatalf("NormalizeLanguage(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestCanonicalExtension(t *testing.T) {
	got, err := CanonicalExtension("shell")
	if err != nil {
		t.Fatal(err)
	}
	if got != ".sh" {
		t.Fatalf("CanonicalExtension(shell) = %q, want .sh", got)
	}
}

func TestCanonicalExtensionNewLanguages(t *testing.T) {
	cases := map[string]string{
		"java":   ".java",
		"c++":    ".cpp",
		"kotlin": ".kt",
	}
	for input, want := range cases {
		got, err := CanonicalExtension(input)
		if err != nil {
			t.Fatalf("CanonicalExtension(%q): %v", input, err)
		}
		if got != want {
			t.Fatalf("CanonicalExtension(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestDetectLanguageFromPath(t *testing.T) {
	cases := map[string]string{
		"foo.go":   "go",
		"foo.py":   "python",
		"foo.rs":   "rust",
		"foo.ts":   "typescript",
		"foo.sql":  "sql",
		"foo.java": "java",
		"foo.cpp":  "c++",
	}
	for path, want := range cases {
		got, ok := DetectLanguageFromPath(path)
		if !ok {
			t.Fatalf("DetectLanguageFromPath(%q) not found", path)
		}
		if got != want {
			t.Fatalf("DetectLanguageFromPath(%q) = %q, want %q", path, got, want)
		}
	}
}

func TestNormalizeName(t *testing.T) {
	got, err := NormalizeName("Retry Decorator!")
	if err != nil {
		t.Fatal(err)
	}
	if got != "retry_decorator" {
		t.Fatalf("NormalizeName = %q, want retry_decorator", got)
	}
}
