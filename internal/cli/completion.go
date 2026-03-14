package cli

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
)

const zshDescribeHook = `__%[1]s_debug "Calling _describe"
        if eval _describe $keepOrder "completions" completions $flagPrefix $noSpace; then`

const zshCompaddHelper = `
__%[1]s_compadd_described_completions()
{
    local rawComp completionChoice completionDescription
    local -a completionChoices completionDisplay compaddArgs

    for rawComp in "${completions[@]}"; do
        completionChoice="${rawComp%%%%:*}"
        completionDescription="${rawComp#*:}"
        if [ "${completionChoice}" = "${rawComp}" ]; then
            completionDescription=""
        fi

        completionChoice="${completionChoice//\\:/:}"
        completionDescription="${completionDescription//\\:/:}"
        completionChoices+=("${completionChoice}")
        if [ -n "${completionDescription}" ]; then
            completionDisplay+=("${completionChoice} -- ${completionDescription}")
        else
            completionDisplay+=("${completionChoice}")
        fi
    done

    if [ ${#completionChoices} -eq 0 ]; then
        return 1
    fi

    if [ -n "${keepOrder}" ]; then
        compaddArgs+=(${(z)keepOrder})
    fi
    if [ -n "${flagPrefix}" ]; then
        compaddArgs+=(${(z)flagPrefix})
    fi
    if [ -n "${noSpace}" ]; then
        compaddArgs+=(${(z)noSpace})
    fi

    __%[1]s_debug "Calling compadd for described completions"
    compadd "${compaddArgs[@]}" -d completionDisplay -a completionChoices
}
`

func newCompletionCmd() *cobra.Command {
	const shortDesc = "Generate the autocompletion script for the %s shell"
	const rootName = "trove"

	completionCmd := &cobra.Command{
		Use:                   "completion",
		Short:                 "Generate shell completion scripts",
		Long:                  "Generate shell completion scripts for bash, zsh, fish, or PowerShell.",
		Args:                  cobra.NoArgs,
		DisableFlagsInUseLine: true,
		ValidArgsFunction:     cobra.NoFileCompletions,
	}

	completionCmd.AddCommand(newCompletionShellCmd(
		"bash",
		fmt.Sprintf(shortDesc, "bash"),
		fmt.Sprintf(`Generate the autocompletion script for the bash shell.

To load completions in your current shell session:

	source <(%[1]s completion bash)

To load completions for every new session, execute once:

#### Linux:

	%[1]s completion bash > /etc/bash_completion.d/%[1]s

#### macOS:

	%[1]s completion bash > $(brew --prefix)/etc/bash_completion.d/%[1]s
`, rootName),
		func(root *cobra.Command, out io.Writer, noDesc bool) error {
			return root.GenBashCompletionV2(out, !noDesc)
		},
	))
	completionCmd.AddCommand(newCompletionShellCmd(
		"zsh",
		fmt.Sprintf(shortDesc, "zsh"),
		fmt.Sprintf(`Generate the autocompletion script for the zsh shell.

If shell completion is not already enabled in your environment you will need
to enable it. You can execute the following once:

	echo "autoload -U compinit; compinit" >> ~/.zshrc

To load completions in your current shell session:

	source <(%[1]s completion zsh)

To load completions for every new session, execute once:

#### Linux:

	%[1]s completion zsh > "${fpath[1]}/_%[1]s"

#### macOS:

	%[1]s completion zsh > $(brew --prefix)/share/zsh/site-functions/_%[1]s
`, rootName),
		func(root *cobra.Command, out io.Writer, noDesc bool) error {
			return generateZshCompletion(root, out, noDesc)
		},
	))
	completionCmd.AddCommand(newCompletionShellCmd(
		"fish",
		fmt.Sprintf(shortDesc, "fish"),
		fmt.Sprintf(`Generate the autocompletion script for the fish shell.

To load completions in your current shell session:

	%[1]s completion fish | source

To load completions for every new session, execute once:

	%[1]s completion fish > ~/.config/fish/completions/%[1]s.fish
`, rootName),
		func(root *cobra.Command, out io.Writer, noDesc bool) error {
			return root.GenFishCompletion(out, !noDesc)
		},
	))
	completionCmd.AddCommand(newCompletionShellCmd(
		"powershell",
		fmt.Sprintf(shortDesc, "powershell"),
		`Generate the autocompletion script for PowerShell.

To load completions in your current shell session:

	trove completion powershell | Out-String | Invoke-Expression
`,
		func(root *cobra.Command, out io.Writer, noDesc bool) error {
			if noDesc {
				return root.GenPowerShellCompletion(out)
			}
			return root.GenPowerShellCompletionWithDesc(out)
		},
	))

	return completionCmd
}

func newCompletionShellCmd(use string, short string, long string, run func(root *cobra.Command, out io.Writer, noDesc bool) error) *cobra.Command {
	var noDesc bool

	cmd := &cobra.Command{
		Use:                   use,
		Short:                 short,
		Long:                  long,
		Args:                  cobra.NoArgs,
		DisableFlagsInUseLine: true,
		ValidArgsFunction:     cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cmd.Root(), cmd.OutOrStdout(), noDesc)
		},
	}
	cmd.Flags().BoolVar(&noDesc, "no-descriptions", false, "disable completion descriptions")

	return cmd
}

func generateZshCompletion(root *cobra.Command, out io.Writer, noDesc bool) error {
	if noDesc {
		return root.GenZshCompletionNoDesc(out)
	}

	var buf bytes.Buffer
	if err := root.GenZshCompletion(&buf); err != nil {
		return err
	}

	script, err := patchZshCompletionScript(root.Name(), buf.String())
	if err != nil {
		return err
	}
	_, err = io.WriteString(out, script)
	return err
}

func patchZshCompletionScript(name string, script string) (string, error) {
	insertionPoint := fmt.Sprintf("}\n\n_%s()\n{", name)
	helper := fmt.Sprintf(zshCompaddHelper, name)
	if !strings.Contains(script, insertionPoint) {
		return "", fmt.Errorf("unexpected zsh completion format: missing helper insertion point")
	}
	script = strings.Replace(script, insertionPoint, "}\n"+helper+"\n_"+name+"()\n{", 1)

	describeHook := fmt.Sprintf(zshDescribeHook, name)
	replacement := fmt.Sprintf(`__%[1]s_debug "Calling compadd"
        if __%[1]s_compadd_described_completions; then`, name)
	if !strings.Contains(script, describeHook) {
		return "", fmt.Errorf("unexpected zsh completion format: missing describe hook")
	}
	script = strings.Replace(script, describeHook, replacement, 1)

	return script, nil
}
