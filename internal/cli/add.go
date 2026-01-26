package cli

import (
	"fmt"

	"github.com/jakoblorz/go-changesets/internal/filesystem"
	"github.com/jakoblorz/go-changesets/internal/tui/add"
	"github.com/jakoblorz/go-changesets/internal/workspace"
	"github.com/spf13/cobra"
)

// AddCommand handles the add command
type AddCommand struct {
	fs filesystem.FileSystem
}

// NewAddCommand creates a new add command
func NewAddCommand(fs filesystem.FileSystem) *cobra.Command {
	cmd := &AddCommand{fs: fs}

	cobraCmd := &cobra.Command{
		Use:   "add",
		Short: "Create a new changeset",
		Long:  `Create a new changeset by selecting projects and describing changes.`,
		RunE:  cmd.Run,
	}

	return cobraCmd
}

// Run executes the add command
func (c *AddCommand) Run(cmd *cobra.Command, args []string) error {
	// Detect workspace
	ws := workspace.New(c.fs)
	if err := ws.Detect(); err != nil {
		return fmt.Errorf("failed to detect workspace: %w", err)
	}

	flow := add.NewFlow(c.fs, ws)
	result, err := flow.Run()
	if err != nil {
		return fmt.Errorf("failed to run TUI: %w", err)
	}

	if result == nil {
		return nil
	}

	_, _ = fmt.Fprintln(cmd.OutOrStdout(), add.RenderSuccess(result))

	return nil
}
