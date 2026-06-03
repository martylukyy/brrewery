// Package paths defines fixed production paths for brrewery.
package paths

import (
	"os"
	"path/filepath"
	"strings"
)

const (
	BinaryPath           = "/usr/local/bin/brrewery"
	BackendListenAddress = "127.0.0.1:8080"
	DevBackendListenAddr = "127.0.0.1:8081"
	LogFile              = "/var/log/brrewery/brrewery.log"
	WebRoot              = "/var/www/brrewery"
	UserStorePath        = "/var/lib/brrewery/users.json"
	SessionSecretPath    = "/var/lib/brrewery/session.key" //nolint:gosec // filesystem path, not a secret value
	AnsibleRoot          = "/usr/share/brrewery/ansible"
	NginxSitesAvailable  = "/etc/nginx/sites-available"
	NginxSitesEnabled    = "/etc/nginx/sites-enabled"
	TLSCertPath          = "/etc/ssl/brrewery/fullchain.pem"
	TLSKeyPath           = "/etc/ssl/brrewery/privkey.pem"
)

// ListenAddress returns the HTTP listen address for the API server.
func ListenAddress() string {
	if addr := strings.TrimSpace(os.Getenv("BRREWERY_LISTEN_ADDR")); addr != "" {
		return addr
	}
	return BackendListenAddress
}

// ResolveJobsDir returns a directory for persisted install jobs, or empty for in-memory only.
func ResolveJobsDir() string {
	return strings.TrimSpace(os.Getenv("BRREWERY_JOBS_DIR"))
}

// ResolveAnsibleRoot returns the ansible tree used for package playbooks.
func ResolveAnsibleRoot() string {
	if env := strings.TrimSpace(os.Getenv("BRREWERY_ANSIBLE_ROOT")); env != "" {
		return env
	}

	if root := resolveRepoRoot(); root != "" {
		candidate := filepath.Join(root, "ansible")
		if isAnsibleRoot(candidate) {
			return absPath(candidate)
		}
	}

	candidates := []string{
		filepath.Join("ansible"),
		filepath.Join("/etc/brrewery/ansible"),
		AnsibleRoot,
	}
	for _, candidate := range candidates {
		if isAnsibleRoot(candidate) {
			return absPath(candidate)
		}
	}

	return AnsibleRoot
}

func resolveRepoRoot() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return ""
	}

	dir := filepath.Dir(exe)
	if filepath.Base(dir) == "tmp" {
		return filepath.Dir(dir)
	}

	return ""
}

func isAnsibleRoot(path string) bool {
	info, err := os.Stat(filepath.Join(path, "playbooks", "packages"))
	return err == nil && info.IsDir()
}

func absPath(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return abs
}
