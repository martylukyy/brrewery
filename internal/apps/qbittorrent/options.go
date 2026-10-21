package qbittorrent

import (
	"github.com/autobrr/brrewery/internal/apps/extravars"
	"github.com/autobrr/brrewery/internal/apps/model"
)

// branchOrder is the order the install UI lists the libtorrent branches in.
var branchOrder = []string{BranchRC12, BranchRC20, BranchRC21}

func branchLabel(branch string) string {
	switch branch {
	case BranchRC20:
		return "libtorrent 2.0"
	case BranchRC21:
		return "libtorrent 2.1"
	default:
		return "libtorrent 1.2"
	}
}

// InstallOptions builds the catalog install options for qBittorrent from the
// vendored manifest: a version picker (one choice per minor line) and a
// libtorrent branch picker shown only for versions that offer more than one
// branch. Each branch choice is limited to the versions whose manifest line
// lists it (e.g. libtorrent 2.0 up to 5.2, libtorrent 2.1 from 5.3).
// It returns nil when the manifest cannot be loaded.
func InstallOptions() []model.InstallOption {
	m, err := LoadManifest()
	if err != nil {
		return nil
	}

	versionChoices := make([]model.InstallOptionChoice, 0, len(m.Lines))
	branchVersions := make([]string, 0, len(m.Lines))
	versionsByBranch := make(map[string][]string)
	for _, line := range m.Lines {
		versionChoices = append(versionChoices, model.InstallOptionChoice{
			Value: line.Version,
			Label: line.Version,
		})
		if len(line.Libtorrent.Branches) > 1 {
			branchVersions = append(branchVersions, line.Version)
			for branch := range line.Libtorrent.Branches {
				versionsByBranch[branch] = append(versionsByBranch[branch], line.Version)
			}
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
			Key:     extravars.LibtorrentBranch,
			Label:   "libtorrent version",
			Type:    "select",
			Choices: branchChoices(versionsByBranch),
			When: &model.InstallOptionWhen{
				Key:   extravars.QbittorrentVersion,
				OneOf: branchVersions,
			},
		})
	}

	return options
}

// branchChoices returns one choice per libtorrent branch, each gated on the
// qBittorrent versions that can build against it.
func branchChoices(versionsByBranch map[string][]string) []model.InstallOptionChoice {
	choices := make([]model.InstallOptionChoice, 0, len(versionsByBranch))
	for _, branch := range branchOrder {
		versions, ok := versionsByBranch[branch]
		if !ok {
			continue
		}
		choices = append(choices, model.InstallOptionChoice{
			Value: branch,
			Label: branchLabel(branch),
			When: &model.InstallOptionWhen{
				Key:   extravars.QbittorrentVersion,
				OneOf: versions,
			},
		})
	}
	return choices
}
