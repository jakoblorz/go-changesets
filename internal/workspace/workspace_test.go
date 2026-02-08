package workspace

import (
	"bytes"
	"errors"
	"testing"

	"github.com/jakoblorz/go-changesets/internal/changeset"
	"github.com/jakoblorz/go-changesets/internal/filesystem"
	"github.com/jakoblorz/go-changesets/internal/models"
	"github.com/stretchr/testify/require"
)

const testWorkspaceRoot = "/test-workspace"

type stubGoEnvReader struct {
	env GoEnv
	err error
}

func (r stubGoEnvReader) Read() (GoEnv, error) {
	return r.env, r.err
}

func buildWorkspace(t *testing.T, setup func(*WorkspaceBuilder)) (*Workspace, *filesystem.MockFileSystem) {
	t.Helper()
	wb := NewWorkspaceBuilder(testWorkspaceRoot)
	if setup != nil {
		setup(wb)
	}
	fs := wb.Build()
	ws := New(fs, WithGoEnv(NewMockGoEnvReader(fs)))
	require.NoError(t, ws.Detect())
	return ws, fs
}

func TestWorkspaceDetect_GoSingleProject(t *testing.T) {
	ws, _ := buildWorkspace(t, func(wb *WorkspaceBuilder) {
		wb.AddProject("auth", "auth", "github.com/test/auth")
	})

	require.Len(t, ws.Projects, 1)
	project := ws.Projects[0]
	require.Equal(t, "auth", project.Name)
	require.Equal(t, models.ProjectTypeGo, project.Type)
	require.Equal(t, testWorkspaceRoot+"/auth/go.mod", project.ManifestPath)
}

func TestWorkspaceDetect_GoDisabledProject(t *testing.T) {
	ws, _ := buildWorkspace(t, func(wb *WorkspaceBuilder) {
		wb.AddProject("enabled", "enabled", "github.com/test/enabled")
		wb.AddProject("disabled", "disabled", "github.com/test/disabled")
		wb.SetVersion("disabled", "false")
	})

	require.Len(t, ws.Projects, 1)
	require.Equal(t, "enabled", ws.Projects[0].Name)
}

func TestWorkspaceDetect_GoNameCollision(t *testing.T) {
	ws, _ := buildWorkspace(t, func(wb *WorkspaceBuilder) {
		wb.AddProject("app1", "app1", "github.com/test/web")
		wb.AddProject("app2", "app2", "github.com/other/web")
	})

	require.Len(t, ws.Projects, 2)

	names := map[string]bool{}
	for _, p := range ws.Projects {
		names[p.Name] = true
		require.Equal(t, models.ProjectTypeGo, p.Type)
	}

	require.Truef(t, names["web-go"], "expected web-go in %v", names)
	require.Truef(t, names["web-go-2"], "expected web-go-2 in %v", names)
}

func TestWorkspaceDetect_NodeSingleProject(t *testing.T) {
	fs := filesystem.NewMockFileSystem()
	fs.AddFile("/workspace/package.json", []byte(`{"name":"root-app","version":"0.1.0"}`))
	fs.SetCurrentDir("/workspace")

	ws := New(fs, WithGoEnv(NewMockGoEnvReader(fs)))
	require.NoError(t, ws.Detect())

	require.Len(t, ws.Projects, 1)

	project := ws.Projects[0]
	require.Equal(t, "root-app", project.Name)
	require.Equal(t, models.ProjectTypeNode, project.Type)
	require.Equal(t, "/workspace/package.json", project.ManifestPath)
}

