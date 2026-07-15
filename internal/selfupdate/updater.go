package selfupdate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/rs/zerolog"

	"github.com/autobrr/brrewery/internal/apps/jobs"
	"github.com/autobrr/brrewery/internal/apps/model"
	"github.com/autobrr/brrewery/internal/buildinfo"
	"github.com/autobrr/brrewery/internal/paths"
)

var (
	ErrUpdateInProgress = errors.New("update already in progress")
	ErrNoUpdate         = errors.New("no update available")
	ErrUnsupported      = errors.New("self-update is not supported for this installation")
	ErrNoPendingRestart = errors.New("no installed update is waiting for a restart")
)

// selfUpdateAppID is the pseudo app id self-update jobs carry in the shared
// job store, so the existing /jobs endpoints serve their progress unchanged.
const selfUpdateAppID = "brrewery"

// minFreeBytes is required in the staging filesystem before downloading; the
// extracted release plus staged copies stay well under it.
const minFreeBytes = 500 * 1024 * 1024

// nginxFiles are the contrib/nginx files install.sh manages, relative to both
// the archive's contrib/nginx dir and /etc/nginx.
var nginxFiles = []string{
	"nginx.conf",
	"general.conf",
	"security.conf",
	"proxy.conf",
	"ssl.conf",
	filepath.Join("sites-available", "default"),
}

const systemdUnitArchivePath = "contrib/systemd/brrewery.service"

// Config carries every path and side effect the updater touches, so tests can
// point it all at temp dirs and a command recorder.
type Config struct {
	Repo            string
	CurrentVersion  string
	BinaryPath      string
	WebRoot         string
	AnsibleRoot     string
	NginxEtc        string
	SystemdUnitPath string
	StagingDir      string
	MarkerPath      string
	// DownloadBaseURL prefixes "{repo}/releases/download/{tag}/{asset}".
	DownloadBaseURL string
	RunCmd          func(ctx context.Context, name string, args ...string) (string, error)
	Executable      func() (string, error)
	// SkipPlatformCheck lets tests run the updater on non-linux/amd64 hosts.
	SkipPlatformCheck bool
}

// DefaultConfig returns the production configuration.
func DefaultConfig() Config {
	return Config{
		Repo:            RepoFromEnv(),
		CurrentVersion:  buildinfo.Version,
		BinaryPath:      paths.BinaryPath,
		WebRoot:         paths.WebRoot,
		AnsibleRoot:     paths.AnsibleRoot,
		NginxEtc:        "/etc/nginx",
		SystemdUnitPath: "/etc/systemd/system/brrewery.service",
		StagingDir:      "/var/lib/brrewery/selfupdate",
		MarkerPath:      "/var/lib/brrewery/selfupdate-pending.json",
		DownloadBaseURL: "https://github.com",
		RunCmd:          runCommand,
		Executable:      os.Executable,
	}
}

