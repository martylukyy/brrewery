package selfupdate

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/autobrr/brrewery/internal/apps/jobs"
	"github.com/autobrr/brrewery/internal/apps/model"
)

const (
	testCurrentVersion = "1.0.0"
	testTargetTag      = "v1.1.0"
	testTargetVersion  = "1.1.0"
)

var nopLogger = zerolog.Nop()

type archiveEntry struct {
	name    string
	mode    int64
	content string
}

func defaultArchiveEntries() []archiveEntry {
	return []archiveEntry{
		{"brrewery", 0o755, "new-binary"},
		{"web/dist/index.html", 0o644, "<html>new</html>"},
		{"ansible/playbooks/apps/install.yml", 0o644, "- hosts: localhost"},
		{"contrib/nginx/nginx.conf", 0o644, "new nginx.conf"},
		{"contrib/nginx/general.conf", 0o644, "new general.conf"},
		{"contrib/nginx/security.conf", 0o644, "new security.conf"},
		{"contrib/nginx/proxy.conf", 0o644, "new proxy.conf"},
		{"contrib/nginx/ssl.conf", 0o644, "new ssl.conf"},
		{"contrib/nginx/sites-available/default", 0o644, "server_name _;\nlisten 80;\nserver_name _;\n"},
		{"contrib/systemd/brrewery.service", 0o644, "[Service]\nExecStart=/usr/local/bin/brrewery serve\n"},
	}
}

func buildArchive(t *testing.T, entries []archiveEntry) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for _, entry := range entries {
		require.NoError(t, tw.WriteHeader(&tar.Header{
			Name:     entry.name,
			Mode:     entry.mode,
			Size:     int64(len(entry.content)),
			Typeflag: tar.TypeReg,
		}))
		_, err := tw.Write([]byte(entry.content))
		require.NoError(t, err)
	}
	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())
	return buf.Bytes()
}

type cmdRecorder struct {
	mu     sync.Mutex
	calls  []string
	failOn map[string]error
}

func (r *cmdRecorder) run(_ context.Context, name string, args ...string) (string, error) {
	cmd := strings.Join(append([]string{name}, args...), " ")
	r.mu.Lock()
	r.calls = append(r.calls, cmd)
	r.mu.Unlock()
	if err, ok := r.failOn[cmd]; ok {
		return "simulated failure", err
	}
	return "", nil
}

func (r *cmdRecorder) recorded() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.calls...)
}

func (r *cmdRecorder) has(cmd string) bool {
	for _, call := range r.recorded() {
		if call == cmd {
			return true
		}
	}
	return false
}

type fixture struct {
	updater *Updater
	store   *jobs.Store
	rec     *cmdRecorder
	cfg     Config
}

