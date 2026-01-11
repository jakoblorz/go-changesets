package e2e_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/jakoblorz/go-changesets/internal/changeset"
	"github.com/jakoblorz/go-changesets/internal/models"
	"github.com/jakoblorz/go-changesets/internal/versioning"
	"github.com/jakoblorz/go-changesets/internal/workspace"
	"github.com/jakoblorz/go-changesets/test/testutil"
)

func TestChangelogPreview_InProjectContext(t *testing.T) {
	// Setup mock workspace with multiple projects
	wb := testutil.NewWorkspaceBuilder("/test-workspace")
	wb.AddProject("auth", "packages/auth", "github.com/test/auth")
	wb.AddProject("api", "packages/api", "github.com/test/api")

	// Add changesets
	wb.AddChangeset("auth-minor", "auth", "minor", "Add OAuth2 authentication support")
	wb.AddChangeset("auth-patch", "auth", "patch", "Fix memory leak in session handler")
	wb.AddChangeset("api-minor", "api", "minor", "Add GraphQL endpoint")

	fs := wb.Build()

	// Detect workspace
	ws := workspace.New(fs)
	if err := ws.Detect(); err != nil {
		t.Fatalf("failed to detect workspace: %v", err)
	}

	// Read changesets
	csManager := changeset.NewManager(fs, ws.ChangesetDir())
	allChangesets, err := csManager.ReadAll()
	if err != nil {
		t.Fatalf("failed to read changesets: %v", err)
	}

	if len(allChangesets) != 3 {
		t.Errorf("expected 3 changesets, got %d", len(allChangesets))
	}

	// Test auth project changelog preview
	authChangesets := csManager.FilterByProject(allChangesets, "auth")
	if len(authChangesets) != 2 {
		t.Fatalf("expected 2 auth changesets, got %d", len(authChangesets))
	}

	changelog := versioning.NewChangelog(fs)
	authProject, _ := ws.GetProject("auth")
	authPreview, err := changelog.FormatEntry(authChangesets, "auth", authProject.RootPath)
	if err != nil {
		t.Fatalf("failed to format auth preview: %v", err)
	}

	// Verify auth preview contains both changes grouped by type
	if !strings.Contains(authPreview, "### Minor Changes") {
		t.Errorf("Expected '### Minor Changes' in auth preview")
	}
	if !strings.Contains(authPreview, "### Patch Changes") {
		t.Errorf("Expected '### Patch Changes' in auth preview")
	}
	if !strings.Contains(authPreview, "Add OAuth2 authentication support") {
		t.Errorf("Expected OAuth2 message in auth preview")
	}
	if !strings.Contains(authPreview, "Fix memory leak in session handler") {
		t.Errorf("Expected memory leak fix in auth preview")
	}
}

func TestChangelogPreview_MultilineMessages(t *testing.T) {
	// Setup workspace
	wb := testutil.NewWorkspaceBuilder("/test-workspace")
	wb.AddProject("auth", "packages/auth", "github.com/test/auth")

	// Add changeset with multiline message
	multilineMsg := `Add comprehensive OAuth2 support

This change includes:
- Google OAuth2 provider
- GitHub OAuth2 provider
- Token refresh mechanism`

	wb.AddChangeset("multiline", "auth", "minor", multilineMsg)
	fs := wb.Build()

	ws := workspace.New(fs)
	if err := ws.Detect(); err != nil {
		t.Fatalf("failed to detect workspace: %v", err)
	}

	csManager := changeset.NewManager(fs, ws.ChangesetDir())
	allChangesets, err := csManager.ReadAll()
	if err != nil {
		t.Fatalf("failed to read changesets: %v", err)
	}

	authChangesets := csManager.FilterByProject(allChangesets, "auth")
	changelog := versioning.NewChangelog(fs)
	authProject, _ := ws.GetProject("auth")
	preview, err := changelog.FormatEntry(authChangesets, "auth", authProject.RootPath)
	if err != nil {
		t.Fatalf("failed to format preview: %v", err)
	}

	// Verify multiline formatting
	if !strings.Contains(preview, "Add comprehensive OAuth2 support") {
		t.Errorf("Expected first line in preview")
	}
	if !strings.Contains(preview, "Google OAuth2 provider") {
		t.Errorf("Expected detail line in preview")
	}
	// Lines should be indented
	lines := strings.Split(preview, "\n")
	foundIndented := false
	for _, line := range lines {
		if strings.HasPrefix(line, "  ") && strings.Contains(line, "Google") {
			foundIndented = true
			break
		}
	}
	if !foundIndented {
		t.Errorf("Expected indented lines for multiline message")
	}
}

