package workspace

import (
	"fmt"
	"path/filepath"

	"github.com/jakoblorz/go-changesets/internal/filesystem"
)

// WorkspaceBuilder helps create test workspaces
type WorkspaceBuilder struct {
	fs       *filesystem.MockFileSystem
	root     string
	projects []ProjectConfig
}

// ProjectConfig represents a project configuration
type ProjectConfig struct {
	Name       string
	Path       string
	ModulePath string
	Version    string
}

// NewWorkspaceBuilder creates a new WorkspaceBuilder
func NewWorkspaceBuilder(root string) *WorkspaceBuilder {
	fs := filesystem.NewMockFileSystem()
	fs.AddDir(root)
	fs.AddDir(filepath.Join(root, ".changeset"))
	fs.SetCurrentDir(root)

	return &WorkspaceBuilder{
		fs:   fs,
		root: root,
	}
}

// AddProject adds a project to the workspace
func (wb *WorkspaceBuilder) AddProject(name, path, modulePath string) *WorkspaceBuilder {
	wb.projects = append(wb.projects, ProjectConfig{
		Name:       name,
		Path:       path,
		ModulePath: modulePath,
		Version:    "0.0.0",
	})

	projectRoot := filepath.Join(wb.root, path)
	wb.fs.AddDir(projectRoot)

	// Create go.mod
	goMod := fmt.Sprintf("module %s\n\ngo 1.24\n", modulePath)
	wb.fs.AddFile(filepath.Join(projectRoot, "go.mod"), []byte(goMod))

	return wb
}

// AddChangeset adds a changeset to the workspace
func (wb *WorkspaceBuilder) AddChangeset(id, project, bump, message string) *WorkspaceBuilder {
	content := fmt.Sprintf("---\n%s: %s\n---\n\n%s\n", project, bump, message)
	changesetPath := filepath.Join(wb.root, ".changeset", id+".md")
	wb.fs.AddFile(changesetPath, []byte(content))
	return wb
}

// SetVersion sets the version for a project
func (wb *WorkspaceBuilder) SetVersion(project, version string) *WorkspaceBuilder {
	for i, p := range wb.projects {
		if p.Name == project {
			wb.projects[i].Version = version
			versionPath := filepath.Join(wb.root, p.Path, "version.txt")
			wb.fs.AddFile(versionPath, []byte(version+"\n"))
			break
		}
	}
	return wb
}

// DisableProject disables a project (sets version.txt to "false")
func (wb *WorkspaceBuilder) DisableProject(project string) *WorkspaceBuilder {
	return wb.SetVersion(project, "false")
}

// AddChangelog adds a changelog for a project
func (wb *WorkspaceBuilder) AddChangelog(project, content string) *WorkspaceBuilder {
	for _, p := range wb.projects {
		if p.Name == project {
			changelogPath := filepath.Join(wb.root, p.Path, "CHANGELOG.md")
			wb.fs.AddFile(changelogPath, []byte(content))
			break
		}
	}
	return wb
}

// Build finalizes the workspace and returns the filesystem
func (wb *WorkspaceBuilder) Build() *filesystem.MockFileSystem {
	// Create go.work file
	goWork := "go 1.24\n\nuse (\n"
	for _, p := range wb.projects {
		goWork += fmt.Sprintf("\t./%s\n", p.Path)
	}
	goWork += ")\n"

	wb.fs.AddFile(filepath.Join(wb.root, "go.work"), []byte(goWork))

	return wb.fs
}

// FileSystem returns the mock filesystem
func (wb *WorkspaceBuilder) FileSystem() *filesystem.MockFileSystem {
	return wb.fs
}
