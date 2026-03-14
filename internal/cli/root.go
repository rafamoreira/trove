package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/rafamoreira/trove/internal/config"
	"github.com/rafamoreira/trove/internal/diag"
	"github.com/rafamoreira/trove/internal/editor"
	"github.com/rafamoreira/trove/internal/search"
	"github.com/rafamoreira/trove/internal/vault"
)

type Options struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
	Now    func() time.Time
}

type outputEnvelope struct {
	Data     any            `json:"data"`
	Warnings []diag.Warning `json:"warnings"`
}

type runtime struct {
	cfg    *config.Config
	vault  *vault.Vault
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
	json   bool
	now    func() time.Time
}

func Execute(args []string, opts Options) error {
	cmd := NewRoot(opts)
	cmd.SetArgs(args)
	return cmd.Execute()
}

func NewRoot(opts Options) *cobra.Command {
	if opts.Stdin == nil {
		opts.Stdin = os.Stdin
	}
	if opts.Stdout == nil {
		opts.Stdout = os.Stdout
	}
	if opts.Stderr == nil {
		opts.Stderr = os.Stderr
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}

	root := &cobra.Command{
		Use:           "trove",
		Short:         "A local-first snippet vault",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.CompletionOptions.DisableDescriptions = true

	root.SetIn(opts.Stdin)
	root.SetOut(opts.Stdout)
	root.SetErr(opts.Stderr)
	root.PersistentFlags().String("vault", "", "override vault path")
	root.PersistentFlags().String("config", "", "override config path")
	root.PersistentFlags().Bool("json", false, "output machine-readable JSON")

	root.AddCommand(newInitCmd(opts))
	root.AddCommand(newNewCmd(opts))
	root.AddCommand(newAddCmd(opts))
	root.AddCommand(newEditCmd(opts))
	root.AddCommand(newShowCmd(opts))
	root.AddCommand(newListCmd(opts))
	root.AddCommand(newSearchCmd(opts))
	root.AddCommand(newRmCmd(opts))
	root.AddCommand(newSyncCmd(opts))
	root.AddCommand(newStatusCmd(opts))
	root.AddCommand(newConfigCmd(opts))

	return root
}

func newInitCmd(opts Options) *cobra.Command {
	var remote string

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a trove vault",
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := loadRuntime(cmd, opts)
			if err != nil {
				return err
			}
			warnings, err := rt.vault.Init(remote)
			if err != nil {
				return err
			}
			wroteConfig, err := rt.vault.EnsureConfigFile()
			if err != nil {
				return err
			}

			data := map[string]any{
				"vault_path":         rt.vault.Path,
				"config_path":        rt.cfg.FilePath,
				"config_created":     wroteConfig,
				"git_repo":           rt.vault.GitIsRepo(),
				"git_available":      vault.GitAvailable(),
				"default_git_remote": rt.cfg.GitRemote,
			}
			return rt.emit(data, warnings, renderMap)
		},
	}
	cmd.Flags().StringVar(&remote, "remote", "", "git remote URL")
	return cmd
}

func newNewCmd(opts Options) *cobra.Command {
	var lang string
	var desc string
	var tags string

	cmd := &cobra.Command{
		Use:   "new <name>",
		Short: "Create a new snippet",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := loadRuntime(cmd, opts)
			if err != nil {
				return err
			}

			name, language, err := resolveNameAndLanguage(args[0], lang, true)
			if err != nil {
				return err
			}

			snippet, err := rt.vault.CreateSnippet(language, name, nil, desc, splitCSV(tags))
			if err != nil {
				return err
			}
			if err := editor.Open(rt.cfg.Editor, snippet.Path, rt.stdin, rt.stdout, rt.stderr); err != nil {
				return err
			}
			body, err := os.ReadFile(snippet.Path)
			if err != nil {
				return err
			}
			if len(bytes.TrimSpace(body)) == 0 {
				if err := rt.vault.DeleteSnippet(snippet); err != nil {
					return err
				}
				return fmt.Errorf("snippet creation aborted: empty body")
			}

			warnings := rt.vault.CommitSnippet("add "+filepath.Base(filepath.Dir(snippet.Path))+"/"+filepath.Base(snippet.Path), relativeToVault(rt.vault.Path, snippet.Path), relativeToVault(rt.vault.Path, snippet.MetaPath))
			return rt.emit(snippetData(snippet), warnings, renderSnippetSummary)
		},
	}
	cmd.Flags().StringVar(&lang, "lang", "", "snippet language")
	cmd.Flags().StringVar(&desc, "desc", "", "snippet description")
	cmd.Flags().StringVar(&tags, "tags", "", "comma-separated tags")
	return cmd
}

