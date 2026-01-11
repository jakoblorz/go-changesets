package cli

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jakoblorz/go-changesets/internal/changeset"
	"github.com/jakoblorz/go-changesets/internal/filesystem"
	"github.com/jakoblorz/go-changesets/internal/git"
	"github.com/jakoblorz/go-changesets/internal/github"
	"github.com/jakoblorz/go-changesets/internal/models"
	"github.com/jakoblorz/go-changesets/internal/versioning"
	"github.com/spf13/cobra"
)

// VersionCommand handles the version command
type VersionCommand struct {
	fs       filesystem.FileSystem
	git      git.GitClient
	ghClient github.GitHubClient
}

// NewVersionCommand creates a new version command
func NewVersionCommand(fs filesystem.FileSystem, gitClient git.GitClient, ghClient github.GitHubClient) *cobra.Command {
	cmd := &VersionCommand{
		fs:       fs,
		git:      gitClient,
		ghClient: ghClient,
	}

	cobraCmd := &cobra.Command{
		Use:   "version",
		Short: "Version a project based on changesets",
		Long:  `Applies all changesets for a project, updates version.txt and CHANGELOG.md.`,
		RunE:  cmd.Run,
	}

	cobraCmd.Flags().StringP("project", "p", "", "Project name to version (required unless run via 'changeset each')")
	cobraCmd.Flags().StringP("owner", "o", "", "GitHub repository owner (optional, enables PR links in changelog)")
	cobraCmd.Flags().StringP("repo", "r", "", "GitHub repository name (optional, enables PR links in changelog)")

	return cobraCmd
}

// Run executes the version command
func (c *VersionCommand) Run(cmd *cobra.Command, args []string) error {
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
		fmt.Printf("📦 Versioning %s (via changeset each)\n\n", resolved.Name)
	} else {
		fmt.Printf("📦 Versioning project: %s\n\n", resolved.Name)
	}

	csManager := changeset.NewManager(c.fs, resolved.Workspace.ChangesetDir())
	allChangesets, err := csManager.ReadAll()
	if err != nil {
		return fmt.Errorf("failed to read changesets: %w", err)
	}

	projectChangesets := csManager.FilterByProject(allChangesets, resolved.Name)
	if len(projectChangesets) == 0 {
		fmt.Println("⚠️  No changesets found for this project")
		return nil
	}

	fmt.Printf("Found %d changeset(s) for %s:\n", len(projectChangesets), resolved.Name)
	for _, cs := range projectChangesets {
		bump, _ := cs.GetBumpForProject(resolved.Name)
		fmt.Printf("  - %s (%s)\n", cs.ID, bump)
	}
	fmt.Println()

	if owner != "" && repo != "" {
		if err := c.enrichChangesetsWithPRInfo(projectChangesets, owner, repo); err != nil {
			return err
		}
	}

	highestBump := csManager.GetHighestBump(projectChangesets, resolved.Name)
	fmt.Printf("Highest bump type: %s\n\n", highestBump)

	versionStore := versioning.NewVersionStore(c.fs, resolved.Project.Type)
	currentVersion, err := versionStore.Read(resolved.Project.RootPath)
	if err != nil {
		return fmt.Errorf("failed to read current version: %w", err)
	}

	fmt.Printf("Current version: %s\n", currentVersion.String())

	newVersion := currentVersion.Bump(highestBump)
	fmt.Printf("New version: %s\n\n", newVersion.String())

	if err := versionStore.Write(resolved.Project.RootPath, newVersion); err != nil {
		return fmt.Errorf("failed to write version: %w", err)
	}

	fmt.Printf("✓ Updated %s/version.txt\n", resolved.Name)

	changelog := versioning.NewChangelog(c.fs)
	entry := &versioning.ChangelogEntry{
		Version:    newVersion,
		Date:       time.Now(),
		Changesets: projectChangesets,
	}

	if err := changelog.Append(resolved.Project.RootPath, entry); err != nil {
		return fmt.Errorf("failed to update changelog: %w", err)
	}

	fmt.Printf("✓ Updated %s/CHANGELOG.md\n\n", resolved.Name)

	fmt.Println("Removing consumed changesets...")
	for _, cs := range projectChangesets {
		if err := csManager.Delete(cs); err != nil {
			fmt.Printf("⚠️  Warning: failed to delete %s: %v\n", cs.ID, err)
			continue
		}
		fmt.Printf("  ✓ Removed %s.md\n", cs.ID)
	}

	fmt.Printf("\n🎉 Successfully versioned %s to %s\n", resolved.Name, newVersion.String())
	return nil
}

func (c *VersionCommand) enrichChangesetsWithPRInfo(changesets []*models.Changeset, owner, repo string) error {
	if c.git == nil {
		fmt.Println("⚠️  Git client not available, skipping PR enrichment")
		return nil
	}

	if c.ghClient == nil {
		token := os.Getenv("GITHUB_TOKEN")
		if token != "" {
			c.ghClient = github.NewClient(token)
		} else {
			fmt.Println("⚠️  GITHUB_TOKEN not set; PR enrichment may fail for private/internal repos")
			c.ghClient = github.NewClientWithoutAuth()
		}
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