func TestWorkspaceDetect_NodeWorkspaces(t *testing.T) {
	fs := filesystem.NewMockFileSystem()
	fs.AddFile("/workspace/package.json", []byte(`{"name":"root","private":true,"workspaces":["packages/*"]}`))
	fs.AddFile("/workspace/packages/api/package.json", []byte(`{"name":"api","version":"0.1.0"}`))
	fs.AddFile("/workspace/packages/web/package.json", []byte(`{"name":"web","version":"0.2.0"}`))
	fs.SetCurrentDir("/workspace")

	ws := New(fs, WithGoEnv(NewMockGoEnvReader(fs)))
	require.NoError(t, ws.Detect())

	require.Len(t, ws.Projects, 2)

	var names []string
	for _, project := range ws.Projects {
		names = append(names, project.Name)
	}
	require.Contains(t, names, "api")
	require.Contains(t, names, "web")
}

func TestWorkspaceDetect_NodeWorkspacesSkipsPrivate(t *testing.T) {
	fs := filesystem.NewMockFileSystem()
	fs.AddFile("/workspace/package.json", []byte(`{"name":"root","private":true,"workspaces":["packages/*"]}`))
	fs.AddFile("/workspace/packages/api/package.json", []byte(`{"name":"api","private":true,"version":"0.1.0"}`))
	fs.AddFile("/workspace/packages/web/package.json", []byte(`{"name":"web","version":"0.2.0"}`))
	fs.SetCurrentDir("/workspace")

	ws := New(fs, WithGoEnv(NewMockGoEnvReader(fs)))
	require.NoError(t, ws.Detect())

	require.Len(t, ws.Projects, 1)
	require.Equal(t, "web", ws.Projects[0].Name)
	require.Equal(t, models.ProjectTypeNode, ws.Projects[0].Type)
}

func TestWorkspaceDetect_NodeWorkspacesIncludesUnlistedPackagesByDefault(t *testing.T) {
	fs := filesystem.NewMockFileSystem()
	fs.AddFile("/workspace/package.json", []byte(`{"name":"root","private":true,"workspaces":["packages/*"]}`))
	fs.AddFile("/workspace/packages/api/package.json", []byte(`{"name":"api","version":"0.1.0"}`))
	fs.AddFile("/workspace/packages/web/package.json", []byte(`{"name":"web","version":"0.2.0"}`))
	fs.AddFile("/workspace/tools/cli/package.json", []byte(`{"name":"cli","version":"0.1.0"}`))
	fs.SetCurrentDir("/workspace")

	ws := New(fs, WithGoEnv(NewMockGoEnvReader(fs)))
	require.NoError(t, ws.Detect())

	require.Len(t, ws.Projects, 3)

	var names []string
	for _, project := range ws.Projects {
		names = append(names, project.Name)
	}
	require.Contains(t, names, "api")
	require.Contains(t, names, "web")
	require.Contains(t, names, "cli")
}

func TestWorkspaceDetect_NodeStrictWorkspaceSkipsUnlistedPackages(t *testing.T) {
	fs := filesystem.NewMockFileSystem()
	fs.AddFile("/workspace/package.json", []byte(`{"name":"root","private":true,"workspaces":["packages/*"]}`))
	fs.AddFile("/workspace/packages/api/package.json", []byte(`{"name":"api","version":"0.1.0"}`))
	fs.AddFile("/workspace/packages/web/package.json", []byte(`{"name":"web","version":"0.2.0"}`))
	fs.AddFile("/workspace/tools/cli/package.json", []byte(`{"name":"cli","version":"0.1.0"}`))
	fs.SetCurrentDir("/workspace")

	ws := New(fs, WithNodeStrictWorkspace(true), WithGoEnv(NewMockGoEnvReader(fs)))
	require.NoError(t, ws.Detect())

	require.Len(t, ws.Projects, 2)

	var names []string
	for _, project := range ws.Projects {
		names = append(names, project.Name)
	}
	require.Contains(t, names, "api")
	require.Contains(t, names, "web")
	require.NotContains(t, names, "cli")
}

