package libtorrent_test

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/autobrr/brrewery/internal/apps/libtorrent"
)

func TestValidatePatch(t *testing.T) {
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
		{name: "not base64", patch: "%%%not base64%%%", wantErr: libtorrent.ErrPatchInvalid},
		{name: "not a diff", patch: enc("just some text\nwith no diff markers\n"), wantErr: libtorrent.ErrPatchInvalid},
		{name: "too large", patch: enc(validDiff + strings.Repeat("x", libtorrent.MaxPatchBytes)), wantErr: libtorrent.ErrPatchTooLarge},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := libtorrent.ValidatePatch(tt.patch)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}
