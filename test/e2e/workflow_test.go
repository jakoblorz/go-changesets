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
	"github.com/jakoblorz/go-changesets/test/testutil"
)

func TestFullWorkflow(t *testing.T) {
	// Setup mock workspace
	wb := testutil.NewWorkspaceBuilder("/test-workspace")
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
	if err := ws.Detect(); err != nil {
		t.Fatalf("failed to detect workspace: %v", err)
	}

	if len(ws.Projects) != 2 {
		t.Errorf("expected 2 projects, got %d", len(ws.Projects))
	}

	// Test: Read changesets
	csManager := changeset.NewManager(fs, ws.ChangesetDir())
	changesets, err := csManager.ReadAll()
	if err != nil {
		t.Fatalf("failed to read changesets: %v", err)
	}

	if len(changesets) != 2 {
		t.Errorf("expected 2 changesets, got %d", len(changesets))
	}

	// Test: Filter changesets by project
	authChangesets := csManager.FilterByProject(changesets, "auth")
	if len(authChangesets) != 2 {
		t.Errorf("expected 2 changesets for auth, got %d", len(authChangesets))
	}

	// Test: Get highest bump
	highestBump := csManager.GetHighestBump(authChangesets, "auth")
	if highestBump != models.BumpMinor {
		t.Errorf("expected minor bump, got %s", highestBump)
	}

	// Test: Version management
	authProject, _ := ws.GetProject("auth")
	versionFile := versioning.NewVersionFile(fs)

	// Read initial version (should default to 0.0.0)
	currentVersion, err := versionFile.Read(authProject.RootPath)
	if err != nil {
		t.Fatalf("failed to read version: %v", err)
	}

	if currentVersion.String() != "0.0.0" {
		t.Errorf("expected version 0.0.0, got %s", currentVersion.String())
	}

	// Bump version
	newVersion := currentVersion.Bump(highestBump)
	if newVersion.String() != "0.1.0" {
		t.Errorf("expected version 0.1.0, got %s", newVersion.String())
	}

	// Write new version
	if err := versionFile.Write(authProject.RootPath, newVersion); err != nil {
		t.Fatalf("failed to write version: %v", err)
	}

	// Verify version was written
	readVersion, err := versionFile.Read(authProject.RootPath)
	if err != nil {
		t.Fatalf("failed to read version after write: %v", err)
	}

	if readVersion.String() != "0.1.0" {
		t.Errorf("expected version 0.1.0 after write, got %s", readVersion.String())
	}

	// Test: Changelog generation
	cl := changelog.NewChangelog(fs)
	entry := &changelog.Entry{
		Version:    newVersion,
		Date:       time.Now(),
		Changesets: authChangesets,
	}

	if err := cl.Append(authProject.RootPath, "auth", entry); err != nil {
		t.Fatalf("failed to append to changelog: %v", err)
	}

	// Verify changelog was created
	changelogPath := authProject.RootPath + "/CHANGELOG.md"
	changelogData, err := fs.ReadFile(changelogPath)
	if err != nil {
		t.Fatalf("failed to read changelog: %v", err)
	}

	changelogContent := string(changelogData)
	if !strings.Contains(changelogContent, "0.1.0") {
		t.Error("changelog missing version 0.1.0")
	}
	if !strings.Contains(changelogContent, "Add new login feature") {
		t.Error("changelog missing expected entry")
	}

	// Test: Delete changesets
	for _, cs := range authChangesets {
		if err := csManager.Delete(cs); err != nil {
			t.Fatalf("failed to delete changeset: %v", err)
		}
	}

	// Verify changesets were deleted
	remainingChangesets, err := csManager.ReadAll()
	if err != nil {
		t.Fatalf("failed to read changesets after deletion: %v", err)
	}

	if len(remainingChangesets) != 0 {
		t.Errorf("expected 0 changesets after deletion, got %d", len(remainingChangesets))
	}

	// Test: GitHub release creation
	ctx := context.Background()
	tag := "auth@v0.1.0"

	release, err := ghMock.CreateRelease(ctx, "test", "monorepo", &github.CreateReleaseRequest{
		TagName: tag,
		Name:    tag,
		Body:    "Release notes",
	})
	if err != nil {
		t.Fatalf("failed to create release: %v", err)
	}

	if release.TagName != tag {
		t.Errorf("expected tag %s, got %s", tag, release.TagName)
	}

	// Test: Check if release exists
	existingRelease, err := ghMock.GetReleaseByTag(ctx, "test", "monorepo", tag)
	if err != nil {
		t.Fatalf("failed to get release by tag: %v", err)
	}

	if existingRelease.TagName != tag {
		t.Errorf("expected tag %s, got %s", tag, existingRelease.TagName)
	}

	// Test: Duplicate release should fail
	_, err = ghMock.CreateRelease(ctx, "test", "monorepo", &github.CreateReleaseRequest{
		TagName: tag,
		Name:    tag,
		Body:    "Duplicate",
	})
	if err == nil {
		t.Error("expected error when creating duplicate release, got nil")
	}
}

