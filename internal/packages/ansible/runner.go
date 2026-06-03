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
	if len(req.ExtraVars) > 0 {
		payload, marshalErr := json.Marshal(req.ExtraVars)
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

func readLines(r io.Reader, onOutput func(line string)) {
	if onOutput == nil {
		return
	}
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		onOutput(scanner.Text())
	}
}
