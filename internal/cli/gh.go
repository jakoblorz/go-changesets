package cli

import (
	"github.com/jakoblorz/go-changesets/internal/filesystem"
	"github.com/jakoblorz/go-changesets/internal/git"
	"github.com/jakoblorz/go-changesets/internal/github"
	"github.com/spf13/cobra"
)

func NewGHCommand(fs filesystem.FileSystem, git git.GitClient, ghClient github.GitHubClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gh",
		Short: "GitHub operations for release PR management",
		Long: `GitHub operations for managing release pull requests.

Includes commands for creating, linking, and closing release PRs.`,
	}

	cmd.PersistentFlags().String("owner", "", "GitHub repository owner (required)")
	cmd.PersistentFlags().String("repo", "", "GitHub repository name (required)")

	prCmd := &cobra.Command{
		Use:   "pr",
		Short: "Pull request operations",
		Long:  "Operations for managing release pull requests.",
	}
	prCmd.AddCommand(NewGHOpenCommand(fs, git, ghClient))
	prCmd.AddCommand(NewGHLinkCommand(fs, ghClient))
	// prCmd.AddCommand(NewGHCloseCommand(fs, ghClient))

	cmd.AddCommand(prCmd)

	return cmd
}
