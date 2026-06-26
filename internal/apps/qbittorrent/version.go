package qbittorrent

import (
	"strconv"
	"strings"
)

// compareVersions compares two dotted-numeric versions, returning a negative,
// zero, or positive value when left is less than, equal to, or greater than right.
func compareVersions(left, right string) int {
	lp := parseVersionParts(left)
	rp := parseVersionParts(right)
	n := len(lp)
	if len(rp) > n {
		n = len(rp)
	}
	for i := 0; i < n; i++ {
		var lv, rv int
		if i < len(lp) {
			lv = lp[i]
		}
		if i < len(rp) {
			rv = rp[i]
		}
		if lv != rv {
			return lv - rv
		}
	}
	return 0
}

func parseVersionParts(version string) []int {
	segments := strings.Split(version, ".")
	out := make([]int, 0, len(segments))
	for _, segment := range segments {
		n, err := strconv.Atoi(segment)
		if err != nil {
			continue
		}
		out = append(out, n)
	}
	return out
}

func maxVersion(versions []string) string {
	best := versions[0]
	for _, candidate := range versions[1:] {
		if compareVersions(candidate, best) > 0 {
			best = candidate
		}
	}
	return best
}