func TestMultiProjectChangesets(t *testing.T) {
	// Setup: Create workspace with two projects and changesets for both
	wb := testutil.NewWorkspaceBuilder("/test-workspace")
	wb.AddProject("auth", "packages/auth", "github.com/test/auth")
	wb.AddProject("api", "packages/api", "github.com/test/api")

	// Simulate what happens when a user selects both projects:
	// Two separate changeset files are created, one per project
	wb.AddChangeset("abc123", "auth", "minor", "Add OAuth support")
	wb.AddChangeset("def456", "api", "minor", "Add OAuth support")

	fs := wb.Build()

	// Setup workspace
	ws := workspace.New(fs)
	if err := ws.Detect(); err != nil {
		t.Fatalf("failed to detect workspace: %v", err)
	}

	csManager := changeset.NewManager(fs, ws.ChangesetDir())

	// Read all changesets
	allChangesets, err := csManager.ReadAll()
	if err != nil {
		t.Fatalf("failed to read changesets: %v", err)
	}

	if len(allChangesets) != 2 {
		t.Fatalf("expected 2 changesets, got %d", len(allChangesets))
	}

	// Version the 'auth' project
	authProject, _ := ws.GetProject("auth")
	authChangesets := csManager.FilterByProject(allChangesets, "auth")

	if len(authChangesets) != 1 {
		t.Errorf("expected 1 changeset for auth, got %d", len(authChangesets))
	}

	// Apply version to auth
	versionFile := versioning.NewVersionFile(fs)
	currentVersion, _ := versionFile.Read(authProject.RootPath)
	newVersion := currentVersion.Bump(models.BumpMinor)
	versionFile.Write(authProject.RootPath, newVersion)

	// Delete auth changesets
	for _, cs := range authChangesets {
		if err := csManager.Delete(cs); err != nil {
			t.Fatalf("failed to delete changeset: %v", err)
		}
	}

	// Verify: Auth changeset is deleted
	remainingChangesets, err := csManager.ReadAll()
	if err != nil {
		t.Fatalf("failed to read changesets after deletion: %v", err)
	}

	if len(remainingChangesets) != 1 {
		t.Errorf("expected 1 changeset remaining, got %d", len(remainingChangesets))
	}

	// Verify: API changeset still exists
	apiChangesets := csManager.FilterByProject(remainingChangesets, "api")
	if len(apiChangesets) != 1 {
		t.Errorf("expected 1 changeset for api after versioning auth, got %d", len(apiChangesets))
	}

	// Verify the remaining changeset is for API
	if !apiChangesets[0].AffectsProject("api") {
		t.Error("remaining changeset should affect api project")
	}

	if apiChangesets[0].AffectsProject("auth") {
		t.Error("remaining changeset should not affect auth project")
	}
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
			if err != nil {
				t.Fatalf("failed to parse version: %v", err)
			}

			bumped := version.Bump(tt.bump)
			if bumped.String() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, bumped.String())
			}
		})
	}
}

