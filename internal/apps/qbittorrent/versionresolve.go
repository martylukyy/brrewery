package qbittorrent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const qbittorrentTagsURL = "https://api.github.com/repos/qbittorrent/qBittorrent/tags?per_page=100"

var (
	// ErrReleaseResolveFailed indicates no suitable qBittorrent release tag was found.
	ErrReleaseResolveFailed = errors.New("could not resolve latest qBittorrent release from GitHub")
)

// ReleaseResolver looks up the newest qBittorrent patch for a manifest minor line.
type ReleaseResolver struct {
	Client  *http.Client
	TagsURL string
}

// DefaultReleaseResolver returns a resolver with a production HTTP client.
func DefaultReleaseResolver() *ReleaseResolver {
	return &ReleaseResolver{
		Client:  &http.Client{Timeout: 60 * time.Second},
		TagsURL: qbittorrentTagsURL,
	}
}

type githubTag struct {
	Name string `json:"name"`
}

// ResolveLatest returns the newest release-{major.minor}.patch tag on GitHub.
func (r *ReleaseResolver) ResolveLatest(ctx context.Context, minor string) (string, error) {
	minor = strings.TrimSpace(minor)
	if minor == "" {
		return "", fmt.Errorf("%w: empty minor line", ErrReleaseResolveFailed)
	}

	tags, err := r.listTags(ctx)
	if err != nil {
		return "", err
	}

	var candidates []string
	for _, tag := range tags {
		if version, ok := releaseVersionForMinor(tag.Name, minor); ok {
			candidates = append(candidates, version)
		}
	}
	if len(candidates) == 0 {
		return "", fmt.Errorf("%w: no release tags for minor %s", ErrReleaseResolveFailed, minor)
	}
	return maxVersion(candidates), nil
}

func (r *ReleaseResolver) listTags(ctx context.Context) ([]githubTag, error) {
	url := r.TagsURL
	if url == "" {
		url = qbittorrentTagsURL
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := r.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch GitHub tags: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("fetch GitHub tags: HTTP %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var tags []githubTag
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return nil, fmt.Errorf("decode GitHub tags: %w", err)
	}
	return tags, nil
}

func releaseVersionForMinor(tag, minor string) (string, bool) {
	const prefix = "release-"
	if !strings.HasPrefix(tag, prefix) {
		return "", false
	}
	version := strings.TrimPrefix(tag, prefix)
	minParts := strings.Split(minor, ".")
	verParts := strings.Split(version, ".")
	if len(verParts) < len(minParts)+1 {
		return "", false
	}
	for i, part := range minParts {
		if verParts[i] != part {
			return "", false
		}
	}
	for _, part := range verParts {
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
