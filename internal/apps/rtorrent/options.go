package rtorrent

import (
	"github.com/autobrr/brrewery/internal/apps/extravars"
	"github.com/autobrr/brrewery/internal/apps/model"
)

// InstallOptions builds the catalog install options for rtorrent from the
// vendored manifest: a single version picker with one choice per release line
// (e.g. 0.16.x, 0.15.x, 0.10.0, 0.9.8, 0.9.6). It returns nil when the manifest
// cannot be loaded, in which case the app installs with no version choice.
func InstallOptions() []model.InstallOption {
	m, err := LoadManifest()
	if err != nil {
		return nil
	}

	choices := make([]model.InstallOptionChoice, 0, len(m.Lines))
	for _, line := range m.Lines {
		choices = append(choices, model.InstallOptionChoice{
			Value: line.Version,
			Label: line.Version,
		})
	}

	return []model.InstallOption{{
		Key:     extravars.RtorrentVersion,
		Label:   "rTorrent version",
		Type:    "select",
		Choices: choices,
	}}
}
