package cli

import (
	"fmt"

	"github.com/jakoblorz/go-changesets/internal/filesystem"
	"github.com/jakoblorz/go-changesets/internal/git"
	"github.com/jakoblorz/go-changesets/internal/github"
	"github.com/jakoblorz/go-changesets/internal/models"
	"github.com/jakoblorz/go-changesets/internal/versioning"
	"github.com/jakoblorz/go-changesets/internal/workspace"
	"github.com/spf13/cobra"
)

type GHCloseCommand struct {
	fs       filesystem.FileSystem
	git      git.GitClient
	ghClient github.GitHubClient
}

func NewGHCloseCommand(fs filesystem.FileSystem, git git.GitClient, ghClient github.GitHubClient) *cobra.Command {
	cmd := &GHCloseCommand{
		fs:       fs,
		git:      git,
		ghClient: ghClient,
	}

	cobraCmd := &cobra.Command{
		Use:   "close",
		Short: "Close obsolete release PRs",
		Long: `Close obsolete release PRs that are no longer needed.

This command finds and closes release PRs that have no remaining changesets.`,
		Example: `  # Close obsolete PRs
  changeset gh pr close --owner myorg --repo myrepo

  # For specific project
  changeset gh pr close --owner myorg --repo myrepo --project auth`,
		RunE: cmd.Run,
	}

	cobraCmd.Flags().String("project", "", "Project name (required unless run via 'changeset each')")

	return cobraCmd
}

func (c *GHCloseCommand) Run(cmd *cobra.Command, args []string) error {
	owner, _ := cmd.Flags().GetString("owner")
	repo, _ := cmd.Flags().GetString("repo")
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

	var ctx *models.ProjectContext
	if resolved.ViaEach {
		ctx, err = newProjectContextBuilder(c.fs, c.git).BuildFromEnv()
		if err != nil {
			return fmt.Errorf("failed to build project context from environment: %w", err)
		}

		// when receiving context via each, we nee to update a few fields in case they are outdated (when run via each --from-tree-file, etc)

		// always read in the current version, even if set via each. we need the "new" version (after running 'version') for the PR title/body
		versionStore := versioning.NewVersionStore(c.fs, resolved.Project.Type)
		if currentVer, err := versionStore.Read(resolved.Project.RootPath); err == nil {
			ctx.CurrentVersion = currentVer.String()
		} else {
			ctx.CurrentVersion = "0.0.0"
		}

		// we are on the "latest" version after 'changeset version', so we are not "outdated"
		ctx.IsOutdated = false
	} else {
		ctxs, err := newProjectContextBuilder(c.fs, c.git).BuildFromWorkspace(ws)
		if err != nil {
			return fmt.Errorf("failed to build project contexts: %w", err)
		}

		ctxs, err = filterContextsByName(ctxs, []string{resolved.Name})
		if err != nil {
			return fmt.Errorf("failed to filter project contexts: %w", err)
		}
		if len(ctxs) != 1 {
			return fmt.Errorf("project context for %s not found", resolved.Name)
		}
		ctx = ctxs[0]
	}

	branchName, err := c.git.GetCurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current git branch: %w", err)
	}

	pr, err := c.ghClient.GetPullRequestByHead(cmd.Context(), owner, repo, branchName)
	if err != nil {
		return fmt.Errorf("failed to get open PR for %s: %w", ctx.Project, err)
	}
	if pr == nil {
		fmt.Printf("⚠️  No open PR found for %s, skipping\n", ctx.Project)
		return nil
	}

	// fmt.Sprintf("✅ This release PR is no longer needed (no changesets remaining for %s). If new changesets are added, a new PR will be created automatically.", ctx.Project)

	if err := c.ghClient.ClosePullRequest(cmd.Context(), owner, repo, pr.Number); err != nil {
		return fmt.Errorf("failed to close PR: %w", err)
	}

	fmt.Printf("✓ Closed PR #%d for %s\n", pr.Number, ctx.Project)

	if err := c.ghClient.DeleteBranch(cmd.Context(), owner, repo, branchName); err != nil {
		fmt.Printf("⚠️  Failed to delete branch %s: %v\n", branchName, err)
	} else {
		fmt.Printf("  Deleted branch %s\n", branchName)
	}

	return nil
}
