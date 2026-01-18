package workspace

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/jakoblorz/go-changesets/internal/filesystem"
	"github.com/jakoblorz/go-changesets/internal/models"
	"github.com/jakoblorz/go-changesets/internal/versioning"
	"golang.org/x/mod/modfile"
)

// Workspace represents a Go workspace containing one or more Go projects.
type Workspace struct {
	fs           filesystem.FileSystem
	RootPath     string
	WorkFilePath string
	Projects     []*models.Project
}

// New creates a new Workspace instance.
func New(fs filesystem.FileSystem) *Workspace {
	return &Workspace{
		fs:       fs,
		Projects: []*models.Project{},
	}
}

// Detect finds and loads the workspace from the current directory.
func (w *Workspace) Detect() error {
	root, err := w.findWorkspaceRoot()
	if err != nil {
		return err
	}

	w.RootPath = root
	w.WorkFilePath = filepath.Join(root, "go.work")

	if err := w.loadProjects(); err != nil {
		return fmt.Errorf("failed to load projects: %w", err)
	}

	return nil
}

// findWorkspaceRoot walks up the directory tree looking for go.work.
func (w *Workspace) findWorkspaceRoot() (string, error) {
	cwd, err := w.fs.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}

	dir := cwd
	for {
		goWorkPath := filepath.Join(dir, "go.work")
		if w.fs.Exists(goWorkPath) {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("workspace not found")
		}
		dir = parent
	}
}

func (w *Workspace) loadProjects() error {
	goProjects, err := w.loadGoProjects()
	if err != nil {
		return err
	}

	projects := dedupeProjectNames(goProjects)
	if len(projects) == 0 {
		return fmt.Errorf("no projects found in workspace")
	}

	w.Projects = projects
	return nil
}

// loadGoProjects parses go.work and loads all Go projects.
func (w *Workspace) loadGoProjects() ([]*models.Project, error) {
	data, err := w.fs.ReadFile(w.WorkFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read go.work: %w", err)
	}

	workFile, err := modfile.ParseWork(w.WorkFilePath, data, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to parse go.work: %w", err)
	}

	versionStore := versioning.NewVersionStore(w.fs, models.ProjectTypeGo)

	var projects []*models.Project
	for _, use := range workFile.Use {
		projectPath := filepath.Join(w.RootPath, use.Path)

		if !versionStore.IsEnabled(projectPath) {
			continue
		}

		project, err := w.loadGoProject(projectPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load project at %s: %w", projectPath, err)
		}

		projects = append(projects, project)
	}

	return projects, nil
}

// loadGoProject loads a single Go project from a directory.
func (w *Workspace) loadGoProject(projectPath string) (*models.Project, error) {
	goModPath := filepath.Join(projectPath, "go.mod")
	if !w.fs.Exists(goModPath) {
		return nil, fmt.Errorf("go.mod not found at %s", goModPath)
	}

	data, err := w.fs.ReadFile(goModPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read go.mod: %w", err)
	}

	modFile, err := modfile.Parse(goModPath, data, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to parse go.mod: %w", err)
	}

	modulePath := modFile.Module.Mod.Path
	name := extractProjectName(modulePath)

	return models.NewProject(name, projectPath, modulePath, goModPath, models.ProjectTypeGo), nil
}

// GetProject returns a project by name.
func (w *Workspace) GetProject(name string) (*models.Project, error) {
	for _, p := range w.Projects {
		if p.Name == name {
			return p, nil
		}
	}
	return nil, fmt.Errorf("project %s not found in workspace", name)
}

// GetProjectNames returns a list of all project names.
func (w *Workspace) GetProjectNames() []string {
	names := make([]string, len(w.Projects))
	for i, p := range w.Projects {
		names[i] = p.Name
	}
	return names
}

// ChangesetDir returns the path to the .changeset directory.
func (w *Workspace) ChangesetDir() string {
	return filepath.Join(w.RootPath, ".changeset")
}

// FileSystem returns the filesystem used by this workspace.
func (w *Workspace) FileSystem() filesystem.FileSystem {
	return w.fs
}

// extractProjectName extracts the project name from a module path.
// e.g., "github.com/user/project" -> "project".
func extractProjectName(modulePath string) string {
	parts := strings.Split(modulePath, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return modulePath
}

func dedupeProjectNames(projects []*models.Project) []*models.Project {
	counts := make(map[string]int)
	for _, p := range projects {
		counts[p.Name]++
	}

	used := make(map[string]int)
	for _, p := range projects {
		name := p.Name
		if counts[p.Name] > 1 {
			name = fmt.Sprintf("%s-%s", p.Name, string(p.Type))
		}

		if used[name] > 0 {
			name = fmt.Sprintf("%s-%d", name, used[name]+1)
		}

		used[name]++
		p.Name = name
	}

	return projects
}
