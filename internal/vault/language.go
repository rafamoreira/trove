package vault

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	enry "github.com/go-enry/go-enry/v2"
)

// disambiguations overrides enry for extensions that are ambiguous in Linguist
// but have an obvious default for a snippet manager.
var disambiguations = map[string]string{
	".rs":  "Rust",
	".sql": "SQL",
	".ts":  "TypeScript",
}

var invalidSepPattern = regexp.MustCompile(`[\\/]+`)
var nameSanitizePattern = regexp.MustCompile(`[^a-z0-9]+`)

// NormalizeLanguage resolves a language name or alias to a canonical lowercase
// form suitable for use as a directory name. It accepts full language names
// (e.g., "python"), common aliases (e.g., "py"), and file extensions without
// dots (e.g., "rs").
func NormalizeLanguage(input string) (string, error) {
	value := strings.ToLower(strings.TrimSpace(input))
	if value == "" {
		return "", fmt.Errorf("language cannot be empty")
	}

	if value == "plaintext" || value == "text" || value == "txt" {
		return "plaintext", nil
	}
	if value == "prompt" {
		return "prompt", nil
	}

	// Try alias lookup (handles "js"→"JavaScript", "go"→"Go", etc.)
	if lang, ok := enry.GetLanguageByAlias(value); ok {
		return strings.ToLower(lang), nil
	}

	// Check disambiguation map (handles "rs", "sql", "ts" as bare input)
	if lang, ok := disambiguations["."+value]; ok {
		return strings.ToLower(lang), nil
	}

	// Try treating input as a file extension (handles "py", etc.)
	if lang, ok := enry.GetLanguageByExtension("file." + value); ok {
		return strings.ToLower(lang), nil
	}

	return "", fmt.Errorf("unsupported language: %s", input)
}

// DetectLanguageFromPath returns the language for a file path based on its
// extension. It returns false if the extension is unrecognized.
func DetectLanguageFromPath(path string) (string, bool) {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == "" {
		return "", false
	}

	if ext == ".prompt" {
		return "prompt", true
	}

	// Check disambiguations first for known ambiguous extensions
	if lang, ok := disambiguations[ext]; ok {
		return strings.ToLower(lang), true
	}

	// Use enry extension detection (unambiguous matches)
	if lang, ok := enry.GetLanguageByExtension(path); ok {
		return strings.ToLower(lang), true
	}

	// For ambiguous extensions, try alias lookup on the extension without dot
	if lang, ok := enry.GetLanguageByAlias(ext[1:]); ok {
		return strings.ToLower(lang), true
	}

	return "", false
}

// CanonicalExtension returns the primary file extension for a language
// (e.g., "python" → ".py").
func CanonicalExtension(language string) (string, error) {
	lang, err := NormalizeLanguage(language)
	if err != nil {
		return "", err
	}
	if lang == "plaintext" {
		return "", nil
	}
	if lang == "prompt" {
		return ".prompt", nil
	}

	// enry needs title-case canonical names; resolve via alias lookup
	canonical, ok := enry.GetLanguageByAlias(lang)
	if !ok {
		return "", fmt.Errorf("no canonical extension for %s", lang)
	}
	if exts := enry.GetLanguageExtensions(canonical); len(exts) > 0 {
		return exts[0], nil
	}
	return "", fmt.Errorf("no canonical extension for %s", lang)
}

// LanguageExtensions returns the known file extensions for a language.
// The language should be a normalized lowercase name.
func LanguageExtensions(language string) []string {
	if language == "prompt" {
		return []string{".prompt"}
	}
	if canonical, ok := enry.GetLanguageByAlias(language); ok {
		return enry.GetLanguageExtensions(canonical)
	}
	return nil
}

func NormalizeName(input string) (string, error) {
	value := strings.TrimSpace(input)
	if value == "" {
		return "", fmt.Errorf("name cannot be empty")
	}
	if invalidSepPattern.MatchString(value) {
		return "", fmt.Errorf("name cannot contain path separators")
	}
	value = strings.ToLower(value)
	value = nameSanitizePattern.ReplaceAllString(value, "_")
	value = strings.Trim(value, "_")
	if value == "" {
		return "", fmt.Errorf("name cannot be empty after normalization")
	}
	return value, nil
}

func SplitFilenameInput(input string) (string, string) {
	base := filepath.Base(strings.TrimSpace(input))
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	return stem, ext
}
