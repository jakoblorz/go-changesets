package cli

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/jakoblorz/go-changesets/internal/changeset"
	"github.com/jakoblorz/go-changesets/internal/filesystem"
	"github.com/jakoblorz/go-changesets/internal/git"
	"github.com/jakoblorz/go-changesets/internal/models"
	"github.com/jakoblorz/go-changesets/internal/workspace"
	"github.com/spf13/cobra"
)

// TreeCommand handles the tree command
type TreeCommand struct {
	fs  filesystem.FileSystem
	git git.GitClient
}

// ChangesetGroup represents a group of related changesets (from same commit)
type ChangesetGroup struct {
	Commit      string                         `json:"commit"`
	CommitShort string                         `json:"commitShort"`
	Message     string                         `json:"message"`
	Projects    []ProjectChangesetsInfo        `json:"projects"`
	projectsMap map[string][]*models.Changeset // Internal use only
}

// ProjectChangesetsInfo represents changesets for a project in a group
type ProjectChangesetsInfo struct {
	Name       string          `json:"name"`
	Changesets []ChangesetInfo `json:"changesets"`
}

// ChangesetInfo represents a single changeset's info
type ChangesetInfo struct {
	ID      string `json:"id"`
	File    string `json:"file"`
	Bump    string `json:"bump"`
	Message string `json:"message"`
}

// TreeOutput represents the complete tree output
type TreeOutput struct {
	Groups []ChangesetGroup `json:"groups"`
}

// NewTreeCommand creates a new tree command
func NewTreeCommand(fs filesystem.FileSystem, gitClient git.GitClient) *cobra.Command {
	cmd := &TreeCommand{
		fs:  fs,
		git: gitClient,
	}

	cobraCmd := &cobra.Command{
		Use:   "tree",
		Short: "Show changeset relationships and groupings",
		Long: `Analyzes changesets to determine which are related.

Changesets created in the same git commit are considered related.
This is useful for understanding which release PRs are related and should
be reviewed together.

The command groups changesets by the commit that created them, helping you
coordinate reviews when a single feature affects multiple projects.`,
		Example: `  # Show tree in human-readable format
  changeset tree --filter open-changesets
  
  # Output JSON for scripting
  changeset tree --filter open-changesets --format json > tree.json
  
  # Show all changesets (no filter)
  changeset tree`,
		RunE: cmd.Run,
	}

	cobraCmd.Flags().String("filter", "", "Filter projects (same filters as 'each' command)")
	cobraCmd.Flags().String("format", "text", "Output format: text or json")

	return cobraCmd
}

// Run executes the tree command
func (c *TreeCommand) Run(cmd *cobra.Command, args []string) error {
	format, _ := cmd.Flags().GetString("format")
	filter, _ := cmd.Flags().GetString("filter")

	// Detect workspace
	ws := workspace.New(c.fs)
	if err := ws.Detect(); err != nil {
		return fmt.Errorf("failed to detect workspace: %w", err)
	}

	// Read all changesets
	csManager := changeset.NewManager(c.fs, ws.ChangesetDir())
	allChangesets, err := csManager.ReadAll()
	if err != nil {
		return fmt.Errorf("failed to read changesets: %w", err)
	}

	if len(allChangesets) == 0 {
		if format == "json" {
			fmt.Println(`{"groups":[]}`)
		}
		return nil
	}

	// Group changesets by commit SHA
	groups, err := c.groupByCommit(allChangesets)
	if err != nil {
		return fmt.Errorf("failed to group changesets: %w", err)
	}

	// Apply filter if specified
	if filter != "" {
		groups, err = c.applyFilter(groups, ws, csManager, filter)
		if err != nil {
			return fmt.Errorf("failed to apply filter: %w", err)
		}
	}

	// Output in requested format
	if format == "json" {
		return c.outputJSON(groups)
	}
	return c.outputText(groups)
}

// groupByCommit groups changesets by their creation commit
func (c *TreeCommand) groupByCommit(changesets []*models.Changeset) ([]*ChangesetGroup, error) {
	groupMap := make(map[string]*ChangesetGroup)

	for _, cs := range changesets {
		// Get commit that created this changeset
		commit, err := c.git.GetFileCreationCommit(cs.FilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to get commit for %s: %w", cs.FilePath, err)
		}

		if commit == "" {
			// File not in git history, use "unknown" group
			commit = "unknown"
		}

		// Create group if doesn't exist
		if _, exists := groupMap[commit]; !exists {
			commitMsg := ""
			commitShort := commit
			if commit != "unknown" && len(commit) >= 7 {
				commitShort = commit[:7]
				// Get commit message
				msg, err := c.git.GetCommitMessage(commit)
				if err == nil {
					commitMsg = msg
				}
			}

			groupMap[commit] = &ChangesetGroup{
				Commit:      commit,
				CommitShort: commitShort,
				Message:     commitMsg,
				projectsMap: make(map[string][]*models.Changeset),
			}
		}

		// Add changeset to all affected projects in this group
		for projectName := range cs.Projects {
			groupMap[commit].projectsMap[projectName] = append(
				groupMap[commit].projectsMap[projectName], cs)
		}
	}

	// Convert map to slice and sort by commit
	var groups []*ChangesetGroup
	for _, group := range groupMap {
		groups = append(groups, group)
	}

	// Sort groups by commit SHA for consistent output
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].Commit < groups[j].Commit
	})

	return groups, nil
}

