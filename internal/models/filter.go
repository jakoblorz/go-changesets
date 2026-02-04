package models

import "fmt"

// FilterType represents a filter for selecting projects
type FilterType string

const (
	// FilterAll selects all projects
	FilterAll FilterType = "all"

	// FilterOpenChangesets selects projects with changesets in .changeset/
	FilterOpenChangesets FilterType = "open-changesets"

	// FilterOutdatedVersions selects projects where the version source > latest git tag
	FilterOutdatedVersions FilterType = "outdated-versions"

	// FilterHasVersion selects projects with a version source file
	FilterHasVersion FilterType = "has-version"

	// FilterNoVersion selects projects without a version source file
	FilterNoVersion FilterType = "no-version"

	// FilterUnchanged selects projects with no changesets
	FilterUnchanged FilterType = "unchanged"
)

// IsValid checks if the filter type is valid
func (f FilterType) IsValid() bool {
	switch f {
	case FilterAll, FilterOpenChangesets, FilterOutdatedVersions, FilterHasVersion, FilterNoVersion, FilterUnchanged:
		return true
	default:
		return false
	}
}

// String returns the string representation of FilterType
func (f FilterType) String() string {
	return string(f)
}

// ParseFilterType parses a string into a FilterType
func ParseFilterType(s string) (FilterType, error) {
	ft := FilterType(s)
	if !ft.IsValid() {
		return "", fmt.Errorf("invalid filter type: %s (must be all, open-changesets, outdated-versions, has-version, no-version, or unchanged)", s)
	}
	return ft, nil
}

// MatchesContext checks if a ProjectContext matches this filter
func (f FilterType) MatchesContext(ctx *ProjectContext) bool {
	switch f {
	case FilterAll:
		return true
	case FilterOpenChangesets:
		return ctx.HasChangesets
	case FilterOutdatedVersions:
		return ctx.IsOutdated
	case FilterHasVersion:
		return ctx.HasVersionFile
	case FilterNoVersion:
		return !ctx.HasVersionFile
	case FilterUnchanged:
		// No changesets (no pending changes)
		return !ctx.HasChangesets
	default:
		return false
	}
}