// newFixture stands up a fake GitHub (releases API + download assets) and an
// updater whose every path points into a temp dir.
func newFixture(t *testing.T, archive []byte, checksums string) *fixture {
	t.Helper()
	root := t.TempDir()

	repo := "martylukyy/brrewery"
	archiveName := fmt.Sprintf("brrewery_%s_linux_amd64.tar.gz", testTargetVersion)
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/"+repo+"/releases", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprintf(w, `[{"tag_name": %q, "draft": false}]`, testTargetTag)
	})
	mux.HandleFunc("/"+repo+"/releases/download/"+testTargetTag+"/"+archiveName, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(archive)
	})
	mux.HandleFunc("/"+repo+"/releases/download/"+testTargetTag+"/checksums.txt", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(checksums))
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	binDir := filepath.Join(root, "bin")
	nginxEtc := filepath.Join(root, "nginx")
	systemdDir := filepath.Join(root, "systemd")
	for _, dir := range []string{binDir, nginxEtc, filepath.Join(nginxEtc, "sites-available"), systemdDir} {
		require.NoError(t, os.MkdirAll(dir, 0o755))
	}
	cfg := Config{
		Repo:            repo,
		CurrentVersion:  testCurrentVersion,
		BinaryPath:      filepath.Join(binDir, "brrewery"),
		WebRoot:         filepath.Join(root, "www"),
		AnsibleRoot:     filepath.Join(root, "ansible"),
		NginxEtc:        nginxEtc,
		SystemdUnitPath: filepath.Join(systemdDir, "brrewery.service"),
		StagingDir:      filepath.Join(root, "staging"),
		MarkerPath:      filepath.Join(root, "selfupdate-pending.json"),
		DownloadBaseURL: server.URL,
	}
	cfg.Executable = func() (string, error) { return cfg.BinaryPath, nil }

	// Pre-existing install state the update must replace.
	require.NoError(t, os.WriteFile(cfg.BinaryPath, []byte("old-binary"), 0o600))
	require.NoError(t, os.MkdirAll(cfg.WebRoot, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(cfg.WebRoot, "index.html"), []byte("<html>old</html>"), 0o600))
	require.NoError(t, os.MkdirAll(filepath.Join(cfg.AnsibleRoot, "playbooks"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(cfg.AnsibleRoot, "stale.yml"), []byte("old"), 0o600))
	for _, rel := range nginxFiles {
		require.NoError(t, os.WriteFile(filepath.Join(nginxEtc, rel), []byte("old "+filepath.ToSlash(rel)), 0o600))
	}

	rec := &cmdRecorder{failOn: map[string]error{}}
	cfg.RunCmd = rec.run

	store := jobs.NewStore()
	checker := NewChecker(repo)
	checker.apiBase = server.URL
	checker.currentVersion = testCurrentVersion

	return &fixture{
		updater: NewUpdater(&cfg, store, checker, &nopLogger),
		store:   store,
		rec:     rec,
		cfg:     cfg,
	}
}

func newDefaultFixture(t *testing.T) *fixture {
	t.Helper()
	archive := buildArchive(t, defaultArchiveEntries())
	sum := sha256.Sum256(archive)
	checksums := fmt.Sprintf("%s  brrewery_%s_linux_amd64.tar.gz\n", hex.EncodeToString(sum[:]), testTargetVersion)
	return newFixture(t, archive, checksums)
}

const restartCmd = "systemctl --no-block restart brrewery"

func waitForSuccess(t *testing.T, f *fixture, jobID string) {
	t.Helper()
	require.Eventually(t, func() bool {
		job, ok := f.store.Get(jobID)
		return ok && job.Status == model.JobStatusSucceeded
	}, 5*time.Second, 10*time.Millisecond, "update job never succeeded")
}

func waitForFailure(t *testing.T, f *fixture, jobID string) model.Job {
	t.Helper()
	var job model.Job
	require.Eventually(t, func() bool {
		var ok bool
		job, ok = f.store.Get(jobID)
		return ok && job.Status == model.JobStatusFailed
	}, 5*time.Second, 10*time.Millisecond, "update job never failed")
	return job
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(data)
}

