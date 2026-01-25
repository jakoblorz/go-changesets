package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/jakoblorz/go-changesets/internal/filesystem"
	"github.com/jakoblorz/go-changesets/internal/git"
	"github.com/jakoblorz/go-changesets/internal/github"
	"github.com/jakoblorz/go-changesets/internal/models"
	"github.com/jakoblorz/go-changesets/internal/versioning"
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

func filterSlices[T any](slice []T, predicate func(T) bool) []T {
	result := make([]T, 0, len(slice))
	for _, item := range slice {
		if predicate(item) {
			result = append(result, item)
		}
	}

	return result
}

func findSlices[T any](slice []T, predicate func(T) bool) (T, bool) {
	for _, item := range slice {
		if predicate(item) {
			return item, true
		}
	}

	var zero T
	return zero, false
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

	mapping, err := github.ReadPRMapping(mappingFile)
	if err != nil {
		return fmt.Errorf("failed to read mapping file: %w", err)
	}

	pr, err := c.ghClient.GetPullRequestByHead(cmd.Context(), owner, repo, branchName)
	if err != nil {
		fmt.Printf("⚠️  Failed to get open PR for %s (skipping): %v\n", ctx.Project, err)
		return nil
	}

	group := tree.GetGroupForProject(ctx.Project)
	if group == nil {
		fmt.Printf("⚠️  Failed to get group for project %s (skipping)\n", ctx.Project)
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

// func (c *GHLinkCommand) Run(cmd *cobra.Command, args []string) error {
// 	owner, _ := cmd.Flags().GetString("owner")
// 	repo, _ := cmd.Flags().GetString("repo")
// 	treeFile, _ := cmd.Flags().GetString("tree-file")
// 	mappingFile, _ := cmd.Flags().GetString("mapping-file")

// 	if owner == "" {
// 		return fmt.Errorf("--owner is required")
// 	}
// 	if repo == "" {
// 		return fmt.Errorf("--repo is required")
// 	}

// 	treeData, err := os.ReadFile(treeFile)
// 	if err != nil {
// 		return fmt.Errorf("failed to read tree file: %w", err)
// 	}

// 	var tree TreeOutput
// 	if err := json.Unmarshal(treeData, &tree); err != nil {
// 		return fmt.Errorf("failed to parse tree JSON: %w", err)
// 	}

// 	mapping, err := github.ReadPRMapping(mappingFile)
// 	if err != nil {
// 		return fmt.Errorf("failed to read mapping file: %w", err)
// 	}

// 	ws := workspace.New(c.fs)
// 	if err := ws.Detect(); err != nil {
// 		return fmt.Errorf("failed to detect workspace: %w", err)
// 	}

// 	updated := 0
// 	for _, group := range tree.Groups {
// 		if len(group.Projects) <= 1 {
// 			continue
// 		}

// 		var relatedPRs []github.RelatedPRInfo
// 		for _, proj := range group.Projects {
// 			entry, ok := mapping.Get(proj.Name)
// 			if ok {
// 				relatedPRs = append(relatedPRs, github.RelatedPRInfo{
// 					Number:  entry.Number,
// 					Project: proj.Name,
// 					Version: entry.Version,
// 				})
// 			}
// 		}

// 		if len(relatedPRs) <= 1 {
// 			continue
// 		}

// 		for _, relatedPR := range relatedPRs {
// 			entry, ok := mapping.Get(relatedPR.Project)
// 			if !ok {
// 				continue
// 			}

// 			projectInfo, ok := findSlices(group.Projects, func(p ProjectChangesetsInfo) bool {
// 				return p.Name == relatedPR.Project
// 			})
// 			if !ok {
// 				fmt.Printf("⚠️  Could not find project info for %s in tree group\n", relatedPR.Project)
// 				continue
// 			}

// 			project, err := ws.GetProject(relatedPR.Project)
// 			if err != nil {
// 				fmt.Printf("⚠️  Failed to get project %s: %v\n", relatedPR.Project, err)
// 				continue
// 			}

// 			pr, err := c.ghClient.GetPullRequest(cmd.Context(), owner, repo, entry.Number)
// 			if err != nil {
// 				fmt.Printf("⚠️  Failed to get PR #%d for %s: %v\n", entry.Number, relatedPR.Project, err)
// 				continue
// 			}

// 			relatedPRsFiltered := filterSlices(relatedPRs, func(r github.RelatedPRInfo) bool {
// 				return r.Project != relatedPR.Project
// 			})

// 			body, err := github.NewPRRenderer(c.fs).RenderBody(github.TemplateData{
// 				Project: projectInfo.Name,
// 				Version: entry.Version,
// 				// CurrentVersion:   entry.Version,
// 				ChangelogPreview: projectInfo.ChangelogPreview,
// 				RelatedPRs:       relatedPRsFiltered,
// 			}, project.RootPath)
// 			if err != nil {
// 				return fmt.Errorf("failed to build PR body: %w", err)
// 			}

// 			title, err := github.NewPRRenderer(c.fs).RenderTitle(github.TemplateData{
// 				Project: projectInfo.Name,
// 				Version: entry.Version,
// 				// CurrentVersion:   entry.Version,
// 				ChangelogPreview: projectInfo.ChangelogPreview,
// 				RelatedPRs:       relatedPRsFiltered,
// 			}, project.RootPath)
// 			if err != nil {
// 				return fmt.Errorf("failed to build PR title: %w", err)
// 			}

// 			_, err = c.ghClient.UpdatePullRequest(cmd.Context(), owner, repo, pr.Number, &github.UpdatePullRequestRequest{
// 				Title: title,
// 				Body:  body,
// 			})
// 			if err != nil {
// 				fmt.Printf("⚠️  Failed to update PR #%d for %s: %v\n", entry.Number, relatedPR.Project, err)
// 				continue
// 			}

// 			fmt.Printf("✓ Updated PR #%d (%s) with related PR links\n", pr.Number, relatedPR.Project)
// 			updated++
// 		}
// 	}

// 	if updated == 0 {
// 		fmt.Println("No PRs needed linking")
// 	} else {
// 		fmt.Printf("\n✓ Linked %d PR(s)\n", updated)
// 	}

// 	return nil
// }
