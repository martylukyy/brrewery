package qbittorrent

import (
	"context"
	"crypto/rand"
	"crypto/sha512"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/pbkdf2"

	"github.com/autobrr/brrewery/internal/packages/extravars"
)

const qtArchiveIndex = "https://download.qt.io/archive/qt/"

var (
	// ErrQtResolveFailed indicates no suitable Qt release was found on download.qt.io.
	ErrQtResolveFailed = errors.New("could not resolve a compatible Qt version from download.qt.io")
	// ErrQtBelowMinimum indicates a fixed Qt override is below the line minimum.
	ErrQtBelowMinimum = errors.New("Qt version is below the minimum for this qBittorrent line")
)

var seriesDirRE = regexp.MustCompile(`href="(\d+\.\d+)/"`)

// QtResolver looks up the newest Qt patch on download.qt.io that satisfies a minimum version.
type QtResolver struct {
	Client  *http.Client
	BaseURL string
}

// DefaultQtResolver returns a resolver with a production HTTP client.
func DefaultQtResolver() *QtResolver {
	return &QtResolver{
		Client:  &http.Client{Timeout: 60 * time.Second},
		BaseURL: qtArchiveIndex,
	}
}

// ResolveLatest returns the newest Qt patch >= min with published qtbase sources.
// When override is non-empty it is returned after verifying override >= min.
func (r *QtResolver) ResolveLatest(ctx context.Context, min, override string) (string, error) {
	min = strings.TrimSpace(min)
	override = strings.TrimSpace(override)
	if min == "" {
		return "", fmt.Errorf("%w: empty minimum", ErrQtResolveFailed)
	}
	if override != "" {
		if !versionAtLeast(override, min) {
			return "", fmt.Errorf("%w: %s < %s", ErrQtBelowMinimum, override, min)
		}
		return override, nil
	}

	major := strings.Split(min, ".")[0]
	seriesList, err := r.listSeries(ctx, major)
	if err != nil {
		return "", err
	}

	minSeries := seriesPrefix(min)
	var candidates []string
	for _, series := range seriesList {
		if !versionAtLeast(series, minSeries) {
			continue
		}
		patch, patchErr := r.latestPatch(ctx, series)
		if patchErr != nil || !versionAtLeast(patch, min) {
			continue
		}
		if r.archiveAvailable(ctx, r.qtbaseArchiveURL(patch)) {
			candidates = append(candidates, patch)
		}
	}
	if len(candidates) == 0 {
		return "", fmt.Errorf("%w: no qtbase sources for Qt >= %s", ErrQtResolveFailed, min)
	}
	return maxVersion(candidates), nil
}

// EnrichAnsibleVars resolves qBittorrent, Qt, zlib, Boost, and OpenSSL versions for Ansible.
func EnrichAnsibleVars(
	ctx context.Context,
	vars map[string]string,
	releaseResolver *ReleaseResolver,
	qtResolver *QtResolver,
	zlibResolver *ZlibResolver,
	boostResolver *BoostResolver,
	opensslResolver *OpensslResolver,
) error {
	if releaseResolver == nil {
		releaseResolver = DefaultReleaseResolver()
	}
	if qtResolver == nil {
		qtResolver = DefaultQtResolver()
	}
	if zlibResolver == nil {
		zlibResolver = DefaultZlibResolver()
	}
	if boostResolver == nil {
		boostResolver = DefaultBoostResolver()
	}
	if opensslResolver == nil {
		opensslResolver = DefaultOpensslResolver()
	}

	version := strings.TrimSpace(vars[extravars.QbittorrentVersion])
	if version == "" {
		return errors.New("qbittorrent_version is required")
	}

	m, err := LoadManifest()
	if err != nil {
		return err
	}
	line, err := m.ResolveSelection(version)
	if err != nil {
		return err
	}
	vars[extravars.QbittorrentVersion] = line.Version

	patch, err := releaseResolver.ResolveLatest(ctx, line.Version)
	if err != nil {
		return err
	}
	vars[extravars.QbittorrentRelease] = patch

	qtVersion, err := qtResolver.ResolveLatest(ctx, line.Qt.Min, line.QtVersionOverride())
	if err != nil {
		return err
	}
	vars[extravars.QbittorrentQtVersion] = qtVersion

	zlibVersion, err := zlibResolver.ResolveLatest(ctx)
	if err != nil {
		return err
	}
	vars[extravars.QbittorrentZlibVersion] = zlibVersion

	branch := strings.TrimSpace(vars[extravars.LibtorrentBranch])
	if branch == "" {
		branch = line.Libtorrent.Default
	}
	var boostVersion string
	if branch == BranchRC12 {
		boostVersion = strings.TrimSpace(m.Defaults.BoostRC12)
		if boostVersion == "" {
			return errors.New("manifest defaults.boost_rc_1_2 is required for RC_1_2 builds")
		}
	} else {
		boostVersion, err = boostResolver.ResolveLatest(ctx, "")
		if err != nil {
			return err
		}
	}
	vars[extravars.QbittorrentBoostVersion] = boostVersion

	opensslVersion, err := opensslResolver.ResolveLatest(ctx)
	if err != nil {
		return err
	}
	vars[extravars.QbittorrentOpensslVersion] = opensslVersion

	pw := vars[extravars.BrreweryUserPassword]
	if pw == "" {
		return errors.New("brrewery_user_password is required for qBittorrent WebUI credentials")
	}
	hash, err := qbtWebUIPasswordHash(pw)
	if err != nil {
		return fmt.Errorf("hash qBittorrent WebUI password: %w", err)
	}
	vars[extravars.QbittorrentWebUIPasswordHash] = hash
	delete(vars, extravars.BrreweryUserPassword)

	return nil
}

