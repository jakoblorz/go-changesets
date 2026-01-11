package e2e_test

import (
	"testing"

	"github.com/jakoblorz/go-changesets/internal/changeset"
	"github.com/jakoblorz/go-changesets/internal/workspace"
	"github.com/jakoblorz/go-changesets/test/testutil"
)

func TestDisabledProject_NotInWorkspace(t *testing.T) {
	// Setup workspace with enabled and disabled projects
	wb := testutil.NewWorkspaceBuilder("/test-workspace")
	wb.AddProject("backend", "apps/backend", "github.com/test/backend")
	wb.AddProject("www", "apps/www", "github.com/test/www")
	wb.AddProject("internal", "apps/internal", "github.com/test/internal")

	// Disable internal project
	wb.SetVersion("internal", "false")

	fs := wb.Build()

	// Detect workspace
	ws := workspace.New(fs)
	err := ws.Detect()
	if err != nil {
		t.Fatalf("Workspace detection failed: %v", err)
	}

	// Should only have 2 projects (backend and www)
	if len(ws.Projects) != 2 {
		t.Errorf("Expected 2 enabled projects, got %d", len(ws.Projects))
	}

	// Verify disabled project not in list
	for _, p := range ws.Projects {
		if p.Name == "internal" {
			t.Error("Disabled project 'internal' should not be in workspace")
		}
	}

	// Verify enabled projects are present
	found := map[string]bool{"backend": false, "www": false}
	for _, p := range ws.Projects {
		if _, exists := found[p.Name]; exists {
			found[p.Name] = true
		}
	}

	for name, wasFound := range found {
		if !wasFound {
			t.Errorf("Expected to find enabled project %s", name)
		}
	}
}

func TestDisabledProject_NotInProjectNames(t *testing.T) {
	wb := testutil.NewWorkspaceBuilder("/test-workspace")
	wb.AddProject("backend", "apps/backend", "github.com/test/backend")
	wb.AddProject("www", "apps/www", "github.com/test/www")
	wb.AddProject("internal", "apps/internal", "github.com/test/internal")

	// Disable internal project
	wb.SetVersion("internal", "false")

	fs := wb.Build()
	ws := workspace.New(fs)
	ws.Detect()

	// GetProjectNames should only return enabled projects
	names := ws.GetProjectNames()

	if len(names) != 2 {
		t.Errorf("Expected 2 project names, got %d: %v", len(names), names)
	}

	// Check disabled project not in names
	for _, name := range names {
		if name == "internal" {
			t.Error("Disabled project 'internal' should not be in project names")
		}
	}
}

func TestDisabledProject_NotFoundByGetProject(t *testing.T) {
	wb := testutil.NewWorkspaceBuilder("/test-workspace")
	wb.AddProject("backend", "apps/backend", "github.com/test/backend")
	wb.AddProject("internal", "apps/internal", "github.com/test/internal")

	// Disable internal project
	wb.SetVersion("internal", "false")

	fs := wb.Build()
	ws := workspace.New(fs)
	ws.Detect()

	// Try to get disabled project by name
	_, err := ws.GetProject("internal")
	if err == nil {
		t.Error("Expected error when getting disabled project, got nil")
	}

	if err != nil && err.Error() != "project internal not found in workspace" {
		t.Errorf("Expected 'not found' error, got: %v", err)
	}

	// Enabled project should still work
	project, err := ws.GetProject("backend")
	if err != nil {
		t.Errorf("Expected to find enabled project, got error: %v", err)
	}
	if project == nil || project.Name != "backend" {
		t.Error("Expected to get backend project")
	}
}

func TestDisabledProject_WithChangesets(t *testing.T) {
	wb := testutil.NewWorkspaceBuilder("/test-workspace")
	wb.AddProject("backend", "apps/backend", "github.com/test/backend")
	wb.AddProject("internal", "apps/internal", "github.com/test/internal")

	// Add changesets for both projects
	wb.AddChangeset("abc123", "backend", "minor", "backend feature")
	wb.AddChangeset("def456", "internal", "minor", "Internal feature")

	// Disable internal project
	wb.SetVersion("internal", "false")

	fs := wb.Build()
	ws := workspace.New(fs)
	ws.Detect()

	// Read all changesets
	csManager := changeset.NewManager(fs, ws.ChangesetDir())
	allChangesets, err := csManager.ReadAll()
	if err != nil {
		t.Fatalf("Failed to read changesets: %v", err)
	}

	// Should have 2 changesets (both exist in filesystem)
	if len(allChangesets) != 2 {
		t.Errorf("Expected 2 changesets, got %d", len(allChangesets))
	}

	// Filter by backend - should get 1
	backendCS := csManager.FilterByProject(allChangesets, "backend")
	if len(backendCS) != 1 {
		t.Errorf("Expected 1 changeset for backend, got %d", len(backendCS))
	}

	// Filter by internal - should get 1 (changesets still exist)
	internalCS := csManager.FilterByProject(allChangesets, "internal")
	if len(internalCS) != 1 {
		t.Errorf("Expected 1 changeset for internal, got %d", len(internalCS))
	}

	// Note: The changeset still exists in the filesystem, but the project
	// won't appear in commands because it's filtered at workspace level
}

func TestDisabledProject_AllProjectsDisabled(t *testing.T) {
	wb := testutil.NewWorkspaceBuilder("/test-workspace")
	wb.AddProject("internal1", "apps/internal1", "github.com/test/internal1")
	wb.AddProject("internal2", "apps/internal2", "github.com/test/internal2")

	// Disable both projects
	wb.SetVersion("internal1", "false")
	wb.SetVersion("internal2", "false")

	fs := wb.Build()
	ws := workspace.New(fs)

	// Workspace detection should fail (no enabled projects)
	err := ws.Detect()
	if err == nil {
		t.Error("Expected error when all projects disabled, got nil")
	}

	if err != nil && err.Error() != "no projects found in workspace" {
		t.Logf("Got error: %v", err)
	}
}
