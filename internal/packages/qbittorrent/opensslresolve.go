package qbittorrent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const opensslReleasesURL = "https://api.github.com/repos/openssl/openssl/releases?per_page=100"

var (
	// ErrOpensslResolveFailed indicates no suitable OpenSSL 3.x release was found.
	ErrOpensslResolveFailed = errors.New("could not resolve latest OpenSSL 3.x release from GitHub")
)

const opensslReleaseTagPrefix = "openssl-"

// OpensslResolver looks up OpenSSL 3.x releases on GitHub.
type OpensslResolver struct {
	Client      *http.Client
	ReleasesURL string
}

// DefaultOpensslResolver returns a resolver with a production HTTP client.
func DefaultOpensslResolver() *OpensslResolver {
	return &OpensslResolver{
		Client:      &http.Client{Timeout: 60 * time.Second},
		ReleasesURL: opensslReleasesURL,
	}
}

// ResolveLatest returns the newest OpenSSL 3.x release with a published source tarball.
// OpenSSL 4.x and pre-releases are excluded so qBittorrent builds stay on the 3.x API.
func (r *OpensslResolver) ResolveLatest(ctx context.Context) (string, error) {
	releases, err := r.listReleases(ctx)
	if err != nil {
		return "", err
	}

	var candidates []string
	for _, release := range releases {
		version, ok := opensslVersionFromTag(release.TagName)
		if !ok {
			continue
		}
		if releaseHasSourceTarball(release.Assets, version) {
			candidates = append(candidates, version)
		}
	}
	if len(candidates) == 0 {
		return "", fmt.Errorf("%w: no OpenSSL 3.x source tarball found", ErrOpensslResolveFailed)
	}
	return maxVersion(candidates), nil
}

type opensslRelease struct {
	TagName string
	Assets  []opensslReleaseAsset
}

type opensslReleaseAsset struct {
	Name string
}

func releaseHasSourceTarball(assets []opensslReleaseAsset, version string) bool {
	want := fmt.Sprintf("openssl-%s.tar.gz", version)
	for _, asset := range assets {
		if asset.Name == want {
			return true
		}
	}
	return false
}

func (r *OpensslResolver) listReleases(ctx context.Context) ([]opensslRelease, error) {
	url := r.ReleasesURL
	if url == "" {
		url = opensslReleasesURL
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := r.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch GitHub releases: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("fetch GitHub releases: HTTP %s", resp.Status)
	}

	var payload []struct {
		TagName string `json:"tag_name"`
		Assets  []struct {
			Name string `json:"name"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode GitHub releases: %w", err)
	}

	out := make([]opensslRelease, 0, len(payload))
	for _, release := range payload {
		if release.TagName == "" {
			continue
		}
		assets := make([]opensslReleaseAsset, 0, len(release.Assets))
		for _, asset := range release.Assets {
			if asset.Name == "" {
				continue
			}
			assets = append(assets, opensslReleaseAsset{Name: asset.Name})
		}
		out = append(out, opensslRelease{TagName: release.TagName, Assets: assets})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%w: no releases found", ErrOpensslResolveFailed)
	}
	return out, nil
}

func opensslVersionFromTag(tag string) (string, bool) {
	if !strings.HasPrefix(tag, opensslReleaseTagPrefix) {
		return "", false
	}
	version := strings.TrimPrefix(tag, opensslReleaseTagPrefix)
	if version == "" {
		return "", false
	}
	lower := strings.ToLower(version)
	if strings.Contains(lower, "alpha") || strings.Contains(lower, "beta") || strings.Contains(lower, "rc") {
		return "", false
	}
	parts := strings.Split(version, ".")
	if len(parts) < 3 || parts[0] != "3" {
		return "", false
	}
	for _, part := range parts {
		if part == "" {
			return "", false
		}
		for _, ch := range part {
			if ch < '0' || ch > '9' {
				return "", false
			}
		}
	}
	return version, true
}
