package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/jakoblorz/go-changesets/internal/changelog"
	"github.com/jakoblorz/go-changesets/internal/filesystem"
	"github.com/jakoblorz/go-changesets/internal/git"
	"github.com/jakoblorz/go-changesets/internal/github"
	"github.com/jakoblorz/go-changesets/internal/models"
	"github.com/jakoblorz/go-changesets/internal/versioning"
	"github.com/spf13/cobra"
)

// PublishCommand handles the publish command
type PublishCommand struct {
	fs       filesystem.FileSystem
	git      git.GitClient
	ghClient github.GitHubClient
}

// NewPublishCommand creates a new publish command
func NewPublishCommand(fs filesystem.FileSystem, gitClient git.GitClient, ghClient github.GitHubClient) *cobra.Command {
	cmd := &PublishCommand{
		fs:       fs,
		git:      gitClient,
		ghClient: ghClient,
	}

	cobraCmd := &cobra.Command{
		Use:   "publish",
		Short: "Creates & pushes the most recent git tag; Optionally creates a release on GitHub",
		Long:  `Creates & pushes the most recent git tag; Optionally creates a release on GitHub.`,
		RunE:  cmd.Run,
	}

	cobraCmd.Flags().StringP("project", "p", "", "Project name to publish (required unless run via 'changeset each')")
	cobraCmd.Flags().StringP("owner", "o", "", "GitHub repository owner (optional, enables creating a release)")
	cobraCmd.Flags().StringP("repo", "r", "", "GitHub repository name (optional, enables creating a release)")

	return cobraCmd
}

// Run executes the publish command
func (c *PublishCommand) Run(cmd *cobra.Command, args []string) error {
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
		fmt.Printf("📦 Publishing %s (via changeset each)\n\n", resolved.Name)
	} else {
		fmt.Printf("📦 Publishing project: %s\n\n", resolved.Name)
	}

	versionStore := versioning.NewVersionStore(c.fs, resolved.Project.Type)
	fileVersion, err := versionStore.Read(resolved.Project.RootPath)
	if err != nil {
		return fmt.Errorf("failed to read version: %w", err)
	}

	fmt.Printf("Version from version.txt: %s\n", fileVersion.String())

	tagVersion, err := getLatestNonRCVersion(c.git, resolved.Name)
	if err != nil {
		tagVersion = &models.Version{Major: 0, Minor: 0, Patch: 0}
		fmt.Printf("No existing git tag found (first release)\n")
	} else {
		fmt.Printf("Latest git tag: %s\n", tagVersion.String())
	}

	if fileVersion.Compare(tagVersion) <= 0 {
		fmt.Printf("\n⚠️  Version %s already published (skipping)\n", fileVersion.String())
		return nil
	}

	fmt.Printf("\n🚀 Publishing new version: %s -> %s\n\n", tagVersion.String(), fileVersion.String())

	tag := fmt.Sprintf("%s@%s", resolved.Name, fileVersion.Tag())
	fmt.Printf("Creating git tag: %s\n", tag)

	changelogMsg, _ := c.getChangelogForVersion(resolved.Project.RootPath, fileVersion)
	if err := c.git.CreateTag(tag, changelogMsg); err != nil {
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

	if c.ghClient == nil {
		if owner != "" || repo != "" {
			return fmt.Errorf("--owner and --repo flags require a GitHub client: authenticated GitHub client required to create a release: %w", github.ErrGitHubTokenNotFound)
		}
	}
	if c.ghClient != nil {
		if owner == "" {
			return fmt.Errorf("--owner flag required")
		}
		if repo == "" {
			return fmt.Errorf("--repo flag required")
		}

		ctx := context.Background()
		existingRelease, err := c.ghClient.GetReleaseByTag(ctx, owner, repo, tag)
		if err == nil && existingRelease != nil {
			fmt.Printf("⚠️  Release %s already exists\n", tag)
			fmt.Printf("Release URL: https://github.com/%s/%s/releases/tag/%s\n", owner, repo, tag)
			return nil
		}

		changelog := changelog.NewChangelog(c.fs)
		changelogEntry, err := changelog.GetEntryForVersion(resolved.Project.RootPath, fileVersion)
		if err != nil {
			fmt.Printf("⚠️  Warning: could not read changelog entry: %v\n", err)
			changelogEntry = fmt.Sprintf("Release %s", fileVersion.String())
		}

		releaseNotes := extractReleaseNotes(changelogEntry)

		fmt.Println("Creating GitHub release...")
		_, err = c.ghClient.CreateRelease(ctx, owner, repo, &github.CreateReleaseRequest{
			TagName: tag,
			Name:    tag,
			Body:    releaseNotes,
		})
		if err != nil {
			return fmt.Errorf("failed to create release: %w", err)
		}

		fmt.Printf("Release URL: https://github.com/%s/%s/releases/tag/%s\n", owner, repo, tag)
	}

	fmt.Printf("\n🎉 Successfully published %s@%s\n", resolved.Name, fileVersion.String())

	return nil
}

func (c *PublishCommand) getChangelogForVersion(projectRoot string, version *models.Version) (string, error) {
	changelog := changelog.NewChangelog(c.fs)
	return changelog.GetEntryForVersion(projectRoot, version)
}

func extractReleaseNotes(entry string) string {
	lines := strings.Split(entry, "\n")
	if len(lines) == 0 {
		return entry
	}

	var noteLines []string
	skipFirst := false
	for _, line := range lines {
		if strings.HasPrefix(line, "## ") && !skipFirst {
			skipFirst = true
			continue
		}
		noteLines = append(noteLines, line)
	}

	return strings.TrimSpace(strings.Join(noteLines, "\n"))
}
