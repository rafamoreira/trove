package editor

import (
	"fmt"
	"io"
	"os/exec"
	"strings"
)

func Open(command string, path string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	command = strings.TrimSpace(command)
	if command == "" {
		return fmt.Errorf("editor command cannot be empty")
	}

	parts := strings.Fields(command)
	cmd := exec.Command(parts[0], append(parts[1:], path)...)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	return cmd.Run()
}
