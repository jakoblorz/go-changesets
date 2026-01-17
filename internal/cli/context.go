package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/jakoblorz/go-changesets/internal/changelog"
	"github.com/jakoblorz/go-changesets/internal/changeset"
	"github.com/jakoblorz/go-changesets/internal/filesystem"
	"github.com/jakoblorz/go-changesets/internal/git"
	"github.com/jakoblorz/go-changesets/internal/github"
	"github.com/jakoblorz/go-changesets/internal/models"
	"github.com/jakoblorz/go-changesets/internal/versioning"
	"github.com/jakoblorz/go-changesets/internal/workspace"
)

type resolvedProject struct {
	Name    string
	ViaEach bool

	Workspace *workspace.Workspace
	Project   *models.Project
}

func resolveProjectName(projectFlag string) (string, bool, error) {
	if projectFlag != "" {
		return projectFlag, false, nil
	}

	ctx, err := readProjectContextFromStdin()
	if err != nil {
		return "", false, err
	}

	return ctx.Project, true, nil
}

func resolveWorkspaceProject(fs filesystem.FileSystem, projectName string) (*workspace.Workspace, *models.Project, error) {
	ws := workspace.New(fs)
	if err := ws.Detect(); err != nil {
		return nil, nil, fmt.Errorf("failed to detect workspace: %w", err)
	}

	project, err := ws.GetProject(projectName)
	if err != nil {
		return nil, nil, fmt.Errorf("project not found: %w", err)
	}

	return ws, project, nil
}

func resolveProject(fs filesystem.FileSystem, projectFlag string) (*resolvedProject, error) {
	projectName, viaEach, err := resolveProjectName(projectFlag)
	if err != nil {
		return nil, err
	}

	ws, project, err := resolveWorkspaceProject(fs, projectName)
	if err != nil {
		return nil, err
	}

	return &resolvedProject{
		Name:      projectName,
		ViaEach:   viaEach,
		Workspace: ws,
		Project:   project,
	}, nil
}

// projectContextBuilder creates ProjectContext values for workspace projects.
type projectContextBuilder struct {
	fs  filesystem.FileSystem
	git git.GitClient
}

func newProjectContextBuilder(fs filesystem.FileSystem, gitClient git.GitClient) *projectContextBuilder {
	return &projectContextBuilder{fs: fs, git: gitClient}
}

func (b *projectContextBuilder) Build(ws *workspace.Workspace) ([]*models.ProjectContext, error) {
	csManager := changeset.NewManager(b.fs, ws.ChangesetDir())
	allChangesets, err := csManager.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read changesets: %w", err)
	}

	contexts := make([]*models.ProjectContext, 0, len(ws.Projects))

	for _, project := range ws.Projects {
		ctx := &models.ProjectContext{
			Project:          project.Name,
			ProjectPath:      project.RootPath,
			ModulePath:       project.ModulePath,
			Changesets:       []models.ChangesetSummary{},
			HasVersionFile:   hasVersionFile(b.fs, project),
			ChangelogPreview: "",
		}

		projectChangesets := changeset.FilterByProject(allChangesets, project.Name)
		ctx.HasChangesets = len(projectChangesets) > 0

		for _, cs := range projectChangesets {
			bump, _ := cs.GetBumpForProject(project.Name)
			ctx.Changesets = append(ctx.Changesets, models.ChangesetSummary{
				ID:       cs.ID,
				BumpType: bump,
				Message:  cs.Message,
			})
		}

		versionStore := versioning.NewVersionStore(b.fs, project.Type)
		if currentVer, err := versionStore.Read(project.RootPath); err == nil {
			ctx.CurrentVersion = currentVer.String()
		} else {
			ctx.CurrentVersion = "0.0.0"
		}

		ctx.LatestTag = latestTagVersion(b.git, project.Name)

		currentVer, _ := models.ParseVersion(ctx.CurrentVersion)
		latestVer, _ := models.ParseVersion(ctx.LatestTag)
		ctx.IsOutdated = currentVer.Compare(latestVer) > 0

		if len(projectChangesets) > 0 {
			changelog := changelog.NewChangelog(b.fs)
			preview, err := changelog.FormatEntry(projectChangesets, project.Name, project.RootPath)
			if err != nil {
				return nil, fmt.Errorf("failed to format changelog preview for %s: %w", project.Name, err)
			}
			ctx.ChangelogPreview = preview
		}

		contexts = append(contexts, ctx)
	}

	return contexts, nil
}

func hasVersionFile(fs filesystem.FileSystem, project *models.Project) bool {
	return fs.Exists(filepath.Join(project.RootPath, "version.txt"))
}

