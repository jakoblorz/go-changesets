package workspace

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	gitignore "github.com/denormal/go-gitignore"
	"github.com/jakoblorz/go-changesets/internal/filesystem"
	"github.com/jakoblorz/go-changesets/internal/models"
	"github.com/jakoblorz/go-changesets/internal/versioning"
	"golang.org/x/mod/modfile"
)

// Workspace represents a workspace containing Go and/or Node projects.
type Workspace struct {
	fs                  filesystem.FileSystem
	RootPath            string
	WorkFilePath        string
	Projects            []*models.Project
	nodeStrictWorkspace bool
}

// Option configures workspace behavior.
type Option func(*Workspace)

// WithNodeStrictWorkspace limits Node discovery to workspace manifests only.
func WithNodeStrictWorkspace(enabled bool) Option {
	return func(w *Workspace) {
		w.nodeStrictWorkspace = enabled
	}
}

// New creates a new Workspace instance.
func New(fs filesystem.FileSystem, options ...Option) *Workspace {
	ws := &Workspace{
		fs:       fs,
		Projects: []*models.Project{},
	}

	for _, option := range options {
		option(ws)
	}

	return ws
}

// Detect finds and loads the workspace from the current directory.
func (w *Workspace) Detect() error {
	root, hasGoWork, hasPackageJSON, err := w.findWorkspaceRoot()
	if err != nil {
		return err
	}

	w.RootPath = root
	if hasGoWork {
		w.WorkFilePath = filepath.Join(root, "go.work")
	}

	if err := w.loadProjects(hasGoWork, hasPackageJSON); err != nil {
		return fmt.Errorf("failed to load projects: %w", err)
	}

	return nil
}

// findWorkspaceRoot walks up the directory tree looking for go.work or package.json.
func (w *Workspace) findWorkspaceRoot() (string, bool, bool, error) {
	cwd, err := w.fs.Getwd()
	if err != nil {
		return "", false, false, fmt.Errorf("failed to get working directory: %w", err)
	}

	dir := cwd
	for {
		goWorkPath := filepath.Join(dir, "go.work")
		pkgJSONPath := filepath.Join(dir, "package.json")

		hasGoWork := w.fs.Exists(goWorkPath)
		hasPackage := w.fs.Exists(pkgJSONPath)

		if hasGoWork || hasPackage {
			return dir, hasGoWork, hasPackage, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false, false, fmt.Errorf("workspace not found")
		}
		dir = parent
	}
}

