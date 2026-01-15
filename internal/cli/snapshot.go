package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/jakoblorz/go-changesets/internal/changeset"
	"github.com/jakoblorz/go-changesets/internal/filesystem"
	"github.com/jakoblorz/go-changesets/internal/git"
	"github.com/jakoblorz/go-changesets/internal/github"
	"github.com/jakoblorz/go-changesets/internal/models"
	"github.com/jakoblorz/go-changesets/internal/versioning"
	"github.com/spf13/cobra"
)

// SnapshotCommand handles the snapshot command
type SnapshotCommand struct {
	fs       filesystem.FileSystem
	git      git.GitClient
	ghClient github.GitHubClient
}

// NewSnapshotCommand creates a new snapshot command
func NewSnapshotCommand(fs filesystem.FileSystem, gitClient git.GitClient, ghClient github.GitHubClient) *cobra.Command {
	cmd := &SnapshotCommand{
		fs:       fs,
		git:      gitClient,
		ghClient: ghClient,
	}

	cobraCmd := &cobra.Command{
		Use:   "snapshot",
		Short: "Create a release candidate (RC) snapshot",
		Long:  `Creates a pre-release snapshot with an -rc{N} suffix. Does not modify changesets or version files.`,
		RunE:  cmd.Run,
	}

	cobraCmd.Flags().StringP("project", "p", "", "Project name to snapshot (required unless run via 'changeset each')")
	cobraCmd.Flags().StringP("owner", "o", "", "GitHub repository owner (required)")
	cobraCmd.Flags().StringP("repo", "r", "", "GitHub repository name (required)")
	cobraCmd.MarkFlagRequired("owner")
	cobraCmd.MarkFlagRequired("repo")

	return cobraCmd
}

// Run executes the snapshot command
func (c *SnapshotCommand) Run(cmd *cobra.Command, args []string) error {
	projectFlag, _ := cmd.Flags().GetString("project")
	owner, _ := cmd.Flags().GetString("owner")
	repo, _ := cmd.Flags().GetString("repo")

	resolved, err := resolveProject(c.fs, projectFlag)
	if err != nil {
		if projectFlag == "" {
			return fmt.Errorf("--project flag required (or run via 'changeset each'): %w", err)
		}
		return err
	}

	if resolved.ViaEach {
		fmt.Printf("📸 Creating snapshot for %s (via changeset each)\n\n", resolved.Name)
	} else {
		fmt.Printf("📸 Creating snapshot for project: %s\n\n", resolved.Name)
	}

	if owner == "" {
		return fmt.Errorf("--owner flag required")
	}
	if repo == "" {
		return fmt.Errorf("--repo flag required")
	}

	if c.ghClient == nil {
		token := os.Getenv("GH_TOKEN")
		if token == "" {
			token = os.Getenv("GITHUB_TOKEN")
		}
		if token == "" {
			return fmt.Errorf("GITHUB_TOKEN or GH_TOKEN environment variable required for snapshot")
		}
		c.ghClient = github.NewClient(token)
	}

	csManager := changeset.NewManager(c.fs, resolved.Workspace.ChangesetDir())
	allChangesets, err := csManager.ReadAll()
	if err != nil {
		return fmt.Errorf("failed to read changesets: %w", err)
	}

	projectChangesets := csManager.FilterByProject(allChangesets, resolved.Name)
	if len(projectChangesets) == 0 {
		return fmt.Errorf("no changesets found for project %s", resolved.Name)
	}

	fmt.Printf("Found %d changeset(s) for %s:\n", len(projectChangesets), resolved.Name)
	for _, cs := range projectChangesets {
		bump, _ := cs.GetBumpForProject(resolved.Name)
		fmt.Printf("  - %s (%s)\n", cs.ID, bump)
	}
	fmt.Println()

	if err := c.enrichChangesetsWithPRInfo(projectChangesets, owner, repo); err != nil {
		return err
	}

	highestBump := csManager.GetHighestBump(projectChangesets, resolved.Name)
	fmt.Printf("Highest bump type: %s\n\n", highestBump)

	nextVersion, err := c.calculateNextVersion(resolved.Name, highestBump)
	if err != nil {
		return fmt.Errorf("failed to calculate next version: %w", err)
	}

	fmt.Printf("Next version: %s\n", nextVersion.String())

	rcNumber, err := c.findNextRCNumber(resolved.Name, nextVersion)
	if err != nil {
		return fmt.Errorf("failed to find next RC number: %w", err)
	}

	fmt.Printf("Next RC number: rc%d\n\n", rcNumber)

	rcVersion := nextVersion.WithPrerelease(fmt.Sprintf("rc%d", rcNumber))
	tag := fmt.Sprintf("%s@%s", resolved.Name, rcVersion.Tag())

	fmt.Printf("Creating snapshot tag: %s\n", tag)

	changelog := versioning.NewChangelog(c.fs)
	summary, err := changelog.FormatEntry(projectChangesets, resolved.Project.Name, resolved.Project.RootPath)
	if err != nil {
		return fmt.Errorf("failed to format changelog entry: %w", err)
	}

	if c.git != nil {

		if err := c.git.CreateTag(tag, summary); err != nil {
			exists, _ := c.git.TagExists(tag)
			if !exists {
				return fmt.Errorf("failed to create tag: %w", err)
			}
			fmt.Printf("Tag already exists locally\n")
		}

		fmt.Printf("Pushing tag to remote...\n")
		if err := c.git.PushTag(tag); err != nil {
			fmt.Printf("⚠️  Warning: failed to push tag: %v\n", err)
		}
	} else {
		fmt.Printf("⚠️  Skipping git tag creation (no git client)\n")
	}

	ctx := context.Background()
	existingRelease, err := c.ghClient.GetReleaseByTag(ctx, owner, repo, tag)
	if err == nil && existingRelease != nil {
		fmt.Printf("⚠️  Release %s already exists\n", tag)
		fmt.Printf("Release URL: https://github.com/%s/%s/releases/tag/%s\n", owner, repo, tag)
		return nil
	}

	fmt.Println("Creating GitHub pre-release...")
	release, err := c.ghClient.CreateRelease(ctx, owner, repo, &github.CreateReleaseRequest{
		TagName:    tag,
		Name:       tag,
		Body:       summary,
		Prerelease: true,
	})
	if err != nil {
		return fmt.Errorf("failed to create release: %w", err)
	}

	fmt.Printf("\n🎉 Successfully created snapshot %s@%s\n", resolved.Name, rcVersion.String())
	fmt.Printf("Release URL: https://github.com/%s/%s/releases/tag/%s\n", owner, repo, tag)

	_ = release
	return nil
}

