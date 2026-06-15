package apps

import (
	"context"
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

type fakeController struct {
	calls int
	units []string
	on    bool
}

func (f *fakeController) SetEnabled(_ context.Context, units []string, on bool) error {
	f.calls++
	f.units = units
	f.on = on
	return nil
}

func TestService_SetServiceEnabled_Errors(t *testing.T) {
	t.Parallel()

	svc := NewService()

	_, err := svc.SetServiceEnabled(context.Background(), "missing", "admin", true)
	require.ErrorIs(t, err, ErrAppNotFound)

	_, err = svc.SetServiceEnabled(context.Background(), "autobrr", "  ", true)
	require.ErrorIs(t, err, ErrInstallUserMissing)
}

func TestService_SetServiceEnabled_InvokesController(t *testing.T) {
	t.Parallel()

	svc := NewService()
	fake := &fakeController{}
	svc.controller = fake

	// Find an app the host actually has installed that exposes a service; the
	// toggle only runs for installed apps. Skip when the host has none.
	var target string
	for _, status := range svc.List("") {
		if status.Installed && status.Service != nil {
			target = status.ID
			break
		}
	}
	if target == "" {
		t.Skip("no installed app with a controllable service on this host")
	}

	_, err := svc.SetServiceEnabled(context.Background(), target, "admin", false)
	require.NoError(t, err)
	assert.Equal(t, 1, fake.calls)
	assert.False(t, fake.on)
	assert.NotEmpty(t, fake.units)
}
