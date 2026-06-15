package rtorrent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	defaultTagsURL     = "https://api.github.com/repos/rakshasa/rtorrent/tags?per_page=100"
	defaultReleaseTmpl = "https://api.github.com/repos/rakshasa/rtorrent/releases/tags/%s"
)

// ErrReleaseResolveFailed indicates no suitable rtorrent release could be resolved.
var ErrReleaseResolveFailed = errors.New("could not resolve rtorrent release from GitHub")

// ReleaseResolver resolves the concrete rtorrent and libtorrent versions for a
// manifest line from the rakshasa/rtorrent GitHub repository.
type ReleaseResolver struct {
	Client *http.Client
	// TagsURL lists rtorrent tags (overridable in tests).
	TagsURL string
	// ReleaseTmpl is a fmt template taking the tag, returning the release-by-tag
	// API URL (overridable in tests).
	ReleaseTmpl string
}

// DefaultReleaseResolver returns a resolver with a production HTTP client.
func DefaultReleaseResolver() *ReleaseResolver {
	return &ReleaseResolver{
		Client:      &http.Client{Timeout: 60 * time.Second},
		TagsURL:     defaultTagsURL,
		ReleaseTmpl: defaultReleaseTmpl,
	}
}

type githubTag struct {
	Name string `json:"name"`
}

type githubRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name string `json:"name"`
	} `json:"assets"`
}

var (
	rtorrentAssetRE   = regexp.MustCompile(`^rtorrent-(\d+\.\d+\.\d+)\.tar\.gz$`)
	libtorrentAssetRE = regexp.MustCompile(`^libtorrent-(\d+\.\d+\.\d+)\.tar\.gz$`)
)

// ResolveVersions returns the rtorrent release and the matching libtorrent
// release for the given line. The rtorrent version comes from the resolved tag
// (newest patch in the series for "latest" lines, or the pinned tag otherwise);
// the libtorrent version comes from the libtorrent-*.tar.gz asset bundled in the
// rtorrent release (falling back to the matched rtorrent version), or from the
// manifest's libtorrent_tag for git-archive lines.
func (r *ReleaseResolver) ResolveVersions(ctx context.Context, line Line) (rtorrentRelease, libtorrentRelease string, err error) {
	tag := strings.TrimSpace(line.Tag)
	if line.Resolve == ResolveLatest {
		tag, err = r.resolveLatestTag(ctx, line.Series)
		if err != nil {
			return "", "", err
		}
	}
	if tag == "" {
		return "", "", fmt.Errorf("%w: no tag for line %q", ErrReleaseResolveFailed, line.Version)
	}
	rtorrentRelease = stripV(tag)

	if line.Source == SourceGitArchive {
		libtorrentRelease = stripV(line.LibtorrentTag)
		if libtorrentRelease == "" {
			return "", "", fmt.Errorf("%w: line %q is git-archive but has no libtorrent_tag", ErrReleaseResolveFailed, line.Version)
		}
		return rtorrentRelease, libtorrentRelease, nil
	}

	libtorrentRelease, err = r.resolveLibtorrentFromRelease(ctx, tag)
	if err != nil {
		return "", "", err
	}
	if libtorrentRelease == "" {
		// No libtorrent asset on the release: rakshasa pairs rtorrent X.Y.Z with
		// libtorrent X.Y.Z for the release-era lines, so fall back to that.
		libtorrentRelease = rtorrentRelease
	}
	return rtorrentRelease, libtorrentRelease, nil
}

// resolveLatestTag returns the newest v<series>.<patch> tag for the series.
func (r *ReleaseResolver) resolveLatestTag(ctx context.Context, series string) (string, error) {
	series = strings.TrimSpace(series)
	if series == "" {
		return "", fmt.Errorf("%w: empty series", ErrReleaseResolveFailed)
	}
	tags, err := r.listTags(ctx)
	if err != nil {
		return "", err
	}

	prefix := "v" + series + "."
	best := ""
	for _, t := range tags {
		if !strings.HasPrefix(t.Name, prefix) {
			continue
		}
		// Only accept v<series>.<patch> with a numeric patch (no suffixes).
		if !numericVersion(stripV(t.Name)) {
			continue
		}
		if best == "" || compareVersions(stripV(t.Name), stripV(best)) > 0 {
			best = t.Name
		}
	}
	if best == "" {
		return "", fmt.Errorf("%w: no v%s.* tags found", ErrReleaseResolveFailed, series)
	}
	return best, nil
}

func (r *ReleaseResolver) resolveLibtorrentFromRelease(ctx context.Context, tag string) (string, error) {
	rel, err := r.fetchRelease(ctx, tag)
	if err != nil {
		return "", err
	}
	for _, a := range rel.Assets {
		if m := libtorrentAssetRE.FindStringSubmatch(a.Name); m != nil {
			return m[1], nil
		}
	}
	return "", nil
}

func (r *ReleaseResolver) listTags(ctx context.Context) ([]githubTag, error) {
	url := r.TagsURL
	if url == "" {
		url = defaultTagsURL
	}
	var tags []githubTag
	if err := r.getJSON(ctx, url, &tags); err != nil {
		return nil, err
	}
	return tags, nil
}

func (r *ReleaseResolver) fetchRelease(ctx context.Context, tag string) (*githubRelease, error) {
	tmpl := r.ReleaseTmpl
	if tmpl == "" {
		tmpl = defaultReleaseTmpl
	}
	var rel githubRelease
	if err := r.getJSON(ctx, fmt.Sprintf(tmpl, tag), &rel); err != nil {
		return nil, err
	}
	return &rel, nil
}

func (r *ReleaseResolver) getJSON(ctx context.Context, url string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
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