func latestTagVersion(gitClient git.GitClient, projectName string) string {
	if gitClient == nil {
		return "0.0.0"
	}

	latestTag, err := gitClient.GetLatestTag(projectName)
	if err != nil {
		return "0.0.0"
	}

	parts := strings.Split(latestTag, "@")
	if len(parts) != 2 {
		return "0.0.0"
	}

	return strings.TrimPrefix(parts[1], "v")
}

func parseFilters(filters []string) ([]models.FilterType, error) {
	if len(filters) == 0 {
		return []models.FilterType{models.FilterAll}, nil
	}

	out := make([]models.FilterType, 0, len(filters))
	for _, f := range filters {
		ft, err := models.ParseFilterType(f)
		if err != nil {
			return nil, err
		}
		out = append(out, ft)
	}
	return out, nil
}

// filterContexts applies AND logic across filters.
func filterContexts(contexts []*models.ProjectContext, filters []models.FilterType) ([]*models.ProjectContext, error) {
	if len(filters) == 0 {
		return contexts, nil
	}

	for _, f := range filters {
		if f == models.FilterAll {
			return contexts, nil
		}
	}

	filtered := make([]*models.ProjectContext, 0, len(contexts))
	for _, ctx := range contexts {
		matches := true
		for _, filter := range filters {
			if !filter.MatchesContext(ctx) {
				matches = false
				break
			}
		}
		if matches {
			filtered = append(filtered, ctx)
		}
	}

	return filtered, nil
}

// readProjectContextFromStdin attempts to read project context from STDIN
// Returns nil error if valid context found, error otherwise
//
// This function is used by commands to auto-detect when they're being
// executed via 'changeset each' and receive project context via STDIN.
func readProjectContextFromStdin() (*models.ProjectContext, error) {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat STDIN: %w", err)
	}

	// Check if there's data on STDIN (not a terminal)
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		// STDIN is a terminal, no piped data
		return nil, fmt.Errorf("no context on STDIN")
	}

	// Read JSON from STDIN
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil, fmt.Errorf("failed to read STDIN: %w", err)
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("empty STDIN")
	}

	// Parse JSON
	var ctx models.ProjectContext
	if err := json.Unmarshal(data, &ctx); err != nil {
		return nil, fmt.Errorf("failed to parse context JSON: %w", err)
	}

	// Validate required fields
	if ctx.Project == "" {
		return nil, fmt.Errorf("invalid context: project name is required")
	}

	return &ctx, nil
}

type gitOperator struct {
	git      git.GitClient
	ghClient github.GitHubClient
}

func enrichChangesetsWithPRInfo(git git.GitClient, ghClient github.GitHubClient, changesets []*models.Changeset, owner, repo string) error {
	return (&gitOperator{
		git:      git,
		ghClient: ghClient,
	}).EnrichChangesetsWithPRInfo(changesets, owner, repo)
}

func getLatestNonRCVersion(git git.GitClient, projectName string) (*models.Version, error) {
	return (&gitOperator{
		git: git,
	}).GetLatestNonRCVersion(projectName)
}

func (c *gitOperator) EnrichChangesetsWithPRInfo(changesets []*models.Changeset, owner, repo string) error {
	if c.git == nil {
		fmt.Println("⚠️  Git client not available, skipping PR enrichment")
		return nil
	}

	ghClient := c.ghClient
	if ghClient == nil {
		fmt.Printf("⚠️  GitHub client not authenticated; PR enrichment may fail for private/internal repos: %+v\n", github.ErrGitHubTokenNotFound)
		ghClient = github.NewClientWithoutAuth()
	}

	enricher := github.NewPREnricher(c.git, ghClient)
	res, err := enricher.Enrich(context.Background(), changesets, owner, repo)
	if err != nil {
		return fmt.Errorf("failed to enrich changesets with PR info: %w", err)
	}

	for _, warn := range res.Warnings {
		fmt.Printf("⚠️  Warning: %v\n", warn)
	}

	if res.Enriched > 0 {
		fmt.Printf("✓ Enriched %d changeset(s) with PR information\n\n", res.Enriched)
	}

	return nil
}

func (c *gitOperator) GetLatestNonRCVersion(projectName string) (*models.Version, error) {
	if c.git == nil {
		return nil, fmt.Errorf("git client not available")
	}

	prefix := fmt.Sprintf("%s@v*", projectName)
	tags, err := c.git.GetTagsWithPrefix(prefix)
	if err != nil {
		return nil, fmt.Errorf("failed to get tags: %w", err)
	}

	for _, tag := range tags {
		rcNum, _ := c.git.ExtractRCNumber(tag)
		if rcNum >= 0 {
			continue
		}

		parts := strings.Split(tag, "@")
		if len(parts) != 2 {
			continue
		}

		version, err := models.ParseVersion(parts[1])
		if err != nil {
			continue
		}

		return version, nil
	}

	return nil, fmt.Errorf("no non-RC tags found")
}
