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

func TestCanonicalExtension(t *testing.T) {
	got, err := CanonicalExtension("shell")
	if err != nil {
		t.Fatal(err)
	}
	if got != ".sh" {
		t.Fatalf("CanonicalExtension(shell) = %q, want .sh", got)
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