// applyFilter filters groups to only include projects matching the filter
func (c *TreeCommand) applyFilter(groups []*ChangesetGroup, ws *workspace.Workspace,
	csManager *changeset.Manager, filter string) ([]*ChangesetGroup, error) {

	// Simple filter implementation
	// For tree command, we only care about "open-changesets" filter mainly
	// Other filters don't make much sense for viewing changeset relationships

	if filter == "open-changesets" || filter == "" {
		// No filtering needed - we already only have projects with changesets
		return groups, nil
	}

	// For other filters, filter to only projects matching
	filteredProjects := make(map[string]bool)

	switch filter {
	case "all":
		// Include all projects
		for _, project := range ws.Projects {
			filteredProjects[project.Name] = true
		}
	case "has-version", "no-version", "outdated-versions", "unchanged":
		builder := newProjectContextBuilder(c.fs, c.git)
		contexts, err := builder.Build(ws)
		if err != nil {
			return nil, err
		}

		filterTypes, err := parseFilters([]string{filter})
		if err != nil {
			return nil, err
		}

		filtered, err := filterContexts(contexts, filterTypes)
		if err != nil {
			return nil, err
		}

		for _, ctx := range filtered {
			filteredProjects[ctx.Project] = true
		}
	default:
		return nil, fmt.Errorf("unknown filter: %s", filter)
	}

	// Filter groups
	var result []*ChangesetGroup
	for _, group := range groups {
		newGroup := &ChangesetGroup{
			Commit:      group.Commit,
			CommitShort: group.CommitShort,
			Message:     group.Message,
			projectsMap: make(map[string][]*models.Changeset),
		}

		for projectName, changesets := range group.projectsMap {
			if filteredProjects[projectName] {
				newGroup.projectsMap[projectName] = changesets
			}
		}

		// Only include group if it has projects after filtering
		if len(newGroup.projectsMap) > 0 {
			result = append(result, newGroup)
		}
	}

	return result, nil
}

// outputText outputs the tree in human-readable format
func (c *TreeCommand) outputText(groups []*ChangesetGroup) error {
	if len(groups) == 0 {
		return nil
	}

	fmt.Println("📊 Changeset Relationship Tree")
	fmt.Println()

	totalChangesets := 0
	projectsAffected := make(map[string]bool)

	for _, group := range groups {
		// Build sorted project list
		var projectNames []string
		for projectName := range group.projectsMap {
			projectNames = append(projectNames, projectName)
		}
		sort.Strings(projectNames)

		// Output group header
		if group.Commit == "unknown" {
			fmt.Println("Ungrouped changesets (not in git history):")
		} else {
			commitInfo := group.CommitShort
			if group.Message != "" {
				commitInfo = fmt.Sprintf("%s (%s)", group.CommitShort, group.Message)
			}
			fmt.Printf("Commit: %s\n", commitInfo)
		}

		// Output each project's changesets
		for i, projectName := range projectNames {
			projectsAffected[projectName] = true
			changesets := group.projectsMap[projectName]
			totalChangesets += len(changesets)

			isLast := i == len(projectNames)-1
			prefix := "├─"
			if isLast {
				prefix = "└─"
			}

			fmt.Printf("%s %s (%d changeset(s))\n", prefix, projectName, len(changesets))

			// List changesets for this project
			for j, cs := range changesets {
				bump, _ := cs.GetBumpForProject(projectName)
				isLastCS := j == len(changesets)-1

				csPrefix := "│  ├─"
				if isLast {
					csPrefix = "   ├─"
				}
				if isLastCS {
					if isLast {
						csPrefix = "   └─"
					} else {
						csPrefix = "│  └─"
					}
				}

				// Truncate message for display
				msg := cs.Message
				if len(msg) > 60 {
					msg = msg[:57] + "..."
				}
				// Only first line
				if idx := strings.Index(msg, "\n"); idx > 0 {
					msg = msg[:idx]
				}

				fmt.Printf("%s %s.md (%s) - %s\n", csPrefix, cs.ID, bump, msg)
			}
		}
		fmt.Println()
	}

	// Summary
	fmt.Println("Summary:")
	fmt.Printf("- %d commit group(s)\n", len(groups))
	fmt.Printf("- %d project(s) affected\n", len(projectsAffected))
	fmt.Printf("- %d total changeset(s)\n", totalChangesets)

	return nil
}

// outputJSON outputs the tree in JSON format
func (c *TreeCommand) outputJSON(groups []*ChangesetGroup) error {
	// Convert internal structure to output structure
	output := TreeOutput{
		Groups: make([]ChangesetGroup, 0, len(groups)),
	}

	for _, group := range groups {
		outGroup := ChangesetGroup{
			Commit:      group.Commit,
			CommitShort: group.CommitShort,
			Message:     group.Message,
			Projects:    make([]ProjectChangesetsInfo, 0),
		}

		// Sort projects for consistent output
		var projectNames []string
		for projectName := range group.projectsMap {
			projectNames = append(projectNames, projectName)
		}
		sort.Strings(projectNames)

		// Build project info
		for _, projectName := range projectNames {
			changesets := group.projectsMap[projectName]

			projectInfo := ProjectChangesetsInfo{
				Name:       projectName,
				Changesets: make([]ChangesetInfo, 0, len(changesets)),
			}

			for _, cs := range changesets {
				bump, _ := cs.GetBumpForProject(projectName)
				projectInfo.Changesets = append(projectInfo.Changesets, ChangesetInfo{
					ID:      cs.ID,
					File:    cs.FilePath,
					Bump:    bump.String(),
					Message: cs.Message,
				})
			}

			outGroup.Projects = append(outGroup.Projects, projectInfo)
		}

		output.Groups = append(output.Groups, outGroup)
	}

	// Marshal and output
	jsonData, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	fmt.Println(string(jsonData))
	return nil
}
