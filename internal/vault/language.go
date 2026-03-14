package vault

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

var languageAliases = map[string]string{
	"bash":       "shell",
	"go":         "go",
	"golang":     "go",
	"javascript": "javascript",
	"js":         "javascript",
	"lua":        "lua",
	"plaintext":  "plaintext",
	"py":         "python",
	"python":     "python",
	"rb":         "ruby",
	"ruby":       "ruby",
	"rs":         "rust",
	"rust":       "rust",
	"sh":         "shell",
	"shell":      "shell",
	"sql":        "sql",
	"ts":         "typescript",
	"typescript": "typescript",
}

var languageExtensions = map[string]string{
	"go":         ".go",
	"javascript": ".js",
	"lua":        ".lua",
	"plaintext":  "",
	"python":     ".py",
	"ruby":       ".rb",
	"rust":       ".rs",
	"shell":      ".sh",
	"sql":        ".sql",
	"typescript": ".ts",
}

var extToLanguage = map[string]string{
	".bash": "shell",
	".go":   "go",
	".js":   "javascript",
	".lua":  "lua",
	".py":   "python",
	".rb":   "ruby",
	".rs":   "rust",
	".sh":   "shell",
	".sql":  "sql",
	".ts":   "typescript",
}

var invalidSepPattern = regexp.MustCompile(`[\\/]+`)
var nameSanitizePattern = regexp.MustCompile(`[^a-z0-9]+`)

func NormalizeLanguage(input string) (string, error) {
	value := strings.ToLower(strings.TrimSpace(input))
	if value == "" {
		return "", fmt.Errorf("language cannot be empty")
	}
	lang, ok := languageAliases[value]
	if !ok {
		return "", fmt.Errorf("unsupported language: %s", input)
	}
	return lang, nil
}

func DetectLanguageFromPath(path string) (string, bool) {
	ext := strings.ToLower(filepath.Ext(path))
	lang, ok := extToLanguage[ext]
	return lang, ok
}

func CanonicalExtension(language string) (string, error) {
	lang, err := NormalizeLanguage(language)
	if err != nil {
		return "", err
	}
	ext, ok := languageExtensions[lang]
	if !ok {
		return "", fmt.Errorf("no canonical extension for %s", lang)
	}
	return ext, nil
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