func newAddCmd(opts Options) *cobra.Command {
	var name string
	var lang string
	var desc string
	var tags string

	cmd := &cobra.Command{
		Use:   "add [file]",
		Short: "Add an existing snippet",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := loadRuntime(cmd, opts)
			if err != nil {
				return err
			}

			var sourceFile string
			if len(args) == 1 {
				sourceFile = args[0]
			}

			if sourceFile == "" && strings.TrimSpace(lang) == "" {
				return fmt.Errorf("--lang is required when adding from an editor buffer")
			}
			if sourceFile == "" && strings.TrimSpace(name) == "" {
				return fmt.Errorf("--name is required when adding from an editor buffer")
			}

			var snippetName string
			var language string
			if sourceFile != "" {
				base := strings.TrimSuffix(filepath.Base(sourceFile), filepath.Ext(sourceFile))
				if strings.TrimSpace(name) == "" {
					name = base
				}
				snippetName, language, err = resolveNameAndLanguageWithSource(name, lang, sourceFile)
				if err != nil {
					return err
				}
				content, err := os.ReadFile(sourceFile)
				if err != nil {
					return err
				}
				snippet, err := rt.vault.CreateSnippet(language, snippetName, content, desc, splitCSV(tags))
				if err != nil {
					return err
				}
				warnings := rt.vault.CommitSnippet("add "+filepath.Base(filepath.Dir(snippet.Path))+"/"+filepath.Base(snippet.Path), relativeToVault(rt.vault.Path, snippet.Path), relativeToVault(rt.vault.Path, snippet.MetaPath))
				return rt.emit(snippetData(snippet), warnings, renderSnippetSummary)
			}

			snippetName, language, err = resolveNameAndLanguage(name, lang, false)
			if err != nil {
				return err
			}

			tempDir, err := os.MkdirTemp("", "trove-add-*")
			if err != nil {
				return err
			}
			defer os.RemoveAll(tempDir)

			ext, err := vault.CanonicalExtension(language)
			if err != nil {
				return err
			}
			tempPath := filepath.Join(tempDir, snippetName+ext)
			if err := os.WriteFile(tempPath, nil, 0o644); err != nil {
				return err
			}
			if err := editor.Open(rt.cfg.Editor, tempPath, rt.stdin, rt.stdout, rt.stderr); err != nil {
				return err
			}
			body, err := os.ReadFile(tempPath)
			if err != nil {
				return err
			}
			if len(bytes.TrimSpace(body)) == 0 {
				return fmt.Errorf("snippet creation aborted: empty body")
			}

			snippet, err := rt.vault.CreateSnippet(language, snippetName, body, desc, splitCSV(tags))
			if err != nil {
				return err
			}
			warnings := rt.vault.CommitSnippet("add "+filepath.Base(filepath.Dir(snippet.Path))+"/"+filepath.Base(snippet.Path), relativeToVault(rt.vault.Path, snippet.Path), relativeToVault(rt.vault.Path, snippet.MetaPath))
			return rt.emit(snippetData(snippet), warnings, renderSnippetSummary)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "snippet name")
	cmd.Flags().StringVar(&lang, "lang", "", "snippet language")
	cmd.Flags().StringVar(&desc, "desc", "", "snippet description")
	cmd.Flags().StringVar(&tags, "tags", "", "comma-separated tags")
	return cmd
}

