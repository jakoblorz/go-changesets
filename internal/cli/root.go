package cli

import (
	"fmt"

	"github.com/jakoblorz/go-changesets/internal/filesystem"
	"github.com/jakoblorz/go-changesets/internal/git"
	"github.com/jakoblorz/go-changesets/internal/github"
	"github.com/spf13/cobra"
)

// NewRootCommand creates the root command
func NewRootCommand(fs filesystem.FileSystem, gitClient git.GitClient, ghClient github.GitHubClient) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "changeset",
		Short: "Manage changesets for Go monorepos",
		Long: `A CLI tool for managing changesets in Go monorepos.
		
Changesets help track changes, version projects, and publish releases.`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Default to `changeset add` when no subcommand is provided.
			return (&AddCommand{fs: fs}).Run(cmd, args)
		},
	}

	// Add subcommands
	rootCmd.AddCommand(NewAddCommand(fs))
	rootCmd.AddCommand(NewVersionCommand(fs, gitClient, ghClient))
	rootCmd.AddCommand(NewChangelogCommand(fs))
	rootCmd.AddCommand(NewTreeCommand(fs, gitClient))
	rootCmd.AddCommand(NewPublishCommand(fs, gitClient, ghClient))
	rootCmd.AddCommand(NewSnapshotCommand(fs, gitClient, ghClient))
	rootCmd.AddCommand(NewEachCommand(fs, gitClient, nil))

	return rootCmd
}

// Execute runs the root command
func Execute() error {
	fs := filesystem.NewOSFileSystem()
	gitClient := git.NewOSGitClient()
	ghClient, _ := github.NewClientFromEnv()

	rootCmd := NewRootCommand(fs, gitClient, ghClient)

	if err := rootCmd.Execute(); err != nil {
		return fmt.Errorf("command failed: %w", err)
	}

	return nil
}
