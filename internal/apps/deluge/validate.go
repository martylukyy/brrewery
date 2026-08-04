package deluge

import (
	"fmt"
	"strings"

	"github.com/autobrr/brrewery/internal/apps/extravars"
	"github.com/autobrr/brrewery/internal/apps/libtorrent"
)

// Validate enforces the Deluge install options when appID is the Deluge catalog
// id. Other apps pass through unchanged. It checks the version exists in the
// manifest, the libtorrent branch (when supplied) is allowed for that line (an
// empty branch is accepted; Ansible falls back to the line default), and that an
// uploaded libtorrent patch is a plausible unified diff within the size cap.
func Validate(appID string, extra map[string]string) error {
	if appID != AppID {
		return nil
	}

	version := strings.TrimSpace(extra[extravars.DelugeVersion])
	branch := strings.TrimSpace(extra[extravars.LibtorrentBranch])

	m, err := LoadManifest()
	if err != nil {
		return err
	}
	line, err := m.ResolveSelection(version)
	if err != nil {
		return err
	}

	if branch != "" && !line.AllowsBranch(branch) {
		return fmt.Errorf("%w: %s with Deluge %s", ErrBranchNotAllowed, branch, line.Version)
	}

	return libtorrent.ValidatePatch(extra[extravars.LibtorrentPatch])
}