func newEditCmd(opts Options) *cobra.Command {
	var desc string
	var tags string

	cmd := &cobra.Command{
		Use:   "edit <selector>",
		Short: "Edit an existing snippet",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := loadRuntime(cmd, opts)
			if err != nil {
				return err
			}

			snippet, warnings, err := rt.vault.Resolve(args[0])
			if err != nil {
				return err
			}

			descChanged := cmd.Flags().Changed("desc")
			tagsChanged := cmd.Flags().Changed("tags")
			if descChanged || tagsChanged {
				var descPtr *string
				var tagsPtr *[]string
				if descChanged {
					descCopy := desc
					descPtr = &descCopy
				}
				if tagsChanged {
					tagCopy := splitCSV(tags)
					tagsPtr = &tagCopy
				}
				if err := rt.vault.UpdateSnippet(snippet, descPtr, tagsPtr); err != nil {
					return err
				}
				warnings = append(warnings, rt.vault.CommitSnippet("edit "+filepath.Base(filepath.Dir(snippet.Path))+"/"+filepath.Base(snippet.Path), relativeToVault(rt.vault.Path, snippet.MetaPath))...)
				return rt.emit(snippetData(snippet), warnings, renderSnippetSummary)
			}

			before, err := os.ReadFile(snippet.Path)
			if err != nil {
				return err
			}
			if err := editor.Open(rt.cfg.Editor, snippet.Path, rt.stdin, rt.stdout, rt.stderr); err != nil {
				return err
			}
			after, err := os.ReadFile(snippet.Path)
			if err != nil {
				return err
			}
			if bytes.Equal(before, after) {
				return rt.emit(map[string]any{
					"id":      snippet.ID,
					"changed": false,
				}, warnings, renderMap)
			}
			warnings = append(warnings, rt.vault.CommitSnippet("edit "+filepath.Base(filepath.Dir(snippet.Path))+"/"+filepath.Base(snippet.Path), relativeToVault(rt.vault.Path, snippet.Path))...)
			return rt.emit(map[string]any{
				"id":      snippet.ID,
				"changed": true,
			}, warnings, renderMap)
		},
	}
	cmd.Flags().StringVar(&desc, "desc", "", "update snippet description")
	cmd.Flags().StringVar(&tags, "tags", "", "update snippet tags")
	return cmd
}

func newShowCmd(opts Options) *cobra.Command {
	var includeMeta bool

	cmd := &cobra.Command{
		Use:   "show <selector>",
		Short: "Show a snippet",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := loadRuntime(cmd, opts)
			if err != nil {
				return err
			}
			snippet, warnings, err := rt.vault.Resolve(args[0])
			if err != nil {
				return err
			}
			body, err := snippet.Body()
			if err != nil {
				return err
			}
			data := map[string]any{
				"snippet": snippetData(snippet),
				"body":    body,
			}
			return rt.emit(data, warnings, func(w io.Writer, value any) error {
				payload := value.(map[string]any)
				if includeMeta {
					s := payload["snippet"].(map[string]any)
					fmt.Fprintf(w, "ID: %s\nLanguage: %s\nDescription: %s\nTags: %s\nCreated: %s\n\n",
						s["id"], s["language"], s["description"], strings.Join(anyToStringSlice(s["tags"]), ", "), s["created"])
				}
				_, err := fmt.Fprint(w, payload["body"].(string))
				return err
			})
		},
	}
	cmd.Flags().BoolVar(&includeMeta, "meta", false, "include metadata header")
	return cmd
}

func newListCmd(opts Options) *cobra.Command {
	var lang string
	var tag string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List snippets",
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := loadRuntime(cmd, opts)
			if err != nil {
				return err
			}
			items, warnings, err := rt.vault.List(vault.ListOptions{
				Language: lang,
				Tag:      tag,
			})
			if err != nil {
				return err
			}
			data := make([]map[string]any, 0, len(items))
			for _, item := range items {
				data = append(data, snippetData(item))
			}
			return rt.emit(data, warnings, renderSnippetTable)
		},
	}
	cmd.Flags().StringVar(&lang, "lang", "", "filter by language")
	cmd.Flags().StringVar(&tag, "tag", "", "filter by tag")
	return cmd
}

