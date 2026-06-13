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

func TestOpensslResolver_ResolveLatest(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/openssl/openssl/releases" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`[
			{
				"tag_name":"openssl-4.0.0",
				"assets":[{"name":"openssl-4.0.0.tar.gz"}]
			},
			{
				"tag_name":"openssl-3.6.2",
				"assets":[{"name":"openssl-3.6.2.tar.gz"}]
			},
			{
				"tag_name":"openssl-3.5.6",
				"assets":[{"name":"openssl-3.5.6.tar.gz"}]
			},
			{
				"tag_name":"openssl-3.4.3",
				"assets":[{"name":"openssl-3.4.3.tar.gz"}]
			}
		]`))
	}))
	t.Cleanup(server.Close)

	resolver := &qbittorrent.OpensslResolver{
		Client:      server.Client(),
		ReleasesURL: server.URL + "/repos/openssl/openssl/releases",
	}

	got, err := resolver.ResolveLatest(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "3.6.2", got)
}

func TestOpensslResolver_ResolveLatest_noThreeSeries(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/repos/openssl/openssl/releases" {
			_, _ = w.Write([]byte(`[{"tag_name":"openssl-4.0.0","assets":[{"name":"openssl-4.0.0.tar.gz"}]}]`))
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(server.Close)

	resolver := &qbittorrent.OpensslResolver{
		Client:      server.Client(),
		ReleasesURL: server.URL + "/repos/openssl/openssl/releases",
	}

	_, err := resolver.ResolveLatest(context.Background())
	require.Error(t, err)
	assert.ErrorIs(t, err, qbittorrent.ErrOpensslResolveFailed)
}

func TestOpensslResolver_ResolveLatest_requiresSourceAsset(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/repos/openssl/openssl/releases" {
			_, _ = w.Write([]byte(`[
				{"tag_name":"openssl-3.6.2","assets":[{"name":"openssl-3.6.2.tar.gz.asc"}]},
				{"tag_name":"openssl-3.5.6","assets":[{"name":"openssl-3.5.6.tar.gz"}]}
			]`))
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(server.Close)

	resolver := &qbittorrent.OpensslResolver{
		Client:      server.Client(),
		ReleasesURL: server.URL + "/repos/openssl/openssl/releases",
	}

	got, err := resolver.ResolveLatest(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "3.5.6", got)
}
