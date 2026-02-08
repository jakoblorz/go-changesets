package workspace

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"strings"

	gitignore "github.com/denormal/go-gitignore"
	"github.com/jakoblorz/go-changesets/internal/filesystem"
	"github.com/jakoblorz/go-changesets/internal/models"
	"github.com/jakoblorz/go-changesets/internal/versioning"
	"golang.org/x/mod/modfile"
)

var (
	rootSkipDirs = map[string]struct{}{
		".git": {},
	}
)

// Workspace represents a workspace containing Go and/or Node projects.
type Workspace struct {
	fs                  filesystem.FileSystem
	goEnv               GoEnvReader
	warningWriter       io.Writer
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

// WithGoEnv overrides the go env reader (useful for tests).
func WithGoEnv(goEnv GoEnvReader) Option {
	return func(w *Workspace) {
		w.goEnv = goEnv
	}
}

// WithWarningWriter sets the sink for non-fatal warnings (nil disables).
func WithWarningWriter(writer io.Writer) Option {
	return func(w *Workspace) {
		w.warningWriter = writer
	}
}

// New creates a new Workspace instance.
func New(fs filesystem.FileSystem, options ...Option) *Workspace {
	goEnv := newOSGoEnvReader(fs)
	if _, ok := fs.(*filesystem.MockFileSystem); ok {
		goEnv = NewMockGoEnvReader(fs)
	}

	ws := &Workspace{
		fs:       fs,
		goEnv:    goEnv,
		Projects: []*models.Project{},
	}

	for _, option := range options {
		option(ws)
	}

	return ws
}

// Detect finds and loads the workspace from the current directory.
func (w *Workspace) Detect() error {
	goProjects, goRoot, err := w.detectGoProjects()
	if err != nil {
		return fmt.Errorf("failed to load projects: %w", err)
	}

	nodeRoot, err := w.findNodeRoot()
	if err != nil {
		return err
	}

	scanRoot := goRoot
	if scanRoot == "" {
		scanRoot = nodeRoot
	}

	nodeProjects, err := w.loadNodeProjects(nodeRoot, scanRoot)
	if err != nil {
		return fmt.Errorf("failed to load projects: %w", err)
	}

	if goRoot == "" && nodeRoot == "" {
		return fmt.Errorf("workspace not found")
	}

	if goRoot != "" {
		w.RootPath = goRoot
	} else {
		w.RootPath = nodeRoot
	}

	projects := append(goProjects, nodeProjects...)
	projects = dedupeProjectNames(projects)
	if len(projects) == 0 {
		return fmt.Errorf("failed to load projects: no projects found in workspace")
	}

	w.Projects = projects
	return nil
}

func (w *Workspace) warnf(format string, args ...interface{}) {
	if w.warningWriter == nil {
		return
	}

	_, _ = fmt.Fprintf(w.warningWriter, format+"\n", args...)
}

func (w *Workspace) detectGoProjects() ([]*models.Project, string, error) {
	if w.goEnv == nil {
		return nil, "", nil
	}

	goEnv, err := w.goEnv.Read()
	if err != nil {
		w.warnf("warning: failed to read go env; skipping Go project detection: %v", err)
		return nil, "", nil
	}

	if goEnv.GoWork != "" {
		w.WorkFilePath = goEnv.GoWork
		root := filepath.Dir(goEnv.GoWork)
		projects, err := w.loadGoProjects(root, goEnv.GoWork)
		if err != nil {
			return nil, "", err
		}
		return projects, root, nil
	}

	if goEnv.GoMod != "" {
		root := filepath.Dir(goEnv.GoMod)
		versionStore := versioning.NewVersionStore(w.fs, models.ProjectTypeGo)
		if !versionStore.IsEnabled(root) {
			return nil, root, nil
		}

		project, err := w.loadGoProject(root)
		if err != nil {
			return nil, "", err
		}

		return []*models.Project{project}, root, nil
	}

	return nil, "", nil
}

// findNodeRoot walks up the directory tree looking for package.json.
func (w *Workspace) findNodeRoot() (string, error) {
	cwd, err := w.fs.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}

	pkgPath, found, err := findFileUp(w.fs, cwd, "package.json")
	if err != nil {
		return "", err
	}
	if !found {
		return "", nil
	}

	return filepath.Dir(pkgPath), nil
}

// loadGoProjects parses go.work and loads all Go projects.
func (w *Workspace) loadGoProjects(root, workFilePath string) ([]*models.Project, error) {
	data, err := w.fs.ReadFile(workFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read go.work: %w", err)
	}

	workFile, err := modfile.ParseWork(workFilePath, data, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to parse go.work: %w", err)
	}

	versionStore := versioning.NewVersionStore(w.fs, models.ProjectTypeGo)

	var projects []*models.Project
	for _, use := range workFile.Use {
		projectPath := filepath.Join(root, use.Path)

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
func (w *Workspace) loadNodeManifestProjects(rootPath string) ([]*models.Project, error) {
	rootPackagePath := filepath.Join(rootPath, "package.json")
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
		project := nodeProjectFromPackageJSON(rootPkg, rootPath, rootPackagePath)
		return []*models.Project{project}, nil
	}

	var projects []*models.Project
	for _, pattern := range workspaces {
		globPattern := filepath.Join(rootPath, pattern)
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

func (w *Workspace) fuzzyLoadNodeProjects(rootPath string) ([]*models.Project, error) {
	ignore, err := w.loadRootGitIgnore(rootPath)
	if err != nil {
		return nil, err
	}

	ignoredDirs := make(map[string]struct{})
	var projects []*models.Project
	if err := w.fs.WalkDir(rootPath, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == rootPath {
			return nil
		}

		rel, relErr := filepath.Rel(rootPath, path)
		if relErr != nil {
			return relErr
		}
		rel = filepath.ToSlash(rel)

		for rootSkipDir := range rootSkipDirs {
			if rel == rootSkipDir || strings.HasPrefix(rel, rootSkipDir+"/") {
				if entry.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

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

		pkg, err := readPackageJSON(w.fs, path)
		if err != nil {
			return fmt.Errorf("failed to read package.json at %s: %w", path, err)
		}

		if pkg.Private {
			return nil
		}

		projectRoot := filepath.Dir(path)
		projects = append(projects, nodeProjectFromPackageJSON(pkg, projectRoot, path))

		return nil
	}); err != nil {
		return nil, err
	}

	return projects, nil
}

func (w *Workspace) loadNodeProjects(manifestRoot, scanRoot string) ([]*models.Project, error) {
	var normal []*models.Project
	if manifestRoot != "" {
		projects, err := w.loadNodeManifestProjects(manifestRoot)
		if err != nil {
			return nil, err
		}
		normal = projects
	}

	if w.nodeStrictWorkspace || scanRoot == "" {
		return normal, nil
	}

	fuzzyProjects, err := w.fuzzyLoadNodeProjects(scanRoot)
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

func (w *Workspace) loadRootGitIgnore(rootPath string) (gitignore.GitIgnore, error) {
	ignorePath := filepath.Join(rootPath, ".gitignore")
	if !w.fs.Exists(ignorePath) {
		return nil, nil
	}

	data, err := w.fs.ReadFile(ignorePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read .gitignore: %w", err)
	}

	return gitignore.New(bytes.NewReader(data), rootPath, nil), nil
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
