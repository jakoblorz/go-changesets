package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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
	fs           filesystem.FileSystem
	git          git.GitClient
	filters      []string
	command      []string
	fromTreeFile string

	stdoutWriter io.Writer
}

// NewEachCommand creates a new each command
func NewEachCommand(fs filesystem.FileSystem, gitClient git.GitClient, stdoutWriter io.Writer) *cobra.Command {
	cmd := &EachCommand{
		fs:           fs,
		git:          gitClient,
		stdoutWriter: stdoutWriter,
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
  changeset each --filter=open-changesets -- bash -c 'echo "Releasing $PROJECT"'

  # Link PRs using tree file (after versioning)
  changeset each --from-tree-file=/tmp/tree.json -- bash -c 'echo "Releasing $PROJECT"'`,
		RunE: cmd.Run,
	}

	cobraCmd.Flags().StringSliceVar(&cmd.filters, "filter", []string{"all"},
		"Filter projects (open-changesets, outdated-versions, has-version, no-version, unchanged, all)")
	cobraCmd.Flags().StringVar(&cmd.fromTreeFile, "from-tree-file", "",
		"Read projects from a tree JSON file instead of workspace filters")

	return cobraCmd
}

// getStdoutWriter returns the output writer or stdout if not set
func (c *EachCommand) getStdoutWriter() io.Writer {
	if c.stdoutWriter == nil {
		return os.Stdout
	}
	return c.stdoutWriter
}

// Run executes the each command
func (c *EachCommand) Run(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("no command specified (use -- before command)")
	}
	c.command = args

	if c.fromTreeFile != "" {
		return c.runFromTreeFile()
	}

	return c.runFromFilter()
}

func (c *EachCommand) runFromFilter() error {
	ws := workspace.New(c.fs)
	if err := ws.Detect(); err != nil {
		return fmt.Errorf("failed to detect workspace: %w", err)
	}

	builder := newProjectContextBuilder(c.fs, c.git)
	contexts, err := builder.BuildFromWorkspace(ws)
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
		fmt.Fprintln(c.getStdoutWriter(), "No projects match the specified filters")
		return nil
	}

	return c.executeForContexts(filtered)
}

func (c *EachCommand) runFromTreeFile() error {
	data, err := c.fs.ReadFile(c.fromTreeFile)
	if err != nil {
		return fmt.Errorf("failed to read tree file: %w", err)
	}

	var tree TreeOutput
	if err := json.Unmarshal(data, &tree); err != nil {
		return fmt.Errorf("failed to parse tree JSON: %w", err)
	}

	contexts, err := newProjectContextBuilder(c.fs, c.git).BuildFromTreeFile(tree)
	if err != nil {
		return fmt.Errorf("failed to build project contexts from tree file: %w", err)
	}

	return c.executeForContexts(contexts)
}

func (c *EachCommand) executeForContexts(contexts []*models.ProjectContext) error {
	fmt.Fprintf(c.getStdoutWriter(), "Running command for %d project(s)...\n\n", len(contexts))

	var failed []string
	for i, ctx := range contexts {
		if i > 0 {
			fmt.Fprintln(c.getStdoutWriter(), "\n"+strings.Repeat("-", 60)+"\n")
		}

		fmt.Fprintf(c.getStdoutWriter(), "📦 [%d/%d] %s\n", i+1, len(contexts), ctx.Project)

		if err := c.executeForProject(ctx); err != nil {
			fmt.Fprintf(c.getStdoutWriter(), "❌ Failed: %v\n", err)
			failed = append(failed, ctx.Project)
			continue
		}

		fmt.Fprintln(c.getStdoutWriter(), "✓ Success")
	}

	if len(failed) > 0 {
		fmt.Fprintf(c.getStdoutWriter(), "\n⚠️  %d project(s) failed: %s\n", len(failed), strings.Join(failed, ", "))
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
	execCmd.Stdout = c.getStdoutWriter()
	execCmd.Stderr = os.Stderr

	execCmd.Env = append(os.Environ(),
		fmt.Sprintf("PROJECT=%s", ctx.Project),
		fmt.Sprintf("PROJECT_PATH=%s", ctx.ProjectPath),
		fmt.Sprintf("CURRENT_VERSION=%s", ctx.CurrentVersion),
		fmt.Sprintf("LATEST_TAG=%s", ctx.LatestTag),
		fmt.Sprintf("CHANGELOG_PREVIEW=%s", ctx.ChangelogPreview),
		fmt.Sprintf("CHANGESET_CONTEXT=%s", string(jsonData)),
		// make sure to update the each_test.go env cases if you add more variables
	)

	return execCmd.Run()
}
