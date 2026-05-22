package updater

import (
	"strconv"
	"strings"
)

type Version struct {
	Major int
	Minor int
	Patch int
}

func ParseVersion(s string) (Version, bool) {
	s = strings.TrimPrefix(s, "v")
	parts := strings.SplitN(s, ".", 3)
	if len(parts) != 3 {
		return Version{}, false
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return Version{}, false
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return Version{}, false
	}
	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return Version{}, false
	}
	return Version{Major: major, Minor: minor, Patch: patch}, true
}

func (v Version) GreaterThan(other Version) bool {
	if v.Major != other.Major {
		return v.Major > other.Major
	}
	if v.Minor != other.Minor {
		return v.Minor > other.Minor
	}
	return v.Patch > other.Patch
}
