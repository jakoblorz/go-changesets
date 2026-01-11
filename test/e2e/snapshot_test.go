package e2e_test

import (
	"strings"
	"testing"

	"github.com/jakoblorz/go-changesets/internal/changeset"
	"github.com/jakoblorz/go-changesets/internal/cli"
	"github.com/jakoblorz/go-changesets/internal/git"
	"github.com/jakoblorz/go-changesets/internal/github"
	"github.com/jakoblorz/go-changesets/internal/versioning"
	"github.com/jakoblorz/go-changesets/internal/workspace"
	"github.com/jakoblorz/go-changesets/test/testutil"
	"github.com/spf13/cobra"
)

func TestSnapshotWorkflow(t *testing.T) {
	// Setup mock workspace with changesets
	wb := testutil.NewWorkspaceBuilder("/test-workspace")
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
		if err != nil {
			t.Fatalf("snapshot command failed: %v", err)
		}

		// Verify tag was created
		tags := gitMock.GetAllTags()
		if len(tags) != 1 {
			t.Errorf("expected 1 tag, got %d", len(tags))
		}

		expectedTag := "backend@v0.1.0-rc0"
		if _, exists := tags[expectedTag]; !exists {
			t.Errorf("expected tag %s not found, got tags: %v", expectedTag, tagNames(tags))
		}

		// Verify GitHub release was created as pre-release
		releases := ghMock.GetAllReleases("testorg", "testrepo")
		if len(releases) != 1 {
			t.Errorf("expected 1 GitHub release, got %d", len(releases))
		}

		release := releases[0]
		if release.TagName != expectedTag {
			t.Errorf("expected release tag %s, got %s", expectedTag, release.TagName)
		}
		if !release.Prerelease {
			t.Errorf("expected release to be marked as prerelease")
		}

		// Verify changesets were NOT deleted
		ws := workspace.New(fs)
		ws.Detect()
		csManager := changeset.NewManager(fs, ws.ChangesetDir())
		remainingChangesets, _ := csManager.ReadAll()
		if len(remainingChangesets) != 2 {
			t.Errorf("expected changesets to remain, got %d changesets", len(remainingChangesets))
		}
	})

	// Test 2: Second snapshot (should create rc1)
	t.Run("second snapshot creates rc1", func(t *testing.T) {
		cmd := cli.NewSnapshotCommand(fs, gitMock, ghMock)
		cmd.SetArgs([]string{"--project", "backend", "--owner", "testorg", "--repo", "testrepo"})

		err := cmd.Execute()
		if err != nil {
			t.Fatalf("second snapshot command failed: %v", err)
		}

		// Verify rc1 tag was created
		tags := gitMock.GetAllTags()
		if len(tags) != 2 {
			t.Errorf("expected 2 tags, got %d", len(tags))
		}

		expectedTag := "backend@v0.1.0-rc1"
		if _, exists := tags[expectedTag]; !exists {
			t.Errorf("expected tag %s not found, got tags: %v", expectedTag, tagNames(tags))
		}
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

		if version.String() != "0.1.0" {
			t.Errorf("expected version 0.1.0, got %s", version.String())
		}

		// Verify changesets were deleted
		csManager := changeset.NewManager(fs, ws.ChangesetDir())
		remainingChangesets, _ := csManager.ReadAll()
		if len(remainingChangesets) != 0 {
			t.Errorf("expected changesets to be deleted, got %d changesets", len(remainingChangesets))
		}
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
		if _, exists := tags[expectedTag]; !exists {
			t.Errorf("expected final tag %s not found, got tags: %v", expectedTag, tagNames(tags))
		}

		// Verify GitHub release was created as normal release (not prerelease)
		releases := ghMock.GetAllReleases("testorg", "testrepo")
		var finalRelease *github.Release
		for _, r := range releases {
			if r.TagName == expectedTag {
				finalRelease = r
				break
			}
		}

		if finalRelease == nil {
			t.Fatalf("final release %s not found", expectedTag)
		}

		if finalRelease.Prerelease {
			t.Errorf("expected final release to NOT be marked as prerelease")
		}
	})
}

func TestSnapshotWithNoChangesets(t *testing.T) {
	// Setup mock workspace with NO changesets
	wb := testutil.NewWorkspaceBuilder("/test-workspace")
	wb.AddProject("backend", "apps/backend", "github.com/test/backend")

	fs := wb.Build()
	gitMock := git.NewMockGitClient()
	ghMock := github.NewMockClient()
	ghMock.SetupRepository("testorg", "testrepo")

	cmd := cli.NewSnapshotCommand(fs, gitMock, ghMock)
	cmd.SetArgs([]string{"--project", "backend", "--owner", "testorg", "--repo", "testrepo"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when no changesets found, got nil")
	}

	if !strings.Contains(err.Error(), "no changesets found") {
		t.Errorf("expected 'no changesets found' error, got: %v", err)
	}
}

func TestSnapshotViaEach(t *testing.T) {
	// Setup workspace with multiple projects
	wb := testutil.NewWorkspaceBuilder("/test-workspace")
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
			if err != nil {
				t.Fatalf("snapshot failed for %s: %v", projectName, err)
			}
		}

		// Verify both projects got RC tags
		tags := gitMock.GetAllTags()
		if len(tags) != 2 {
			t.Errorf("expected 2 RC tags, got %d: %v", len(tags), tagNames(tags))
		}

		expectedTags := []string{"backend@v0.1.0-rc0", "www@v0.0.1-rc0"}
		for _, expectedTag := range expectedTags {
			if _, exists := tags[expectedTag]; !exists {
				t.Errorf("expected tag %s not found", expectedTag)
			}
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
