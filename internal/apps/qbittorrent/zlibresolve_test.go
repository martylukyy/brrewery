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

func TestZlibResolver_ResolveLatest(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[
			{"name":"v1.3.1"},
			{"name":"v1.3.0"},
			{"name":"v1.2.13"},
			{"name":"bogus"}
		]`))
	}))
	t.Cleanup(server.Close)

	resolver := &qbittorrent.ZlibResolver{
		Client:  server.Client(),
		TagsURL: server.URL,
	}

	got, err := resolver.ResolveLatest(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "1.3.1", got)
}