// qBittorrent's WebUI authentication (Utils::Password::PBKDF2 in
// src/base/utils/password.cpp) verifies the stored credential by recomputing
// PBKDF2-HMAC-SHA512 over the entered password with the stored salt. These
// parameters must match it exactly or the WebUI login rejects the password:
// a 16-byte salt, 100000 iterations and a 64-byte derived key, serialized as
// @ByteArray(<salt_b64>:<hash_b64>).
const (
	qbtPBKDF2SaltLen    = 16
	qbtPBKDF2Iterations = 100000
	qbtPBKDF2KeyLen     = 64
)

// qbtWebUIPasswordHash returns a qBittorrent-compatible PBKDF2-HMAC-SHA512 hash
// in the @ByteArray(<salt_b64>:<hash_b64>) format written to qBittorrent.conf.
func qbtWebUIPasswordHash(password string) (string, error) {
	salt := make([]byte, qbtPBKDF2SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := pbkdf2.Key([]byte(password), salt, qbtPBKDF2Iterations, qbtPBKDF2KeyLen, sha512.New)
	return "@ByteArray(" + base64.StdEncoding.EncodeToString(salt) + ":" + base64.StdEncoding.EncodeToString(key) + ")", nil
}

func (r *QtResolver) listSeries(ctx context.Context, major string) ([]string, error) {
	html, err := r.fetch(ctx, r.indexURL())
	if err != nil {
		return nil, err
	}
	prefix := major + "."
	var out []string
	seen := make(map[string]struct{})
	for _, match := range seriesDirRE.FindAllStringSubmatch(html, -1) {
		series := match[1]
		if !strings.HasPrefix(series, prefix) {
			continue
		}
		if _, ok := seen[series]; ok {
			continue
		}
		seen[series] = struct{}{}
		out = append(out, series)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%w: no Qt %s.x series found", ErrQtResolveFailed, major)
	}
	return out, nil
}

func (r *QtResolver) latestPatch(ctx context.Context, series string) (string, error) {
	html, err := r.fetch(ctx, r.indexURL()+series+"/")
	if err != nil {
		return "", err
	}
	re := regexp.MustCompile(fmt.Sprintf(`href="%s\.(\d+)/"`, regexp.QuoteMeta(series)))
	best := -1
	for _, match := range re.FindAllStringSubmatch(html, -1) {
		n, convErr := strconv.Atoi(match[1])
		if convErr == nil && n > best {
			best = n
		}
	}
	if best < 0 {
		return "", fmt.Errorf("%w: no patches under %s", ErrQtResolveFailed, series)
	}
	return fmt.Sprintf("%s.%d", series, best), nil
}

func (r *QtResolver) archiveAvailable(ctx context.Context, url string) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		return false
	}
	resp, err := r.Client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	return resp.StatusCode >= 200 && resp.StatusCode < 400
}

func (r *QtResolver) fetch(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := r.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("fetch %s: HTTP %s", url, resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", url, err)
	}
	return string(body), nil
}

func (r *QtResolver) indexURL() string {
	if r.BaseURL == "" {
		return qtArchiveIndex
	}
	return r.BaseURL
}

func (r *QtResolver) qtbaseArchiveURL(version string) string {
	parts := strings.Split(version, ".")
	majmin := strings.Join(parts[:2], ".")
	return fmt.Sprintf(
		"%s%s/%s/submodules/qtbase-everywhere-src-%s.tar.xz",
		r.indexURL(), majmin, version, version,
	)
}

func seriesPrefix(min string) string {
	parts := strings.Split(min, ".")
	if len(parts) < 2 {
		return min
	}
	return parts[0] + "." + parts[1]
}

func versionAtLeast(left, right string) bool {
	return compareVersions(left, right) >= 0
}

func compareVersions(left, right string) int {
	lp := parseVersionParts(left)
	rp := parseVersionParts(right)
	n := len(lp)
	if len(rp) > n {
		n = len(rp)
	}
	for i := 0; i < n; i++ {
		var lv, rv int
		if i < len(lp) {
			lv = lp[i]
		}
		if i < len(rp) {
			rv = rp[i]
		}
		if lv != rv {
			return lv - rv
		}
	}
	return 0
}

func parseVersionParts(version string) []int {
	segments := strings.Split(version, ".")
	out := make([]int, 0, len(segments))
	for _, segment := range segments {
		n, err := strconv.Atoi(segment)
		if err != nil {
			continue
		}
		out = append(out, n)
	}
	return out
}

func maxVersion(versions []string) string {
	best := versions[0]
	for _, candidate := range versions[1:] {
		if compareVersions(candidate, best) > 0 {
			best = candidate
		}
	}
	return best
}
