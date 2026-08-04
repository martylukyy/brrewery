package deluge

import (
	"slices"

	"github.com/autobrr/brrewery/internal/apps/extravars"
	"github.com/autobrr/brrewery/internal/apps/model"
)

func branchLabel(branch string) string {
	switch branch {
	case BranchRC21:
		return "libtorrent 2.1"
	case BranchRC20:
		return "libtorrent 2.0"
	case BranchRC11:
		return "libtorrent 1.1"
	default:
		return "libtorrent 1.2"
	}
}

// InstallOptions builds the catalog install options for Deluge from the vendored
// manifest: a version picker (one choice per release line) and a libtorrent
// branch picker shown only for the lines that offer a choice (the Python 3
// lines). Both the branch list and which Deluge lines each branch is offered on
// come from the manifest, so vendoring a new libtorrent branch needs no Go
// change beyond a label. It returns nil when the manifest cannot be loaded, in
// which case the app installs with no version choice.
func InstallOptions() []model.InstallOption {
	m, err := LoadManifest()
	if err != nil {
		return nil
	}

	versionChoices := make([]model.InstallOptionChoice, 0, len(m.Lines))
	branchVersions := make([]string, 0, len(m.Lines))
	// branch -> the version lines allowing it, in manifest order.
	branchLines := make(map[string][]string)
	for _, line := range m.Lines {
		versionChoices = append(versionChoices, model.InstallOptionChoice{
			Value: line.Version,
			Label: line.Version,
		})
		if !line.HasBranchChoice() {
			continue
		}
		branchVersions = append(branchVersions, line.Version)
		for branch := range line.Libtorrent.Branches {
			branchLines[branch] = append(branchLines[branch], line.Version)
		}
	}

	options := []model.InstallOption{{
		Key:     extravars.DelugeVersion,
		Label:   "Deluge version",
		Type:    "select",
		Choices: versionChoices,
	}}

	if len(branchVersions) > 0 {
		options = append(options, model.InstallOption{
			Key:     extravars.LibtorrentBranch,
			Label:   "libtorrent version",
			Type:    "select",
			Choices: branchChoices(branchLines, branchVersions),
			When: &model.InstallOptionWhen{
				Key:   extravars.DelugeVersion,
				OneOf: branchVersions,
			},
		})
	}

	return options
}

// branchChoices renders one choice per libtorrent branch in branchOrder. A
// branch every line allows is offered unconditionally; one only some lines allow
// carries a When so the picker hides it for the rest (e.g. RC_2_1, which the
// 2.0.x line cannot run). Validate rejects the same combinations server-side.
func branchChoices(branchLines map[string][]string, allVersions []string) []model.InstallOptionChoice {
	ordered := make([]string, 0, len(branchLines))
	for _, branch := range branchOrder {
		if _, ok := branchLines[branch]; ok {
			ordered = append(ordered, branch)
		}
	}
	// A branch vendored but not yet in branchOrder still gets offered; sort those
	// so the option list stays stable across calls (map order is not).
	unordered := make([]string, 0)
	for branch := range branchLines {
		if !slices.Contains(ordered, branch) {
			unordered = append(unordered, branch)
		}
	}
	slices.Sort(unordered)
	ordered = append(ordered, unordered...)

	choices := make([]model.InstallOptionChoice, 0, len(ordered))
	for _, branch := range ordered {
		choice := model.InstallOptionChoice{Value: branch, Label: branchLabel(branch)}
		if lines := branchLines[branch]; len(lines) < len(allVersions) {
			choice.When = &model.InstallOptionWhen{
				Key:   extravars.DelugeVersion,
				OneOf: lines,
			}
		}
		choices = append(choices, choice)
	}
	return choices
}
