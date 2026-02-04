package models

// ProjectContext represents the context passed to commands via STDIN
// when executed through 'changeset each'
type ProjectContext struct {
	// Project is the project name
	Project string `json:"project"`

	// ProjectPath is the absolute path to the project root
	ProjectPath string `json:"projectPath"`

	// ModulePath is the full module path from go.mod
	ModulePath string `json:"modulePath"`

	// DirtyOnly indicates the project was discovered via dirty mode only.
	DirtyOnly bool `json:"dirtyOnly"`

	// Changesets contains summaries of all changesets affecting this project
	Changesets []ChangesetSummary `json:"changesets"`

	// CurrentVersion is the parsed project version (defaults to "0.0.0")
	CurrentVersion string `json:"currentVersion"`

	// HasVersionFile indicates whether the version source exists for this project.
	// (version.txt for Go, package.json for Node).
	HasVersionFile bool `json:"hasVersionFile"`

	// LatestTag is the latest git tag for this project (or "0.0.0" if none)
	LatestTag string `json:"latestTag"`

	// HasChangesets indicates if there are any changesets for this project
	HasChangesets bool `json:"hasChangesets"`

	// IsOutdated indicates if CurrentVersion > LatestTag
	IsOutdated bool `json:"isOutdated"`

	// ChangelogPreview contains the markdown that will be added to CHANGELOG.md
	// Empty string if no changesets
	ChangelogPreview string `json:"changelogPreview"`
}

// ChangesetSummary is a simplified changeset representation for context
type ChangesetSummary struct {
	// ID is the changeset identifier (filename without extension)
	ID string `json:"id"`

	// BumpType is the semantic version bump type for this project
	BumpType BumpType `json:"bumpType"`

	// Message is the changeset description
	Message string `json:"message"`
}