func runCommand(ctx context.Context, name string, args ...string) (string, error) {
	out, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// pendingMarker survives the restart and tells the new process which job to
// resolve; the job model itself has no target-version field.
type pendingMarker struct {
	JobID         string    `json:"job_id"`
	TargetVersion string    `json:"target_version"`
	StartedAt     time.Time `json:"started_at"`
}

// Updater downloads the newest release archive and installs everything
// install.sh installs, then restarts the service via systemd.
type Updater struct {
	cfg     Config
	jobs    *jobs.Store
	checker *Checker
	logger  zerolog.Logger
	client  *http.Client
	running atomic.Bool
}

func NewUpdater(cfg *Config, store *jobs.Store, checker *Checker, logger *zerolog.Logger) *Updater {
	if cfg.RunCmd == nil {
		cfg.RunCmd = runCommand
	}
	if cfg.Executable == nil {
		cfg.Executable = os.Executable
	}
	return &Updater{
		cfg:     *cfg,
		jobs:    store,
		checker: checker,
		logger:  *logger,
		client:  &http.Client{},
	}
}

// Start validates that an update can run, creates the job and installs in the
// background. The job is intentionally left running when the process restarts;
// ReconcileOnStartup resolves it from the new binary.
func (u *Updater) Start(ctx context.Context) (model.Job, error) {
	if !u.running.CompareAndSwap(false, true) {
		return model.Job{}, ErrUpdateInProgress
	}

	if err := u.supported(); err != nil {
		u.running.Store(false)
		return model.Job{}, err
	}

	status, err := u.checker.Refresh(ctx)
	if err != nil {
		u.running.Store(false)
		return model.Job{}, fmt.Errorf("check for updates: %w", err)
	}
	if !status.UpdateAvailable {
		u.running.Store(false)
		return model.Job{}, ErrNoUpdate
	}

	job := u.jobs.Create(selfUpdateAppID, model.JobActionSelfUpdate)
	// The request context dies with the HTTP response; the update must not.
	go u.run(context.Background(), job.ID, status.LatestTag) //nolint:contextcheck,gosec // deliberately detached from the request
	return job, nil
}

func (u *Updater) supported() error {
	if !u.cfg.SkipPlatformCheck && (runtime.GOOS != "linux" || runtime.GOARCH != "amd64") {
		return fmt.Errorf("%w: release binaries are linux/amd64 only", ErrUnsupported)
	}
	if IsDevBuild(u.cfg.CurrentVersion) {
		return fmt.Errorf("%w: this is a development build", ErrUnsupported)
	}
	exe, err := u.cfg.Executable()
	if err != nil {
		return fmt.Errorf("%w: cannot resolve running binary: %w", ErrUnsupported, err)
	}
	// A staged update renames the running binary to brrewery.bak until the
	// restart, and /proc/self/exe follows that rename — with a " (deleted)"
	// suffix once a further staging removes the backup. Both still identify
	// the installed binary, so updating again before the restart stays allowed.
	exe = strings.TrimSuffix(exe, " (deleted)")
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	if exe != u.cfg.BinaryPath && exe != u.cfg.BinaryPath+".bak" {
		return fmt.Errorf("%w: running binary %s is not %s", ErrUnsupported, exe, u.cfg.BinaryPath)
	}
	return nil
}

func (u *Updater) run(ctx context.Context, jobID, tag string) {
	u.jobs.MarkRunning(jobID)

	if err := u.install(ctx, jobID, tag); err != nil {
		u.logger.Error().Err(err).Str("tag", tag).Msg("self-update failed")
		u.jobs.AppendLog(jobID, "Error: "+err.Error())
		u.jobs.SetStatus(jobID, model.JobStatusFailed, err.Error())
		_ = os.RemoveAll(u.cfg.StagingDir)
		u.running.Store(false)
		return
	}

	// Everything is installed; the job is done. The old process keeps serving
	// until the operator confirms the result and triggers Restart, so nobody
	// is kicked out of their session mid-review.
	u.jobs.SetStatus(jobID, model.JobStatusSucceeded, "")
	_ = os.RemoveAll(u.cfg.StagingDir)
	u.running.Store(false)
}

// Restart finishes an installed update by restarting the brrewery service.
// It is gated on the pending marker the install wrote, so it can only ever
// complete a staged update, never bounce the service on its own.
func (u *Updater) Restart(ctx context.Context) error {
	// A marker from an already-installed update may still be on disk while a
	// newer install is running; restarting now would cut that install short.
	if u.running.Load() {
		return ErrUpdateInProgress
	}

	data, err := os.ReadFile(u.cfg.MarkerPath)
	if err != nil {
		return ErrNoPendingRestart
	}
	var marker pendingMarker
	if err := json.Unmarshal(data, &marker); err != nil || marker.JobID == "" {
		return ErrNoPendingRestart
	}

	u.log(marker.JobID, "Restarting brrewery to finish the update")
	if out, err := u.cfg.RunCmd(ctx, "systemctl", "--no-block", "restart", "brrewery"); err != nil {
		// systemctl runs inside brrewery's own cgroup, so the restart it
		// triggers tears it down along with the service before it can exit
		// cleanly. A child killed by that shutdown is the restart working,
		// not a failure — only report commands that ran and refused.
		if killedByShutdown(err) {
			return nil
		}
		return fmt.Errorf("systemctl restart brrewery: %s: %w", out, err)
	}
	return nil
}

// killedByShutdown reports whether a command died from SIGTERM/SIGKILL rather
// than failing on its own — the signature of the service shutdown reaping the
// command's process along with the rest of the cgroup.
func killedByShutdown(err error) bool {
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		return false
	}
	waitStatus, ok := exitErr.Sys().(syscall.WaitStatus)
	if !ok || !waitStatus.Signaled() {
		return false
	}
	return waitStatus.Signal() == syscall.SIGTERM || waitStatus.Signal() == syscall.SIGKILL
}

