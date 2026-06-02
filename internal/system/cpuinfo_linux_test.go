//go:build linux

package system

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReadCPUModelName(t *testing.T) {
	t.Parallel()

	name, err := readCPUModelName()
	require.NoError(t, err)
	require.NotEmpty(t, name)
}
