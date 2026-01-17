package e2e_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jakoblorz/go-changesets/internal/changelog"
	"github.com/jakoblorz/go-changesets/internal/changeset"
	"github.com/jakoblorz/go-changesets/internal/git"
	"github.com/jakoblorz/go-changesets/internal/github"
	"github.com/jakoblorz/go-changesets/internal/models"
	"github.com/jakoblorz/go-changesets/internal/versioning"
	"github.com/jakoblorz/go-changesets/internal/workspace"
	"github.com/stretchr/testify/require"
)

func TestFullWorkflow(t *testing.T) {
	// Setup mock workspace
	wb := workspace.NewWorkspaceBuilder("/test-workspace")
	wb.AddProject("auth", "packages/auth", "github.com/test/auth")
	wb.AddProject("api", "packages/api", "github.com/test/api")
	wb.AddChangeset("abc123", "auth", "minor", "Add new login feature")
	wb.AddChangeset("def456", "auth", "patch", "Fix session bug")

	fs := wb.Build()

	// Setup mock GitHub
	ghMock := github.NewMockClient()
	ghMock.SetupRepository("test", "monorepo")

	// Test: Workspace detection
	ws := workspace.New(fs)
	require.NoError(t, ws.Detect())

	require.Len(t, ws.Projects, 2)

	// Test: Read changesets
	csManager := changeset.NewManager(fs, ws.ChangesetDir())
	changesets, err := csManager.ReadAll()
	require.NoError(t, err)

	require.Len(t, changesets, 2)

	// Test: Filter changesets by project
	authChangesets := changeset.FilterByProject(changesets, "auth")
	require.Len(t, authChangesets, 2)

	// Test: Get highest bump
	highestBump := csManager.GetHighestBump(authChangesets, "auth")
	require.Equal(t, models.BumpMinor, highestBump)

	// Test: Version management
	authProject, _ := ws.GetProject("auth")
	versionFile := versioning.NewVersionFile(fs)

	// Read initial version (should default to 0.0.0)
	currentVersion, err := versionFile.Read(authProject.RootPath)
	require.NoError(t, err)

	require.Equal(t, "0.0.0", currentVersion.String())

	// Bump version
	newVersion := currentVersion.Bump(highestBump)
	require.Equal(t, "0.1.0", newVersion.String())

	// Write new version
	require.NoError(t, versionFile.Write(authProject.RootPath, newVersion))

	// Verify version was written
	readVersion, err := versionFile.Read(authProject.RootPath)
	require.NoError(t, err)

	require.Equal(t, "0.1.0", readVersion.String())

	// Test: Changelog generation
	cl := changelog.NewChangelog(fs)
	entry := &changelog.Entry{
		Version:    newVersion,
		Date:       time.Now(),
		Changesets: authChangesets,
	}

	require.NoError(t, cl.Append(authProject.RootPath, "auth", entry))

	// Verify changelog was created
	changelogPath := authProject.RootPath + "/CHANGELOG.md"
	changelogData, err := fs.ReadFile(changelogPath)
	require.NoError(t, err)

	changelogContent := string(changelogData)
	require.Contains(t, changelogContent, "0.1.0")
	require.Contains(t, changelogContent, "Add new login feature")

	// Test: Delete changesets
	for _, cs := range authChangesets {
		require.NoError(t, csManager.Delete(cs))
	}

	// Verify changesets were deleted
	remainingChangesets, err := csManager.ReadAll()
	require.NoError(t, err)

	require.Len(t, remainingChangesets, 0)

	// Test: GitHub release creation
	ctx := context.Background()
	tag := "auth@v0.1.0"

	release, err := ghMock.CreateRelease(ctx, "test", "monorepo", &github.CreateReleaseRequest{
		TagName: tag,
		Name:    tag,
		Body:    "Release notes",
	})
	require.NoError(t, err)

	require.Equal(t, tag, release.TagName)

	// Test: Check if release exists
	existingRelease, err := ghMock.GetReleaseByTag(ctx, "test", "monorepo", tag)
	require.NoError(t, err)

	require.Equal(t, tag, existingRelease.TagName)

	// Test: Duplicate release should fail
	_, err = ghMock.CreateRelease(ctx, "test", "monorepo", &github.CreateReleaseRequest{
		TagName: tag,
		Name:    tag,
		Body:    "Duplicate",
	})
	require.Error(t, err)
}

