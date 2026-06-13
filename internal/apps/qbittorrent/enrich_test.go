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

func TestEnrichAnsibleVars_resolvesPatchAndQt(t *testing.T) {
	t.Parallel()

	github := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/qbittorrent/qBittorrent/tags":
			_, _ = w.Write([]byte(`[{"name":"release-5.2.1"}]`))
		case "/repos/madler/zlib/tags":
			_, _ = w.Write([]byte(`[{"name":"v1.3.1"}]`))
		case "/repos/openssl/openssl/releases":
			_, _ = w.Write([]byte(`[{"tag_name":"openssl-3.6.2","assets":[{"name":"openssl-3.6.2.tar.gz"}]}]`))
		default:
			http.NotFound(w, r)
		}
	}))
	boost := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/release/":
			_, _ = w.Write([]byte(`<a href="1.88.0/">1.88.0</a>`))
		case r.Method == http.MethodHead:
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	qt := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/archive/qt/":
			_, _ = w.Write([]byte(`<a href="6.8/">6.8</a>`))
		case "/archive/qt/6.8/":
			_, _ = w.Write([]byte(`<a href="6.8.3/">6.8.3</a>`))
		default:
			if r.Method == http.MethodHead {
				w.WriteHeader(http.StatusOK)
			}
		}
	}))
	t.Cleanup(github.Close)
	t.Cleanup(boost.Close)
	t.Cleanup(qt.Close)

	vars := map[string]string{
		extravars.QbittorrentVersion:   "5.2",
		extravars.LibtorrentBranch:     qbittorrent.BranchRC20,
		extravars.BrreweryUserPassword: "testpassword",
	}
	err := qbittorrent.EnrichAnsibleVars(
		context.Background(),
		vars,
		&qbittorrent.ReleaseResolver{Client: github.Client(), TagsURL: github.URL + "/repos/qbittorrent/qBittorrent/tags"},
		&qbittorrent.QtResolver{Client: qt.Client(), BaseURL: qt.URL + "/archive/qt/"},
		&qbittorrent.ZlibResolver{Client: github.Client(), TagsURL: github.URL + "/repos/madler/zlib/tags"},
		&qbittorrent.BoostResolver{Client: boost.Client(), BaseURL: boost.URL + "/release/"},
		&qbittorrent.OpensslResolver{
			Client:      github.Client(),
			ReleasesURL: github.URL + "/repos/openssl/openssl/releases",
		},
	)
	require.NoError(t, err)
	assert.Equal(t, "5.2", vars[extravars.QbittorrentVersion])
	assert.Equal(t, "5.2.1", vars[extravars.QbittorrentRelease])
	assert.Equal(t, "6.8.3", vars[extravars.QbittorrentQtVersion])
	assert.Equal(t, "1.3.1", vars[extravars.QbittorrentZlibVersion])
	assert.Equal(t, "1_88_0", vars[extravars.QbittorrentBoostVersion])
	assert.Equal(t, "3.6.2", vars[extravars.QbittorrentOpensslVersion])
	assert.True(t, strings.HasPrefix(vars[extravars.QbittorrentWebUIPasswordHash], "@ByteArray("))
}

func TestEnrichAnsibleVars_RC12_usesManifestBoostCap(t *testing.T) {
	t.Parallel()

	github := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/qbittorrent/qBittorrent/tags":
			_, _ = w.Write([]byte(`[{"name":"release-5.2.1"}]`))
		case "/repos/madler/zlib/tags":
			_, _ = w.Write([]byte(`[{"name":"v1.3.1"}]`))
		case "/repos/openssl/openssl/releases":
			_, _ = w.Write([]byte(`[{"tag_name":"openssl-3.6.2","assets":[{"name":"openssl-3.6.2.tar.gz"}]}]`))
		default:
			http.NotFound(w, r)
		}
	}))
	qt := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/archive/qt/":
			_, _ = w.Write([]byte(`<a href="6.8/">6.8</a>`))
		case "/archive/qt/6.8/":
			_, _ = w.Write([]byte(`<a href="6.8.3/">6.8.3</a>`))
		default:
			if r.Method == http.MethodHead {
				w.WriteHeader(http.StatusOK)
			}
		}
	}))
	t.Cleanup(github.Close)
	t.Cleanup(qt.Close)

	vars := map[string]string{
		extravars.QbittorrentVersion:   "5.2",
		extravars.LibtorrentBranch:     qbittorrent.BranchRC12,
		extravars.BrreweryUserPassword: "testpassword",
	}
	err := qbittorrent.EnrichAnsibleVars(
		context.Background(),
		vars,
		&qbittorrent.ReleaseResolver{Client: github.Client(), TagsURL: github.URL + "/repos/qbittorrent/qBittorrent/tags"},
		&qbittorrent.QtResolver{Client: qt.Client(), BaseURL: qt.URL + "/archive/qt/"},
		&qbittorrent.ZlibResolver{Client: github.Client(), TagsURL: github.URL + "/repos/madler/zlib/tags"},
		nil,
		&qbittorrent.OpensslResolver{
			Client:      github.Client(),
			ReleasesURL: github.URL + "/repos/openssl/openssl/releases",
		},
	)
	require.NoError(t, err)
	assert.Equal(t, "1_86_0", vars[extravars.QbittorrentBoostVersion])
}
