package qbittorrent_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/autobrr/brrewery/internal/apps/qbittorrent"
)

func TestReleaseResolver_ResolveLatest(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[
			{"name":"release-5.2.0"},
			{"name":"release-5.2.1"},
			{"name":"release-5.1.4"},
			{"name":"v5.2.1"}
		]`))
	}))
	t.Cleanup(server.Close)

	resolver := &qbittorrent.ReleaseResolver{
		Client:  server.Client(),
		TagsURL: server.URL,
	}

	got, err := resolver.ResolveLatest(context.Background(), "5.2")
	require.NoError(t, err)
	assert.Equal(t, "5.2.1", got)
}

func TestManifest_ResolveSelection(t *testing.T) {
	t.Parallel()

	m, err := qbittorrent.LoadManifest()
	require.NoError(t, err)

	line, err := m.ResolveSelection("5.2")
	require.NoError(t, err)
	assert.Equal(t, "5.2", line.Version)

	_, err = m.ResolveSelection("5.2.0")
	require.ErrorIs(t, err, qbittorrent.ErrUnknownVersion)
}