func TestMultiProjectChangesets(t *testing.T) {
	// Setup: Create workspace with two projects and changesets for both
	wb := workspace.NewWorkspaceBuilder("/test-workspace")
	wb.AddProject("auth", "packages/auth", "github.com/test/auth")
	wb.AddProject("api", "packages/api", "github.com/test/api")

	// Simulate what happens when a user selects both projects:
	// Two separate changeset files are created, one per project
	wb.AddChangeset("abc123", "auth", "minor", "Add OAuth support")
	wb.AddChangeset("def456", "api", "minor", "Add OAuth support")

	fs := wb.Build()

	// Setup workspace
	ws := workspace.New(fs)
	require.NoError(t, ws.Detect())

	csManager := changeset.NewManager(fs, ws.ChangesetDir())

	// Read all changesets
	allChangesets, err := csManager.ReadAll()
	require.NoError(t, err)

	require.Len(t, allChangesets, 2)

	// Version the 'auth' project
	authProject, _ := ws.GetProject("auth")
	authChangesets := changeset.FilterByProject(allChangesets, "auth")

	require.Len(t, authChangesets, 1)

	// Apply version to auth
	versionFile := versioning.NewVersionFile(fs)
	currentVersion, _ := versionFile.Read(authProject.RootPath)
	newVersion := currentVersion.Bump(models.BumpMinor)
	versionFile.Write(authProject.RootPath, newVersion)

	// Delete auth changesets
	for _, cs := range authChangesets {
		require.NoError(t, csManager.Delete(cs))
	}

	// Verify: Auth changeset is deleted
	remainingChangesets, err := csManager.ReadAll()
	require.NoError(t, err)

	require.Len(t, remainingChangesets, 1)

	// Verify: API changeset still exists
	apiChangesets := changeset.FilterByProject(remainingChangesets, "api")
	require.Len(t, apiChangesets, 1)

	// Verify the remaining changeset is for API
	require.True(t, apiChangesets[0].AffectsProject("api"), "remaining changeset should affect api project")

	require.False(t, apiChangesets[0].AffectsProject("auth"), "remaining changeset should not affect auth project")
}

func TestVersionBumping(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		bump     models.BumpType
		expected string
	}{
		{"patch bump", "1.2.3", models.BumpPatch, "1.2.4"},
		{"minor bump", "1.2.3", models.BumpMinor, "1.3.0"},
		{"major bump", "1.2.3", models.BumpMajor, "2.0.0"},
		{"patch from zero", "0.0.0", models.BumpPatch, "0.0.1"},
		{"minor from zero", "0.0.0", models.BumpMinor, "0.1.0"},
		{"major from zero", "0.0.0", models.BumpMajor, "1.0.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version, err := models.ParseVersion(tt.version)
			require.NoError(t, err)

			bumped := version.Bump(tt.bump)
			require.Equal(t, tt.expected, bumped.String())
		})
	}
}

