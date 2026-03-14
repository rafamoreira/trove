package vault

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var (
	ErrGitUnavailable = errors.New("git unavailable")
	ErrNotGitRepo     = errors.New("vault is not a git repository")
)

type Commit struct {
	Hash      string    `json:"hash"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

func GitAvailable() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

func (v *Vault) GitInit() error {
	if !GitAvailable() {
		return ErrGitUnavailable
	}
	_, err := v.git("init")
	return err
}

func (v *Vault) GitIsRepo() bool {
	if !GitAvailable() {
		return false
	}
	_, err := os.Stat(filepath.Join(v.Path, ".git"))
	return err == nil
}

func (v *Vault) GitAdd(paths ...string) error {
	if !GitAvailable() {
		return ErrGitUnavailable
	}
	if !v.GitIsRepo() {
		return ErrNotGitRepo
	}
	args := append([]string{"add"}, paths...)
	_, err := v.git(args...)
	return err
}

func (v *Vault) GitAddAll() error {
	if !GitAvailable() {
		return ErrGitUnavailable
	}
	if !v.GitIsRepo() {
		return ErrNotGitRepo
	}
	_, err := v.git("add", "-A")
	return err
}

func (v *Vault) GitCommit(message string) (bool, error) {
	if !GitAvailable() {
		return false, ErrGitUnavailable
	}
	if !v.GitIsRepo() {
		return false, ErrNotGitRepo
	}

	cmd := exec.Command("git", "diff", "--cached", "--quiet", "--exit-code")
	cmd.Dir = v.Path
	if err := cmd.Run(); err == nil {
		return false, nil
	}

	_, err := v.git("commit", "-m", message)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (v *Vault) GitPush(remote string, branch string) error {
	if !GitAvailable() {
		return ErrGitUnavailable
	}
	if !v.GitIsRepo() {
		return ErrNotGitRepo
	}
	if strings.TrimSpace(remote) == "" {
		return nil
	}
	_, err := v.git("push", remote, branch)
	return err
}

func (v *Vault) GitPull(remote string, branch string) error {
	if !GitAvailable() {
		return ErrGitUnavailable
	}
	if !v.GitIsRepo() {
		return ErrNotGitRepo
	}
	if strings.TrimSpace(remote) == "" {
		return nil
	}
	_, err := v.git("pull", "--ff-only", remote, branch)
	return err
}

func (v *Vault) GitStatus() ([]string, error) {
	if !GitAvailable() {
		return nil, ErrGitUnavailable
	}
	if !v.GitIsRepo() {
		return nil, ErrNotGitRepo
	}
	out, err := v.git("status", "--short")
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return nil, nil
	}
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		result = append(result, strings.TrimSpace(line))
	}
	return result, nil
}

func (v *Vault) GitLog(n int) ([]Commit, error) {
	if !GitAvailable() {
		return nil, ErrGitUnavailable
	}
	if !v.GitIsRepo() {
		return nil, ErrNotGitRepo
	}
	if n <= 0 {
		n = 1
	}
	format := "%H%x1f%s%x1f%cI"
	out, err := v.git("log", fmt.Sprintf("-%d", n), "--format="+format)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(out), "\n")
	commits := make([]Commit, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.Split(line, "\x1f")
		if len(parts) != 3 {
			continue
		}
		ts, err := time.Parse(time.RFC3339, parts[2])
		if err != nil {
			return nil, err
		}
		commits = append(commits, Commit{
			Hash:      parts[0],
			Message:   parts[1],
			Timestamp: ts,
		})
	}
	return commits, nil
}

func (v *Vault) git(args ...string) (string, error) {
	var stderr bytes.Buffer
	cmd := exec.Command("git", args...)
	cmd.Dir = v.Path
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = err.Error()
		}
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), message)
	}
	return string(out), nil
}