func TestUpdateHappyPath(t *testing.T) {
	f := newDefaultFixture(t)

	job, err := f.updater.Start(context.Background())
	require.NoError(t, err)
	assert.Equal(t, selfUpdateAppID, job.AppID)
	assert.Equal(t, model.JobActionSelfUpdate, job.Action)

	waitForSuccess(t, f, job.ID)

	assert.Equal(t, "new-binary", readFile(t, f.cfg.BinaryPath))
	assert.Equal(t, "old-binary", readFile(t, f.cfg.BinaryPath+".bak"))
	assert.Equal(t, "<html>new</html>", readFile(t, filepath.Join(f.cfg.WebRoot, "index.html")))
	assert.NoFileExists(t, filepath.Join(f.cfg.AnsibleRoot, "stale.yml"))
	assert.FileExists(t, filepath.Join(f.cfg.AnsibleRoot, "playbooks", "apps", "install.yml"))
	assert.Equal(t, "new nginx.conf", readFile(t, filepath.Join(f.cfg.NginxEtc, "nginx.conf")))
	assert.Contains(t, readFile(t, f.cfg.SystemdUnitPath), "ExecStart")

	link, err := os.Readlink(filepath.Join(f.cfg.NginxEtc, "sites-enabled", "default"))
	require.NoError(t, err)
	assert.Equal(t, filepath.Join("..", "sites-available", "default"), link)

	var marker pendingMarker
	require.NoError(t, json.Unmarshal([]byte(readFile(t, f.cfg.MarkerPath)), &marker))
	assert.Equal(t, job.ID, marker.JobID)
	assert.Equal(t, testTargetVersion, marker.TargetVersion)

	// No restart yet: the operator reviews the succeeded job first.
	calls := f.rec.recorded()
	assert.Equal(t, []string{"nginx -t", "systemctl reload nginx", "systemctl daemon-reload"}, calls)
	assert.False(t, f.updater.running.Load())
	assert.True(t, f.updater.RestartPending())
	assert.FileExists(t, f.cfg.MarkerPath)
	assert.NoDirExists(t, f.cfg.StagingDir)

	// The operator clicks "Restart brrewery".
	require.NoError(t, f.updater.Restart(context.Background()))
	assert.True(t, f.rec.has(restartCmd))

	// Simulate the new binary starting up.
	f.updater.cfg.CurrentVersion = testTargetVersion
	f.updater.ReconcileOnStartup()

	current, ok := f.store.Get(job.ID)
	require.True(t, ok)
	assert.Equal(t, model.JobStatusSucceeded, current.Status)
	assert.False(t, f.updater.RestartPending())
	assert.NoFileExists(t, f.cfg.MarkerPath)
	assert.NoDirExists(t, f.cfg.StagingDir)
}

// sigtermError produces the real error shape of a child reaped by SIGTERM,
// which is what happens to systemctl when the restart it triggers tears down
// brrewery's cgroup around it.
func sigtermError(t *testing.T) error {
	t.Helper()
	err := exec.CommandContext(context.Background(), "sh", "-c", "kill -TERM $$").Run()
	require.Error(t, err)
	return err
}

func writePendingMarker(t *testing.T, f *fixture) {
	t.Helper()
	marker, err := json.Marshal(pendingMarker{JobID: "job-1", TargetVersion: testTargetVersion, StartedAt: time.Now()})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(f.cfg.MarkerPath, marker, 0o600))
}

func TestRestartToleratesShutdownKillingSystemctl(t *testing.T) {
	f := newDefaultFixture(t)
	writePendingMarker(t, f)
	f.rec.failOn[restartCmd] = sigtermError(t)

	require.NoError(t, f.updater.Restart(context.Background()))
	assert.True(t, f.rec.has(restartCmd))
}

func TestRestartSurfacesRealFailures(t *testing.T) {
	f := newDefaultFixture(t)
	writePendingMarker(t, f)
	f.rec.failOn[restartCmd] = errors.New("exit status 1")

	err := f.updater.Restart(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "systemctl restart brrewery")
}

func TestRestartWithoutPendingUpdate(t *testing.T) {
	f := newDefaultFixture(t)

	err := f.updater.Restart(context.Background())
	require.ErrorIs(t, err, ErrNoPendingRestart)
	assert.False(t, f.rec.has(restartCmd))
}

func TestRestartRefusedWhileUpdateRunning(t *testing.T) {
	f := newDefaultFixture(t)
	// Marker from an already-installed update, plus a newer install in flight.
	require.NoError(t, os.WriteFile(f.cfg.MarkerPath, []byte(`{"job_id":"old","target_version":"1.0.1"}`), 0o600))
	f.updater.running.Store(true)

	err := f.updater.Restart(context.Background())
	require.ErrorIs(t, err, ErrUpdateInProgress)
	assert.False(t, f.rec.has(restartCmd))
}

