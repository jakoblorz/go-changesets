package workspace

import (
	"testing"

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
