package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/jakoblorz/go-changesets/internal/filesystem"
	"github.com/jakoblorz/go-changesets/internal/github"
	"github.com/jakoblorz/go-changesets/internal/workspace"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

const testWorkspaceRootGH = "/test-workspace"

func buildWorkspaceForGH(t *testing.T, setup func(*workspace.WorkspaceBuilder)) (*workspace.Workspace, *filesystem.MockFileSystem) {
	t.Helper()

	wb := workspace.NewWorkspaceBuilder(testWorkspaceRootGH)
	if setup != nil {
		setup(wb)
	}

	fs := wb.Build()
	ws := workspace.New(fs)
	require.NoError(t, ws.Detect())

	return ws, fs
}

func newGHOpenCommandWithParent(fs filesystem.FileSystem, ghClient github.GitHubClient, owner, repo string) *cobra.Command {
	ghCmd := &cobra.Command{
		Use: "gh",
	}

	ghCmd.PersistentFlags().String("owner", owner, "GitHub repository owner (required)")
	ghCmd.PersistentFlags().String("repo", repo, "GitHub repository name (required)")

	prCmd := &cobra.Command{
		Use: "pr",
	}
	prCmd.AddCommand(NewGHOpenCommand(fs, ghClient))

	ghCmd.AddCommand(prCmd)

	return ghCmd
}

func TestGHOpen_CreateNewPR(t *testing.T) {
	ws, fs := buildWorkspaceForGH(t, func(wb *workspace.WorkspaceBuilder) {
		wb.AddProject("auth", "packages/auth", "github.com/test/auth")
		wb.SetVersion("auth", "1.0.0")
	})

	ghClient := github.NewMockClient()

	ghCmd := newGHOpenCommandWithParent(fs, ghClient, "testorg", "testrepo")
	ghCmd.SetArgs([]string{"pr", "open"})

	os.Setenv("PROJECT", "auth")
	os.Setenv("PROJECT_PATH", ws.Projects[0].RootPath)
	defer func() {
		os.Unsetenv("PROJECT")
		os.Unsetenv("PROJECT_PATH")
	}()

	err := ghCmd.Execute()
	require.NoError(t, err)

	prs := ghClient.GetAllPullRequests("testorg", "testrepo")
	require.Len(t, prs, 1)

	pr := prs[0]
	require.Equal(t, "🚀 Release auth v1.0.0", pr.Title)
	require.Equal(t, "changeset-release/auth", pr.Head)
	require.Equal(t, "main", pr.Base)
	snaps.MatchSnapshot(t, pr.Body)
}

func TestGHOpen_UpdateExistingPR(t *testing.T) {
	ws, fs := buildWorkspaceForGH(t, func(wb *workspace.WorkspaceBuilder) {
		wb.AddProject("auth", "packages/auth", "github.com/test/auth")
		wb.SetVersion("auth", "1.1.0")
	})

	ghClient := github.NewMockClient()

	existingPR := &github.PullRequest{
		Number:  42,
		Title:   "🚀 Release auth v1.0.0",
		Body:    "Old body content",
		Head:    "changeset-release/auth",
		Base:    "main",
		State:   "open",
		HTMLURL: "https://github.com/testorg/testrepo/pull/42",
	}
	ghClient.AddPullRequestByHead("testorg", "testrepo", "changeset-release/auth", existingPR)

	ghCmd := newGHOpenCommandWithParent(fs, ghClient, "testorg", "testrepo")
	ghCmd.SetArgs([]string{"pr", "open"})

	os.Setenv("PROJECT", "auth")
	os.Setenv("PROJECT_PATH", ws.Projects[0].RootPath)
	defer func() {
		os.Unsetenv("PROJECT")
		os.Unsetenv("PROJECT_PATH")
	}()

	err := ghCmd.Execute()
	require.NoError(t, err)

	prs := ghClient.GetAllPullRequests("testorg", "testrepo")
	require.Len(t, prs, 1)

	pr := prs[0]
	require.Equal(t, 42, pr.Number)
	require.Equal(t, "🚀 Release auth v1.1.0", pr.Title)
}

func TestGHOpen_DefaultTemplate(t *testing.T) {
	ws, fs := buildWorkspaceForGH(t, func(wb *workspace.WorkspaceBuilder) {
		wb.AddProject("api", "packages/api", "github.com/test/api")
		wb.SetVersion("api", "2.0.0")
	})

	ghClient := github.NewMockClient()

	ghCmd := newGHOpenCommandWithParent(fs, ghClient, "testorg", "testrepo")
	ghCmd.SetArgs([]string{"pr", "open"})

	os.Setenv("PROJECT", "api")
	os.Setenv("PROJECT_PATH", ws.Projects[0].RootPath)
	defer func() {
		os.Unsetenv("PROJECT")
		os.Unsetenv("PROJECT_PATH")
	}()

	err := ghCmd.Execute()
	require.NoError(t, err)

	prs := ghClient.GetAllPullRequests("testorg", "testrepo")
	require.Len(t, prs, 1)

	pr := prs[0]
	snaps.MatchSnapshot(t, pr.Title)
	snaps.MatchSnapshot(t, pr.Body)
}

