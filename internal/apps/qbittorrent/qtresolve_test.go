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

func TestQtResolver_ResolveLatest(t *testing.T) {
	t.Parallel()

	const indexHTML = `
<a href="6.8/">6.8</a>
<a href="6.11/">6.11</a>
<a href="5.15/">5.15</a>
`
	const html68 = `
<a href="6.8.2/">6.8.2</a>
<a href="6.8.3/">6.8.3</a>
`
	const html611 = `<a href="6.11.0/">6.11.0</a>`
	const html515 = `<a href="5.15.19/">5.15.19</a>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/archive/qt/":
			_, _ = w.Write([]byte(indexHTML))
		case "/archive/qt/6.8/":
			_, _ = w.Write([]byte(html68))
		case "/archive/qt/6.11/":
			_, _ = w.Write([]byte(html611))
		case "/archive/qt/5.15/":
			_, _ = w.Write([]byte(html515))
		default:
			if r.Method == http.MethodHead {
				w.WriteHeader(http.StatusOK)
				return
			}
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	resolver := &qbittorrent.QtResolver{
		Client:  server.Client(),
		BaseURL: server.URL + "/archive/qt/",
	}

	got, err := resolver.ResolveLatest(context.Background(), "6.6", "")
	require.NoError(t, err)
	assert.Equal(t, "6.11.0", got)

	got, err = resolver.ResolveLatest(context.Background(), "5.12", "")
	require.NoError(t, err)
	assert.Equal(t, "5.15.19", got)

	_, err = resolver.ResolveLatest(context.Background(), "6.6", "6.5.0")
	require.ErrorIs(t, err, qbittorrent.ErrQtBelowMinimum)
}

func TestQtResolver_ResolveLatest_override(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	resolver := &qbittorrent.QtResolver{Client: server.Client(), BaseURL: server.URL + "/"}
	got, err := resolver.ResolveLatest(context.Background(), "6.0", "6.8.2")
	require.NoError(t, err)
	assert.Equal(t, "6.8.2", got)
}
