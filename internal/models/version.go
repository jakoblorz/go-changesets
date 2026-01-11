package models

import (
	"fmt"
	"strconv"
	"strings"
)

// Version represents a semantic version
type Version struct {
	Major      int
	Minor      int
	Patch      int
	Prerelease string // e.g., "rc0", "rc1", etc.
}

// ParseVersion parses a version string (e.g., "1.2.3", "v1.2.3", "1.2.3-rc0")
func ParseVersion(s string) (*Version, error) {
	// Remove leading 'v' if present
	s = strings.TrimPrefix(s, "v")
	s = strings.TrimSpace(s)

	if s == "" {
		return &Version{Major: 0, Minor: 0, Patch: 0}, nil
	}

	// Check for prerelease suffix (e.g., "1.2.3-rc0")
	var prerelease string
	if idx := strings.Index(s, "-"); idx != -1 {
		prerelease = s[idx+1:]
		s = s[:idx]
	}

	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid version format: %s (expected major.minor.patch)", s)
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid major version: %s", parts[0])
	}

	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid minor version: %s", parts[1])
	}

	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return nil, fmt.Errorf("invalid patch version: %s", parts[2])
	}

	return &Version{
		Major:      major,
		Minor:      minor,
		Patch:      patch,
		Prerelease: prerelease,
	}, nil
}

// String returns the version as a string without 'v' prefix
func (v *Version) String() string {
	if v.Prerelease != "" {
		return fmt.Sprintf("%d.%d.%d-%s", v.Major, v.Minor, v.Patch, v.Prerelease)
	}
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// Tag returns the version as a tag string with 'v' prefix
func (v *Version) Tag() string {
	if v.Prerelease != "" {
		return fmt.Sprintf("v%d.%d.%d-%s", v.Major, v.Minor, v.Patch, v.Prerelease)
	}
	return fmt.Sprintf("v%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// Bump creates a new version by applying a bump type
func (v *Version) Bump(bumpType BumpType) *Version {
	newVersion := &Version{
		Major: v.Major,
		Minor: v.Minor,
		Patch: v.Patch,
	}

	switch bumpType {
	case BumpMajor:
		newVersion.Major++
		newVersion.Minor = 0
		newVersion.Patch = 0
	case BumpMinor:
		newVersion.Minor++
		newVersion.Patch = 0
	case BumpPatch:
		newVersion.Patch++
	}

	return newVersion
}

// Compare compares two versions
// Returns -1 if v < other, 0 if v == other, 1 if v > other
// Prerelease versions are ordered before release versions (e.g., 1.2.3-rc0 < 1.2.3)
func (v *Version) Compare(other *Version) int {
	if v.Major != other.Major {
		if v.Major < other.Major {
			return -1
		}
		return 1
	}

	if v.Minor != other.Minor {
		if v.Minor < other.Minor {
			return -1
		}
		return 1
	}

	if v.Patch != other.Patch {
		if v.Patch < other.Patch {
			return -1
		}
		return 1
	}

	// Same base version (major.minor.patch), compare prerelease
	// No prerelease (release) > prerelease
	if v.Prerelease == "" && other.Prerelease == "" {
		return 0
	}
	if v.Prerelease == "" && other.Prerelease != "" {
		return 1 // Release > prerelease
	}
	if v.Prerelease != "" && other.Prerelease == "" {
		return -1 // Prerelease < release
	}

	// Both have prerelease, compare lexicographically
	if v.Prerelease < other.Prerelease {
		return -1
	}
	if v.Prerelease > other.Prerelease {
		return 1
	}
	return 0
}

// WithPrerelease returns a new version with the specified prerelease suffix
func (v *Version) WithPrerelease(prerelease string) *Version {
	return &Version{
		Major:      v.Major,
		Minor:      v.Minor,
		Patch:      v.Patch,
		Prerelease: prerelease,
	}
}

// IsPrerelease returns true if this version has a prerelease suffix
func (v *Version) IsPrerelease() bool {
	return v.Prerelease != ""
}

// StripPrerelease returns a new version without the prerelease suffix
func (v *Version) StripPrerelease() *Version {
	return &Version{
		Major: v.Major,
		Minor: v.Minor,
		Patch: v.Patch,
	}
}
