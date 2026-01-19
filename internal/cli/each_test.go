package cli

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/jakoblorz/go-changesets/internal/filesystem"
	"github.com/jakoblorz/go-changesets/internal/workspace"
	"github.com/stretchr/testify/require"
)

const testWorkspaceRoot = "/test-workspace"

func buildWorkspace(t *testing.T, setup func(*workspace.WorkspaceBuilder)) (*workspace.Workspace, *filesystem.MockFileSystem) {
	t.Helper()

	wb := workspace.NewWorkspaceBuilder(testWorkspaceRoot)
	if setup != nil {
		setup(wb)
	}

	fs := wb.Build()
	ws := workspace.New(fs)
	require.NoError(t, ws.Detect())

	return ws, fs
}

func TestEach_MissingTreeFile(t *testing.T) {
	_, fs := buildWorkspace(t, func(wb *workspace.WorkspaceBuilder) {
		wb.AddProject("auth", "auth", "github.com/example/auth")
		wb.AddProject("api", "api", "github.com/example/api")
	})

	var buf bytes.Buffer
	cmd := &EachCommand{
		fs:           fs,
		fromTreeFile: "/tmp/tree.json",
		command:      []string{"sh", "-c", "echo $PROJECT:$PROJECT_PATH; echo $CHANGELOG_PREVIEW"},
		stdoutWriter: &buf,
	}

	err := cmd.Run(nil, []string{"echo", "test"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to read tree file")

	snaps.MatchSnapshot(t, buf.String())
}

func TestEach_InvalidTreeFileJSON(t *testing.T) {
	_, fs := buildWorkspace(t, func(wb *workspace.WorkspaceBuilder) {
		wb.AddProject("auth", "auth", "github.com/example/auth")
		wb.AddProject("api", "api", "github.com/example/api")
	})

	fs.AddFile("/tmp/tree.json", []byte(`this is not json`))

	var buf bytes.Buffer
	cmd := &EachCommand{
		fs:           fs,
		fromTreeFile: "/tmp/tree.json",
		command:      []string{"echo", "test"},
		stdoutWriter: &buf,
	}

	err := cmd.Run(nil, []string{"echo", "test"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to parse tree JSON")

	snaps.MatchSnapshot(t, buf.String())
}

func TestEach_FromTreeFile(t *testing.T) {
	ws, fs := buildWorkspace(t, func(wb *workspace.WorkspaceBuilder) {
		wb.AddProject("auth", "auth", "github.com/example/auth")
		wb.AddProject("api", "api", "github.com/example/api")
	})

	fs.AddFile("/tmp/tree.json", []byte(fmt.Sprintf(`{
		"groups": [
			{
				"commit": "abc123def456",
				"projects": [
					{
				    	"name": "auth",
						"changesets": [
							{
								"id": "willing_bobcat_xFt2dlSu",
								"file": "%s/.changeset/willing_bobcat_xFt2dlSu.md",
								"bump": "patch",
								"message": "Improving the performance of the authentication module."
							}
						],
						"changelogPreview": "### Patch Changes\n\n- Improving the performance of the authentication module.\n\n"
					},
					{
				    	"name": "api",
						"changesets": [
							{
								"id": "willing_elephant_xFt2dlSu",
								"file": "%s/.changeset/willing_elephant_xFt2dlSu.md",
								"bump": "patch",
								"message": "Restructuring API endpoints for better clarity."
							}
						],
						"changelogPreview": "### Patch Changes\n\n- Restructuring API endpoints for better clarity.\n\n"
					}
				]
			}
		]
	}`, ws.RootPath, ws.RootPath)))

	var buf bytes.Buffer
	cmd := &EachCommand{
		fs:           fs,
		fromTreeFile: "/tmp/tree.json",
		command:      []string{"sh", "-c", "echo $PROJECT:$PROJECT_PATH; echo $CHANGELOG_PREVIEW"},
		stdoutWriter: &buf,
	}

	err := cmd.runFromTreeFile()
	require.NoError(t, err)

	snaps.MatchSnapshot(t, buf.String())
}

func TestEach_NoCommand(t *testing.T) {
	_, fs := buildWorkspace(t, func(wb *workspace.WorkspaceBuilder) {
		wb.AddProject("auth", "auth", "github.com/example/auth")
		wb.AddProject("api", "api", "github.com/example/api")
	})

	var buf bytes.Buffer
	cmd := &EachCommand{
		fs:           fs,
		command:      []string{},
		stdoutWriter: &buf,
	}

	err := cmd.Run(nil, []string{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "no command specified")

	snaps.MatchSnapshot(t, buf.String())
}