// RestartPending reports whether an installed update is waiting for a restart.
func (u *Updater) RestartPending() bool {
	_, err := os.Stat(u.cfg.MarkerPath)
	return err == nil
}

func (u *Updater) log(jobID, format string, args ...any) {
	line := fmt.Sprintf(format, args...)
	u.jobs.AppendLog(jobID, line)
	u.logger.Info().Str("job", jobID).Msg("self-update: " + line)
}

func (u *Updater) install(ctx context.Context, jobID, tag string) error {
	version := strings.TrimPrefix(tag, "v")

	extractDir, err := u.fetchRelease(ctx, jobID, tag, version)
	if err != nil {
		return err
	}

	u.log(jobID, "Updating ansible playbooks")
	if err := swapDir(filepath.Join(extractDir, "ansible"), u.cfg.AnsibleRoot); err != nil {
		return err
	}

	u.log(jobID, "Updating web assets")
	if err := swapDir(filepath.Join(extractDir, "web", "dist"), u.cfg.WebRoot); err != nil {
		return err
	}

	u.log(jobID, "Updating nginx configuration")
	if err := u.installNginx(ctx, jobID, extractDir); err != nil {
		return err
	}

	u.log(jobID, "Updating binary")
	if err := u.installBinary(extractDir); err != nil {
		return err
	}

	u.log(jobID, "Updating systemd unit")
	if err := copyFile(filepath.Join(extractDir, filepath.FromSlash(systemdUnitArchivePath)), u.cfg.SystemdUnitPath, 0o644); err != nil {
		return err
	}
	if out, err := u.cfg.RunCmd(ctx, "systemctl", "daemon-reload"); err != nil {
		return fmt.Errorf("systemctl daemon-reload: %s: %w", out, err)
	}

	// The marker survives the restart and tells the new process which job the
	// finished update belongs to; its presence is also what arms Restart.
	marker := pendingMarker{JobID: jobID, TargetVersion: version, StartedAt: time.Now().UTC()}
	payload, err := json.Marshal(marker)
	if err != nil {
		return fmt.Errorf("encode pending marker: %w", err)
	}
	if err := os.WriteFile(u.cfg.MarkerPath, payload, 0o600); err != nil {
		return fmt.Errorf("write pending marker: %w", err)
	}

	u.log(jobID, "Update installed. Please restart brrewery to finish the update.")
	return nil
}

