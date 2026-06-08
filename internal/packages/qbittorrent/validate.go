package qbittorrent

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/autobrr/brrewery/internal/packages/extravars"
)

// MaxLibtorrentPatchBytes caps the decoded size of an uploaded libtorrent patch.
const MaxLibtorrentPatchBytes = 512 * 1024

var (
	// ErrManifestUnavailable indicates the vendored build manifest could not be read.
	ErrManifestUnavailable = errors.New("qBittorrent build manifest unavailable")
	// ErrUnknownVersion indicates the requested version is not a manifest entry.
	ErrUnknownVersion = errors.New("unsupported qBittorrent version")
	// ErrBranchNotAllowed indicates the libtorrent branch is not valid for the version.
	ErrBranchNotAllowed = errors.New("libtorrent branch not supported for this qBittorrent version")
	// ErrPatchTooLarge indicates the uploaded libtorrent patch exceeds the size limit.
	ErrPatchTooLarge = errors.New("libtorrent patch exceeds maximum size")
	// ErrPatchInvalid indicates the uploaded libtorrent patch is not a valid unified diff.
	ErrPatchInvalid = errors.New("libtorrent patch is not a valid patch file")
)

// Validate enforces the qBittorrent install options when packageID is the
// qBittorrent catalog id. Other packages pass through unchanged.
func Validate(packageID string, extra map[string]string) error {
	if packageID != PackageID {
		return nil
	}

	version := strings.TrimSpace(extra[extravars.QbittorrentVersion])
	branch := strings.TrimSpace(extra[extravars.LibtorrentBranch])
	if err := ValidateInstallOptions(version, branch); err != nil {
		return err
	}

	return ValidateLibtorrentPatch(extra[extravars.LibtorrentPatch])
}

// ValidateInstallOptions checks the version exists in the manifest and the
// libtorrent branch (when supplied) is allowed for that version. An empty
// branch is accepted; Ansible falls back to the line default (RC_1_2).
func ValidateInstallOptions(version, branch string) error {
	m, err := LoadManifest()
	if err != nil {
		return err
	}

	line, err := m.ResolveSelection(version)
	if err != nil {
		return err
	}

	if branch == "" {
		return nil
	}
	if branch != BranchRC12 && branch != BranchRC20 {
		return fmt.Errorf("%w: %q", ErrBranchNotAllowed, branch)
	}
	if !line.AllowsBranch(branch) {
		return fmt.Errorf("%w: %s with qBittorrent %s", ErrBranchNotAllowed, branch, line.Version)
	}
	return nil
}

// ValidateLibtorrentPatch checks an optional base64-encoded libtorrent patch for
// size and that it resembles a unified diff. An empty value is accepted.
func ValidateLibtorrentPatch(encoded string) error {
	encoded = strings.TrimSpace(encoded)
	if encoded == "" {
		return nil
	}

	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrPatchInvalid, err)
	}
	if len(decoded) > MaxLibtorrentPatchBytes {
		return ErrPatchTooLarge
	}
	if !looksLikeUnifiedDiff(string(decoded)) {
		return ErrPatchInvalid
	}
	return nil
}

func looksLikeUnifiedDiff(content string) bool {
	for _, line := range strings.Split(content, "\n") {
		switch {
		case strings.HasPrefix(line, "diff "),
			strings.HasPrefix(line, "--- "),
			strings.HasPrefix(line, "@@ "),
			strings.HasPrefix(line, "Index: "):
			return true
		}
	}
	return false
}