func TestVersionPublishWithGitTags(t *testing.T) {
	// Setup mock workspace
	wb := testutil.NewWorkspaceBuilder("/test-workspace")
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
	if err := ws.Detect(); err != nil {
		t.Fatalf("failed to detect workspace: %v", err)
	}

	project, _ := ws.GetProject("auth")

	// Step 1: Run version command (simulates what version command does)
	csManager := changeset.NewManager(fs, ws.ChangesetDir())
	allChangesets, _ := csManager.ReadAll()
	projectChangesets := csManager.FilterByProject(allChangesets, "auth")
	highestBump := csManager.GetHighestBump(projectChangesets, "auth")

	versionFile := versioning.NewVersionFile(fs)
	currentVersion, _ := versionFile.Read(project.RootPath)
	newVersion := currentVersion.Bump(highestBump)

	// Write new version.txt
	if err := versionFile.Write(project.RootPath, newVersion); err != nil {
		t.Fatalf("failed to write version: %v", err)
	}

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
	if readVersion.String() != "0.1.0" {
		t.Errorf("expected version 0.1.0, got %s", readVersion.String())
	}

	// Step 2: Run publish command logic
	// Read version from version.txt
	fileVersion, _ := versionFile.Read(project.RootPath)

	// Try to get latest tag (should fail - no tags yet)
	_, err := gitMock.GetLatestTag("auth")
	if err == nil {
		t.Error("expected error for no tags, got nil")
	}

	// Since no tag exists, this is first release - should publish
	// Create git tag
	tag := "auth@v" + fileVersion.String()
	if err := gitMock.CreateTag(tag, "Release 0.1.0"); err != nil {
		t.Fatalf("failed to create tag: %v", err)
	}

	// Push tag
	if err := gitMock.PushTag(tag); err != nil {
		t.Fatalf("failed to push tag: %v", err)
	}

	// Create GitHub release
	ctx := context.Background()
	release, err := ghMock.CreateRelease(ctx, "test", "monorepo", &github.CreateReleaseRequest{
		TagName: tag,
		Name:    tag,
		Body:    "Release notes",
	})
	if err != nil {
		t.Fatalf("failed to create release: %v", err)
	}

	if release.TagName != tag {
		t.Errorf("expected tag %s, got %s", tag, release.TagName)
	}

	// Step 3: Try to publish again - should skip (already published)
	// Get latest tag
	latestTag, err := gitMock.GetLatestTag("auth")
	if err != nil {
		t.Fatalf("expected tag to exist, got error: %v", err)
	}

	if latestTag != tag {
		t.Errorf("expected tag %s, got %s", tag, latestTag)
	}

	// Read version from file
	fileVersion2, _ := versionFile.Read(project.RootPath)

	// Parse tag version
	tagVersionStr := strings.TrimPrefix(strings.Split(latestTag, "@")[1], "v")
	tagVersion, _ := models.ParseVersion(tagVersionStr)

	// Compare: fileVersion <= tagVersion means already published
	if fileVersion2.Compare(tagVersion) > 0 {
		t.Error("expected fileVersion to equal tagVersion (already published)")
	}

	// Step 4: Add another changeset and version again
	wb2 := testutil.NewWorkspaceBuilder("/test-workspace")
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
	projectChangesets2 := csManager2.FilterByProject(allChangesets2, "auth")
	highestBump2 := csManager2.GetHighestBump(projectChangesets2, "auth")

	versionFile2 := versioning.NewVersionFile(fs2)
	currentVersion2, _ := versionFile2.Read(project2.RootPath)
	newVersion2 := currentVersion2.Bump(highestBump2)

	versionFile2.Write(project2.RootPath, newVersion2)

	// Verify new version
	readVersion2, _ := versionFile2.Read(project2.RootPath)
	if readVersion2.String() != "0.1.1" {
		t.Errorf("expected version 0.1.1, got %s", readVersion2.String())
	}

	// Now publish should detect version.txt (0.1.1) > git tag (0.1.0)
	if readVersion2.Compare(tagVersion) <= 0 {
		t.Error("expected new version to be greater than tag version")
	}

	// This would trigger a new publish
	newTag := "auth@v" + readVersion2.String()
	gitMock.CreateTag(newTag, "Release 0.1.1")
	gitMock.PushTag(newTag)

	// Verify both tags exist
	tags := gitMock.GetAllTags()
	if len(tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(tags))
	}
}