func TestUpdateChecksumMismatchAbortsBeforeInstall(t *testing.T) {
	archive := buildArchive(t, defaultArchiveEntries())
	checksums := fmt.Sprintf("%064d  brrewery_%s_linux_amd64.tar.gz\n", 0, testTargetVersion)
	f := newFixture(t, archive, checksums)

	job, err := f.updater.Start(context.Background())
	require.NoError(t, err)

	failed := waitForFailure(t, f, job.ID)
	assert.Contains(t, failed.Error, "checksum mismatch")

	assert.Equal(t, "old-binary", readFile(t, f.cfg.BinaryPath))
	assert.Equal(t, "<html>old</html>", readFile(t, filepath.Join(f.cfg.WebRoot, "index.html")))
	assert.Empty(t, f.rec.recorded())
	assert.NoDirExists(t, f.cfg.StagingDir)

	// A failed update releases the lock so it can be retried.
	assert.False(t, f.updater.running.Load())
}

func TestUpdateInvalidArchiveContents(t *testing.T) {
	archive := buildArchive(t, []archiveEntry{{"brrewery", 0o755, "new-binary"}})
	sum := sha256.Sum256(archive)
	checksums := fmt.Sprintf("%s  brrewery_%s_linux_amd64.tar.gz\n", hex.EncodeToString(sum[:]), testTargetVersion)
	f := newFixture(t, archive, checksums)

	job, err := f.updater.Start(context.Background())
	require.NoError(t, err)

	failed := waitForFailure(t, f, job.ID)
	assert.Contains(t, failed.Error, "missing")
	assert.Equal(t, "old-binary", readFile(t, f.cfg.BinaryPath))
}

func TestUpdateNginxTestFailureRestoresBackups(t *testing.T) {
	f := newDefaultFixture(t)
	f.rec.failOn["nginx -t"] = errors.New("exit status 1")

	job, err := f.updater.Start(context.Background())
	require.NoError(t, err)

	failed := waitForFailure(t, f, job.ID)
	assert.Contains(t, failed.Error, "nginx config test failed")

	for _, rel := range nginxFiles {
		assert.Equal(t, "old "+filepath.ToSlash(rel), readFile(t, filepath.Join(f.cfg.NginxEtc, rel)),
			"nginx file %s was not restored", rel)
	}
	// Binary is installed after nginx, so it must be untouched.
	assert.Equal(t, "old-binary", readFile(t, f.cfg.BinaryPath))
	assert.False(t, f.rec.has(restartCmd))
}

func TestUpdatePreservesServerName(t *testing.T) {
	f := newDefaultFixture(t)
	vhost := filepath.Join(f.cfg.NginxEtc, "sites-available", "default")
	require.NoError(t, os.WriteFile(vhost, []byte("server_name _;\nserver_name box.example.com;\n"), 0o600))

	job, err := f.updater.Start(context.Background())
	require.NoError(t, err)
	waitForSuccess(t, f, job.ID)

	content := readFile(t, vhost)
	assert.NotContains(t, content, "server_name _;")
	assert.Equal(t, 2, strings.Count(content, "server_name box.example.com;"))
}

func TestUpdateAlreadyInProgress(t *testing.T) {
	f := newDefaultFixture(t)
	f.updater.running.Store(true)

	_, err := f.updater.Start(context.Background())
	assert.ErrorIs(t, err, ErrUpdateInProgress)
}

func TestUpdateNoUpdateAvailable(t *testing.T) {
	f := newDefaultFixture(t)
	f.updater.checker.currentVersion = testTargetVersion

	_, err := f.updater.Start(context.Background())
	require.ErrorIs(t, err, ErrNoUpdate)
	assert.False(t, f.updater.running.Load())
}

func TestUpdateDevBuildUnsupported(t *testing.T) {
	f := newDefaultFixture(t)
	f.updater.cfg.CurrentVersion = "0.0.0-dev"

	_, err := f.updater.Start(context.Background())
	require.ErrorIs(t, err, ErrUnsupported)
	assert.False(t, f.updater.running.Load())
}

