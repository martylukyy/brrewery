package qbittorrent

import (
	"github.com/autobrr/brrewery/internal/apps/extravars"
	"github.com/autobrr/brrewery/internal/apps/model"
)

func branchLabel(branch string) string {
	switch branch {
	case BranchRC20:
		return "libtorrent 2.0"
	default:
		return "libtorrent 1.2"
	}
}

// InstallOptions builds the catalog install options for qBittorrent from the
// vendored manifest: a version picker (one choice per minor line) and a
// libtorrent branch picker shown only for versions that support libtorrent 2.0.
// It returns nil when the manifest cannot be loaded.
func InstallOptions() []model.InstallOption {
	m, err := LoadManifest()
	if err != nil {
		return nil
	}

	versionChoices := make([]model.InstallOptionChoice, 0, len(m.Lines))
	branchVersions := make([]string, 0, len(m.Lines))
	for _, line := range m.Lines {
		versionChoices = append(versionChoices, model.InstallOptionChoice{
			Value: line.Version,
			Label: line.Version,
		})
		if len(line.Libtorrent.Branches) > 1 {
			branchVersions = append(branchVersions, line.Version)
		}
	}

	options := []model.InstallOption{{
		Key:     extravars.QbittorrentVersion,
		Label:   "qBittorrent version",
		Type:    "select",
		Choices: versionChoices,
	}}

	if len(branchVersions) > 0 {
		options = append(options, model.InstallOption{
			Key:   extravars.LibtorrentBranch,
			Label: "libtorrent version",
			Type:  "select",
			Choices: []model.InstallOptionChoice{
				{Value: BranchRC12, Label: branchLabel(BranchRC12)},
				{Value: BranchRC20, Label: branchLabel(BranchRC20)},
			},
			When: &model.InstallOptionWhen{
				Key:   extravars.QbittorrentVersion,
				OneOf: branchVersions,
			},
		})
	}

	return options
}
