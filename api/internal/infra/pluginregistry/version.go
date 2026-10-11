package pluginregistry

import (
	"fmt"
	"regexp"
	"strconv"
)

var versionPattern = regexp.MustCompile(`^v?(\d+)\.(\d+)\.(\d+)`)

// Version is a semantic version without pre-release ordering, which plugin compatibility does not
// need.
type Version struct {
	Major, Minor, Patch int
}

// ParseVersion reads the leading X.Y.Z of a version, accepting a "v" prefix and the suffix that
// `git describe` adds ("v0.24.0-3-g614077d" is 0.24.0).
func ParseVersion(raw string) (Version, error) {
	match := versionPattern.FindStringSubmatch(raw)
	if match == nil {
		return Version{}, fmt.Errorf("%q is not a semantic version", raw)
	}
	var parts [3]int
	for index := range parts {
		value, err := strconv.Atoi(match[index+1])
		if err != nil {
			return Version{}, fmt.Errorf("%q is not a semantic version", raw)
		}
		parts[index] = value
	}
	return Version{Major: parts[0], Minor: parts[1], Patch: parts[2]}, nil
}

// AtLeast reports whether v is the same as or newer than minimum.
func (v Version) AtLeast(minimum Version) bool {
	if v.Major != minimum.Major {
		return v.Major > minimum.Major
	}
	if v.Minor != minimum.Minor {
		return v.Minor > minimum.Minor
	}
	return v.Patch >= minimum.Patch
}

func (v Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}
