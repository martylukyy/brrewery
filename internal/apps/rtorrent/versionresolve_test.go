package rtorrent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeGitHub serves the tags list and release-by-tag endpoints the resolver uses.
func fakeGitHub(t *testing.T, tags []string, releases map[string][]string) *ReleaseResolver {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/tags", func(w http.ResponseWriter, _ *http.Request) {
		out := make([]githubTag, 0, len(tags))
		for _, name := range tags {
			out = append(out, githubTag{Name: name})
		}
		_ = json.NewEncoder(w).Encode(out)
	})
	mux.HandleFunc("/releases/tags/", func(w http.ResponseWriter, r *http.Request) {
		tag := strings.TrimPrefix(r.URL.Path, "/releases/tags/")
		assets, ok := releases[tag]
		if !ok {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		rel := githubRelease{TagName: tag}
		for _, name := range assets {
			rel.Assets = append(rel.Assets, struct {
				Name string `json:"name"`
			}{Name: name})
		}
		_ = json.NewEncoder(w).Encode(rel)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	return &ReleaseResolver{
		Client:      srv.Client(),
		TagsURL:     srv.URL + "/tags",
		ReleaseTmpl: srv.URL + "/releases/tags/%s",
	}
}

func TestResolveVersionsLatestUsesReleaseAsset(t *testing.T) {
	r := fakeGitHub(t,
		[]string{"v0.16.14", "v0.16.2", "v0.16.0", "v0.15.7", "v0.10.0", "v0.9.8"},
		map[string][]string{
			"v0.16.14": {"rtorrent-0.16.14.tar.gz", "libtorrent-0.16.14.tar.gz"},
		},
	)
	rt, lt, err := r.ResolveVersions(context.Background(), Line{
		Version: "0.16.x", Series: "0.16", Source: SourceRelease, Resolve: ResolveLatest,
	})
	require.NoError(t, err)
	assert.Equal(t, "0.16.14", rt)
	assert.Equal(t, "0.16.14", lt)
}

func TestResolveVersionsPinnedReleaseUsesBundledLibtorrent(t *testing.T) {
	r := fakeGitHub(t, nil, map[string][]string{
		"v0.10.0": {"rtorrent-0.10.0.tar.gz", "libtorrent-0.14.0.tar.gz"},
	})
	rt, lt, err := r.ResolveVersions(context.Background(), Line{
		Version: "0.10.0", Tag: "v0.10.0", Source: SourceRelease, Resolve: ResolvePinned,
	})
	require.NoError(t, err)
	assert.Equal(t, "0.10.0", rt)
	assert.Equal(t, "0.14.0", lt)
}

func TestResolveVersionsReleaseFallsBackToMatchedLibtorrent(t *testing.T) {
	// Release has the rtorrent asset but no libtorrent asset -> matched fallback.
	r := fakeGitHub(t, nil, map[string][]string{
		"v0.9.8": {"rtorrent-0.9.8.tar.gz"},
	})
	rt, lt, err := r.ResolveVersions(context.Background(), Line{
		Version: "0.9.8", Tag: "v0.9.8", Source: SourceRelease, Resolve: ResolvePinned,
	})
	require.NoError(t, err)
	assert.Equal(t, "0.9.8", rt)
	assert.Equal(t, "0.9.8", lt)
}

func TestResolveVersionsGitArchivePinsLibtorrent(t *testing.T) {
	r := fakeGitHub(t, nil, nil) // no HTTP calls expected
	rt, lt, err := r.ResolveVersions(context.Background(), Line{
		Version: "0.9.6", Tag: "v0.9.6", Source: SourceGitArchive, Resolve: ResolvePinned,
		LibtorrentTag: "v0.13.6",
	})
	require.NoError(t, err)
	assert.Equal(t, "0.9.6", rt)
	assert.Equal(t, "0.13.6", lt)
}

func TestResolveVersionsLatestNoTagsFails(t *testing.T) {
	r := fakeGitHub(t, []string{"v0.9.8", "v0.10.0"}, nil)
	_, _, err := r.ResolveVersions(context.Background(), Line{
		Version: "0.16.x", Series: "0.16", Source: SourceRelease, Resolve: ResolveLatest,
	})
	assert.ErrorIs(t, err, ErrReleaseResolveFailed)
}

func TestEnrichAnsibleVars(t *testing.T) {
	r := fakeGitHub(t, nil, map[string][]string{
		"v0.10.0": {"rtorrent-0.10.0.tar.gz", "libtorrent-0.14.0.tar.gz"},
	})
	vars := map[string]string{"rtorrent_version": "0.10.0"}
	require.NoError(t, EnrichAnsibleVars(context.Background(), vars, r))
	assert.Equal(t, "0.10.0", vars["rtorrent_version"])
	assert.Equal(t, "0.10.0", vars["rtorrent_release"])
	assert.Equal(t, "0.14.0", vars["libtorrent_release"])
}

func TestEnrichAnsibleVarsRequiresVersion(t *testing.T) {
	err := EnrichAnsibleVars(context.Background(), map[string]string{}, DefaultReleaseResolver())
	require.Error(t, err)
}
