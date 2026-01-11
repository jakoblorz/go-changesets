package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/jakoblorz/go-changesets/internal/filesystem"
	"github.com/jakoblorz/go-changesets/internal/git"
	"github.com/jakoblorz/go-changesets/internal/models"
	"github.com/jakoblorz/go-changesets/internal/workspace"
	"github.com/spf13/cobra"
)

// EachCommand handles the each command
type EachCommand struct {
	fs      filesystem.FileSystem
	git     git.GitClient
	filters []string
	command []string
}

// NewEachCommand creates a new each command
func NewEachCommand(fs filesystem.FileSystem, gitClient git.GitClient) *cobra.Command {
	cmd := &EachCommand{
		fs:  fs,
		git: gitClient,
	}

	cobraCmd := &cobra.Command{
		Use:   "each [flags] -- <command> [args...]",
		Short: "Run a command for each project matching filters",
		Long: `Run a command for each project, passing project context as JSON via STDIN.

Filters:
  all               - All projects (default)
  open-changesets   - Projects with changesets in .changeset/
  outdated-versions - Projects where version.txt > latest git tag
  has-version       - Projects with a version source file
  no-version        - Projects without a version source file

The command receives a JSON object via STDIN with project context.
Environment variables are also set: PROJECT, PROJECT_PATH, CURRENT_VERSION, LATEST_TAG`,
		Example: `  # Version all projects with changesets
  changeset each --filter=open-changesets -- changeset version

  # Publish all outdated projects
  changeset each --filter=outdated-versions -- changeset publish --owner org --repo repo

  # Custom script
  changeset each --filter=open-changesets -- bash -c 'echo "Releasing $PROJECT"'`,
		RunE: cmd.Run,
	}

	cobraCmd.Flags().StringSliceVar(&cmd.filters, "filter", []string{"all"},
		"Filter projects (open-changesets, outdated-versions, has-version, no-version, unchanged, all)")

	return cobraCmd
}

// Run executes the each command
func (c *EachCommand) Run(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("no command specified (use -- before command)")
	}
	c.command = args

	ws := workspace.New(c.fs)
	if err := ws.Detect(); err != nil {
		return fmt.Errorf("failed to detect workspace: %w", err)
	}

	builder := newProjectContextBuilder(c.fs, c.git)
	contexts, err := builder.Build(ws)
	if err != nil {
		return fmt.Errorf("failed to build project contexts: %w", err)
	}

	filterTypes, err := parseFilters(c.filters)
	if err != nil {
		return fmt.Errorf("failed to parse filters: %w", err)
	}

	filtered, err := filterContexts(contexts, filterTypes)
	if err != nil {
		return fmt.Errorf("failed to filter projects: %w", err)
	}

	if len(filtered) == 0 {
		fmt.Println("No projects match the specified filters")
		return nil
	}

	fmt.Printf("Running command for %d project(s)...\n\n", len(filtered))

	var failed []string
	for i, ctx := range filtered {
		if i > 0 {
			fmt.Println("\n" + strings.Repeat("-", 60) + "\n")
		}

		fmt.Printf("📦 [%d/%d] %s\n", i+1, len(filtered), ctx.Project)

		if err := c.executeForProject(ctx); err != nil {
			fmt.Printf("❌ Failed: %v\n", err)
			failed = append(failed, ctx.Project)
			continue
		}

		fmt.Printf("✓ Success\n")
	}

	if len(failed) > 0 {
		fmt.Printf("\n⚠️  %d project(s) failed: %s\n", len(failed), strings.Join(failed, ", "))
		return fmt.Errorf("some projects failed")
	}

	return nil
}

// executeForProject executes the command for a single project
func (c *EachCommand) executeForProject(ctx *models.ProjectContext) error {
	jsonData, err := json.MarshalIndent(ctx, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal context: %w", err)
	}

	cmdName := c.command[0]
	cmdArgs := c.command[1:]

	execCmd := exec.Command(cmdName, cmdArgs...)
	execCmd.Stdin = bytes.NewReader(jsonData)
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr

	execCmd.Env = append(os.Environ(),
		fmt.Sprintf("PROJECT=%s", ctx.Project),
		fmt.Sprintf("PROJECT_PATH=%s", ctx.ProjectPath),
		fmt.Sprintf("CURRENT_VERSION=%s", ctx.CurrentVersion),
		fmt.Sprintf("LATEST_TAG=%s", ctx.LatestTag),
		fmt.Sprintf("CHANGELOG_PREVIEW=%s", ctx.ChangelogPreview),
		fmt.Sprintf("CHANGESET_CONTEXT=%s", string(jsonData)),
	)

	return execCmd.Run()
}
