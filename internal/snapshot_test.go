package internal_test

import (
	"testing"

	"github.com/jakoblorz/go-changesets/internal/changeset"
	"github.com/jakoblorz/go-changesets/internal/cli"
	"github.com/jakoblorz/go-changesets/internal/git"
	"github.com/jakoblorz/go-changesets/internal/github"
	"github.com/jakoblorz/go-changesets/internal/versioning"
	"github.com/jakoblorz/go-changesets/internal/workspace"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestSnapshotWorkflow(t *testing.T) {
	// Setup mock workspace with changesets
	wb := workspace.NewWorkspaceBuilder("/test-workspace")
	wb.AddProject("backend", "apps/backend", "github.com/test/backend")
	wb.AddChangeset("abc123", "backend", "minor", "Add new API endpoints")
	wb.AddChangeset("def456", "backend", "patch", "Fix memory leak")

	fs := wb.Build()

	// Setup mock git client
	gitMock := git.NewMockGitClient()

	// Setup mock GitHub client
	ghMock := github.NewMockClient()
	ghMock.SetupRepository("testorg", "testrepo")

	// Test 1: First snapshot (should create rc0)
	t.Run("first snapshot creates rc0", func(t *testing.T) {
		cmd := cli.NewSnapshotCommand(fs, gitMock, ghMock)
		cmd.SetArgs([]string{"--project", "backend", "--owner", "testorg", "--repo", "testrepo"})

		// Mock expects no existing tags
		// Snapshot should calculate version 0.1.0 (minor bump from 0.0.0)
		// And create backend@v0.1.0-rc0

		err := cmd.Execute()
		require.NoError(t, err)

		// Verify tag was created
		tags := gitMock.GetAllTags()
		require.Len(t, tags, 1)

		expectedTag := "backend@v0.1.0-rc0"
		_, exists := tags[expectedTag]
		require.True(t, exists, "expected tag %s not found, got tags: %v", expectedTag, tagNames(tags))

		// Verify GitHub release was created as pre-release
		releases := ghMock.GetAllReleases("testorg", "testrepo")
		require.Len(t, releases, 1)

		release := releases[0]
		require.Equal(t, expectedTag, release.TagName)
		require.True(t, release.Prerelease, "expected release to be marked as prerelease")

		// Verify changesets were NOT deleted
		ws := workspace.New(fs)
		ws.Detect()
		csManager := changeset.NewManager(fs, ws.ChangesetDir())
		remainingChangesets, _ := csManager.ReadAll()
		require.Len(t, remainingChangesets, 2, "expected changesets to remain, got %d changesets", len(remainingChangesets))
	})

	// Test 2: Second snapshot (should create rc1)
	t.Run("second snapshot creates rc1", func(t *testing.T) {
		cmd := cli.NewSnapshotCommand(fs, gitMock, ghMock)
		cmd.SetArgs([]string{"--project", "backend", "--owner", "testorg", "--repo", "testrepo"})

		err := cmd.Execute()
		require.NoError(t, err)

		// Verify rc1 tag was created
		tags := gitMock.GetAllTags()
		require.Len(t, tags, 2)

		expectedTag := "backend@v0.1.0-rc1"
		_, exists := tags[expectedTag]
		require.True(t, exists, "expected tag %s not found, got tags: %v", expectedTag, tagNames(tags))
	})

	// Test 3: Version command (should delete changesets)
	t.Run("version command updates version.txt and deletes changesets", func(t *testing.T) {
		cmd := cli.NewVersionCommand(fs, gitMock, ghMock)
		cmd.SetArgs([]string{"--project", "backend"})

		err := cmd.Execute()
		if err != nil {
			t.Fatalf("version command failed: %v", err)
		}

		// Verify version.txt was updated
		ws := workspace.New(fs)
		ws.Detect()
		project, _ := ws.GetProject("backend")
		versionFile := versioning.NewVersionFile(fs)
		version, _ := versionFile.Read(project.RootPath)

		require.Equal(t, "0.1.0", version.String())

		// Verify changesets were deleted
		csManager := changeset.NewManager(fs, ws.ChangesetDir())
		remainingChangesets, _ := csManager.ReadAll()
		require.Len(t, remainingChangesets, 0, "expected changesets to be deleted, got %d changesets", len(remainingChangesets))
	})

	// Test 4: Publish command (should ignore RC tags)
	t.Run("publish command creates final release and ignores RC tags", func(t *testing.T) {
		// Simulate that version.txt has been updated to 0.1.0
		// Git tags: backend@v0.1.0-rc0, backend@v0.1.0-rc1
		// Publish should create backend@v0.1.0 (final release)

		cmd := cli.NewPublishCommand(fs, gitMock, ghMock)
		cmd.SetArgs([]string{"--project", "backend", "--owner", "testorg", "--repo", "testrepo"})

		err := cmd.Execute()
		if err != nil {
			t.Fatalf("publish command failed: %v", err)
		}

		// Verify final release tag was created (without -rc suffix)
		tags := gitMock.GetAllTags()
		expectedTag := "backend@v0.1.0"
		_, exists := tags[expectedTag]
		require.True(t, exists, "expected final tag %s not found, got tags: %v", expectedTag, tagNames(tags))

		// Verify GitHub release was created as normal release (not prerelease)
		releases := ghMock.GetAllReleases("testorg", "testrepo")
		var finalRelease *github.Release
		for _, r := range releases {
			if r.TagName == expectedTag {
				finalRelease = r
				break
			}
		}

		require.NotNil(t, finalRelease, "final release %s not found", expectedTag)

		require.False(t, finalRelease.Prerelease, "expected final release to NOT be marked as prerelease")
	})
}

func TestSnapshotWithNoChangesets(t *testing.T) {
	// Setup mock workspace with NO changesets
	wb := workspace.NewWorkspaceBuilder("/test-workspace")
	wb.AddProject("backend", "apps/backend", "github.com/test/backend")

	fs := wb.Build()
	gitMock := git.NewMockGitClient()
	ghMock := github.NewMockClient()
	ghMock.SetupRepository("testorg", "testrepo")

	cmd := cli.NewSnapshotCommand(fs, gitMock, ghMock)
	cmd.SetArgs([]string{"--project", "backend", "--owner", "testorg", "--repo", "testrepo"})

	err := cmd.Execute()
	require.Error(t, err)
	require.Contains(t, err.Error(), "no changesets found")
}

func TestSnapshotViaEach(t *testing.T) {
	// Setup workspace with multiple projects
	wb := workspace.NewWorkspaceBuilder("/test-workspace")
	wb.AddProject("backend", "apps/backend", "github.com/test/backend")
	wb.AddProject("www", "apps/www", "github.com/test/www")
	wb.AddChangeset("abc123", "backend", "minor", "Add API")
	wb.AddChangeset("def456", "www", "patch", "Fix UI bug")

	fs := wb.Build()
	gitMock := git.NewMockGitClient()
	ghMock := github.NewMockClient()
	ghMock.SetupRepository("testorg", "testrepo")

	// Test: Use 'each' command with snapshot
	// This simulates: changeset each --filter open-changesets | changeset snapshot
	t.Run("snapshot via each command", func(t *testing.T) {
		ws := workspace.New(fs)
		ws.Detect()

		// Filter projects with open changesets
		csManager := changeset.NewManager(fs, ws.ChangesetDir())
		allChangesets, _ := csManager.ReadAll()

		projectsWithChangesets := make(map[string]bool)
		for _, cs := range allChangesets {
			for project := range cs.Projects {
				projectsWithChangesets[project] = true
			}
		}

		// Snapshot each project
		for projectName := range projectsWithChangesets {
			cmd := cli.NewSnapshotCommand(fs, gitMock, ghMock)
			cmd.SetArgs([]string{"--project", projectName, "--owner", "testorg", "--repo", "testrepo"})

			err := cmd.Execute()
			require.NoError(t, err, "snapshot failed for %s: %v", projectName, err)
		}

		// Verify both projects got RC tags
		tags := gitMock.GetAllTags()
		require.Len(t, tags, 2, "expected 2 RC tags, got %d: %v", len(tags), tagNames(tags))

		expectedTags := []string{"backend@v0.1.0-rc0", "www@v0.0.1-rc0"}
		for _, expectedTag := range expectedTags {
			_, exists := tags[expectedTag]
			require.True(t, exists, "expected tag %s not found", expectedTag)
		}
	})
}

// Helper function to get tag names from mock
func tagNames(tags map[string]*git.MockTag) []string {
	var names []string
	for name := range tags {
		names = append(names, name)
	}
	return names
}

// Helper to suppress cobra output for cleaner tests
func init() {
	cobra.EnableCommandSorting = false
}
