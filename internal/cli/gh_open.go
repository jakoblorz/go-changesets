package cli

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/jakoblorz/go-changesets/internal/filesystem"
	"github.com/jakoblorz/go-changesets/internal/github"
	"github.com/jakoblorz/go-changesets/internal/models"
	"github.com/spf13/cobra"
)

type GHOpenCommand struct {
	fs       filesystem.FileSystem
	ghClient github.GitHubClient
}

func NewGHOpenCommand(fs filesystem.FileSystem, ghClient github.GitHubClient) *cobra.Command {
	cmd := &GHOpenCommand{
		fs:       fs,
		ghClient: ghClient,
	}

	cobraCmd := &cobra.Command{
		Use:   "open",
		Short: "Create or update a release PR",
		Long: `Create or update a release PR for the current project.

This command should be run after 'changeset version' and git commit/push.
Uses .changeset/pr-description.tmpl if present, otherwise uses a default template.`,
		Example: `  # Create release PR for current project
  changeset gh pr open --owner myorg --repo myrepo

  # For specific project
  changeset gh pr open --owner myorg --repo myrepo --project auth

  # With custom title template
  changeset gh pr open --owner myorg --repo myrepo --title "Release {{.Project}} v{{.Version}}"`,
		RunE: cmd.Run,
	}

	cobraCmd.Flags().String("base", "main", "Base branch for PR")
	cobraCmd.Flags().String("labels", "release,automated", "Comma-separated labels for PR")
	cobraCmd.Flags().String("mapping-file", "/tmp/pr-mapping.json", "Path to PR mapping file")
	cobraCmd.Flags().String("project", "", "Project name (required unless run via 'changeset each')")
	cobraCmd.Flags().String("title", "🚀 Release {{.Project}} v{{.Version}}", "PR title template (Go template syntax)")

	return cobraCmd
}

func (c *GHOpenCommand) Run(cmd *cobra.Command, args []string) error {
	owner, _ := cmd.Flags().GetString("owner")
	repo, _ := cmd.Flags().GetString("repo")
	base, _ := cmd.Flags().GetString("base")
	mappingFile, _ := cmd.Flags().GetString("mapping-file")
	projectFlag, _ := cmd.Flags().GetString("project")
	titleTemplate, _ := cmd.Flags().GetString("title")

	if owner == "" {
		return fmt.Errorf("--owner is required")
	}
	if repo == "" {
		return fmt.Errorf("--repo is required")
	}

	resolved, err := resolveProject(c.fs, projectFlag)
	if err != nil {
		if projectFlag == "" {
			return fmt.Errorf("--project flag required (or run via 'changeset each'): %w", err)
		}
		return fmt.Errorf("failed to resolve project: %w", err)
	}
	ctx, err := resolved.ToContext()
	if err != nil {
		return fmt.Errorf("failed to get project context: %w", err)
	}

	version, err := c.readVersion(ctx.ProjectPath)
	if err != nil {
		return fmt.Errorf("failed to read version: %w", err)
	}

	branchName := fmt.Sprintf("changeset-release/%s", ctx.Project)

	body, err := c.buildPRBody(ctx)
	if err != nil {
		return fmt.Errorf("failed to build PR body: %w", err)
	}

	title, err := c.buildTitle(titleTemplate, ctx.Project, version)
	if err != nil {
		return fmt.Errorf("failed to build PR title: %w", err)
	}

	var pr *github.PullRequest
	existingPR, err := c.ghClient.GetPullRequestByHead(cmd.Context(), owner, repo, branchName)
	if err != nil {
		return fmt.Errorf("failed to check for existing PR: %w", err)
	}

	if existingPR != nil {
		pr, err = c.ghClient.UpdatePullRequest(cmd.Context(), owner, repo, existingPR.Number, &github.UpdatePullRequestRequest{
			Title: title,
			Body:  body,
		})
		if err != nil {
			return fmt.Errorf("failed to update PR: %w", err)
		}
		fmt.Printf("✓ Updated PR #%d for %s\n", pr.Number, ctx.Project)
	} else {
		pr, err = c.ghClient.CreatePullRequest(cmd.Context(), owner, repo, &github.CreatePullRequestRequest{
			Title: title,
			Body:  body,
			Head:  branchName,
			Base:  base,
			Draft: false,
		})
		if err != nil {
			return fmt.Errorf("failed to create PR: %w", err)
		}
		fmt.Printf("✓ Created PR #%d for %s\n", pr.Number, ctx.Project)
	}

	if err := c.updateMappingFile(mappingFile, ctx.Project, pr.Number, branchName, version); err != nil {
		return fmt.Errorf("failed to update mapping file: %w", err)
	}

	fmt.Printf("  PR URL: %s\n", pr.HTMLURL)

	return nil
}

func (c *GHOpenCommand) readVersion(projectPath string) (string, error) {
	versionPath := filepath.Join(projectPath, "version.txt")
	data, err := c.fs.ReadFile(versionPath)
	if err != nil {
		return "0.0.0", nil
	}
	return strings.TrimSpace(string(data)), nil
}

func (c *GHOpenCommand) buildPRBody(ctx *models.ProjectContext) (string, error) {
	templatePath := filepath.Join(".changeset", "pr-description.tmpl")

	if !c.fs.Exists(templatePath) {
		return github.ExecuteDefaultTemplate("body", github.TemplateData{
			Project:          ctx.Project,
			Version:          ctx.CurrentVersion,
			CurrentVersion:   ctx.CurrentVersion,
			ChangelogPreview: ctx.ChangelogPreview,
		})
	}

	tmpl, err := github.ParseTemplateFile(templatePath)
	if err != nil {
		return "", err
	}

	return github.ExecuteTemplate(tmpl, "pr-body", github.TemplateData{
		Project:          ctx.Project,
		Version:          ctx.CurrentVersion,
		CurrentVersion:   ctx.CurrentVersion,
		ChangelogPreview: ctx.ChangelogPreview,
	})
}

func (c *GHOpenCommand) buildTitle(templateStr, project, version string) (string, error) {
	data := github.TemplateData{
		Project: project,
		Version: version,
	}

	return github.ExecuteDefaultTemplate("title", data)
}

func (c *GHOpenCommand) updateMappingFile(path, project string, prNumber int, branch, version string) error {
	mapping, err := github.ReadPRMapping(path)
	if err != nil {
		mapping = github.NewPRMapping()
	}

	mapping.Set(project, github.PREntry{
		PRNumber: prNumber,
		Branch:   branch,
		Version:  version,
	})

	return mapping.Write(path)
}
