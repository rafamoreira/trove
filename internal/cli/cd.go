package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	goruntime "runtime"
	"strings"

	"github.com/spf13/cobra"
)

func newCdCmd(opts Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cd",
		Short: "Launch a shell in the vault directory",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := loadRuntime(cmd, opts)
			if err != nil {
				return err
			}

			if rt.json {
				return rt.emit(map[string]any{
					"vault_path": rt.vault.Path,
				}, nil, renderMap)
			}

			return launchShellInDir(rt.vault.Path, rt.stdin, rt.stdout, rt.stderr)
		},
	}

	return cmd
}

func launchShellInDir(dir string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	info, err := os.Stat(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("vault path does not exist: %s", dir)
		}
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("vault path is not a directory: %s", dir)
	}

	shell, err := detectShell()
	if err != nil {
		return err
	}

	cmd := exec.Command(shell)
	cmd.Dir = dir
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Env = append(os.Environ(), "TROVE_VAULT_PATH="+dir)

	return cmd.Run()
}

func detectShell() (string, error) {
	candidates := make([]string, 0, 4)
	if shell := strings.TrimSpace(os.Getenv("SHELL")); shell != "" {
		candidates = append(candidates, shell)
	}

	if goruntime.GOOS == "windows" {
		if shell := strings.TrimSpace(os.Getenv("COMSPEC")); shell != "" {
			candidates = append(candidates, shell)
		}
		candidates = append(candidates, "cmd.exe")
	} else {
		candidates = append(candidates, "/bin/sh", "sh")
	}

	for _, candidate := range candidates {
		path, err := exec.LookPath(candidate)
		if err == nil {
			return path, nil
		}
	}

	return "", errors.New("could not determine a shell to launch")
}
