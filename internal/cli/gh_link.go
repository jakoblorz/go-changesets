package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/jakoblorz/go-changesets/internal/filesystem"
	"github.com/jakoblorz/go-changesets/internal/github"
	"github.com/spf13/cobra"
)

type GHLinkCommand struct {
	fs       filesystem.FileSystem
	ghClient github.GitHubClient
}

func NewGHLinkCommand(fs filesystem.FileSystem, ghClient github.GitHubClient) *cobra.Command {
	cmd := &GHLinkCommand{
		fs:       fs,
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

	return cobraCmd
}

func (c *GHLinkCommand) Run(cmd *cobra.Command, args []string) error {
	owner, _ := cmd.Flags().GetString("owner")
	repo, _ := cmd.Flags().GetString("repo")
	treeFile, _ := cmd.Flags().GetString("tree-file")
	mappingFile, _ := cmd.Flags().GetString("mapping-file")

	if owner == "" {
		return fmt.Errorf("--owner is required")
	}
	if repo == "" {
		return fmt.Errorf("--repo is required")
	}

	treeData, err := os.ReadFile(treeFile)
	if err != nil {
		return fmt.Errorf("failed to read tree file: %w", err)
	}

	var tree TreeOutput
	if err := json.Unmarshal(treeData, &tree); err != nil {
		return fmt.Errorf("failed to parse tree JSON: %w", err)
	}

	mapping, err := github.ReadPRMapping(mappingFile)
	if err != nil {
		return fmt.Errorf("failed to read mapping file: %w", err)
	}

	updated := 0
	for _, group := range tree.Groups {
		if len(group.Projects) <= 1 {
			continue
		}

		var relatedPRs []github.RelatedPRInfo
		for _, proj := range group.Projects {
			entry, ok := mapping.Get(proj.Name)
			if ok {
				relatedPRs = append(relatedPRs, github.RelatedPRInfo{
					Number:  entry.PRNumber,
					Project: proj.Name,
					Version: entry.Version,
				})
			}
		}

		if len(relatedPRs) <= 1 {
			continue
		}

		section := github.BuildRelatedPRsSection(group.Commit, relatedPRs, "")

		for _, relatedPR := range relatedPRs {
			entry, ok := mapping.Get(relatedPR.Project)
			if !ok {
				continue
			}

			pr, err := c.ghClient.GetPullRequest(cmd.Context(), owner, repo, entry.PRNumber)
			if err != nil {
				fmt.Printf("⚠️  Failed to get PR #%d for %s: %v\n", entry.PRNumber, relatedPR.Project, err)
				continue
			}

			newBody := github.ReplaceRelatedPRsPlaceholder(pr.Body, section)

			_, err = c.ghClient.UpdatePullRequest(cmd.Context(), owner, repo, pr.Number, &github.UpdatePullRequestRequest{
				Title: pr.Title,
				Body:  newBody,
			})
			if err != nil {
				fmt.Printf("⚠️  Failed to update PR #%d for %s: %v\n", entry.PRNumber, relatedPR.Project, err)
				continue
			}

			fmt.Printf("✓ Updated PR #%d (%s) with related PR links\n", pr.Number, relatedPR.Project)
			updated++
		}
	}

	if updated == 0 {
		fmt.Println("No PRs needed linking")
	} else {
		fmt.Printf("\n✓ Linked %d PR(s)\n", updated)
	}

	return nil
}
