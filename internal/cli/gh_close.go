package cli

import (
	"fmt"

	"github.com/jakoblorz/go-changesets/internal/filesystem"
	"github.com/jakoblorz/go-changesets/internal/github"
	"github.com/spf13/cobra"
)

type GHCloseCommand struct {
	fs       filesystem.FileSystem
	ghClient github.GitHubClient
}

func NewGHCloseCommand(fs filesystem.FileSystem, ghClient github.GitHubClient) *cobra.Command {
	cmd := &GHCloseCommand{
		fs:       fs,
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
  changeset gh pr close --owner myorg --repo myrepo --project auth

  # With custom comment
  changeset gh pr close --owner myorg --repo myrepo --comment "PR is no longer needed"`,
		RunE: cmd.Run,
	}

	cobraCmd.Flags().String("comment", "", "Custom close comment")
	cobraCmd.Flags().String("project", "", "Project name (required unless run via 'changeset each')")

	return cobraCmd
}

func (c *GHCloseCommand) Run(cmd *cobra.Command, args []string) error {
	owner, _ := cmd.Flags().GetString("owner")
	repo, _ := cmd.Flags().GetString("repo")
	customComment, _ := cmd.Flags().GetString("comment")
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
	ctx, err := resolved.ToContext()
	if err != nil {
		return fmt.Errorf("failed to get project context: %w", err)
	}

	branchName := fmt.Sprintf("changeset-release/%s", ctx.Project)

	pr, err := c.ghClient.GetPullRequestByHead(cmd.Context(), owner, repo, branchName)
	if err != nil {
		return fmt.Errorf("failed to check for PR: %w", err)
	}

	if pr == nil {
		fmt.Printf("No PR found for %s (branch: %s)\n", ctx.Project, branchName)
		return nil
	}

	if pr.State == "closed" {
		fmt.Printf("PR #%d for %s is already closed\n", pr.Number, ctx.Project)
		return nil
	}

	comment := customComment
	if comment == "" {
		comment = fmt.Sprintf("✅ This release PR is no longer needed (no changesets remaining for %s). If new changesets are added, a new PR will be created automatically.", ctx.Project)
	}

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
