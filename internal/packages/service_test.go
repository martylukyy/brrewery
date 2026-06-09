package packages

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/autobrr/brrewery/internal/packages/catalog"
)

func TestService_List(t *testing.T) {
	t.Parallel()

	svc := NewService()
	list := svc.List("")

	all := catalog.All()
	require.NotEmpty(t, list)
	require.Len(t, list, len(all))
	for i := range all {
		assert.Equal(t, all[i].ID, list[i].ID)
	}
}

func TestService_Get(t *testing.T) {
	t.Parallel()

	svc := NewService()
	pkg, ok := svc.Get("radarr", "")
	require.True(t, ok)
	assert.Equal(t, "Radarr", pkg.Name)

	_, ok = svc.Get("missing", "")
	assert.False(t, ok)
}
