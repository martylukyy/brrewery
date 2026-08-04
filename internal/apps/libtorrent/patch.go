// Package libtorrent holds validation shared by the app installers that build
// libtorrent-rasterbar from source and accept an operator-uploaded source patch
// (qBittorrent and Deluge). It deliberately carries no app-specific knowledge so
// neither app package has to import the other.
package libtorrent

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
)

// MaxPatchBytes caps the decoded size of an uploaded libtorrent patch.
const MaxPatchBytes = 512 * 1024

var (
	// ErrPatchTooLarge indicates the uploaded libtorrent patch exceeds the size limit.
	ErrPatchTooLarge = errors.New("libtorrent patch exceeds maximum size")
	// ErrPatchInvalid indicates the uploaded libtorrent patch is not a valid unified diff.
	ErrPatchInvalid = errors.New("libtorrent patch is not a valid patch file")
)

// ValidatePatch checks an optional base64-encoded libtorrent patch for size and
// that it resembles a unified diff. An empty value is accepted: the build then
// falls back to the operator or vendored patch tier.
func ValidatePatch(encoded string) error {
	encoded = strings.TrimSpace(encoded)
	if encoded == "" {
		return nil
	}

	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrPatchInvalid, err)
	}
	if len(decoded) > MaxPatchBytes {
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
