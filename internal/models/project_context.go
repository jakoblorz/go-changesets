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

	// Changesets contains summaries of all changesets affecting this project
	Changesets []Changeset `json:"changesets"`

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
