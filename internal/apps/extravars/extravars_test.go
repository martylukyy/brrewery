package extravars_test

import (
	"testing"

	"github.com/autobrr/brrewery/internal/apps/extravars"
	"github.com/stretchr/testify/assert"
)

func TestForInstall(t *testing.T) {
	t.Parallel()

	vars := extravars.ForInstall("admin", map[string]string{"token": "secret"})
	assert.Equal(t, "admin", vars[extravars.BrreweryUser])
	assert.Equal(t, "secret", vars["token"])
}