// fetchRelease downloads the release archive, verifies its checksum against
// checksums.txt and extracts it into staging, returning the extracted tree.
func (u *Updater) fetchRelease(ctx context.Context, jobID, tag, version string) (string, error) {
	archiveName := fmt.Sprintf("brrewery_%s_linux_amd64.tar.gz", version)

	if err := os.RemoveAll(u.cfg.StagingDir); err != nil {
		return "", fmt.Errorf("clear staging dir: %w", err)
	}
	if err := os.MkdirAll(u.cfg.StagingDir, 0o750); err != nil {
		return "", fmt.Errorf("create staging dir: %w", err)
	}
	if err := checkFreeSpace(u.cfg.StagingDir, minFreeBytes); err != nil {
		return "", err
	}

	baseURL := fmt.Sprintf("%s/%s/releases/download/%s", u.cfg.DownloadBaseURL, u.cfg.Repo, tag)
	archivePath := filepath.Join(u.cfg.StagingDir, archiveName)
	checksumsPath := filepath.Join(u.cfg.StagingDir, "checksums.txt")

	u.log(jobID, "Downloading release archive")
	if err := downloadFile(ctx, u.client, baseURL+"/"+archiveName, archivePath); err != nil {
		return "", err
	}
	if err := downloadFile(ctx, u.client, baseURL+"/checksums.txt", checksumsPath); err != nil {
		return "", err
	}

	u.log(jobID, "Verifying checksum")
	if err := verifyChecksum(archivePath, checksumsPath, archiveName); err != nil {
		return "", err
	}

	extractDir := filepath.Join(u.cfg.StagingDir, "extract")
	u.log(jobID, "Extracting release archive")
	if err := extractTarGz(archivePath, extractDir); err != nil {
		return "", err
	}
	if err := validateArchive(extractDir); err != nil {
		return "", err
	}
	return extractDir, nil
}

// validateArchive checks the extracted release for everything the install
// touches before any live path is modified.
func validateArchive(dir string) error {
	binary, err := os.Stat(filepath.Join(dir, "brrewery"))
	if err != nil || !binary.Mode().IsRegular() || binary.Mode().Perm()&0o100 == 0 {
		return errors.New("release archive is missing an executable brrewery binary")
	}
	for _, sub := range []string{"web/dist", "ansible", "contrib/nginx"} {
		info, err := os.Stat(filepath.Join(dir, filepath.FromSlash(sub)))
		if err != nil || !info.IsDir() {
			return fmt.Errorf("release archive is missing %s", sub)
		}
	}
	for _, rel := range nginxFiles {
		if _, err := os.Stat(filepath.Join(dir, "contrib", "nginx", rel)); err != nil {
			return fmt.Errorf("release archive is missing contrib/nginx/%s", filepath.ToSlash(rel))
		}
	}
	if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(systemdUnitArchivePath))); err != nil {
		return fmt.Errorf("release archive is missing %s", systemdUnitArchivePath)
	}
	return nil
}

// installNginx mirrors install.sh's "Configuring nginx" step, minus the
// fresh-install-only parts (cert generation, acme.sh, systemctl enable). The
// operator's server_name is carried over, and a failed `nginx -t` restores
// the previous configs — a broken nginx would take the whole UI down.
func (u *Updater) installNginx(ctx context.Context, jobID, extractDir string) error {
	srcDir := filepath.Join(extractDir, "contrib", "nginx")
	backupDir := filepath.Join(u.cfg.StagingDir, "nginx-backup")

	sitesAvailable := filepath.Join(u.cfg.NginxEtc, "sites-available")
	sitesEnabled := filepath.Join(u.cfg.NginxEtc, "sites-enabled")
	for _, dir := range []string{sitesAvailable, sitesEnabled} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create %s: %w", dir, err)
		}
	}

	vhostPath := filepath.Join(sitesAvailable, "default")
	serverName := currentServerName(vhostPath)

	existed, err := u.backupNginxFiles(backupDir)
	if err != nil {
		return err
	}

	for _, rel := range nginxFiles {
		if err := copyFile(filepath.Join(srcDir, rel), filepath.Join(u.cfg.NginxEtc, rel), 0o644); err != nil {
			return err
		}
	}
	if serverName != "" {
		u.log(jobID, "Preserving nginx server_name %s", serverName)
		if err := applyServerName(vhostPath, serverName); err != nil {
			return err
		}
	}

	if err := u.enableDefaultSite(); err != nil {
		return err
	}

	if out, err := u.cfg.RunCmd(ctx, "nginx", "-t"); err != nil {
		u.log(jobID, "Nginx config test failed, restoring previous configuration")
		u.restoreNginxBackups(jobID, backupDir, existed)
		_, _ = u.cfg.RunCmd(ctx, "nginx", "-t")
		return fmt.Errorf("nginx config test failed: %s: %w", out, err)
	}

	if out, err := u.cfg.RunCmd(ctx, "systemctl", "reload", "nginx"); err != nil {
		if startOut, startErr := u.cfg.RunCmd(ctx, "systemctl", "start", "nginx"); startErr != nil {
			return fmt.Errorf("reload nginx: %s: %s: %w", out, startOut, startErr)
		}
	}
	return nil
}