func (c *SnapshotCommand) calculateNextVersion(projectName string, bump models.BumpType) (*models.Version, error) {
	if c.git == nil {
		return nil, fmt.Errorf("git client not available")
	}

	latestVersion, err := c.getLatestNonRCVersion(projectName)
	if err != nil {
		latestVersion = &models.Version{Major: 0, Minor: 0, Patch: 0}
		fmt.Printf("No existing tags found (first release)\n")
	} else {
		fmt.Printf("Latest published version: %s\n", latestVersion.String())
	}

	nextVersion := latestVersion.Bump(bump)
	return nextVersion, nil
}

func (c *SnapshotCommand) getLatestNonRCVersion(projectName string) (*models.Version, error) {
	prefix := fmt.Sprintf("%s@v*", projectName)
	tags, err := c.git.GetTagsWithPrefix(prefix)
	if err != nil {
		return nil, err
	}

	for _, tag := range tags {
		rcNum, _ := c.git.ExtractRCNumber(tag)
		if rcNum >= 0 {
			continue
		}

		parts := strings.Split(tag, "@")
		if len(parts) != 2 {
			continue
		}

		version, err := models.ParseVersion(parts[1])
		if err != nil {
			continue
		}

		return version, nil
	}

	return nil, fmt.Errorf("no non-RC tags found")
}

func (c *SnapshotCommand) findNextRCNumber(projectName string, version *models.Version) (int, error) {
	if c.git == nil {
		return 0, nil
	}

	prefix := fmt.Sprintf("%s@v*", projectName)
	allTags, err := c.git.GetTagsWithPrefix(prefix)
	if err != nil {
		return 0, err
	}

	versionPrefix := fmt.Sprintf("%s@%s-rc", projectName, version.Tag())
	var rcTags []string
	for _, tag := range allTags {
		if strings.HasPrefix(tag, versionPrefix) {
			rcTags = append(rcTags, tag)
		}
	}

	if len(rcTags) == 0 {
		return 0, nil
	}

	highestRC := -1
	for _, tag := range rcTags {
		rcNum, err := c.git.ExtractRCNumber(tag)
		if err != nil {
			continue
		}
		if rcNum > highestRC {
			highestRC = rcNum
		}
	}

	return highestRC + 1, nil
}

func (c *SnapshotCommand) enrichChangesetsWithPRInfo(changesets []*models.Changeset, owner, repo string) error {
	if c.git == nil {
		fmt.Println("⚠️  Git client not available, skipping PR enrichment")
		return nil
	}

	enricher := changeset.NewPREnricher(c.git, c.ghClient)
	res, err := enricher.Enrich(context.Background(), changesets, owner, repo)
	if err != nil {
		return fmt.Errorf("failed to enrich changesets with PR info: %w", err)
	}

	for _, warn := range res.Warnings {
		fmt.Printf("⚠️  Warning: %v\n", warn)
	}

	if res.Enriched > 0 {
		fmt.Printf("✓ Enriched %d changeset(s) with PR information\n\n", res.Enriched)
	}

	return nil
}
