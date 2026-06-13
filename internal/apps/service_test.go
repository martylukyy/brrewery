package apps

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/autobrr/brrewery/internal/apps/catalog"
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
	app, ok := svc.Get("radarr", "")
	require.True(t, ok)
	assert.Equal(t, "Radarr", app.Name)

	_, ok = svc.Get("missing", "")
	assert.False(t, ok)
}
