package ansible

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/autobrr/brrewery/internal/packages/extravars"
)

var ErrPlaybookNotFound = errors.New("ansible-playbook not found")

const ansibleRuntimeDir = "/tmp/brrewery-ansible-home"

type RunRequest struct {
	PlaybookPath string
	ExtraVars    map[string]string
	OnOutput     func(line string)
}

type Runner struct {
	ansibleRoot string
	lookPath    func(string) (string, error)
}

func NewRunner(ansibleRoot string) *Runner {
	return &Runner{
		ansibleRoot: ansibleRoot,
		lookPath:    exec.LookPath,
	}
}

func (r *Runner) Run(ctx context.Context, req RunRequest) error {
	if strings.TrimSpace(req.PlaybookPath) == "" {
		return errors.New("playbook path is required")
	}

	ansiblePath, err := r.lookPath("ansible-playbook")
	if err != nil {
		return ErrPlaybookNotFound
	}

	inventory := filepath.Join(r.ansibleRoot, "inventory/localhost.yml")
	args := []string{
		req.PlaybookPath,
		"-i", inventory,
		"--connection=local",
	}

	// Keep the sudo password off the command line and out of the extra-vars
	// JSON: write it to a private temp file consumed via --become-password-file.
	extraVars := req.ExtraVars
	if becomePassword := strings.TrimSpace(extraVars[extravars.BecomePassword]); becomePassword != "" {
		passFile, cleanup, fileErr := writeBecomePasswordFile(becomePassword)
		if fileErr != nil {
			return fileErr
		}
		defer cleanup()
		args = append(args, "--become-password-file", passFile)
		extraVars = withoutKey(extraVars, extravars.BecomePassword)
	}

	if len(extraVars) > 0 {
		payload, marshalErr := json.Marshal(extraVars)
		if marshalErr != nil {
			return fmt.Errorf("encode extra vars: %w", marshalErr)
		}
		args = append(args, "-e", string(payload))
	}

	cmd := exec.CommandContext(ctx, ansiblePath, args...)
	cmd.Dir = r.ansibleRoot
	if err := os.MkdirAll(filepath.Join(ansibleRuntimeDir, ".ansible"), 0o750); err != nil {
		return fmt.Errorf("prepare ansible runtime dir: %w", err)
	}
	cmd.Env = append(os.Environ(),
		"HOME="+ansibleRuntimeDir,
		"ANSIBLE_HOME="+filepath.Join(ansibleRuntimeDir, ".ansible"),
		"ANSIBLE_LOCAL_TEMP=/tmp/brrewery-ansible",
		"ANSIBLE_REMOTE_TEMP=/tmp/brrewery-ansible",
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start ansible-playbook: %w", err)
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		readLines(stdout, req.OnOutput)
	}()
	go func() {
		defer wg.Done()
		readLines(stderr, req.OnOutput)
	}()
	wg.Wait()

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("ansible-playbook failed: %w", err)
	}
	return nil
}

// writeBecomePasswordFile writes the sudo password to a private temp file
// (os.CreateTemp creates it 0600) and returns a cleanup that removes it.
func writeBecomePasswordFile(password string) (path string, cleanup func(), err error) {
	f, err := os.CreateTemp("", "brrewery-become-*.txt")
	if err != nil {
		return "", nil, fmt.Errorf("create become password file: %w", err)
	}
	name := f.Name()
	cleanup = func() { _ = os.Remove(name) }

	if _, err := f.WriteString(password + "\n"); err != nil {
		_ = f.Close()
		cleanup()
		return "", nil, fmt.Errorf("write become password file: %w", err)
	}
	if err := f.Close(); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("close become password file: %w", err)
	}
	return name, cleanup, nil
}

func withoutKey(m map[string]string, key string) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		if k != key {
			out[k] = v
		}
	}
	return out
}

func readLines(r io.Reader, onOutput func(line string)) {
	if onOutput == nil {
		return
	}
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		onOutput(scanner.Text())
	}
}