func newSearchCmd(opts Options) *cobra.Command {
	var lang string
	var tag string

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search snippets",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := loadRuntime(cmd, opts)
			if err != nil {
				return err
			}
			results, warnings, err := search.SearchVault(rt.vault, args[0], vault.ListOptions{
				Language: lang,
				Tag:      tag,
			})
			if err != nil {
				return err
			}
			payload := make([]map[string]any, 0, len(results))
			for _, result := range results {
				payload = append(payload, map[string]any{
					"snippet": snippetData(result.Snippet),
					"matches": result.Matches,
				})
			}
			return rt.emit(payload, warnings, renderSearchResults)
		},
	}
	cmd.Flags().StringVar(&lang, "lang", "", "filter by language")
	cmd.Flags().StringVar(&tag, "tag", "", "filter by tag")
	return cmd
}

func newRmCmd(opts Options) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "rm <selector>",
		Short: "Remove a snippet",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := loadRuntime(cmd, opts)
			if err != nil {
				return err
			}
			snippet, warnings, err := rt.vault.Resolve(args[0])
			if err != nil {
				return err
			}
			if !force {
				fmt.Fprintf(rt.stderr, "Remove %s? [y/N]: ", snippet.ID)
				var answer string
				if _, err := fmt.Fscanln(rt.stdin, &answer); err != nil && err != io.EOF {
					return err
				}
				answer = strings.ToLower(strings.TrimSpace(answer))
				if answer != "y" && answer != "yes" {
					return fmt.Errorf("removal aborted")
				}
			}
			if err := rt.vault.DeleteSnippet(snippet); err != nil {
				return err
			}
			warnings = append(warnings, rt.vault.CommitSnippet("remove "+filepath.Base(filepath.Dir(snippet.Path))+"/"+filepath.Base(snippet.Path), relativeToVault(rt.vault.Path, snippet.Path), relativeToVault(rt.vault.Path, snippet.MetaPath))...)
			return rt.emit(map[string]any{
				"id":      snippet.ID,
				"removed": true,
			}, warnings, renderMap)
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "remove without confirmation")
	return cmd
}

func newSyncCmd(opts Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync the vault",
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := loadRuntime(cmd, opts)
			if err != nil {
				return err
			}
			committed, warnings, err := rt.vault.SyncNow()
			if err != nil {
				return err
			}
			return rt.emit(map[string]any{
				"committed": committed,
				"pushed":    committed && rt.cfg.AutoPush && strings.TrimSpace(rt.cfg.GitRemote) != "",
			}, warnings, renderMap)
		},
	}
	return cmd
}

func newStatusCmd(opts Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show vault status",
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := loadRuntime(cmd, opts)
			if err != nil {
				return err
			}

			data := map[string]any{
				"git_available": vault.GitAvailable(),
				"git_repo":      rt.vault.GitIsRepo(),
				"pending_files": []string{},
			}
			var warnings []diag.Warning
			if vault.GitAvailable() && rt.vault.GitIsRepo() {
				status, err := rt.vault.GitStatus()
				if err != nil {
					return err
				}
				data["pending_files"] = status
				logs, err := rt.vault.GitLog(1)
				if err != nil {
					return err
				}
				if len(logs) > 0 {
					data["last_commit"] = logs[0]
				}
			} else {
				warnings = append(warnings, diag.Warning{
					Code:    "git_unavailable",
					Message: "git status is unavailable until git is installed and the vault is initialized as a repository",
				})
			}
			return rt.emit(data, warnings, renderMap)
		},
	}
	return cmd
}

