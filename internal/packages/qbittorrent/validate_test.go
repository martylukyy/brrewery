package qbittorrent_test

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/autobrr/brrewery/internal/packages/extravars"
	"github.com/autobrr/brrewery/internal/packages/qbittorrent"
)

func TestLoadManifest(t *testing.T) {
	t.Parallel()

	m, err := qbittorrent.LoadManifest()
	require.NoError(t, err)
	require.NotEmpty(t, m.Lines)

	line, ok := m.LineForVersion("5.2")
	require.True(t, ok)
	assert.Equal(t, "5.2", line.Version)
	assert.True(t, line.AllowsBranch(qbittorrent.BranchRC12))
	assert.True(t, line.AllowsBranch(qbittorrent.BranchRC20))

	old, ok := m.LineForVersion("4.3")
	require.True(t, ok)
	assert.True(t, old.AllowsBranch(qbittorrent.BranchRC12))
	assert.False(t, old.AllowsBranch(qbittorrent.BranchRC20))
}

func TestValidateInstallOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		version string
		branch  string
		wantErr error
	}{
		{name: "known minor no branch", version: "4.6", branch: ""},
		{name: "known minor rc12", version: "4.6", branch: qbittorrent.BranchRC12},
		{name: "known minor rc20", version: "4.6", branch: qbittorrent.BranchRC20},
		{name: "4.3 allows rc12", version: "4.3", branch: qbittorrent.BranchRC12},
		{name: "4.3 rejects rc20", version: "4.3", branch: qbittorrent.BranchRC20, wantErr: qbittorrent.ErrBranchNotAllowed},
		{name: "unknown minor", version: "9.9", branch: "", wantErr: qbittorrent.ErrUnknownVersion},
		{name: "empty version", version: "", branch: "", wantErr: qbittorrent.ErrUnknownVersion},
		{name: "garbage branch", version: "5.2", branch: "nope", wantErr: qbittorrent.ErrBranchNotAllowed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := qbittorrent.ValidateInstallOptions(tt.version, tt.branch)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestValidateLibtorrentPatch(t *testing.T) {
	t.Parallel()

	enc := func(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) }
	validDiff := "--- a/src/settings_pack.cpp\n+++ b/src/settings_pack.cpp\n@@ -1 +1 @@\n-old\n+new\n"

	tests := []struct {
		name    string
		patch   string
		wantErr error
	}{
		{name: "empty ok", patch: ""},
		{name: "valid diff", patch: enc(validDiff)},
		{name: "not base64", patch: "%%%not base64%%%", wantErr: qbittorrent.ErrPatchInvalid},
		{name: "not a diff", patch: enc("just some text\nwith no diff markers\n"), wantErr: qbittorrent.ErrPatchInvalid},
		{name: "too large", patch: enc(validDiff + strings.Repeat("x", qbittorrent.MaxLibtorrentPatchBytes)), wantErr: qbittorrent.ErrPatchTooLarge},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := qbittorrent.ValidateLibtorrentPatch(tt.patch)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestValidate_NonQbittorrentPackagePasses(t *testing.T) {
	t.Parallel()

	err := qbittorrent.Validate("autobrr", map[string]string{"anything": "goes"})
	require.NoError(t, err)
}

func TestValidate_QbittorrentChecksVersionAndPatch(t *testing.T) {
	t.Parallel()

	err := qbittorrent.Validate(qbittorrent.PackageID, map[string]string{
		extravars.QbittorrentVersion: "9.9",
	})
	require.ErrorIs(t, err, qbittorrent.ErrUnknownVersion)

	err = qbittorrent.Validate(qbittorrent.PackageID, map[string]string{
		extravars.QbittorrentVersion: "5.2",
		extravars.LibtorrentBranch:   qbittorrent.BranchRC20,
		extravars.LibtorrentPatch:    "not-base64-$$$",
	})
	require.ErrorIs(t, err, qbittorrent.ErrPatchInvalid)
}

func TestInstallOptions(t *testing.T) {
	t.Parallel()

	opts := qbittorrent.InstallOptions()
	require.Len(t, opts, 2)
	assert.Equal(t, extravars.QbittorrentVersion, opts[0].Key)
	assert.NotEmpty(t, opts[0].Choices)

	assert.Equal(t, extravars.LibtorrentBranch, opts[1].Key)
	require.NotNil(t, opts[1].When)
	assert.Equal(t, extravars.QbittorrentVersion, opts[1].When.Key)
	// 4.3 is RC_1_2-only, so it must not gate the branch step.
	assert.NotContains(t, opts[1].When.OneOf, "4.3")
	assert.Contains(t, opts[1].When.OneOf, "5.2")
}