func TestGHOpen_MappingFileUpdated(t *testing.T) {
	ws, fs := buildWorkspaceForGH(t, func(wb *workspace.WorkspaceBuilder) {
		wb.AddProject("auth", "packages/auth", "github.com/test/auth")
		wb.SetVersion("auth", "1.0.0")
	})

	ghClient := github.NewMockClient()

	mappingFile := filepath.Join(t.TempDir(), "pr-mapping.json")

	ghCmd := newGHOpenCommandWithParent(fs, ghClient, "testorg", "testrepo")
	ghCmd.SetArgs([]string{"pr", "open", "--mapping-file", mappingFile})

	os.Setenv("PROJECT", "auth")
	os.Setenv("PROJECT_PATH", ws.Projects[0].RootPath)
	defer func() {
		os.Unsetenv("PROJECT")
		os.Unsetenv("PROJECT_PATH")
	}()

	err := ghCmd.Execute()
	require.NoError(t, err)

	data, err := os.ReadFile(mappingFile)
	require.NoError(t, err)

	var mapping github.PRMapping
	err = json.Unmarshal(data, &mapping)
	require.NoError(t, err)

	entry, ok := mapping.Get("auth")
	require.True(t, ok)
	require.Equal(t, 1, entry.PRNumber)
	require.Equal(t, "changeset-release/auth", entry.Branch)
	require.Equal(t, "1.0.0", entry.Version)
}

func TestGHOpen_MissingOwnerOrRepo(t *testing.T) {
	fs := filesystem.NewMockFileSystem()
	ghClient := github.NewMockClient()

	ghCmd := newGHOpenCommandWithParent(fs, ghClient, "testorg", "")
	ghCmd.SetArgs([]string{"pr", "open"})

	err := ghCmd.Execute()
	require.Error(t, err)
	require.Contains(t, err.Error(), "--repo is required")

	ghCmd2 := newGHOpenCommandWithParent(fs, ghClient, "", "testrepo")
	ghCmd2.SetArgs([]string{"pr", "open"})

	err = ghCmd2.Execute()
	require.Error(t, err)
	require.Contains(t, err.Error(), "--owner is required")
}

func TestGHOpen_ReadVersionFromFile(t *testing.T) {
	ws, fs := buildWorkspaceForGH(t, func(wb *workspace.WorkspaceBuilder) {
		wb.AddProject("auth", "packages/auth", "github.com/test/auth")
		wb.SetVersion("auth", "3.2.1")
	})

	ghClient := github.NewMockClient()

	ghCmd := newGHOpenCommandWithParent(fs, ghClient, "testorg", "testrepo")
	ghCmd.SetArgs([]string{"pr", "open"})

	os.Setenv("PROJECT", "auth")
	os.Setenv("PROJECT_PATH", ws.Projects[0].RootPath)
	defer func() {
		os.Unsetenv("PROJECT")
		os.Unsetenv("PROJECT_PATH")
	}()

	err := ghCmd.Execute()
	require.NoError(t, err)

	prs := ghClient.GetAllPullRequests("testorg", "testrepo")
	require.Len(t, prs, 1)

	pr := prs[0]
	require.Equal(t, "🚀 Release auth v3.2.1", pr.Title)
}

func TestGHOpen_EnvVarContext(t *testing.T) {
	ws, fs := buildWorkspaceForGH(t, func(wb *workspace.WorkspaceBuilder) {
		wb.AddProject("api", "packages/api", "github.com/test/api")
		wb.SetVersion("api", "1.0.0")
	})

	ghClient := github.NewMockClient()

	ghCmd := newGHOpenCommandWithParent(fs, ghClient, "testorg", "testrepo")
	ghCmd.SetArgs([]string{"pr", "open"})

	os.Setenv("PROJECT", "api")
	os.Setenv("PROJECT_PATH", ws.Projects[0].RootPath)
	defer func() {
		os.Unsetenv("PROJECT")
		os.Unsetenv("PROJECT_PATH")
	}()

	err := ghCmd.Execute()
	require.NoError(t, err)

	prs := ghClient.GetAllPullRequests("testorg", "testrepo")
	require.Len(t, prs, 1)
	require.Equal(t, "changeset-release/api", prs[0].Head)
}

func TestGHOpen_CommandRegistration(t *testing.T) {
	fs := filesystem.NewMockFileSystem()
	var ghClient github.GitHubClient = github.NewMockClient()

	cobraCmd := NewGHOpenCommand(fs, ghClient)
	require.NotNil(t, cobraCmd)
	require.Equal(t, "open", cobraCmd.Name())

	flags := cobraCmd.Flags()
	require.NotNil(t, flags.Lookup("base"))
	require.NotNil(t, flags.Lookup("labels"))
	require.NotNil(t, flags.Lookup("mapping-file"))
	require.NotNil(t, flags.Lookup("title"))
}
