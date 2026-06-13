package qbittorrent_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/autobrr/brrewery/internal/apps/qbittorrent"
)

func TestBoostResolver_ResolveLatest(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/release/":
			_, _ = w.Write([]byte(`<a href="1.87.0/">1.87.0</a><a href="1.88.0/">1.88.0</a><a href="1.86.0/">1.86.0</a>`))
		case r.Method == http.MethodHead && strings.Contains(r.URL.Path, "boost_1_88_0.tar.gz"):
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodHead && strings.Contains(r.URL.Path, "boost_1_87_0.tar.gz"):
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodHead && strings.Contains(r.URL.Path, "boost_1_86_0.tar.gz"):
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	resolver := &qbittorrent.BoostResolver{
		Client:  server.Client(),
		BaseURL: server.URL + "/release/",
	}

	got, err := resolver.ResolveLatest(context.Background(), "")
	require.NoError(t, err)
	assert.Equal(t, "1_88_0", got)
}

func TestBoostResolver_ResolveLatest_withMax(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/release/":
			_, _ = w.Write([]byte(`<a href="1.87.0/">1.87.0</a><a href="1.88.0/">1.88.0</a><a href="1.86.0/">1.86.0</a>`))
		case r.Method == http.MethodHead && strings.Contains(r.URL.Path, "boost_1_86_0.tar.gz"):
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	resolver := &qbittorrent.BoostResolver{
		Client:  server.Client(),
		BaseURL: server.URL + "/release/",
	}

	got, err := resolver.ResolveLatest(context.Background(), "1.86.0")
	require.NoError(t, err)
	assert.Equal(t, "1_86_0", got)
}
