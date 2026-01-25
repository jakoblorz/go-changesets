package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/jakoblorz/go-changesets/internal/filesystem"
	"github.com/jakoblorz/go-changesets/internal/git"
	"github.com/jakoblorz/go-changesets/internal/github"
	"github.com/jakoblorz/go-changesets/internal/workspace"
	"github.com/spf13/cobra"
)

type GHLinkCommand struct {
	fs       filesystem.FileSystem
	git      git.GitClient
	ghClient github.GitHubClient
}

func NewGHLinkCommand(fs filesystem.FileSystem, git git.GitClient, ghClient github.GitHubClient) *cobra.Command {
	cmd := &GHLinkCommand{
		fs:       fs,
		git:      git,
		ghClient: ghClient,
	}

	cobraCmd := &cobra.Command{
		Use:   "link",
		Short: "Link related release PRs together",
		Long: `Link related release PRs together using changeset tree data.

This command uses pre-captured tree data to link related PRs together.`,
		Example: `  # Link PRs using default paths
  changeset gh pr link --owner myorg --repo myrepo

  # With custom paths
  changeset gh pr link --owner myorg --repo myrepo --tree-file /tmp/tree.json --mapping-file /tmp/pr-mapping.json`,
		RunE: cmd.Run,
	}

	cobraCmd.Flags().String("tree-file", "/tmp/tree.json", "Path to tree JSON file from 'changeset tree --format json'")
	cobraCmd.Flags().String("mapping-file", "/tmp/pr-mapping.json", "Path to PR mapping file")
	cobraCmd.Flags().String("project", "", "Project name (required unless run via 'changeset each')")

	return cobraCmd
}

func (c *GHLinkCommand) Run(cmd *cobra.Command, args []string) error {
	owner, _ := cmd.Flags().GetString("owner")
	repo, _ := cmd.Flags().GetString("repo")
	treeFile, _ := cmd.Flags().GetString("tree-file")
	mappingFile, _ := cmd.Flags().GetString("mapping-file")
	projectFlag, _ := cmd.Flags().GetString("project")

	if owner == "" {
		return fmt.Errorf("--owner is required")
	}
	if repo == "" {
		return fmt.Errorf("--repo is required")
	}
	if treeFile == "" {
		return fmt.Errorf("--tree-file cannot be empty")
	}
	if mappingFile == "" {
		return fmt.Errorf("--mapping-file cannot be empty")
	}

	treeData, err := os.ReadFile(treeFile)
	if err != nil {
		return fmt.Errorf("failed to read tree file: %w", err)
	}

	var tree TreeOutput
	if err := json.Unmarshal(treeData, &tree); err != nil {
		return fmt.Errorf("failed to parse tree JSON: %w", err)
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

	mapping, err := github.ReadPRMapping(mappingFile)
	if err != nil {
		return fmt.Errorf("failed to read mapping file: %w", err)
	}

	pr, err := c.ghClient.GetPullRequestByHead(cmd.Context(), owner, repo, branchName)
	if err != nil {
		return fmt.Errorf("failed to get open PR for %s: %w", ctx.Project, err)
	}
	if pr == nil {
		fmt.Printf("⚠️  No open PR found for %s, skipping\n", ctx.Project)
		return nil
	}

	group := tree.GetGroupForProject(ctx.Project)
	if group == nil {
		fmt.Printf("⚠️  Failed to get group for project %s, skipping\n", ctx.Project)
		return nil
	}

	var relatedPRs []github.RelatedPRInfo
	for _, proj := range group.Projects {
		entry, ok := mapping.Projects[proj.Name]
		if ok && proj.Name != ctx.Project {
			relatedPRs = append(relatedPRs, github.RelatedPRInfo{
				Number:  entry.Number,
				Project: entry.Project,
				Version: entry.Version,
			})
		}
	}
	if len(relatedPRs) == 0 {
		fmt.Printf("ℹ️  No related PRs found for %s\n", ctx.Project)
		return nil
	}

	body, err := github.NewPRRenderer(c.fs).RenderBody(github.TemplateData{
		Project:          ctx.Project,
		Version:          ctx.CurrentVersion,
		ChangelogPreview: ctx.ChangelogPreview,
		RelatedPRs:       relatedPRs,
	}, ctx.ProjectPath)
	if err != nil {
		return fmt.Errorf("failed to build PR body: %w", err)
	}

	title, err := github.NewPRRenderer(c.fs).RenderTitle(github.TemplateData{
		Project:          ctx.Project,
		Version:          ctx.CurrentVersion,
		ChangelogPreview: ctx.ChangelogPreview,
		RelatedPRs:       relatedPRs,
	}, ctx.ProjectPath)
	if err != nil {
		return fmt.Errorf("failed to build PR title: %w", err)
	}

	pr, err = c.ghClient.UpdatePullRequest(cmd.Context(), owner, repo, pr.Number, &github.UpdatePullRequestRequest{
		Title: title,
		Body:  body,
	})
	if err != nil {
		fmt.Printf("⚠️  Failed to update PR #%d for %s: %v\n", pr.Number, ctx.Project, err)
		return nil
	}

	fmt.Printf("✓ Updated PR #%d for %s\n", pr.Number, ctx.Project)

	return nil
}