func TestUpdateForeignBinaryUnsupported(t *testing.T) {
	f := newDefaultFixture(t)
	f.updater.cfg.Executable = func() (string, error) { return "/somewhere/else/brrewery", nil }

	_, err := f.updater.Start(context.Background())
	assert.ErrorIs(t, err, ErrUnsupported)
}

// After an install but before the restart, the running process's binary was
// renamed to brrewery.bak (and /proc/self/exe follows the rename, gaining a
// " (deleted)" suffix once another staging removes the backup). Updating
// again from that state must stay possible.
func TestUpdateAllowedFromRenamedBackupBinary(t *testing.T) {
	for _, suffix := range []string{".bak", ".bak (deleted)"} {
		t.Run(suffix, func(t *testing.T) {
			f := newDefaultFixture(t)
			f.updater.cfg.Executable = func() (string, error) { return f.cfg.BinaryPath + suffix, nil }

			job, err := f.updater.Start(context.Background())
			require.NoError(t, err)
			waitForSuccess(t, f, job.ID)
		})
	}
}

func TestReconcileVersionMismatchFailsJob(t *testing.T) {
	f := newDefaultFixture(t)
	// The install succeeded and its restart happened, but the old version is
	// somehow still running.
	job := f.store.Create(selfUpdateAppID, model.JobActionSelfUpdate)
	f.store.SetStatus(job.ID, model.JobStatusSucceeded, "")
	marker, err := json.Marshal(pendingMarker{JobID: job.ID, TargetVersion: testTargetVersion, StartedAt: time.Now()})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(f.cfg.MarkerPath, marker, 0o600))

	// Still on the old version after the restart.
	f.updater.ReconcileOnStartup()

	current, ok := f.store.Get(job.ID)
	require.True(t, ok)
	assert.Equal(t, model.JobStatusFailed, current.Status)
	assert.Contains(t, current.Error, "still running version")
	assert.NoFileExists(t, f.cfg.MarkerPath)
}

func TestReconcileSweepsInterruptedJobs(t *testing.T) {
	f := newDefaultFixture(t)
	stale := f.store.Create(selfUpdateAppID, model.JobActionSelfUpdate)
	f.store.MarkRunning(stale.ID)
	appJob := f.store.Create("qbittorrent", model.JobActionInstall)
	f.store.MarkRunning(appJob.ID)

	f.updater.ReconcileOnStartup()

	swept, ok := f.store.Get(stale.ID)
	require.True(t, ok)
	assert.Equal(t, model.JobStatusFailed, swept.Status)
	assert.Equal(t, "interrupted by restart", swept.Error)

	// App jobs are not the updater's to touch.
	untouched, ok := f.store.Get(appJob.ID)
	require.True(t, ok)
	assert.Equal(t, model.JobStatusRunning, untouched.Status)
}

func TestExtractRejectsTraversal(t *testing.T) {
	archive := buildArchive(t, []archiveEntry{{"../evil", 0o644, "boom"}})
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "a.tar.gz")
	require.NoError(t, os.WriteFile(archivePath, archive, 0o600))

	err := extractTarGz(archivePath, filepath.Join(dir, "out"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "escapes extraction dir")
	assert.NoFileExists(t, filepath.Join(dir, "evil"))
}

func TestExtractRejectsSymlinks(t *testing.T) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	require.NoError(t, tw.WriteHeader(&tar.Header{
		Name:     "link",
		Linkname: "/etc/passwd",
		Typeflag: tar.TypeSymlink,
	}))
	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())

	dir := t.TempDir()
	archivePath := filepath.Join(dir, "a.tar.gz")
	require.NoError(t, os.WriteFile(archivePath, buf.Bytes(), 0o600))

	err := extractTarGz(archivePath, filepath.Join(dir, "out"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported archive entry type")
}
