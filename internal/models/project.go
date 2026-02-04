package models

// ProjectType represents the kind of project.
type ProjectType string

const (
	ProjectTypeGo   ProjectType = "go"
	ProjectTypeNode ProjectType = "node"
)

// Project represents a project/module in the workspace.
type Project struct {
	// Name is the project identifier (unique within the workspace)
	Name string

	// RootPath is the absolute path to the project root
	RootPath string

	// ModulePath is the full module path from go.mod (Go projects only).
	ModulePath string

	// ManifestPath is the path to the manifest (go.mod or package.json).
	ManifestPath string

	// Type indicates whether this is a Go or Node project.
	Type ProjectType

	// DirtyOnly indicates the project was discovered via dirty mode only.
	DirtyOnly bool
}

// NewProject creates a new Project instance
func NewProject(name, rootPath, modulePath, manifestPath string, projectType ProjectType) *Project {
	return &Project{
		Name:         name,
		RootPath:     rootPath,
		ModulePath:   modulePath,
		ManifestPath: manifestPath,
		Type:         projectType,
		DirtyOnly:    false,
	}
}
