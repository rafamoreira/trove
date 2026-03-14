package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
)

const (
	FieldVaultPath            = "vault_path"
	FieldEditor               = "editor"
	FieldGitRemote            = "git_remote"
	FieldGitBranch            = "git_branch"
	FieldAutoSync             = "auto_sync"
	FieldSyncDebounceSeconds  = "sync_debounce_seconds"
	FieldAutoPush             = "auto_push"
	defaultConfigRelativePath = ".config/trove/config.toml"
	defaultVaultRelativePath  = ".local/share/trove/vault"
)

type Source string

const (
	SourceDefault Source = "default"
	SourceFile    Source = "file"
	SourceEnv     Source = "env"
	SourceFlag    Source = "flag"
)

type Config struct {
	VaultPath           string            `json:"vault_path"`
	Editor              string            `json:"editor"`
	GitRemote           string            `json:"git_remote"`
	GitBranch           string            `json:"git_branch"`
	AutoSync            bool              `json:"auto_sync"`
	SyncDebounceSeconds int               `json:"sync_debounce_seconds"`
	AutoPush            bool              `json:"auto_push"`
	FilePath            string            `json:"file_path"`
	Sources             map[string]Source `json:"sources"`
}

type Overrides struct {
	VaultPath           *string
	Editor              *string
	GitRemote           *string
	GitBranch           *string
	AutoSync            *bool
	SyncDebounceSeconds *int
	AutoPush            *bool
}

type fileConfig struct {
	VaultPath           *string `toml:"vault_path"`
	Editor              *string `toml:"editor"`
	GitRemote           *string `toml:"git_remote"`
	GitBranch           *string `toml:"git_branch"`
	AutoSync            *bool   `toml:"auto_sync"`
	SyncDebounceSeconds *int    `toml:"sync_debounce_seconds"`
	AutoPush            *bool   `toml:"auto_push"`
}

type Display struct {
	Path   string            `json:"path"`
	Values map[string]any    `json:"values"`
	Source map[string]Source `json:"sources"`
}

func Default() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	editor := strings.TrimSpace(os.Getenv("VISUAL"))
	if editor == "" {
		editor = strings.TrimSpace(os.Getenv("EDITOR"))
	}
	if editor == "" {
		editor = "vi"
	}

	cfg := &Config{
		VaultPath:           filepath.Join(home, defaultVaultRelativePath),
		Editor:              editor,
		GitRemote:           "origin",
		GitBranch:           "main",
		AutoSync:            true,
		SyncDebounceSeconds: 5,
		AutoPush:            true,
		Sources: map[string]Source{
			FieldVaultPath:           SourceDefault,
			FieldEditor:              SourceDefault,
			FieldGitRemote:           SourceDefault,
			FieldGitBranch:           SourceDefault,
			FieldAutoSync:            SourceDefault,
			FieldSyncDebounceSeconds: SourceDefault,
			FieldAutoPush:            SourceDefault,
		},
	}

	path, err := Path("")
	if err != nil {
		return nil, err
	}
	cfg.FilePath = path

	return cfg, nil
}

func Path(override string) (string, error) {
	if strings.TrimSpace(override) != "" {
		return ExpandPath(override)
	}

	if xdg := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME")); xdg != "" {
		return filepath.Join(xdg, "trove", "config.toml"), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, defaultConfigRelativePath), nil
}

func ExpandPath(input string) (string, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", nil
	}
	if input == "~" || strings.HasPrefix(input, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if input == "~" {
			return home, nil
		}
		return filepath.Join(home, input[2:]), nil
	}
	return filepath.Clean(input), nil
}

