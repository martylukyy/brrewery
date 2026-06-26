package qbittorrent_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/autobrr/brrewery/internal/apps/extravars"
	"github.com/autobrr/brrewery/internal/apps/qbittorrent"
)

// newReleaseResolver returns a ReleaseResolver backed by a stub GitHub tags API.
func newReleaseResolver(t *testing.T) *qbittorrent.ReleaseResolver {
	t.Helper()
	github := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/repos/qbittorrent/qBittorrent/tags" {
			_, _ = w.Write([]byte(`[{"name":"release-5.2.1"}]`))
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(github.Close)
	return &qbittorrent.ReleaseResolver{
		Client:  github.Client(),
		TagsURL: github.URL + "/repos/qbittorrent/qBittorrent/tags",
	}
}

func TestEnrichAnsibleVars_resolvesPatchAndHashesPassword(t *testing.T) {
	t.Parallel()

	vars := map[string]string{
		extravars.QbittorrentVersion:   "5.2",
		extravars.LibtorrentBranch:     qbittorrent.BranchRC20,
		extravars.BrreweryUserPassword: "testpassword",
	}
	err := qbittorrent.EnrichAnsibleVars(context.Background(), vars, newReleaseResolver(t))
	require.NoError(t, err)

	// Only the qBittorrent patch is resolved from upstream; the build-dependency
	// versions are pinned in the manifest and read directly by Ansible, so
	// EnrichAnsibleVars does not set them.
	assert.Equal(t, "5.2", vars[extravars.QbittorrentVersion])
	assert.Equal(t, "5.2.1", vars[extravars.QbittorrentRelease])

	assert.True(t, strings.HasPrefix(vars[extravars.QbittorrentWebUIPasswordHash], "@ByteArray("))
	// Plaintext password is dropped after hashing.
	_, ok := vars[extravars.BrreweryUserPassword]
	assert.False(t, ok)
}
