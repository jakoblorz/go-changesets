package cli

import (
	"fmt"

	"github.com/jakoblorz/go-changesets/internal/filesystem"
	"github.com/jakoblorz/go-changesets/internal/git"
	"github.com/jakoblorz/go-changesets/internal/github"
	"github.com/jakoblorz/go-changesets/internal/workspace"
	"github.com/spf13/cobra"
)

type GHOpenCommand struct {
	fs       filesystem.FileSystem
	ghClient github.GitHubClient
	git      git.GitClient
}

func NewGHOpenCommand(fs filesystem.FileSystem, git git.GitClient, ghClient github.GitHubClient) *cobra.Command {
	cmd := &GHOpenCommand{
		fs:       fs,
		ghClient: ghClient,
		git:      git,
	}

	cobraCmd := &cobra.Command{
		Use:   "open",
		Short: "Create or update a release PR",
		Long: `Create or update a release PR for the current project.

This command should be run after 'changeset version' and git commit/push.
Uses .changeset/github_pr_body.tmpl and/or .changeset/github_pr_title.tmpl if present, otherwise uses a default template.`,
		Example: `  # Create release PR for current project (when run via 'changeset each', project is auto-detected)
  changeset gh pr open --owner myorg --repo myrepo

  # For specific project
  changeset gh pr open --owner myorg --repo myrepo --project auth`,
		RunE: cmd.Run,
	}

	cobraCmd.Flags().String("base", "main", "Base branch for PR")
	cobraCmd.Flags().String("labels", "release,automated", "Comma-separated labels for PR")
	cobraCmd.Flags().String("mapping-file", "/tmp/pr-mapping.json", "Path to PR mapping file")
	cobraCmd.Flags().String("project", "", "Project name (required unless run via 'changeset each')")

	return cobraCmd
}

func (c *GHOpenCommand) Run(cmd *cobra.Command, args []string) error {
	owner, _ := cmd.Flags().GetString("owner")
	repo, _ := cmd.Flags().GetString("repo")
	base, _ := cmd.Flags().GetString("base")
	mappingFile, _ := cmd.Flags().GetString("mapping-file")
	projectFlag, _ := cmd.Flags().GetString("project")

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

	ws := workspace.New(c.fs)
	if err := ws.Detect(); err != nil {
		return fmt.Errorf("failed to detect workspace: %w", err)
	}

	ctx, err := resolved.ToCurrentProjectContext(ws, c.fs, c.git)
	if err != nil {
		return fmt.Errorf("failed to obtain project context: %w", err)
	}

	branchName, err := c.git.GetCurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current git branch: %w", err)
	}

	body, err := github.NewPRRenderer(c.fs).RenderBody(github.TemplateData{
		Project:          ctx.Project,
		Version:          ctx.CurrentVersion,
		ChangelogPreview: ctx.ChangelogPreview,
	}, ctx.ProjectPath)
	if err != nil {
		return fmt.Errorf("failed to build PR body: %w", err)
	}

	title, err := github.NewPRRenderer(c.fs).RenderTitle(github.TemplateData{
		Project:          ctx.Project,
		Version:          ctx.CurrentVersion,
		ChangelogPreview: ctx.ChangelogPreview,
	}, ctx.ProjectPath)
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

	if err := c.updateMappingFile(mappingFile, ctx.Project, ctx.CurrentVersion, pr); err != nil {
		return fmt.Errorf("failed to update mapping file: %w", err)
	}

	fmt.Printf("  PR URL: %s\n", pr.HTMLURL)

	return nil
}

func (c *GHOpenCommand) updateMappingFile(path, project, version string, pr *github.PullRequest) error {
	mapping, err := github.ReadPRMapping(path)
	if err != nil {
		mapping = github.NewPRMapping()
	}

	mapping.Set(project, github.PullRequestInfo{
		PullRequest: *pr,
		Version:     version,
		Project:     project,
	})

	return mapping.Write(path)
}
