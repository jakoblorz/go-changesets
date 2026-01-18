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

func newGHCommandWithParent(fs filesystem.FileSystem, ghClient github.GitHubClient, owner, repo string) *cobra.Command {
	ghCmd := &cobra.Command{
		Use: "gh",
	}

	ghCmd.PersistentFlags().String("owner", owner, "GitHub repository owner (required)")
	ghCmd.PersistentFlags().String("repo", repo, "GitHub repository name (required)")

	prCmd := &cobra.Command{
		Use: "pr",
	}
	prCmd.AddCommand(NewGHOpenCommand(fs, ghClient))
	prCmd.AddCommand(NewGHLinkCommand(fs, ghClient))
	prCmd.AddCommand(NewGHCloseCommand(fs, ghClient))

	ghCmd.AddCommand(prCmd)

	return ghCmd
}

func TestGHWorkflow_FullReleaseCycle(t *testing.T) {
	ws, fs := buildWorkspaceForGH(t, func(wb *workspace.WorkspaceBuilder) {
		wb.AddProject("auth", "packages/auth", "github.com/test/auth")
		wb.SetVersion("auth", "1.0.0")
		wb.AddChangeset("changeset-1", "auth", "minor", "Add OAuth2 support")
	})

	ghClient := github.NewMockClient()

	ghCmd := newGHCommandWithParent(fs, ghClient, "testorg", "testrepo")
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

func TestGHWorkflow_MultiProjectCoordinatedRelease(t *testing.T) {
	_, fs := buildWorkspaceForGH(t, func(wb *workspace.WorkspaceBuilder) {
		wb.AddProject("auth", "packages/auth", "github.com/test/auth")
		wb.AddProject("api", "packages/api", "github.com/test/api")
		wb.AddProject("shared", "packages/shared", "github.com/test/shared")
		wb.SetVersion("auth", "1.0.0")
		wb.SetVersion("api", "1.0.0")
		wb.SetVersion("shared", "1.0.0")
	})

	ghClient := github.NewMockClient()

	treeFile := filepath.Join(t.TempDir(), "tree.json")
	mappingFile := filepath.Join(t.TempDir(), "pr-mapping.json")

	treeData := TreeOutput{
		Groups: []ChangesetGroup{
			{
				Commit: "abc123def456",
				Projects: []ProjectChangesetsInfo{
					{Name: "auth"},
					{Name: "api"},
					{Name: "shared"},
				},
			},
		},
	}
	treeJSON, _ := json.Marshal(treeData)

	os.WriteFile(treeFile, treeJSON, 0644)

	ghCmd1 := newGHCommandWithParent(fs, ghClient, "testorg", "testrepo")
	ghCmd1.SetArgs([]string{"pr", "open", "--mapping-file", mappingFile})
	os.Setenv("PROJECT", "auth")
	os.Setenv("PROJECT_PATH", "/test-workspace/packages/auth")
	err := ghCmd1.Execute()
	require.NoError(t, err)

	ghCmd2 := newGHCommandWithParent(fs, ghClient, "testorg", "testrepo")
	ghCmd2.SetArgs([]string{"pr", "open", "--mapping-file", mappingFile})
	os.Setenv("PROJECT", "api")
	os.Setenv("PROJECT_PATH", "/test-workspace/packages/api")
	err = ghCmd2.Execute()
	require.NoError(t, err)

	ghCmd3 := newGHCommandWithParent(fs, ghClient, "testorg", "testrepo")
	ghCmd3.SetArgs([]string{"pr", "open", "--mapping-file", mappingFile})
	os.Setenv("PROJECT", "shared")
	os.Setenv("PROJECT_PATH", "/test-workspace/packages/shared")
	err = ghCmd3.Execute()
	require.NoError(t, err)

	prs := ghClient.GetAllPullRequests("testorg", "testrepo")
	require.Len(t, prs, 3)

	ghCmdLink := newGHCommandWithParent(fs, ghClient, "testorg", "testrepo")
	ghCmdLink.SetArgs([]string{
		"pr", "link",
		"--tree-file", treeFile,
		"--mapping-file", mappingFile,
	})
	err = ghCmdLink.Execute()
	require.NoError(t, err)

	pr1, err := ghClient.GetPullRequest(nilContext, "testorg", "testrepo", 1)
	require.NoError(t, err)
	require.Contains(t, pr1.Body, "Related Release PRs")

	pr2, err := ghClient.GetPullRequest(nilContext, "testorg", "testrepo", 2)
	require.NoError(t, err)
	require.Contains(t, pr2.Body, "Related Release PRs")

	pr3, err := ghClient.GetPullRequest(nilContext, "testorg", "testrepo", 3)
	require.NoError(t, err)
	require.Contains(t, pr3.Body, "Related Release PRs")
}

func TestGHWorkflow_CloseObsoletePR(t *testing.T) {
	ws, fs := buildWorkspaceForGH(t, func(wb *workspace.WorkspaceBuilder) {
		wb.AddProject("auth", "packages/auth", "github.com/test/auth")
		wb.SetVersion("auth", "1.0.0")
	})

	ghClient := github.NewMockClient()

	existingPR := &github.PullRequest{
		Number:  42,
		Title:   "🚀 Release auth v1.0.0",
		Body:    "Release body",
		Head:    "changeset-release/auth",
		Base:    "main",
		State:   "open",
		HTMLURL: "https://github.com/testorg/testrepo/pull/42",
	}
	ghClient.AddPullRequestByHead("testorg", "testrepo", "changeset-release/auth", existingPR)
	ghClient.AddBranch("testorg", "testrepo", "changeset-release/auth")

	ghCmd := newGHCommandWithParent(fs, ghClient, "testorg", "testrepo")
	ghCmd.SetArgs([]string{"pr", "close"})

	os.Setenv("PROJECT", "auth")
	os.Setenv("PROJECT_PATH", ws.Projects[0].RootPath)
	defer func() {
		os.Unsetenv("PROJECT")
		os.Unsetenv("PROJECT_PATH")
	}()

	err := ghCmd.Execute()
	require.NoError(t, err)

	pr, err := ghClient.GetPullRequestByHead(nilContext, "testorg", "testrepo", "changeset-release/auth")
	require.NoError(t, err)
	require.NotNil(t, pr)
	require.Equal(t, "closed", pr.State)
}

func TestGHWorkflow_UpdateExistingReleasePR(t *testing.T) {
	ws, fs := buildWorkspaceForGH(t, func(wb *workspace.WorkspaceBuilder) {
		wb.AddProject("auth", "packages/auth", "github.com/test/auth")
		wb.SetVersion("auth", "1.0.0")
	})

	existingPR := &github.PullRequest{
		Number:  42,
		Title:   "🚀 Release auth v1.0.0",
		Body:    "Original changelog",
		Head:    "changeset-release/auth",
		Base:    "main",
		State:   "open",
		HTMLURL: "https://github.com/testorg/testrepo/pull/42",
	}
	ghClient := github.NewMockClient()
	ghClient.AddPullRequestByHead("testorg", "testrepo", "changeset-release/auth", existingPR)

	versionFile := filepath.Join("/test-workspace/packages/auth", "version.txt")
	fs.AddFile(versionFile, []byte("1.1.0\n"))

	ghCmd := newGHCommandWithParent(fs, ghClient, "testorg", "testrepo")
	ghCmd.SetArgs([]string{"pr", "open"})

	os.Setenv("PROJECT", "auth")
	os.Setenv("PROJECT_PATH", ws.Projects[0].RootPath)
	defer func() {
		os.Unsetenv("PROJECT")
		os.Unsetenv("PROJECT_PATH")
	}()

	err := ghCmd.Execute()
	require.NoError(t, err)

	pr, err := ghClient.GetPullRequest(nilContext, "testorg", "testrepo", 42)
	require.NoError(t, err)
	require.Equal(t, "🚀 Release auth v1.1.0", pr.Title)
	require.NotContains(t, pr.Body, "Original changelog")
}
