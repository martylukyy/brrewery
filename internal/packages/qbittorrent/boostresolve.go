package qbittorrent

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const boostReleaseIndex = "https://archives.boost.io/release/"

var (
	// ErrBoostResolveFailed indicates no suitable Boost release was found on archives.boost.io.
	ErrBoostResolveFailed = errors.New("could not resolve latest Boost release from archives.boost.io")
)

var boostReleaseDirRE = regexp.MustCompile(`href="(\d+\.\d+\.\d+)/"`)

// BoostResolver looks up Boost releases on archives.boost.io.
type BoostResolver struct {
	Client  *http.Client
	BaseURL string
}

// DefaultBoostResolver returns a resolver with a production HTTP client.
func DefaultBoostResolver() *BoostResolver {
	return &BoostResolver{
		Client:  &http.Client{Timeout: 60 * time.Second},
		BaseURL: boostReleaseIndex,
	}
}

// ResolveLatest returns the newest Boost release on archives.boost.io with a published
// source tarball. When maxVersion is non-empty, only versions <= maxVersion are considered
// (used to cap RC_1_2 at 1.86 because Boost 1.87+ drops boost::asio::io_service).
func (r *BoostResolver) ResolveLatest(ctx context.Context, maxVersion string) (string, error) {
	maxVersion = strings.TrimSpace(maxVersion)

	releases, err := r.listReleases(ctx)
	if err != nil {
		return "", err
	}

	var candidates []string
	for _, release := range releases {
		if maxVersion != "" && compareVersions(release, maxVersion) > 0 {
			continue
		}
		underscore := boostVersionToUnderscore(release)
		if r.sourceArchiveAvailable(ctx, release, underscore) {
			candidates = append(candidates, underscore)
		}
	}
	if len(candidates) == 0 {
		if maxVersion != "" {
			return "", fmt.Errorf("%w: no Boost release <= %s with source tarball", ErrBoostResolveFailed, maxVersion)
		}
		return "", fmt.Errorf("%w: no Boost source tarball found", ErrBoostResolveFailed)
	}
	return maxVersionUnderscore(candidates), nil
}

func (r *BoostResolver) listReleases(ctx context.Context) ([]string, error) {
	html, err := r.fetch(ctx, r.indexURL())
	if err != nil {
		return nil, err
	}

	seen := make(map[string]struct{})
	var out []string
	for _, match := range boostReleaseDirRE.FindAllStringSubmatch(html, -1) {
		release := match[1]
		if _, ok := seen[release]; ok {
			continue
		}
		seen[release] = struct{}{}
		out = append(out, release)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%w: no release directories found", ErrBoostResolveFailed)
	}
	return out, nil
}

func (r *BoostResolver) sourceArchiveAvailable(ctx context.Context, dotted, underscore string) bool {
	url := boostSourceArchiveURL(dotted, underscore)
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

func (r *BoostResolver) fetch(ctx context.Context, url string) (string, error) {
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

func (r *BoostResolver) indexURL() string {
	if r.BaseURL == "" {
		return boostReleaseIndex
	}
	return r.BaseURL
}

func boostSourceArchiveURL(dotted, underscore string) string {
	return fmt.Sprintf("%s%ssource/boost_%s.tar.gz", boostReleaseIndex, dotted+"/", underscore)
}

func boostVersionToUnderscore(dotted string) string {
	return strings.ReplaceAll(dotted, ".", "_")
}

func boostUnderscoreToDotted(underscore string) string {
	return strings.ReplaceAll(underscore, "_", ".")
}

func maxVersionUnderscore(versions []string) string {
	best := versions[0]
	for _, candidate := range versions[1:] {
		if compareVersions(boostUnderscoreToDotted(candidate), boostUnderscoreToDotted(best)) > 0 {
			best = candidate
		}
	}
	return best
}