func TestVersionPublishWithGitTags(t *testing.T) {
	// Setup mock workspace
	wb := workspace.NewWorkspaceBuilder("/test-workspace")
	wb.AddProject("auth", "packages/auth", "github.com/test/auth")
	wb.AddChangeset("abc123", "auth", "minor", "Add new feature")
	wb.SetVersion("auth", "0.0.0")

	fs := wb.Build()

	// Setup mock git client
	gitMock := git.NewMockGitClient()
	// Start with no tags (first release)

	// Setup mock GitHub
	ghMock := github.NewMockClient()
	ghMock.SetupRepository("test", "monorepo")

	// Workspace setup
	ws := workspace.New(fs)
	require.NoError(t, ws.Detect())

	project, _ := ws.GetProject("auth")

	// Step 1: Run version command (simulates what version command does)
	csManager := changeset.NewManager(fs, ws.ChangesetDir())
	allChangesets, _ := csManager.ReadAll()
	projectChangesets := changeset.FilterByProject(allChangesets, "auth")
	highestBump := csManager.GetHighestBump(projectChangesets, "auth")

	versionFile := versioning.NewVersionFile(fs)
	currentVersion, _ := versionFile.Read(project.RootPath)
	newVersion := currentVersion.Bump(highestBump)

	// Write new version.txt
	require.NoError(t, versionFile.Write(project.RootPath, newVersion))

	// Update changelog
	cl := changelog.NewChangelog(fs)
	entry := &changelog.Entry{
		Version:    newVersion,
		Date:       time.Now(),
		Changesets: projectChangesets,
	}
	cl.Append(project.RootPath, "auth", entry)

	// Delete changesets
	for _, cs := range projectChangesets {
		csManager.Delete(cs)
	}

	// Verify version.txt was updated
	readVersion, _ := versionFile.Read(project.RootPath)
	require.Equal(t, "0.1.0", readVersion.String())

	// Step 2: Run publish command logic
	// Read version from version.txt
	fileVersion, _ := versionFile.Read(project.RootPath)

	// Try to get latest tag (should fail - no tags yet)
	_, err := gitMock.GetLatestTag("auth")
	require.Error(t, err)

	// Since no tag exists, this is first release - should publish
	// Create git tag
	tag := "auth@v" + fileVersion.String()
	require.NoError(t, gitMock.CreateTag(tag, "Release 0.1.0"))

	// Push tag
	require.NoError(t, gitMock.PushTag(tag))

	// Create GitHub release
	ctx := context.Background()
	release, err := ghMock.CreateRelease(ctx, "test", "monorepo", &github.CreateReleaseRequest{
		TagName: tag,
		Name:    tag,
		Body:    "Release notes",
	})
	require.NoError(t, err)

	require.Equal(t, tag, release.TagName)

	// Step 3: Try to publish again - should skip (already published)
	// Get latest tag
	latestTag, err := gitMock.GetLatestTag("auth")
	require.NoError(t, err)

	require.Equal(t, tag, latestTag)

	// Read version from file
	fileVersion2, _ := versionFile.Read(project.RootPath)

	// Parse tag version
	tagVersionStr := strings.TrimPrefix(strings.Split(latestTag, "@")[1], "v")
	tagVersion, _ := models.ParseVersion(tagVersionStr)

	// Compare: fileVersion <= tagVersion means already published
	require.LessOrEqual(t, fileVersion2.Compare(tagVersion), 0)

	// Step 4: Add another changeset and version again
	wb2 := workspace.NewWorkspaceBuilder("/test-workspace")
	wb2.AddProject("auth", "packages/auth", "github.com/test/auth")
	wb2.AddChangeset("def456", "auth", "patch", "Fix bug")
	wb2.SetVersion("auth", "0.1.0") // Current version from previous release
	fs2 := wb2.Build()

	ws2 := workspace.New(fs2)
	ws2.Detect()
	project2, _ := ws2.GetProject("auth")

	// Version again
	csManager2 := changeset.NewManager(fs2, ws2.ChangesetDir())
	allChangesets2, _ := csManager2.ReadAll()
	projectChangesets2 := changeset.FilterByProject(allChangesets2, "auth")
	highestBump2 := csManager2.GetHighestBump(projectChangesets2, "auth")

	versionFile2 := versioning.NewVersionFile(fs2)
	currentVersion2, _ := versionFile2.Read(project2.RootPath)
	newVersion2 := currentVersion2.Bump(highestBump2)

	versionFile2.Write(project2.RootPath, newVersion2)

	// Verify new version
	readVersion2, _ := versionFile2.Read(project2.RootPath)
	require.Equal(t, "0.1.1", readVersion2.String())

	// Now publish should detect version.txt (0.1.1) > git tag (0.1.0)
	require.Greater(t, readVersion2.Compare(tagVersion), 0)

	// This would trigger a new publish
	newTag := "auth@v" + readVersion2.String()
	gitMock.CreateTag(newTag, "Release 0.1.1")
	gitMock.PushTag(newTag)

	// Verify both tags exist
	tags := gitMock.GetAllTags()
	require.Len(t, tags, 2)
}