func (w *Workspace) loadProjects(hasGoWork, hasPackageJSON bool) error {
	var projects []*models.Project

	if hasGoWork {
		goProjects, err := w.loadGoProjects()
		if err != nil {
			return err
		}
		projects = append(projects, goProjects...)
	}

	nodeProjects, err := w.loadNodeProjects(hasPackageJSON)
	if err != nil {
		return err
	}
	projects = append(projects, nodeProjects...)

	projects = dedupeProjectNames(projects)
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

// extractProjectName extracts the project name from a module path.
// e.g., "github.com/user/project" -> "project".
func extractProjectName(modulePath string) string {
	parts := strings.Split(modulePath, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return modulePath
}

// loadNodeManifestProjects discovers Node projects via package.json workspaces (or root).
func (w *Workspace) loadNodeManifestProjects() ([]*models.Project, error) {
	rootPackagePath := filepath.Join(w.RootPath, "package.json")
	if !w.fs.Exists(rootPackagePath) {
		return nil, nil
	}

	rootPkg, err := readPackageJSON(w.fs, rootPackagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read root package.json: %w", err)
	}

	workspaces := extractWorkspaces(rootPkg)

	if len(workspaces) == 0 {
		if rootPkg.Private {
			return nil, nil
		}
		project := nodeProjectFromPackageJSON(rootPkg, w.RootPath, rootPackagePath)
		return []*models.Project{project}, nil
	}

	var projects []*models.Project
	for _, pattern := range workspaces {
		globPattern := filepath.Join(w.RootPath, pattern)
		matches, err := w.fs.Glob(globPattern)
		if err != nil {
			return nil, fmt.Errorf("failed to glob workspace pattern %s: %w", pattern, err)
		}

		for _, match := range matches {
			pkgPath := match
			info, err := w.fs.Stat(match)
			if err == nil {
				if info.IsDir() {
					pkgPath = filepath.Join(match, "package.json")
				} else if filepath.Base(match) != "package.json" {
					continue
				}
			}

			if filepath.Base(pkgPath) != "package.json" || !w.fs.Exists(pkgPath) {
				continue
			}

			pkg, err := readPackageJSON(w.fs, pkgPath)
			if err != nil {
				return nil, fmt.Errorf("failed to read package.json at %s: %w", pkgPath, err)
			}

			if pkg.Private {
				continue
			}

			projectRoot := filepath.Dir(pkgPath)
			projects = append(projects, nodeProjectFromPackageJSON(pkg, projectRoot, pkgPath))
		}
	}

	return projects, nil
}

func (w *Workspace) fuzzyLoadNodeProjects() ([]*models.Project, error) {
	ignore, err := w.loadRootGitIgnore()
	if err != nil {
		return nil, err
	}

	rootPackagePath := filepath.Join(w.RootPath, "package.json")
	var rootPkg *packageJSON
	skipRootPackage := false
	if w.fs.Exists(rootPackagePath) {
		pkg, err := readPackageJSON(w.fs, rootPackagePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read root package.json: %w", err)
		}
		rootPkg = &pkg
		if len(extractWorkspaces(pkg)) > 0 {
			skipRootPackage = true
		}
	}

	rootUsed := false
	otherFound := false
	ignoredDirs := make(map[string]struct{})
	var projects []*models.Project
	if err := w.fs.WalkDir(w.RootPath, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == w.RootPath {
			return nil
		}

		rel, relErr := filepath.Rel(w.RootPath, path)
		if relErr != nil {
			return relErr
		}
		rel = filepath.ToSlash(rel)

		for ignoredDir := range ignoredDirs {
			if rel == ignoredDir || strings.HasPrefix(rel, ignoredDir+"/") {
				if entry.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		if ignore != nil {
			if match := ignore.Relative(rel, entry.IsDir()); match != nil && match.Ignore() {
				if entry.IsDir() {
					ignoredDirs[rel] = struct{}{}
					return filepath.SkipDir
				}
				return nil
			}
		}

		if entry.IsDir() {
			return nil
		}

		if filepath.Base(path) != "package.json" {
			return nil
		}

		if skipRootPackage && path == rootPackagePath {
			return nil
		}

		pkg, err := readPackageJSON(w.fs, path)
		if err != nil {
			return fmt.Errorf("failed to read package.json at %s: %w", path, err)
		}

		if pkg.Private {
			return nil
		}

		projectRoot := filepath.Dir(path)
		projects = append(projects, nodeProjectFromPackageJSON(pkg, projectRoot, path))
		if path == rootPackagePath {
			rootUsed = true
		} else {
			otherFound = true
		}

		return nil
	}); err != nil {
		return nil, err
	}

	if rootUsed && !otherFound && rootPkg != nil && !rootPkg.Private {
		fmt.Println("Using root package.json because no subprojects were found and it is not private.")
	}

	return projects, nil
}

func (w *Workspace) loadNodeProjects(hasPackageJSON bool) ([]*models.Project, error) {
	var normal []*models.Project
	if hasPackageJSON {
		projects, err := w.loadNodeManifestProjects()
		if err != nil {
			return nil, err
		}
		normal = projects
	}

	if w.nodeStrictWorkspace {
		return normal, nil
	}

	fuzzyProjects, err := w.fuzzyLoadNodeProjects()
	if err != nil {
		return nil, err
	}

	return mergeProjects(normal, fuzzyProjects), nil
}

func mergeProjects(normal, additional []*models.Project) []*models.Project {
	projects := append([]*models.Project{}, normal...)
	if len(additional) == 0 {
		return projects
	}

	byManifest := make(map[string]struct{}, len(normal))
	for _, project := range normal {
		byManifest[project.ManifestPath] = struct{}{}
	}

	for _, project := range additional {
		if _, exists := byManifest[project.ManifestPath]; exists {
			continue
		}
		projects = append(projects, project)
	}

	return projects
}

func (w *Workspace) loadRootGitIgnore() (gitignore.GitIgnore, error) {
	ignorePath := filepath.Join(w.RootPath, ".gitignore")
	if !w.fs.Exists(ignorePath) {
		return nil, nil
	}

	data, err := w.fs.ReadFile(ignorePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read .gitignore: %w", err)
	}

	return gitignore.New(bytes.NewReader(data), w.RootPath, nil), nil
}

func nodeProjectFromPackageJSON(pkg packageJSON, rootPath, manifestPath string) *models.Project {
	name := pkg.Name
	if strings.TrimSpace(name) == "" {
		name = filepath.Base(rootPath)
	}

	return models.NewProject(name, rootPath, "", manifestPath, models.ProjectTypeNode)
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

// packageJSON represents a minimal subset of package.json.
// Workspaces can be an array or an object with a packages array.
// See https://docs.npmjs.com/cli/v10/using-npm/workspaces.
type packageJSON struct {
	Name       string      `json:"name"`
	Private    bool        `json:"private"`
	Workspaces interface{} `json:"workspaces"`
}

func readPackageJSON(fs filesystem.FileSystem, path string) (packageJSON, error) {
	data, err := fs.ReadFile(path)
	if err != nil {
		return packageJSON{}, err
	}

	var pkg packageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return packageJSON{}, err
	}

	return pkg, nil
}

func extractWorkspaces(pkg packageJSON) []string {
	switch v := pkg.Workspaces.(type) {
	case nil:
		return nil
	case []interface{}:
		return convertWorkspaceArray(v)
	case map[string]interface{}:
		if raw, ok := v["packages"]; ok {
			if arr, ok := raw.([]interface{}); ok {
				return convertWorkspaceArray(arr)
			}
		}
	}
	return nil
}

func convertWorkspaceArray(values []interface{}) []string {
	var result []string
	for _, item := range values {
		if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
			result = append(result, s)
		}
	}
	return result
}