// enableDefaultSite removes the legacy site files install.sh also cleans up
// and (re)points sites-enabled/default at the managed vhost.
func (u *Updater) enableDefaultSite() error {
	sitesAvailable := filepath.Join(u.cfg.NginxEtc, "sites-available")
	sitesEnabled := filepath.Join(u.cfg.NginxEtc, "sites-enabled")

	for _, legacy := range []string{
		filepath.Join(sitesEnabled, "brrewery"),
		filepath.Join(sitesEnabled, "brrewery.conf"),
		filepath.Join(sitesAvailable, "brrewery.conf"),
		filepath.Join(u.cfg.NginxEtc, "nginxconfig.io"),
	} {
		_ = os.RemoveAll(legacy)
	}

	enabledLink := filepath.Join(sitesEnabled, "default")
	_ = os.Remove(enabledLink)
	if err := os.Symlink(filepath.Join("..", "sites-available", "default"), enabledLink); err != nil {
		return fmt.Errorf("enable default site: %w", err)
	}
	return nil
}

// backupNginxFiles copies every managed nginx file that currently exists into
// backupDir so a failed nginx -t can put the working configuration back. It
// returns which files existed (missing ones are removed on restore).
func (u *Updater) backupNginxFiles(backupDir string) (map[string]bool, error) {
	existed := make(map[string]bool, len(nginxFiles))
	for _, rel := range nginxFiles {
		target := filepath.Join(u.cfg.NginxEtc, rel)
		if _, err := os.Stat(target); err != nil {
			continue
		}
		existed[rel] = true
		backup := filepath.Join(backupDir, rel)
		if err := os.MkdirAll(filepath.Dir(backup), 0o750); err != nil {
			return nil, fmt.Errorf("create nginx backup dir: %w", err)
		}
		if err := copyFile(target, backup, 0o600); err != nil {
			return nil, fmt.Errorf("back up %s: %w", target, err)
		}
	}
	return existed, nil
}

func (u *Updater) restoreNginxBackups(jobID, backupDir string, existed map[string]bool) {
	for _, rel := range nginxFiles {
		target := filepath.Join(u.cfg.NginxEtc, rel)
		if !existed[rel] {
			_ = os.Remove(target)
			continue
		}
		if err := copyFile(filepath.Join(backupDir, rel), target, 0o644); err != nil {
			u.log(jobID, "Failed to restore %s: %v", target, err)
		}
	}
}

// installBinary replaces the running binary with two same-directory renames,
// both atomic; the previous binary stays behind as brrewery.bak for manual
// rollback until the next update overwrites it.
func (u *Updater) installBinary(extractDir string) error {
	dir := filepath.Dir(u.cfg.BinaryPath)
	staged := filepath.Join(dir, ".brrewery.new")
	if err := copyFile(filepath.Join(extractDir, "brrewery"), staged, 0o755); err != nil {
		return err
	}

	backup := u.cfg.BinaryPath + ".bak"
	if _, err := os.Stat(u.cfg.BinaryPath); err == nil {
		_ = os.Remove(backup)
		if err := os.Rename(u.cfg.BinaryPath, backup); err != nil {
			_ = os.Remove(staged)
			return fmt.Errorf("move aside current binary: %w", err)
		}
	}
	if err := os.Rename(staged, u.cfg.BinaryPath); err != nil {
		_ = os.Rename(backup, u.cfg.BinaryPath)
		_ = os.Remove(staged)
		return fmt.Errorf("install new binary: %w", err)
	}
	return nil
}