func Load(configPath string, overrides Overrides) (*Config, error) {
	cfg, err := Default()
	if err != nil {
		return nil, err
	}

	path, err := Path(configPath)
	if err != nil {
		return nil, err
	}
	cfg.FilePath = path

	if err := applyFile(cfg, path); err != nil {
		return nil, err
	}
	applyEnv(cfg)
	applyOverrides(cfg, overrides)

	cfg.VaultPath, err = ExpandPath(cfg.VaultPath)
	if err != nil {
		return nil, err
	}
	cfg.FilePath, err = ExpandPath(cfg.FilePath)
	if err != nil {
		return nil, err
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	if strings.TrimSpace(c.VaultPath) == "" {
		return errors.New("vault path cannot be empty")
	}
	if strings.TrimSpace(c.Editor) == "" {
		return errors.New("editor cannot be empty")
	}
	if strings.TrimSpace(c.GitBranch) == "" {
		return errors.New("git branch cannot be empty")
	}
	if c.SyncDebounceSeconds <= 0 {
		return errors.New("sync debounce seconds must be greater than zero")
	}
	return nil
}

func (c *Config) Display() Display {
	return Display{
		Path: c.FilePath,
		Values: map[string]any{
			FieldVaultPath:           c.VaultPath,
			FieldEditor:              c.Editor,
			FieldGitRemote:           c.GitRemote,
			FieldGitBranch:           c.GitBranch,
			FieldAutoSync:            c.AutoSync,
			FieldSyncDebounceSeconds: c.SyncDebounceSeconds,
			FieldAutoPush:            c.AutoPush,
		},
		Source: c.Sources,
	}
}

func EnsureParent(path string) error {
	return os.MkdirAll(filepath.Dir(path), 0o755)
}

func DefaultFileContents(vaultPath string) ([]byte, error) {
	expanded, err := ExpandPath(vaultPath)
	if err != nil {
		return nil, err
	}

	fc := fileConfig{
		VaultPath:           &expanded,
		Editor:              stringPtr("vi"),
		GitRemote:           stringPtr("origin"),
		GitBranch:           stringPtr("main"),
		AutoSync:            boolPtr(true),
		SyncDebounceSeconds: intPtr(5),
		AutoPush:            boolPtr(true),
	}

	var buf bytes.Buffer
	enc := toml.NewEncoder(&buf)
	if err := enc.Encode(fc); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func applyFile(cfg *Config, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}

	var fc fileConfig
	if _, err := toml.Decode(string(data), &fc); err != nil {
		return fmt.Errorf("parse config: %w", err)
	}

	if fc.VaultPath != nil {
		cfg.VaultPath = *fc.VaultPath
		cfg.Sources[FieldVaultPath] = SourceFile
	}
	if fc.Editor != nil {
		cfg.Editor = strings.TrimSpace(*fc.Editor)
		cfg.Sources[FieldEditor] = SourceFile
	}
	if fc.GitRemote != nil {
		cfg.GitRemote = strings.TrimSpace(*fc.GitRemote)
		cfg.Sources[FieldGitRemote] = SourceFile
	}
	if fc.GitBranch != nil {
		cfg.GitBranch = strings.TrimSpace(*fc.GitBranch)
		cfg.Sources[FieldGitBranch] = SourceFile
	}
	if fc.AutoSync != nil {
		cfg.AutoSync = *fc.AutoSync
		cfg.Sources[FieldAutoSync] = SourceFile
	}
	if fc.SyncDebounceSeconds != nil {
		cfg.SyncDebounceSeconds = *fc.SyncDebounceSeconds
		cfg.Sources[FieldSyncDebounceSeconds] = SourceFile
	}
	if fc.AutoPush != nil {
		cfg.AutoPush = *fc.AutoPush
		cfg.Sources[FieldAutoPush] = SourceFile
	}
	return nil
}

func applyEnv(cfg *Config) {
	applyEnvString(cfg, "TROVE_VAULT_PATH", FieldVaultPath, func(v string) { cfg.VaultPath = v })
	applyEnvString(cfg, "TROVE_EDITOR", FieldEditor, func(v string) { cfg.Editor = v })
	applyEnvString(cfg, "TROVE_GIT_REMOTE", FieldGitRemote, func(v string) { cfg.GitRemote = v })
	applyEnvString(cfg, "TROVE_GIT_BRANCH", FieldGitBranch, func(v string) { cfg.GitBranch = v })
	applyEnvBool(cfg, "TROVE_AUTO_SYNC", FieldAutoSync, func(v bool) { cfg.AutoSync = v })
	applyEnvInt(cfg, "TROVE_SYNC_DEBOUNCE_SECONDS", FieldSyncDebounceSeconds, func(v int) { cfg.SyncDebounceSeconds = v })
	applyEnvBool(cfg, "TROVE_AUTO_PUSH", FieldAutoPush, func(v bool) { cfg.AutoPush = v })
}

func applyOverrides(cfg *Config, overrides Overrides) {
	if overrides.VaultPath != nil {
		cfg.VaultPath = strings.TrimSpace(*overrides.VaultPath)
		cfg.Sources[FieldVaultPath] = SourceFlag
	}
	if overrides.Editor != nil {
		cfg.Editor = strings.TrimSpace(*overrides.Editor)
		cfg.Sources[FieldEditor] = SourceFlag
	}
	if overrides.GitRemote != nil {
		cfg.GitRemote = strings.TrimSpace(*overrides.GitRemote)
		cfg.Sources[FieldGitRemote] = SourceFlag
	}
	if overrides.GitBranch != nil {
		cfg.GitBranch = strings.TrimSpace(*overrides.GitBranch)
		cfg.Sources[FieldGitBranch] = SourceFlag
	}
	if overrides.AutoSync != nil {
		cfg.AutoSync = *overrides.AutoSync
		cfg.Sources[FieldAutoSync] = SourceFlag
	}
	if overrides.SyncDebounceSeconds != nil {
		cfg.SyncDebounceSeconds = *overrides.SyncDebounceSeconds
		cfg.Sources[FieldSyncDebounceSeconds] = SourceFlag
	}
	if overrides.AutoPush != nil {
		cfg.AutoPush = *overrides.AutoPush
		cfg.Sources[FieldAutoPush] = SourceFlag
	}
}

func applyEnvString(cfg *Config, key string, field string, apply func(string)) {
	if value, ok := os.LookupEnv(key); ok {
		apply(strings.TrimSpace(value))
		cfg.Sources[field] = SourceEnv
	}
}

func applyEnvBool(cfg *Config, key string, field string, apply func(bool)) {
	value, ok := os.LookupEnv(key)
	if !ok {
		return
	}
	parsed, err := strconv.ParseBool(strings.TrimSpace(value))
	if err != nil {
		return
	}
	apply(parsed)
	cfg.Sources[field] = SourceEnv
}

func applyEnvInt(cfg *Config, key string, field string, apply func(int)) {
	value, ok := os.LookupEnv(key)
	if !ok {
		return
	}
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return
	}
	apply(parsed)
	cfg.Sources[field] = SourceEnv
}

func stringPtr(v string) *string { return &v }
func boolPtr(v bool) *bool       { return &v }
func intPtr(v int) *int          { return &v }