func newConfigCmd(opts Options) *cobra.Command {
	var show bool
	var edit bool

	cmd := &cobra.Command{
		Use:   "config",
		Short: "Inspect or edit trove config",
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := loadRuntime(cmd, opts)
			if err != nil {
				return err
			}
			if edit {
				if err := config.EnsureParent(rt.cfg.FilePath); err != nil {
					return err
				}
				if _, err := os.Stat(rt.cfg.FilePath); os.IsNotExist(err) {
					content, err := config.DefaultFileContents(rt.cfg.VaultPath)
					if err != nil {
						return err
					}
					if err := os.WriteFile(rt.cfg.FilePath, content, 0o644); err != nil {
						return err
					}
				}
				return editor.Open(rt.cfg.Editor, rt.cfg.FilePath, rt.stdin, rt.stdout, rt.stderr)
			}
			if !show {
				show = true
			}
			if show {
				return rt.emit(rt.cfg.Display(), nil, renderConfig)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&show, "show", false, "show resolved config")
	cmd.Flags().BoolVar(&edit, "edit", false, "edit config file")
	return cmd
}

func loadRuntime(cmd *cobra.Command, opts Options) (*runtime, error) {
	configPath, err := cmd.Flags().GetString("config")
	if err != nil {
		return nil, err
	}
	vaultOverride, err := cmd.Flags().GetString("vault")
	if err != nil {
		return nil, err
	}
	jsonOutput, err := cmd.Flags().GetBool("json")
	if err != nil {
		return nil, err
	}

	var overrides config.Overrides
	if cmd.Flags().Changed("vault") {
		overrides.VaultPath = &vaultOverride
	}

	cfg, err := config.Load(configPath, overrides)
	if err != nil {
		return nil, err
	}
	v := vault.New(cfg)
	v.Now = opts.Now

	return &runtime{
		cfg:    cfg,
		vault:  v,
		stdin:  opts.Stdin,
		stdout: opts.Stdout,
		stderr: opts.Stderr,
		json:   jsonOutput,
		now:    opts.Now,
	}, nil
}

func (rt *runtime) emit(data any, warnings []diag.Warning, render func(io.Writer, any) error) error {
	if rt.json {
		payload := outputEnvelope{Data: data, Warnings: warnings}
		enc := json.NewEncoder(rt.stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(payload)
	}
	if render != nil {
		if err := render(rt.stdout, data); err != nil {
			return err
		}
	}
	for _, warning := range warnings {
		if warning.Path != "" {
			fmt.Fprintf(rt.stderr, "warning: %s (%s)\n", warning.Message, warning.Path)
			continue
		}
		fmt.Fprintf(rt.stderr, "warning: %s\n", warning.Message)
	}
	return nil
}

func resolveNameAndLanguage(input string, explicitLang string, allowFilename bool) (string, string, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", "", fmt.Errorf("name cannot be empty")
	}

	stem := input
	detectedLang := ""
	if allowFilename {
		var ext string
		stem, ext = vault.SplitFilenameInput(input)
		if ext != "" {
			lang, ok := vault.DetectLanguageFromPath(input)
			if !ok {
				return "", "", fmt.Errorf("unsupported file extension: %s", ext)
			}
			detectedLang = lang
		}
	}

	name, err := vault.NormalizeName(stem)
	if err != nil {
		return "", "", err
	}

	if strings.TrimSpace(explicitLang) != "" {
		lang, err := vault.NormalizeLanguage(explicitLang)
		if err != nil {
			return "", "", err
		}
		if detectedLang != "" && detectedLang != lang {
			return "", "", fmt.Errorf("language flag conflicts with detected file extension")
		}
		return name, lang, nil
	}
	if detectedLang != "" {
		return name, detectedLang, nil
	}
	return name, "plaintext", nil
}

func resolveNameAndLanguageWithSource(input string, explicitLang string, sourcePath string) (string, string, error) {
	name, err := vault.NormalizeName(input)
	if err != nil {
		return "", "", err
	}

	detectedLang := ""
	if lang, ok := vault.DetectLanguageFromPath(sourcePath); ok {
		detectedLang = lang
	}

	if strings.TrimSpace(explicitLang) != "" {
		lang, err := vault.NormalizeLanguage(explicitLang)
		if err != nil {
			return "", "", err
		}
		if detectedLang != "" && detectedLang != lang {
			return "", "", fmt.Errorf("language flag conflicts with detected file extension")
		}
		return name, lang, nil
	}

	if detectedLang == "" {
		return "", "", fmt.Errorf("could not detect language from source file; use --lang")
	}
	return name, detectedLang, nil
}

func splitCSV(input string) []string {
	if strings.TrimSpace(input) == "" {
		return nil
	}
	parts := strings.Split(input, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value == "" {
			continue
		}
		out = append(out, value)
	}
	return out
}

func snippetData(snippet *vault.Snippet) map[string]any {
	created := ""
	if !snippet.Created.IsZero() {
		created = snippet.Created.UTC().Format(time.RFC3339)
	}
	return map[string]any{
		"id":          snippet.ID,
		"name":        snippet.Name,
		"language":    snippet.Language,
		"path":        relativeToVault(filepath.Dir(filepath.Dir(snippet.Path)), snippet.Path),
		"meta_path":   relativeToVault(filepath.Dir(filepath.Dir(snippet.Path)), snippet.MetaPath),
		"description": snippet.Description,
		"tags":        snippet.Tags,
		"created":     created,
	}
}

func relativeToVault(vaultPath string, path string) string {
	relative, err := filepath.Rel(vaultPath, path)
	if err != nil {
		return path
	}
	return relative
}

func renderSnippetSummary(w io.Writer, value any) error {
	item := value.(map[string]any)
	_, err := fmt.Fprintf(w, "%s\t%s\n", item["id"], item["path"])
	return err
}

func renderSnippetTable(w io.Writer, value any) error {
	rows := value.([]map[string]any)
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tLANGUAGE\tDESCRIPTION\tTAGS")
	for _, row := range rows {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", row["id"], row["language"], row["description"], strings.Join(anyToStringSlice(row["tags"]), ", "))
	}
	return tw.Flush()
}

func renderSearchResults(w io.Writer, value any) error {
	rows := value.([]map[string]any)
	for _, row := range rows {
		snippet := row["snippet"].(map[string]any)
		fmt.Fprintf(w, "%s\n", snippet["id"])
		matches := row["matches"].([]vault.SearchMatch)
		for _, match := range matches {
			if match.Line > 0 {
				fmt.Fprintf(w, "  %d: %s\n", match.Line, match.Context)
			} else {
				fmt.Fprintf(w, "  %s\n", match.Context)
			}
		}
	}
	return nil
}

func renderConfig(w io.Writer, value any) error {
	display := value.(config.Display)
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	fmt.Fprintf(tw, "PATH\t%s\n", display.Path)
	keys := []string{
		config.FieldVaultPath,
		config.FieldEditor,
		config.FieldGitRemote,
		config.FieldGitBranch,
		config.FieldAutoSync,
		config.FieldSyncDebounceSeconds,
		config.FieldAutoPush,
	}
	for _, key := range keys {
		fmt.Fprintf(tw, "%s\t%v\t(%s)\n", key, display.Values[key], display.Source[key])
	}
	return tw.Flush()
}

func renderMap(w io.Writer, value any) error {
	m := value.(map[string]any)
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sortStrings(keys)
	for _, key := range keys {
		fmt.Fprintf(tw, "%s\t%v\n", key, m[key])
	}
	return tw.Flush()
}

func anyToStringSlice(value any) []string {
	if value == nil {
		return nil
	}
	switch typed := value.(type) {
	case []string:
		return typed
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			out = append(out, fmt.Sprint(item))
		}
		return out
	default:
		return []string{fmt.Sprint(value)}
	}
}

func sortStrings(items []string) {
	if len(items) < 2 {
		return
	}
	for i := 0; i < len(items)-1; i++ {
		for j := i + 1; j < len(items); j++ {
			if items[j] < items[i] {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
}