func TestWorkspaceDetect_NodeFuzzyRespectsGitIgnore(t *testing.T) {
	fs := filesystem.NewMockFileSystem()
	fs.AddFile("/workspace/package.json", []byte(`{"name":"root","private":true}`))
	fs.AddFile("/workspace/.gitignore", []byte("ignored/\npackages/secret/\n"))
	fs.AddFile("/workspace/ignored/package.json", []byte(`{"name":"ignored","version":"0.1.0"}`))
	fs.AddFile("/workspace/packages/secret/package.json", []byte(`{"name":"secret","version":"0.1.0"}`))
	fs.AddFile("/workspace/packages/public/package.json", []byte(`{"name":"public","version":"0.1.0"}`))
	fs.SetCurrentDir("/workspace")

	ws := New(fs, WithGoEnv(NewMockGoEnvReader(fs)))
	require.NoError(t, ws.Detect())

	require.Len(t, ws.Projects, 1)
	require.Equal(t, "public", ws.Projects[0].Name)
}

func TestWorkspaceDetect_MixedGoAndNodeWithCollision(t *testing.T) {
	fs := filesystem.NewMockFileSystem()
	fs.AddFile("/workspace/go.work", []byte("go 1.21\nuse ./goapp\n"))
	fs.AddFile("/workspace/goapp/go.mod", []byte("module github.com/test/web\n\ngo 1.21\n"))
	fs.AddFile("/workspace/package.json", []byte(`{"name":"root","private":true,"workspaces":["packages/*"]}`))
	fs.AddFile("/workspace/packages/web/package.json", []byte(`{"name":"web","version":"1.0.0"}`))
	fs.SetCurrentDir("/workspace")

	ws := New(fs, WithGoEnv(NewMockGoEnvReader(fs)))
	require.NoError(t, ws.Detect())

	require.Len(t, ws.Projects, 2)

	projectNames := map[string]models.ProjectType{}
	for _, p := range ws.Projects {
		projectNames[p.Name] = p.Type
	}

	require.Equal(t, models.ProjectTypeGo, projectNames["web-go"])
	require.Equal(t, models.ProjectTypeNode, projectNames["web-node"])
}

func TestWorkspaceDetect_WorkspaceNotFound(t *testing.T) {
	fs := NewWorkspaceBuilder(testWorkspaceRoot).FileSystem()
	ws := New(fs, WithGoEnv(NewMockGoEnvReader(fs)))
	err := ws.Detect()
	require.Error(t, err)
	require.Equal(t, "workspace not found", err.Error())
}

func TestWorkspaceDetect_NotInWorkspace(t *testing.T) {
	ws, _ := buildWorkspace(t, func(wb *WorkspaceBuilder) {
		wb.AddProject("backend", "apps/backend", "github.com/test/backend")
		wb.AddProject("www", "apps/www", "github.com/test/www")
		wb.AddProject("internal", "apps/internal", "github.com/test/internal")
		wb.SetVersion("internal", "false")
	})

	require.Len(t, ws.Projects, 2)
	for _, p := range ws.Projects {
		require.NotEqual(t, "internal", p.Name)
	}

	found := map[string]bool{"backend": false, "www": false}
	for _, p := range ws.Projects {
		if _, ok := found[p.Name]; ok {
			found[p.Name] = true
		}
	}
	for name, ok := range found {
		require.Truef(t, ok, "expected to find project %s", name)
	}
}

func TestWorkspaceDetect_NotInProjectNames(t *testing.T) {
	ws, _ := buildWorkspace(t, func(wb *WorkspaceBuilder) {
		wb.AddProject("backend", "apps/backend", "github.com/test/backend")
		wb.AddProject("www", "apps/www", "github.com/test/www")
		wb.AddProject("internal", "apps/internal", "github.com/test/internal")
		wb.SetVersion("internal", "false")
	})

	names := ws.GetProjectNames()
	require.Len(t, names, 2)
	for _, name := range names {
		require.NotEqual(t, "internal", name)
	}
}

