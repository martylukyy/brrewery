package packages

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_List(t *testing.T) {
	t.Parallel()

	svc := NewService()
	list := svc.List()
	require.Len(t, list, 16)
	assert.Equal(t, "qbittorrent", list[0].ID)
}

func TestService_Get(t *testing.T) {
	t.Parallel()

	svc := NewService()
	pkg, ok := svc.Get("radarr")
	require.True(t, ok)
	assert.Equal(t, "Radarr", pkg.Name)

	_, ok = svc.Get("missing")
	assert.False(t, ok)
}
