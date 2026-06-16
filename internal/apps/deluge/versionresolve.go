package deluge

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	defaultDelugeTagsURL = "https://api.github.com/repos/deluge-torrent/deluge/tags"
	// delugeTagPrefix is the git tag prefix Deluge uses for releases.
	delugeTagPrefix = "deluge-"
	// tagsPerPage caps each tags page; pagination walks until a short page.
	tagsPerPage = 100
	// maxTagPages bounds pagination so a misbehaving API cannot loop forever.
	maxTagPages = 6
)

// ErrReleaseResolveFailed indicates no suitable Deluge release tag was found.
var ErrReleaseResolveFailed = errors.New("could not resolve Deluge release from GitHub")

// ReleaseResolver resolves the newest Deluge patch for a manifest line series
// from the deluge-torrent/deluge GitHub repository.
type ReleaseResolver struct {
	Client *http.Client
	// TagsURL is the deluge tags API base (overridable in tests).
	TagsURL string
}

// DefaultReleaseResolver returns a resolver with a production HTTP client.
func DefaultReleaseResolver() *ReleaseResolver {
	return &ReleaseResolver{
		Client:  &http.Client{Timeout: 60 * time.Second},
		TagsURL: defaultDelugeTagsURL,
	}
}

type githubTag struct {
	Name string `json:"name"`
}

// ResolveLatest returns the newest deluge-<series>.<patch> release version
// (e.g. "2.2.0" for series "2.2"), the bare version without the tag prefix.
func (r *ReleaseResolver) ResolveLatest(ctx context.Context, series string) (string, error) {
	series = strings.TrimSpace(series)
	if series == "" {
		return "", fmt.Errorf("%w: empty series", ErrReleaseResolveFailed)
	}

	tags, err := r.listAllTags(ctx)
	if err != nil {
		return "", err
	}

	prefix := delugeTagPrefix + series + "."
	best := ""
	for _, t := range tags {
		if !strings.HasPrefix(t.Name, prefix) {
			continue
		}
		version := strings.TrimPrefix(t.Name, delugeTagPrefix)
		// Only accept plain numeric versions (skip dev/rc/post/alpha tags).
		if !numericVersion(version) {
			continue
		}
		if best == "" || compareVersions(version, best) > 0 {
			best = version
		}
	}
	if best == "" {
		return "", fmt.Errorf("%w: no %s* release tags found", ErrReleaseResolveFailed, prefix)
	}
	return best, nil
}

func (r *ReleaseResolver) listAllTags(ctx context.Context) ([]githubTag, error) {
	base := r.TagsURL
	if base == "" {
		base = defaultDelugeTagsURL
	}

	var all []githubTag
	for page := 1; page <= maxTagPages; page++ {
		sep := "?"
		if strings.Contains(base, "?") {
			sep = "&"
		}
		url := fmt.Sprintf("%s%sper_page=%d&page=%d", base, sep, tagsPerPage, page)

		var tags []githubTag
		if err := r.getJSON(ctx, url, &tags); err != nil {
			return nil, err
		}
		all = append(all, tags...)
		if len(tags) < tagsPerPage {
			break
		}
	}
	return all, nil
}

func (r *ReleaseResolver) getJSON(ctx context.Context, url string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := r.Client.Do(req)
	if err != nil {
		return fmt.Errorf("fetch %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("fetch %s: HTTP %s: %s", url, resp.Status, strings.TrimSpace(string(body)))
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode %s: %w", url, err)
	}
	return nil
}

func numericVersion(v string) bool {
	parts := strings.Split(v, ".")
	if len(parts) == 0 {
		return false
	}
	for _, p := range parts {
		if p == "" {
			return false
		}
		if _, err := strconv.Atoi(p); err != nil {
			return false
		}
	}
	return true
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
