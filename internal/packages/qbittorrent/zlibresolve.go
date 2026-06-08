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

const zlibTagsURL = "https://api.github.com/repos/madler/zlib/tags?per_page=100"

var (
	// ErrZlibResolveFailed indicates no suitable zlib release tag was found.
	ErrZlibResolveFailed = errors.New("could not resolve latest zlib release from GitHub")
)

// ZlibResolver looks up the newest zlib release on GitHub.
type ZlibResolver struct {
	Client  *http.Client
	TagsURL string
}

// DefaultZlibResolver returns a resolver with a production HTTP client.
func DefaultZlibResolver() *ZlibResolver {
	return &ZlibResolver{
		Client:  &http.Client{Timeout: 60 * time.Second},
		TagsURL: zlibTagsURL,
	}
}

// ResolveLatest returns the newest vX.Y.Z tag on github.com/madler/zlib.
func (r *ZlibResolver) ResolveLatest(ctx context.Context) (string, error) {
	tags, err := r.listTags(ctx)
	if err != nil {
		return "", err
	}

	var candidates []string
	for _, tag := range tags {
		if version, ok := zlibVersionFromTag(tag.Name); ok {
			candidates = append(candidates, version)
		}
	}
	if len(candidates) == 0 {
		return "", fmt.Errorf("%w: no release tags found", ErrZlibResolveFailed)
	}
	return maxVersion(candidates), nil
}

func (r *ZlibResolver) listTags(ctx context.Context) ([]githubTag, error) {
	url := r.TagsURL
	if url == "" {
		url = zlibTagsURL
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
		return nil, fmt.Errorf("fetch GitHub tags: HTTP %s", resp.Status)
	}

	var tags []githubTag
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return nil, fmt.Errorf("decode GitHub tags: %w", err)
	}
	return tags, nil
}

func zlibVersionFromTag(tag string) (string, bool) {
	if !strings.HasPrefix(tag, "v") {
		return "", false
	}
	version := strings.TrimPrefix(tag, "v")
	if version == "" {
		return "", false
	}
	for _, part := range strings.Split(version, ".") {
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