func TestWorkspaceDetect_NotFoundByGetProject(t *testing.T) {
	ws, _ := buildWorkspace(t, func(wb *WorkspaceBuilder) {
		wb.AddProject("backend", "apps/backend", "github.com/test/backend")
		wb.AddProject("internal", "apps/internal", "github.com/test/internal")
		wb.SetVersion("internal", "false")
	})

	_, err := ws.GetProject("internal")
	require.Error(t, err)
	require.Equal(t, "project internal not found in workspace", err.Error())

	project, err := ws.GetProject("backend")
	require.NoError(t, err)
	require.Equal(t, "backend", project.Name)
}

func TestWorkspaceDetect_WithChangesets(t *testing.T) {
	ws, fs := buildWorkspace(t, func(wb *WorkspaceBuilder) {
		wb.AddProject("backend", "apps/backend", "github.com/test/backend")
		wb.AddProject("internal", "apps/internal", "github.com/test/internal")
		wb.AddChangeset("abc123", "backend", "minor", "backend feature")
		wb.AddChangeset("def456", "internal", "minor", "Internal feature")
		wb.SetVersion("internal", "false")
	})

	csManager := changeset.NewManager(fs, ws.ChangesetDir())
	allChangesets, err := csManager.ReadAll()
	require.NoError(t, err)
	require.Len(t, allChangesets, 2)

	require.Len(t, changeset.FilterByProject(allChangesets, "backend"), 1)
	require.Len(t, changeset.FilterByProject(allChangesets, "internal"), 1)
}

func TestWorkspaceDetect_AllProjectsDisabled(t *testing.T) {
	wb := NewWorkspaceBuilder(testWorkspaceRoot)
	wb.AddProject("internal1", "apps/internal1", "github.com/test/internal1")
	wb.AddProject("internal2", "apps/internal2", "github.com/test/internal2")
	wb.SetVersion("internal1", "false")
	wb.SetVersion("internal2", "false")

	fs := wb.Build()
	ws := New(fs, WithGoEnv(NewMockGoEnvReader(fs)))
	err := ws.Detect()
	require.Error(t, err)
	require.Equal(t, "failed to load projects: no projects found in workspace", err.Error())
}

func TestWorkspaceDetect_GoEnvErrorWarnsAndContinues(t *testing.T) {
	fs := filesystem.NewMockFileSystem()
	fs.AddFile("/workspace/package.json", []byte(`{"name":"root-app","version":"0.1.0"}`))
	fs.SetCurrentDir("/workspace")

	var warnings bytes.Buffer
	ws := New(
		fs,
		WithGoEnv(stubGoEnvReader{err: errors.New("go env unavailable")}),
		WithWarningWriter(&warnings),
	)
	require.NoError(t, ws.Detect())
	require.Contains(t, warnings.String(), "warning: failed to read go env")

	require.Len(t, ws.Projects, 1)
	require.Equal(t, models.ProjectTypeNode, ws.Projects[0].Type)
}

func TestWorkspaceDetect_GoModSingleProject(t *testing.T) {
	fs := filesystem.NewMockFileSystem()
	fs.AddFile("/workspace/go.mod", []byte("module github.com/test/root\n\ngo 1.21\n"))
	fs.SetCurrentDir("/workspace")

	ws := New(fs, WithGoEnv(stubGoEnvReader{env: GoEnv{GoMod: "/workspace/go.mod"}}))
	require.NoError(t, ws.Detect())

	require.Len(t, ws.Projects, 1)
	project := ws.Projects[0]
	require.Equal(t, "root", project.Name)
	require.Equal(t, models.ProjectTypeGo, project.Type)
	require.Equal(t, "/workspace/go.mod", project.ManifestPath)
}

func TestNormalizeGoEnv_StripsDevNullSentinel(t *testing.T) {
	goEnv := normalizeGoEnv("", "NUL")
	require.Empty(t, goEnv.GoMod)
}
