package vault

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/rafamoreira/trove/internal/config"
	"github.com/rafamoreira/trove/internal/diag"
)

type Vault struct {
	Path   string
	Config *config.Config
	Now    func() time.Time
}

type ListOptions struct {
	Language string
	Tag      string
}

type SearchMatch struct {
	Line    int    `json:"line"`
	Context string `json:"context"`
}

type SearchResult struct {
	Snippet *Snippet      `json:"snippet"`
	Matches []SearchMatch `json:"matches"`
}

func New(cfg *config.Config) *Vault {
	return &Vault{
		Path:   cfg.VaultPath,
		Config: cfg,
		Now:    time.Now,
	}
}

func (v *Vault) Init(remote string) ([]diag.Warning, error) {
	if err := os.MkdirAll(v.Path, 0o755); err != nil {
		return nil, err
	}

	if err := os.WriteFile(filepath.Join(v.Path, ".gitignore"), []byte(".DS_Store\n"), 0o644); err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(v.Path, "README.md"), []byte("# Trove Vault\n"), 0o644); err != nil {
		return nil, err
	}

	var warnings []diag.Warning
	if GitAvailable() {
		if !v.GitIsRepo() {
			if err := v.GitInit(); err != nil {
				return nil, err
			}
		}
		if strings.TrimSpace(remote) != "" {
			if _, err := v.git("remote", "add", v.Config.GitRemote, remote); err != nil && !strings.Contains(err.Error(), "already exists") {
				return nil, err
			}
		}
	} else {
		warnings = append(warnings, diag.Warning{
			Code:    "git_unavailable",
			Message: "git is not available; trove will work locally without sync features",
		})
	}

	return warnings, nil
}

func (v *Vault) EnsureConfigFile() (bool, error) {
	if _, err := os.Stat(v.Config.FilePath); err == nil {
		return false, nil
	} else if err != nil && !os.IsNotExist(err) {
		return false, err
	}

	if err := config.EnsureParent(v.Config.FilePath); err != nil {
		return false, err
	}
	content, err := config.DefaultFileContents(v.Path)
	if err != nil {
		return false, err
	}
	if err := os.WriteFile(v.Config.FilePath, content, 0o644); err != nil {
		return false, err
	}
	return true, nil
}

func (v *Vault) CodePath(language string, name string) (string, error) {
	ext, err := CanonicalExtension(language)
	if err != nil {
		return "", err
	}
	filename := name + ext
	return filepath.Join(v.Path, language, filename), nil
}

func (v *Vault) MetaPath(language string, name string) string {
	return filepath.Join(v.Path, language, name+".toml")
}

func (v *Vault) CreateSnippet(language string, name string, body []byte, description string, tags []string) (*Snippet, error) {
	codePath, err := v.CodePath(language, name)
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(codePath); err == nil {
		return nil, fmt.Errorf("snippet already exists: %s", LogicalID(language, name))
	} else if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	if err := os.MkdirAll(filepath.Dir(codePath), 0o755); err != nil {
		return nil, err
	}

	if err := os.WriteFile(codePath, body, 0o644); err != nil {
		return nil, err
	}

	snippet := &Snippet{
		ID:          LogicalID(language, name),
		Name:        name,
		Language:    language,
		Path:        codePath,
		MetaPath:    v.MetaPath(language, name),
		Description: strings.TrimSpace(description),
		Tags:        NormalizeTags(tags),
		Created:     v.Now().UTC(),
	}

	if err := snippet.SaveMeta(); err != nil {
		return nil, err
	}

	return snippet, nil
}

func (v *Vault) NextGeneratedName(language string) (string, error) {
	lang, err := NormalizeLanguage(language)
	if err != nil {
		return "", err
	}

	snippets, _, err := v.List(ListOptions{})
	if err != nil {
		return "", err
	}

	next := len(snippets) + 1
	for {
		name := fmt.Sprintf("trove_%d", next)
		codePath, err := v.CodePath(lang, name)
		if err != nil {
			return "", err
		}
		if _, err := os.Stat(codePath); os.IsNotExist(err) {
			return name, nil
		} else if err != nil {
			return "", err
		}
		next++
	}
}