// ReconcileOnStartup cleans up after the restart that finished an update. The
// job was already marked succeeded when the install completed; here the new
// process only confirms the running version, clears the marker, and demotes
// the job to failed if the restart somehow brought the old version back.
// Self-update jobs still queued/running (crash mid-install) are swept to failed.
func (u *Updater) ReconcileOnStartup() {
	data, err := os.ReadFile(u.cfg.MarkerPath)
	if err == nil {
		var marker pendingMarker
		if jsonErr := json.Unmarshal(data, &marker); jsonErr == nil && marker.JobID != "" {
			if marker.TargetVersion == u.cfg.CurrentVersion {
				u.jobs.AppendLog(marker.JobID, "Restarted, now running version "+u.cfg.CurrentVersion)
				u.logger.Info().Str("version", u.cfg.CurrentVersion).Msg("self-update finished")
			} else {
				msg := fmt.Sprintf("Restarted but still running version %s (expected %s)", u.cfg.CurrentVersion, marker.TargetVersion)
				u.jobs.AppendLog(marker.JobID, msg)
				u.jobs.SetStatus(marker.JobID, model.JobStatusFailed, msg)
				u.logger.Error().Str("expected", marker.TargetVersion).Str("running", u.cfg.CurrentVersion).Msg("self-update did not take effect")
			}
		}
		_ = os.Remove(u.cfg.MarkerPath)
		_ = os.RemoveAll(u.cfg.StagingDir)
	}

	for _, job := range u.jobs.List() {
		if job.Action != model.JobActionSelfUpdate {
			continue
		}
		if job.Status == model.JobStatusQueued || job.Status == model.JobStatusRunning {
			u.jobs.SetStatus(job.ID, model.JobStatusFailed, "interrupted by restart")
		}
	}
}

func checkFreeSpace(dir string, minBytes uint64) error {
	var st syscall.Statfs_t
	if err := syscall.Statfs(dir, &st); err != nil {
		return nil // best-effort; don't block the update on an exotic fs
	}
	free := st.Bavail * uint64(st.Bsize) //nolint:gosec,unconvert // Bsize is int64 on linux
	if free < minBytes {
		return fmt.Errorf("not enough free disk space in %s: %d MiB free, %d MiB required",
			dir, free/(1024*1024), minBytes/(1024*1024))
	}
	return nil
}

var serverNameRe = regexp.MustCompile(`(?m)^\s*server_name\s+(.+?);`)

// currentServerName returns the operator-facing server_name from the current
// vhost, if install.sh ever wrote a domain into it ("_" is the shipped
// placeholder). Empty when the file is missing or unmodified.
func currentServerName(vhostPath string) string {
	data, err := os.ReadFile(vhostPath)
	if err != nil {
		return ""
	}
	for _, match := range serverNameRe.FindAllStringSubmatch(string(data), -1) {
		name := strings.TrimSpace(match[1])
		if name != "" && name != "_" {
			return name
		}
	}
	return ""
}

// applyServerName re-applies the operator's domain to the freshly installed
// vhost, mirroring the sed install.sh runs when a domain is configured.
func applyServerName(vhostPath, serverName string) error {
	data, err := os.ReadFile(vhostPath)
	if err != nil {
		return fmt.Errorf("read vhost: %w", err)
	}
	updated := strings.ReplaceAll(string(data), "server_name _;", "server_name "+serverName+";")
	//nolint:gosec // 0644 matches install.sh's install -m 0644 for nginx configs
	if err := os.WriteFile(vhostPath, []byte(updated), 0o644); err != nil {
		return fmt.Errorf("write vhost: %w", err)
	}
	return nil
}
