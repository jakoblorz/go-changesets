package workspace

import (
	"testing"

	"github.com/jakoblorz/go-changesets/internal/changeset"
	"github.com/jakoblorz/go-changesets/internal/filesystem"
	"github.com/jakoblorz/go-changesets/internal/models"
)

func TestWorkspaceDetect_GoSingleProject(t *testing.T) {
	fs := filesystem.NewMockFileSystem()
	fs.AddFile("/workspace/go.work", []byte("go 1.21\nuse ./auth\n"))
	fs.AddFile("/workspace/auth/go.mod", []byte("module github.com/test/auth\n\ngo 1.21\n"))

	ws := New(fs)
	if err := ws.Detect(); err != nil {
		t.Fatalf("Detect() error = %v", err)
	}

	if len(ws.Projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(ws.Projects))
	}

	project := ws.Projects[0]
	if project.Name != "auth" {
		t.Fatalf("unexpected project name: %s", project.Name)
	}
	if project.Type != models.ProjectTypeGo {
		t.Fatalf("expected project type go, got %s", project.Type)
	}
	if project.ManifestPath != "/workspace/auth/go.mod" {
		t.Fatalf("unexpected manifest path: %s", project.ManifestPath)
	}
}

func TestWorkspaceDetect_GoDisabledProject(t *testing.T) {
	fs := filesystem.NewMockFileSystem()
	fs.AddFile("/workspace/go.work", []byte("go 1.21\nuse ./enabled\nuse ./disabled\n"))

	fs.AddFile("/workspace/enabled/go.mod", []byte("module github.com/test/enabled\n\ngo 1.21\n"))
	fs.AddFile("/workspace/disabled/go.mod", []byte("module github.com/test/disabled\n\ngo 1.21\n"))
	fs.AddFile("/workspace/disabled/version.txt", []byte("false\n"))

	ws := New(fs)
	if err := ws.Detect(); err != nil {
		t.Fatalf("Detect() error = %v", err)
	}

	if len(ws.Projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(ws.Projects))
	}
	if ws.Projects[0].Name != "enabled" {
		t.Fatalf("unexpected project name: %s", ws.Projects[0].Name)
	}
}

func TestWorkspaceDetect_GoNameCollision(t *testing.T) {
	fs := filesystem.NewMockFileSystem()
	fs.AddFile("/workspace/go.work", []byte("go 1.21\nuse ./app1\nuse ./app2\n"))
	fs.AddFile("/workspace/app1/go.mod", []byte("module github.com/test/web\n\ngo 1.21\n"))
	fs.AddFile("/workspace/app2/go.mod", []byte("module github.com/other/web\n\ngo 1.21\n"))

	ws := New(fs)
	if err := ws.Detect(); err != nil {
		t.Fatalf("Detect() error = %v", err)
	}

	if len(ws.Projects) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(ws.Projects))
	}

	names := map[string]bool{}
	for _, p := range ws.Projects {
		names[p.Name] = true
		if p.Type != models.ProjectTypeGo {
			t.Fatalf("expected project type go, got %s", p.Type)
		}
	}

	if !names["web-go"] || !names["web-go-2"] {
		t.Fatalf("unexpected deduped names: %v", names)
	}
}

func TestWorkspaceDetect_WorkspaceNotFound(t *testing.T) {
	fs := filesystem.NewMockFileSystem()

	ws := New(fs)
	if err := ws.Detect(); err == nil {
		t.Fatalf("expected error")
	} else if err.Error() != "workspace not found" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWorkspaceDetect_NotInWorkspace(t *testing.T) {
	// Setup workspace with enabled and disabled projects
	wb := NewWorkspaceBuilder("/test-workspace")
	wb.AddProject("backend", "apps/backend", "github.com/test/backend")
	wb.AddProject("www", "apps/www", "github.com/test/www")
	wb.AddProject("internal", "apps/internal", "github.com/test/internal")

	// Disable internal project
	wb.SetVersion("internal", "false")

	fs := wb.Build()

	// Detect workspace
	ws := New(fs)
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

func TestWorkspaceDetect_NotInProjectNames(t *testing.T) {
	wb := NewWorkspaceBuilder("/test-workspace")
	wb.AddProject("backend", "apps/backend", "github.com/test/backend")
	wb.AddProject("www", "apps/www", "github.com/test/www")
	wb.AddProject("internal", "apps/internal", "github.com/test/internal")

	// Disable internal project
	wb.SetVersion("internal", "false")

	fs := wb.Build()
	ws := New(fs)
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

func TestWorkspaceDetect_NotFoundByGetProject(t *testing.T) {
	wb := NewWorkspaceBuilder("/test-workspace")
	wb.AddProject("backend", "apps/backend", "github.com/test/backend")
	wb.AddProject("internal", "apps/internal", "github.com/test/internal")

	// Disable internal project
	wb.SetVersion("internal", "false")

	fs := wb.Build()
	ws := New(fs)
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

func TestWorkspaceDetect_WithChangesets(t *testing.T) {
	wb := NewWorkspaceBuilder("/test-workspace")
	wb.AddProject("backend", "apps/backend", "github.com/test/backend")
	wb.AddProject("internal", "apps/internal", "github.com/test/internal")

	// Add changesets for both projects
	wb.AddChangeset("abc123", "backend", "minor", "backend feature")
	wb.AddChangeset("def456", "internal", "minor", "Internal feature")

	// Disable internal project
	wb.SetVersion("internal", "false")

	fs := wb.Build()
	ws := New(fs)
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
	backendCS := changeset.FilterByProject(allChangesets, "backend")
	if len(backendCS) != 1 {
		t.Errorf("Expected 1 changeset for backend, got %d", len(backendCS))
	}

	// Filter by internal - should get 1 (changesets still exist)
	internalCS := changeset.FilterByProject(allChangesets, "internal")
	if len(internalCS) != 1 {
		t.Errorf("Expected 1 changeset for internal, got %d", len(internalCS))
	}

	// Note: The changeset still exists in the filesystem, but the project
	// won't appear in commands because it's filtered at workspace level
}

func TestWorkspaceDetect_AllProjectsDisabled(t *testing.T) {
	wb := NewWorkspaceBuilder("/test-workspace")
	wb.AddProject("internal1", "apps/internal1", "github.com/test/internal1")
	wb.AddProject("internal2", "apps/internal2", "github.com/test/internal2")

	// Disable both projects
	wb.SetVersion("internal1", "false")
	wb.SetVersion("internal2", "false")

	fs := wb.Build()
	ws := New(fs)

	// Workspace detection should fail (no enabled projects)
	err := ws.Detect()
	if err == nil {
		t.Error("Expected error when all projects disabled, got nil")
	}

	if err != nil && err.Error() != "no projects found in workspace" {
		t.Logf("Got error: %v", err)
	}
}
