package deluge

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/autobrr/brrewery/internal/apps/extravars"
)

// fakeGitHub serves the deluge tags endpoint the resolver paginates over.
func fakeGitHub(t *testing.T, tags []string) *ReleaseResolver {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/tags", func(w http.ResponseWriter, _ *http.Request) {
		out := make([]githubTag, 0, len(tags))
		for _, name := range tags {
			out = append(out, githubTag{Name: name})
		}
		_ = json.NewEncoder(w).Encode(out)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	return &ReleaseResolver{
		Client:  srv.Client(),
		TagsURL: srv.URL + "/tags",
	}
}

var sampleDelugeTags = []string{
	"deluge-2.2.0", "deluge-2.1.1", "deluge-2.1.0",
	"deluge-2.0.5", "deluge-2.0.3", "deluge-2.0.0",
	"deluge-1.3.15", "deluge-1.3.14", "deluge-1.3.0",
	// noise that must be ignored
	"deluge-2.2.1.dev43", "deluge-2.0.0rc1", "deluge-1.3.15.post0",
}

func TestResolveLatestPicksNewestPatch(t *testing.T) {
	r := fakeGitHub(t, sampleDelugeTags)

	for series, want := range map[string]string{
		"2.2": "2.2.0",
		"2.1": "2.1.1",
		"2.0": "2.0.5",
		"1.3": "1.3.15",
	} {
		got, err := r.ResolveLatest(context.Background(), series)
		require.NoError(t, err, "series %s", series)
		assert.Equal(t, want, got, "series %s", series)
	}
}

func TestResolveLatestNoMatchFails(t *testing.T) {
	r := fakeGitHub(t, sampleDelugeTags)
	_, err := r.ResolveLatest(context.Background(), "3.0")
	require.ErrorIs(t, err, ErrReleaseResolveFailed)
}

func TestEnrichAnsibleVarsDefaultsBranch(t *testing.T) {
	r := fakeGitHub(t, sampleDelugeTags)
	vars := map[string]string{extravars.DelugeVersion: "2.2.x"}
	require.NoError(t, EnrichAnsibleVars(context.Background(), vars, r))
	assert.Equal(t, "2.2.x", vars[extravars.DelugeVersion])
	assert.Equal(t, "2.2.0", vars[extravars.DelugeRelease])
	assert.Equal(t, BranchRC12, vars[extravars.LibtorrentBranch])
}

func TestEnrichAnsibleVarsHonorsBranch(t *testing.T) {
	r := fakeGitHub(t, sampleDelugeTags)
	vars := map[string]string{
		extravars.DelugeVersion:    "2.1.x",
		extravars.LibtorrentBranch: BranchRC20,
	}
	require.NoError(t, EnrichAnsibleVars(context.Background(), vars, r))
	assert.Equal(t, "2.1.1", vars[extravars.DelugeRelease])
	assert.Equal(t, BranchRC20, vars[extravars.LibtorrentBranch])
}

func TestEnrichAnsibleVarsLegacyForcesRC11(t *testing.T) {
	r := fakeGitHub(t, sampleDelugeTags)
	vars := map[string]string{extravars.DelugeVersion: "1.3.x"}
	require.NoError(t, EnrichAnsibleVars(context.Background(), vars, r))
	assert.Equal(t, "1.3.15", vars[extravars.DelugeRelease])
	assert.Equal(t, BranchRC11, vars[extravars.LibtorrentBranch])
}

func TestEnrichAnsibleVarsRejectsBadBranch(t *testing.T) {
	r := fakeGitHub(t, sampleDelugeTags)
	vars := map[string]string{
		extravars.DelugeVersion:    "1.3.x",
		extravars.LibtorrentBranch: BranchRC20,
	}
	err := EnrichAnsibleVars(context.Background(), vars, r)
	require.ErrorIs(t, err, ErrBranchNotAllowed)
}

func TestEnrichAnsibleVarsRequiresVersion(t *testing.T) {
	err := EnrichAnsibleVars(context.Background(), map[string]string{}, DefaultReleaseResolver())
	require.Error(t, err)
}

func TestValidate(t *testing.T) {
	// Non-deluge app passes through.
	require.NoError(t, Validate("other", map[string]string{}))

	require.NoError(t, Validate(AppID, map[string]string{
		extravars.DelugeVersion: "2.2.x", extravars.LibtorrentBranch: BranchRC20,
	}))

	// Empty branch is accepted (role falls back to the default).
	require.NoError(t, Validate(AppID, map[string]string{
		extravars.DelugeVersion: "1.3.x",
	}))

	err := Validate(AppID, map[string]string{
		extravars.DelugeVersion: "1.3.x", extravars.LibtorrentBranch: BranchRC20,
	})
	require.ErrorIs(t, err, ErrBranchNotAllowed)

	err = Validate(AppID, map[string]string{extravars.DelugeVersion: "9.9.x"})
	require.ErrorIs(t, err, ErrUnknownVersion)
}
