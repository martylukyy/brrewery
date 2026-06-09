// Package paths defines fixed production paths for brrewery.
package paths

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	BinaryPath           = "/usr/local/bin/brrewery"
	BackendListenAddress = "127.0.0.1:8080"
	DevBackendListenAddr = "127.0.0.1:8081"
	LogFile              = "/var/log/brrewery/brrewery.log"
	WebRoot              = "/var/www/brrewery"
	UserStorePath        = "/var/lib/brrewery/users.json"
	JobsDir              = "/var/lib/brrewery/jobs"
	SessionSecretPath    = "/var/lib/brrewery/session.key" //nolint:gosec // filesystem path, not a secret value
	AnsibleRoot          = "/usr/share/brrewery/ansible"
	VendorRoot           = "/usr/share/brrewery/vendor"
	// QBittorrentOperatorPatchesDir holds optional operator-supplied libtorrent
	// patches applied at build time. Web UI uploads are never written here.
	QBittorrentOperatorPatchesDir = "/var/lib/brrewery/patches/qbittorrent"
	NginxSitesAvailable           = "/etc/nginx/sites-available"
	NginxSitesEnabled             = "/etc/nginx/sites-enabled"
	TLSCertPath                   = "/etc/ssl/brrewery/fullchain.pem"
	TLSKeyPath                    = "/etc/ssl/brrewery/privkey.pem"

	qbittorrentBuildFilesDir = "roles/qbittorrent_build/files/qbittorrent"
)

// ListenAddress returns the HTTP listen address for the API server.
func ListenAddress() string {
	if addr := strings.TrimSpace(os.Getenv("BRREWERY_LISTEN_ADDR")); addr != "" {
		return addr
	}
	return BackendListenAddress
}

// ResolveJobsDir returns the directory for persisted install jobs. Override with
// BRREWERY_JOBS_DIR; set it to "-" for an in-memory-only store (tests).
func ResolveJobsDir() string {
	if env := strings.TrimSpace(os.Getenv("BRREWERY_JOBS_DIR")); env != "" {
		if env == "-" {
			return ""
		}
		return env
	}
	return JobsDir
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

// ResolveVendorQBittorrentRoot returns the qBittorrent build manifest and
// patches tree. In development this is ansible/roles/qbittorrent_build/files/qbittorrent;
// in production installs use /usr/share/brrewery/vendor/qbittorrent for the
// downloaded source cache (manifest/patches are copied from the role at install time).
func ResolveVendorQBittorrentRoot() string {
	if env := strings.TrimSpace(os.Getenv("BRREWERY_QBITTORRENT_VENDOR_ROOT")); env != "" {
		return env
	}

	for _, candidate := range qbittorrentManifestCandidates() {
		if isVendorQBittorrentRoot(candidate) {
			return absPath(candidate)
		}
	}

	return filepath.Join(VendorRoot, "qbittorrent")
}

func qbittorrentManifestCandidates() []string {
	candidates := make([]string, 0, 4)
	if root := resolveRepoRoot(); root != "" {
		candidates = append(candidates, filepath.Join(root, "ansible", qbittorrentBuildFilesDir))
	}
	candidates = append(candidates,
		filepath.Join(ResolveAnsibleRoot(), qbittorrentBuildFilesDir),
		filepath.Join("ansible", qbittorrentBuildFilesDir),
		filepath.Join("/etc/brrewery/ansible", qbittorrentBuildFilesDir),
		filepath.Join(VendorRoot, "qbittorrent"),
	)
	return candidates
}

func isVendorQBittorrentRoot(path string) bool {
	info, err := os.Stat(filepath.Join(path, "manifest.yml"))
	return err == nil && !info.IsDir()
}

// resolveRepoRoot locates the repository root at runtime so development and CI
// test runs can read the ansible tree and vendored manifests straight from the
// checkout. It first checks the running binary's location (the dev server builds
// to <repo>/tmp/brrewery) and falls back to this source file's compile-time path
// (present for `go test`/`go build` from a checkout). On deployed hosts neither
// resolves and callers fall through to the fixed install locations.
func resolveRepoRoot() string {
	if root := repoRootFromExecutable(); root != "" {
		return root
	}
	return repoRootFromSource()
}

func repoRootFromExecutable() string {
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

// repoRootFromSource derives the repo root from this file's compile-time path
// (<repo>/internal/paths/paths.go). It only resolves when the checkout is still
// present, e.g. when running tests; the existence guard keeps it from returning a
// stale path on deployed hosts where the build tree is gone.
func repoRootFromSource() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return ""
	}
	root := filepath.Dir(filepath.Dir(filepath.Dir(file)))
	if isAnsibleRoot(filepath.Join(root, "ansible")) {
		return root
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