func TestChangelogPreview_ContextIntegration(t *testing.T) {
	// Test that ProjectContext would include changelog preview
	// This simulates what the 'each' command does
	wb := testutil.NewWorkspaceBuilder("/test-workspace")
	wb.AddProject("auth", "packages/auth", "github.com/test/auth")
	wb.SetVersion("auth", "0.1.0")
	wb.AddChangeset("test", "auth", "minor", "Add new feature")
	fs := wb.Build()

	ws := workspace.New(fs)
	if err := ws.Detect(); err != nil {
		t.Fatalf("failed to detect workspace: %v", err)
	}

	// Read changesets
	csManager := changeset.NewManager(fs, ws.ChangesetDir())
	allChangesets, err := csManager.ReadAll()
	if err != nil {
		t.Fatalf("failed to read changesets: %v", err)
	}

	// Build context for auth project (simulating what each command does)
	project := ws.Projects[0]
	projectChangesets := csManager.FilterByProject(allChangesets, project.Name)

	// Generate changelog preview
	var changelogPreview string
	if len(projectChangesets) > 0 {
		changelog := versioning.NewChangelog(fs)
		changelogPreview, err = changelog.FormatEntry(projectChangesets, project.Name, project.RootPath)
		if err != nil {
			t.Fatalf("failed to format changelog preview: %v", err)
		}
	}

	// Create context
	ctx := &models.ProjectContext{
		Project:          project.Name,
		ProjectPath:      project.RootPath,
		ModulePath:       project.ModulePath,
		HasChangesets:    len(projectChangesets) > 0,
		ChangelogPreview: changelogPreview,
	}

	// Verify context has changelog preview
	if ctx.ChangelogPreview == "" {
		t.Error("Expected ChangelogPreview to be populated in context")
	}

	if !strings.Contains(ctx.ChangelogPreview, "Add new feature") {
		t.Errorf("Expected changeset message in preview, got: %s", ctx.ChangelogPreview)
	}

	// Verify it can be marshaled to JSON
	jsonData, err := json.Marshal(ctx)
	if err != nil {
		t.Fatalf("Failed to marshal context to JSON: %v", err)
	}

	// Verify JSON contains changelogPreview field
	jsonStr := string(jsonData)
	if !strings.Contains(jsonStr, "changelogPreview") {
		t.Error("Expected 'changelogPreview' field in JSON")
	}

	// Unmarshal and verify
	var ctxCopy models.ProjectContext
	if err := json.Unmarshal(jsonData, &ctxCopy); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if ctxCopy.ChangelogPreview != ctx.ChangelogPreview {
		t.Error("ChangelogPreview not preserved through JSON round-trip")
	}
}

func TestChangelogPreview_OrderByBumpType(t *testing.T) {
	// Verify that changesets are ordered by bump type (Major, Minor, Patch)
	wb := testutil.NewWorkspaceBuilder("/test-workspace")
	wb.AddProject("auth", "packages/auth", "github.com/test/auth")

	// Add in random order
	wb.AddChangeset("patch1", "auth", "patch", "Fix bug 1")
	wb.AddChangeset("major1", "auth", "major", "Breaking change")
	wb.AddChangeset("minor1", "auth", "minor", "New feature")
	wb.AddChangeset("patch2", "auth", "patch", "Fix bug 2")

	fs := wb.Build()
	ws := workspace.New(fs)
	if err := ws.Detect(); err != nil {
		t.Fatalf("failed to detect workspace: %v", err)
	}

	csManager := changeset.NewManager(fs, ws.ChangesetDir())
	allChangesets, err := csManager.ReadAll()
	if err != nil {
		t.Fatalf("failed to read changesets: %v", err)
	}

	authChangesets := csManager.FilterByProject(allChangesets, "auth")
	changelog := versioning.NewChangelog(fs)
	authProject, _ := ws.GetProject("auth")
	preview, err := changelog.FormatEntry(authChangesets, "auth", authProject.RootPath)
	if err != nil {
		t.Fatalf("failed to format preview: %v", err)
	}

	// Find positions of each section
	majorIdx := strings.Index(preview, "### Major Changes")
	minorIdx := strings.Index(preview, "### Minor Changes")
	patchIdx := strings.Index(preview, "### Patch Changes")

	if majorIdx == -1 || minorIdx == -1 || patchIdx == -1 {
		t.Fatalf("Missing sections in preview:\n%s", preview)
	}

	// Verify order: Major before Minor before Patch
	if majorIdx > minorIdx {
		t.Error("Major Changes should come before Minor Changes")
	}
	if minorIdx > patchIdx {
		t.Error("Minor Changes should come before Patch Changes")
	}
}
