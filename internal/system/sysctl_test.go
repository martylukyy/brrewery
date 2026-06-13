package system_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/autobrr/brrewery/internal/system"
)

func TestSysctlCatalogIsWellFormed(t *testing.T) {
	t.Parallel()

	catalog := system.SysctlCatalog()
	require.NotEmpty(t, catalog)

	seen := make(map[string]struct{})
	for _, param := range catalog {
		assert.NotEmpty(t, param.Key, "key must be set")
		assert.NotEmpty(t, param.Label, "label must be set for %s", param.Key)
		assert.NotEmpty(t, param.Group, "group must be set for %s", param.Key)
		assert.NotEmpty(t, param.Recommended, "recommended must be set for %s", param.Key)

		if _, dup := seen[param.Key]; dup {
			t.Fatalf("duplicate catalog key %q", param.Key)
		}
		seen[param.Key] = struct{}{}

		// Every recommended value must itself pass validation, otherwise the UI
		// would offer a value the apply endpoint rejects.
		_, err := system.ValidateSysctl(map[string]string{param.Key: param.Recommended})
		assert.NoErrorf(t, err, "recommended value for %s should validate", param.Key)
	}
}

func TestValidateSysctl(t *testing.T) {
	t.Parallel()

	t.Run("accepts valid values", func(t *testing.T) {
		t.Parallel()
		out, err := system.ValidateSysctl(map[string]string{
			"vm.swappiness":                   "10",
			"net.ipv4.tcp_congestion_control": "bbr",
			"net.ipv4.tcp_rmem":               "4096  87380   16777216",
		})
		require.NoError(t, err)
		assert.Equal(t, "10", out["vm.swappiness"])
		assert.Equal(t, "bbr", out["net.ipv4.tcp_congestion_control"])
		// Whitespace runs are normalized to single spaces.
		assert.Equal(t, "4096 87380 16777216", out["net.ipv4.tcp_rmem"])
	})

	t.Run("accepts fs.file-max at the kernel long maximum", func(t *testing.T) {
		t.Parallel()
		// Hosts commonly default fs.file-max to LONG_MAX; the live value must
		// validate so applying the panel does not fail on an untouched field.
		_, err := system.ValidateSysctl(map[string]string{"fs.file-max": "9223372036854775807"})
		require.NoError(t, err)
	})

	t.Run("rejects unknown key", func(t *testing.T) {
		t.Parallel()
		_, err := system.ValidateSysctl(map[string]string{"kernel.shmmax": "1234"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown sysctl parameter")
	})

	t.Run("rejects empty input", func(t *testing.T) {
		t.Parallel()
		_, err := system.ValidateSysctl(map[string]string{})
		require.Error(t, err)
	})

	t.Run("rejects non-integer", func(t *testing.T) {
		t.Parallel()
		_, err := system.ValidateSysctl(map[string]string{"vm.swappiness": "lots"})
		require.Error(t, err)
	})

	t.Run("rejects out-of-range integer", func(t *testing.T) {
		t.Parallel()
		_, err := system.ValidateSysctl(map[string]string{"vm.swappiness": "200"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "at most")
	})

	t.Run("rejects unknown enum value", func(t *testing.T) {
		t.Parallel()
		_, err := system.ValidateSysctl(map[string]string{"net.ipv4.tcp_congestion_control": "rm -rf"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must be one of")
	})

	t.Run("rejects wrong list arity", func(t *testing.T) {
		t.Parallel()
		_, err := system.ValidateSysctl(map[string]string{"net.ipv4.tcp_rmem": "4096 16777216"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "3 space-separated")
	})

	t.Run("rejects injection attempt in list", func(t *testing.T) {
		t.Parallel()
		_, err := system.ValidateSysctl(map[string]string{"net.ipv4.tcp_rmem": "4096 87380 16777216; reboot"})
		require.Error(t, err)
	})
}

func TestSysctlConfContent(t *testing.T) {
	t.Parallel()

	content := system.SysctlConfContent(map[string]string{
		"vm.swappiness":     "10",
		"net.core.rmem_max": "16777216",
	})

	assert.True(t, strings.HasPrefix(content, "# Managed by brrewery"), "should carry the managed header")
	// Keys are sorted, so net.core.* precedes vm.* regardless of map iteration order.
	assert.True(t,
		strings.HasSuffix(content, "net.core.rmem_max = 16777216\nvm.swappiness = 10\n"),
		"settings should be written sorted as 'key = value' lines, got:\n%s", content,
	)
}

func TestReadSysctlReturnsCatalog(t *testing.T) {
	t.Parallel()

	report := system.ReadSysctl()
	assert.Len(t, report.Settings, len(system.SysctlCatalog()))
	for _, setting := range report.Settings {
		assert.NotEmpty(t, setting.Key)
	}
}
