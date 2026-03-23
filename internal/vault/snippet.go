package vault

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

type Snippet struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Language    string    `json:"language"`
	Path        string    `json:"path"`
	MetaPath    string    `json:"meta_path"`
	Description string    `json:"description"`
	Tags        []string  `json:"tags"`
	Public      bool      `json:"public"`
	Created     time.Time `json:"created"`
}

type sidecar struct {
	Description string    `toml:"description"`
	Tags        []string  `toml:"tags"`
	Public      bool      `toml:"public"`
	Created     time.Time `toml:"created"`
}

func (s *Snippet) Body() (string, error) {
	data, err := os.ReadFile(s.Path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (s *Snippet) SaveMeta() error {
	meta := sidecar{
		Description: s.Description,
		Tags:        append([]string(nil), s.Tags...),
		Public:      s.Public,
		Created:     s.Created.UTC(),
	}

	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(meta); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(s.MetaPath), 0o755); err != nil {
		return err
	}

	return os.WriteFile(s.MetaPath, buf.Bytes(), 0o644)
}

func LoadSnippet(vaultPath string, codePath string) (*Snippet, error) {
	relative, err := filepath.Rel(vaultPath, codePath)
	if err != nil {
		return nil, err
	}

	lang, name, err := SnippetIdentityFromRelativePath(relative)
	if err != nil {
		return nil, err
	}

	metaPath := MetaPathForCodePath(codePath)
	data, err := os.ReadFile(metaPath)
	if err != nil {
		return nil, err
	}

	var meta sidecar
	if _, err := toml.Decode(string(data), &meta); err != nil {
		return nil, fmt.Errorf("parse sidecar %s: %w", metaPath, err)
	}

	return &Snippet{
		ID:          LogicalID(lang, name),
		Name:        name,
		Language:    lang,
		Path:        codePath,
		MetaPath:    metaPath,
		Description: meta.Description,
		Tags:        append([]string(nil), meta.Tags...),
		Public:      meta.Public,
		Created:     meta.Created.UTC(),
	}, nil
}

func MetaPathForCodePath(codePath string) string {
	dir := filepath.Dir(codePath)
	base := filepath.Base(codePath)
	ext := filepath.Ext(base)
	if ext != "" {
		base = strings.TrimSuffix(base, ext)
	}
	return filepath.Join(dir, base+".toml")
}

func SnippetIdentityFromRelativePath(relative string) (string, string, error) {
	dir := filepath.Dir(relative)
	if dir == "." {
		return "", "", fmt.Errorf("invalid snippet path: %s", relative)
	}
	lang, err := NormalizeLanguage(filepath.Base(dir))
	if err != nil {
		return "", "", fmt.Errorf("invalid snippet language path %s: %w", relative, err)
	}
	base := filepath.Base(relative)
	ext := filepath.Ext(base)
	if ext != "" {
		base = strings.TrimSuffix(base, ext)
	}
	name, err := NormalizeName(base)
	if err != nil {
		return "", "", err
	}
	return lang, name, nil
}

func NormalizeTags(tags []string) []string {
	out := make([]string, 0, len(tags))
	for _, tag := range tags {
		value := strings.TrimSpace(tag)
		if value == "" {
			continue
		}
		out = append(out, value)
	}
	return out
}

func LowerTags(tags []string) []string {
	out := make([]string, 0, len(tags))
	for _, tag := range tags {
		value := strings.ToLower(strings.TrimSpace(tag))
		if value == "" {
			continue
		}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func HasTag(tags []string, filter string) bool {
	filter = strings.ToLower(strings.TrimSpace(filter))
	if filter == "" {
		return true
	}
	for _, tag := range tags {
		if strings.EqualFold(strings.TrimSpace(tag), filter) {
			return true
		}
	}
	return false
}

func LogicalID(language string, name string) string {
	return language + "/" + name
}
