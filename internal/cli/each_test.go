package cli

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/jakoblorz/go-changesets/internal/filesystem"
	"github.com/jakoblorz/go-changesets/internal/git"
	"github.com/jakoblorz/go-changesets/internal/models"
	"github.com/stretchr/testify/require"
)

func TestEach_FromTreeFile(t *testing.T) {
	fs := filesystem.NewMockFileSystem()

	tmpDir := t.TempDir()
	treeFile := filepath.Join(tmpDir, "tree.json")

	treeData := TreeOutput{
		Groups: []ChangesetGroup{
			{
				Commit: "abc123def456",
				Projects: []ProjectChangesetsInfo{
					{Name: "auth"},
					{Name: "api"},
				},
			},
		},
	}

	data, _ := json.MarshalIndent(treeData, "", "  ")
	fs.AddFile(treeFile, data)

	fs.AddDir(filepath.Join(tmpDir, "project1"))
	fs.AddDir(filepath.Join(tmpDir, "project2"))

	project1Path := filepath.Join(tmpDir, "project1")
	project2Path := filepath.Join(tmpDir, "project2")

	project1Version := filepath.Join(project1Path, "version.txt")
	project2Version := filepath.Join(project2Path, "version.txt")

	fs.AddFile(project1Version, []byte("1.0.0"))
	fs.AddFile(project2Version, []byte("2.0.0"))

	var buf bytes.Buffer
	cmd := &EachCommand{
		fs:           fs,
		fromTreeFile: treeFile,
		command:      []string{"echo", "$PROJECT:$PROJECT_PATH"},
		outputWriter: &buf,
	}

	err := cmd.runFromTreeFile(nil)
	require.NoError(t, err)

	snaps.MatchSnapshot(t, buf.String())
}

func TestEach_MultipleProjects_Snapshot(t *testing.T) {
	fs := filesystem.NewMockFileSystem()

	tmpDir := t.TempDir()
	treeFile := filepath.Join(tmpDir, "tree.json")

	treeData := TreeOutput{
		Groups: []ChangesetGroup{
			{
				Commit: "def789abc123",
				Projects: []ProjectChangesetsInfo{
					{Name: "auth"},
					{Name: "api"},
					{Name: "shared"},
				},
			},
		},
	}

	data, _ := json.MarshalIndent(treeData, "", "  ")
	fs.AddFile(treeFile, data)

	for _, name := range []string{"auth", "api", "shared"} {
		fs.AddDir(filepath.Join(tmpDir, name))
		fs.AddFile(filepath.Join(tmpDir, name, "version.txt"), []byte("1.0.0"))
	}

	var buf bytes.Buffer
	cmd := &EachCommand{
		fs:           fs,
		fromTreeFile: treeFile,
		command:      []string{"echo", "project: $PROJECT"},
		outputWriter: &buf,
	}

	err := cmd.runFromTreeFile(nil)
	require.NoError(t, err)

	snaps.MatchSnapshot(t, buf.String())
}

func TestEach_ExecuteForProject_EnvVars(t *testing.T) {
	fs := filesystem.NewMockFileSystem()

	ctx := &models.ProjectContext{
		Project:          "test-project",
		ProjectPath:      "/workspace/test-project",
		CurrentVersion:   "1.0.0",
		ChangelogPreview: "## Test changes",
	}

	fs.AddDir("/workspace")
	fs.AddDir("/workspace/test-project")
	fs.AddFile("/workspace/test-project/version.txt", []byte("1.0.0"))

	var buf bytes.Buffer
	cmd := &EachCommand{
		fs:           fs,
		command:      []string{"sh", "-c", "env | grep -E '^(PROJECT|PROJECT_PATH|CURRENT_VERSION|LATEST_TAG|CHANGELOG_PREVIEW|CHANGESET_CONTEXT|MODULE_PATH)='"},
		outputWriter: &buf,
	}

	err := cmd.executeForProject(ctx)
	require.NoError(t, err)

	snaps.MatchSnapshot(t, buf.String())
}

func TestEach_NoCommand(t *testing.T) {
	fs := filesystem.NewMockFileSystem()

	var buf bytes.Buffer
	cmd := &EachCommand{
		fs:           fs,
		command:      []string{},
		outputWriter: &buf,
	}

	err := cmd.Run(nil, []string{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "no command specified")

	snaps.MatchSnapshot(t, buf.String())
}

func TestEach_InvalidTreeFile(t *testing.T) {
	fs := filesystem.NewMockFileSystem()

	var buf bytes.Buffer
	cmd := &EachCommand{
		fs:           fs,
		fromTreeFile: "/nonexistent/tree.json",
		command:      []string{"echo", "test"},
		outputWriter: &buf,
	}

	err := cmd.Run(nil, []string{"echo", "test"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to read tree file")

	snaps.MatchSnapshot(t, buf.String())
}

func TestEach_InvalidTreeJSON(t *testing.T) {
	fs := filesystem.NewMockFileSystem()

	tmpDir := t.TempDir()
	treeFile := filepath.Join(tmpDir, "tree.json")

	fs.AddFile(treeFile, []byte("not valid json"))

	var buf bytes.Buffer
	cmd := &EachCommand{
		fs:           fs,
		fromTreeFile: treeFile,
		command:      []string{"echo", "test"},
		outputWriter: &buf,
	}

	err := cmd.Run(nil, []string{"echo", "test"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to parse tree JSON")

	snaps.MatchSnapshot(t, buf.String())
}

func TestEachCommand_CommandRegistration(t *testing.T) {
	fs := filesystem.NewMockFileSystem()
	var gitClient git.GitClient = git.NewMockGitClient()

	cobraCmd := NewEachCommand(fs, gitClient, nil)
	require.NotNil(t, cobraCmd)
	require.Equal(t, "each", cobraCmd.Name())

	flags := cobraCmd.Flags()
	require.NotNil(t, flags.Lookup("filter"))
	require.NotNil(t, flags.Lookup("from-tree-file"))
}
