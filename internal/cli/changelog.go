package cli

import (
	"fmt"

	"github.com/jakoblorz/go-changesets/internal/changelog"
	"github.com/jakoblorz/go-changesets/internal/changeset"
	"github.com/jakoblorz/go-changesets/internal/filesystem"
	"github.com/spf13/cobra"
)

// ChangelogCommand handles the changelog command
type ChangelogCommand struct {
	fs filesystem.FileSystem
}

// NewChangelogCommand creates a new changelog command
func NewChangelogCommand(fs filesystem.FileSystem) *cobra.Command {
	cmd := &ChangelogCommand{fs: fs}

	cobraCmd := &cobra.Command{
		Use:   "changelog",
		Short: "Preview changelog entries for changesets",
		Long: `Displays the changelog content that will be added when versioning.

This command shows what will be added to CHANGELOG.md without modifying any files.
It's useful for previewing changes before running 'changeset version'.

The output can be captured in scripts or CI workflows for use in PR descriptions.`,
		Example: `  # Preview changelog for specific project
  changeset changelog --project auth

  # Preview via 'changeset each' (auto-detects project)
  changeset each --filter open-changesets -- changeset changelog

  # Capture in variable for use in scripts
  PREVIEW=$(changeset changelog --project auth)`,
		RunE: cmd.Run,
	}

	cobraCmd.Flags().StringP("project", "p", "", "Project name to preview (required unless run via 'changeset each')")

	return cobraCmd
}

// Run executes the changelog command
func (c *ChangelogCommand) Run(cmd *cobra.Command, args []string) error {
	projectFlag, _ := cmd.Flags().GetString("project")

	resolved, err := resolveProject(c.fs, projectFlag)
	if err != nil {
		if projectFlag == "" {
			return fmt.Errorf("--project flag required (or run via 'changeset each'): %w", err)
		}
		return err
	}

	csManager := changeset.NewManager(c.fs, resolved.Workspace.ChangesetDir())
	projectChangesets, err := csManager.ReadAllOfProject(resolved.Name)
	if err != nil {
		return fmt.Errorf("failed to read changesets: %w", err)
	}
	if len(projectChangesets) == 0 {
		return nil
	}

	changelog := changelog.NewChangelog(c.fs)
	preview, err := changelog.FormatEntry(projectChangesets, resolved.Project.Name, resolved.Project.RootPath)
	if err != nil {
		return fmt.Errorf("failed to format changelog preview: %w", err)
	}

	fmt.Print(preview)
	return nil
}