func (v *Vault) DeleteSnippet(snippet *Snippet) error {
	if err := os.Remove(snippet.Path); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := os.Remove(snippet.MetaPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (v *Vault) List(opts ListOptions) ([]*Snippet, []diag.Warning, error) {
	var snippets []*Snippet
	var warnings []diag.Warning

	filterLang := ""
	if strings.TrimSpace(opts.Language) != "" {
		lang, err := NormalizeLanguage(opts.Language)
		if err != nil {
			return nil, nil, err
		}
		filterLang = lang
	}

	err := filepath.WalkDir(v.Path, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if path == filepath.Join(v.Path, ".gitignore") || path == filepath.Join(v.Path, "README.md") {
			return nil
		}

		relative, err := filepath.Rel(v.Path, path)
		if err != nil {
			return err
		}
		if strings.HasPrefix(relative, ".git"+string(filepath.Separator)) {
			return nil
		}

		if filepath.Ext(path) == ".toml" {
			base := strings.TrimSuffix(path, ".toml")
			if _, err := os.Stat(base); err == nil {
				return nil
			}
			// Check extensions for the language indicated by the parent directory
			langDir := filepath.Base(filepath.Dir(path))
			for _, ext := range LanguageExtensions(langDir) {
				if _, err := os.Stat(base + ext); err == nil {
					return nil
				}
			}
			warnings = append(warnings, diag.Warning{
				Code:    "orphan_metadata",
				Message: "metadata sidecar has no matching code file",
				Path:    relative,
			})
			return nil
		}

		metaPath := MetaPathForCodePath(path)
		if _, err := os.Stat(metaPath); err != nil {
			if os.IsNotExist(err) {
				warnings = append(warnings, diag.Warning{
					Code:    "orphan_code",
					Message: "code file has no matching metadata sidecar",
					Path:    relative,
				})
				return nil
			}
			return err
		}

		snippet, err := LoadSnippet(v.Path, path)
		if err != nil {
			return err
		}
		if filterLang != "" && snippet.Language != filterLang {
			return nil
		}
		if !HasTag(snippet.Tags, opts.Tag) {
			return nil
		}
		snippets = append(snippets, snippet)
		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	sort.Slice(snippets, func(i int, j int) bool {
		return snippets[i].ID < snippets[j].ID
	})

	return snippets, warnings, nil
}

func (v *Vault) Resolve(selector string) (*Snippet, []diag.Warning, error) {
	items, warnings, err := v.List(ListOptions{})
	if err != nil {
		return nil, nil, err
	}

	selector = strings.TrimSpace(selector)
	if selector == "" {
		return nil, warnings, fmt.Errorf("selector cannot be empty")
	}

	if strings.Contains(selector, "/") {
		parts := strings.Split(selector, "/")
		if len(parts) != 2 {
			return nil, warnings, fmt.Errorf("selector must be <lang>/<name>")
		}
		lang, err := NormalizeLanguage(parts[0])
		if err != nil {
			return nil, warnings, err
		}
		name, err := NormalizeName(parts[1])
		if err != nil {
			return nil, warnings, err
		}
		id := LogicalID(lang, name)
		for _, item := range items {
			if item.ID == id {
				return item, warnings, nil
			}
		}
		return nil, warnings, fmt.Errorf("snippet not found: %s", id)
	}

	name, err := NormalizeName(selector)
	if err != nil {
		return nil, warnings, err
	}
	var matches []*Snippet
	for _, item := range items {
		if item.Name == name {
			matches = append(matches, item)
		}
	}
	if len(matches) == 0 {
		return nil, warnings, fmt.Errorf("snippet not found: %s", name)
	}
	if len(matches) > 1 {
		return nil, warnings, fmt.Errorf("snippet name is ambiguous: %s", name)
	}
	return matches[0], warnings, nil
}

func (v *Vault) UpdateSnippet(snippet *Snippet, description *string, tags *[]string) error {
	if description != nil {
		snippet.Description = strings.TrimSpace(*description)
	}
	if tags != nil {
		snippet.Tags = NormalizeTags(*tags)
	}
	return snippet.SaveMeta()
}

func (v *Vault) SyncNow() (bool, bool, []diag.Warning, error) {
	if !GitAvailable() {
		return false, false, []diag.Warning{{
			Code:    "git_unavailable",
			Message: "git is not available; trove will work locally without sync features",
		}}, nil
	}
	if !v.GitIsRepo() {
		return false, false, []diag.Warning{{
			Code:    "git_unavailable",
			Message: "vault is not a git repository; trove will work locally without sync features",
		}}, nil
	}

	if err := v.GitAddAll(); err != nil {
		return false, false, nil, err
	}
	committed, err := v.GitCommit("sync " + v.Now().UTC().Format(time.RFC3339))
	if err != nil {
		return false, false, nil, err
	}

	var warnings []diag.Warning
	pushed := false
	if v.Config.AutoPush && strings.TrimSpace(v.Config.GitRemote) != "" {
		if err := v.GitPush(v.Config.GitRemote, v.Config.GitBranch); err != nil {
			warnings = append(warnings, diag.Warning{
				Code:    "git_push_failed",
				Message: err.Error(),
			})
		} else {
			pushed = true
		}
	}
	return committed, pushed, warnings, nil
}

func (v *Vault) CommitSnippet(message string, paths ...string) []diag.Warning {
	if !GitAvailable() {
		return []diag.Warning{{
			Code:    "git_unavailable",
			Message: "git is not available; trove will work locally without sync features",
		}}
	}
	if !v.GitIsRepo() {
		return []diag.Warning{{
			Code:    "git_unavailable",
			Message: "vault is not a git repository; trove will work locally without sync features",
		}}
	}
	if err := v.GitAdd(paths...); err != nil {
		return []diag.Warning{{
			Code:    "git_add_failed",
			Message: err.Error(),
		}}
	}
	if _, err := v.GitCommit(message); err != nil {
		return []diag.Warning{{
			Code:    "git_commit_failed",
			Message: err.Error(),
		}}
	}
	return nil
}

func CopyFile(dst string, src string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}
