package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/jakoblorz/go-changesets/internal/filesystem"
	"github.com/jakoblorz/go-changesets/internal/git"
	"github.com/jakoblorz/go-changesets/internal/github"
	"github.com/jakoblorz/go-changesets/internal/models"
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
	fs := filesystem.NewMockFileSystem()
	ghClient := github.NewMockClient()
	gitClient := git.NewMockGitClient()

	ctx := &models.ProjectContext{
		Project:          "auth",
		ProjectPath:      "/workspace/auth",
		CurrentVersion:   "0.1.0",
		ChangelogPreview: "## Minor Changes\n\n- Add OAuth2 support",
	}

	fs.AddDir("/workspace")
	fs.AddDir("/workspace/auth")
	fs.AddFile("/workspace/auth/version.txt", []byte("0.1.0"))
	fs.AddFile("/workspace/go.work", []byte("go 1.24\n\nuse (\n\t./auth\n)"))

	gitClient.AddTag("auth", "v0.1.0", "Release 0.1.0")

	gitClient.CreateCommit("Update version to v1.0.0")
	gitClient.CreateBranch("changeset-release/auth")

	versionFile := filepath.Join("/workspace/auth", "version.txt")
	fs.AddFile(versionFile, []byte("1.0.0"))

	ghCmd := newGHCommandWithParent(fs, ghClient, "testorg", "testrepo")
	ghCmd.SetArgs([]string{"pr", "open"})

	os.Setenv("PROJECT", ctx.Project)
	os.Setenv("PROJECT_PATH", ctx.ProjectPath)
	os.Setenv("CURRENT_VERSION", "1.0.0")
	os.Setenv("CHANGELOG_PREVIEW", ctx.ChangelogPreview)
	defer func() {
		os.Unsetenv("PROJECT")
		os.Unsetenv("PROJECT_PATH")
		os.Unsetenv("CURRENT_VERSION")
		os.Unsetenv("CHANGELOG_PREVIEW")
	}()

	err := ghCmd.Execute()
	require.NoError(t, err)

	prs := ghClient.GetAllPullRequests("testorg", "testrepo")
	require.Len(t, prs, 1)

	pr := prs[0]
	require.Equal(t, "🚀 Release auth v1.0.0", pr.Title)
	require.Contains(t, pr.Body, "Add OAuth2 support")
	require.Equal(t, "changeset-release/auth", pr.Head)
	require.Equal(t, "main", pr.Base)

	snaps.MatchSnapshot(t, pr.Body)
}

func TestGHWorkflow_MultiProjectCoordinatedRelease(t *testing.T) {
	fs := filesystem.NewMockFileSystem()
	ghClient := github.NewMockClient()

	fs.AddDir("/workspace")
	fs.AddDir("/workspace/auth")
	fs.AddDir("/workspace/api")
	fs.AddDir("/workspace/shared")
	fs.AddFile("/workspace/auth/version.txt", []byte("1.0.0"))
	fs.AddFile("/workspace/api/version.txt", []byte("1.0.0"))
	fs.AddFile("/workspace/shared/version.txt", []byte("1.0.0"))
	fs.AddFile("/workspace/go.work", []byte("go 1.24\n\nuse (\n\t./auth\n\t./api\n\t./shared\n)"))

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

	os.Setenv("PROJECT", "auth")
	os.Setenv("PROJECT_PATH", "/workspace/auth")
	os.Setenv("CURRENT_VERSION", "1.0.0")
	os.Setenv("CHANGELOG_PREVIEW", "## Minor Changes\n\n- Auth changes")
	defer func() {
		os.Unsetenv("PROJECT")
		os.Unsetenv("PROJECT_PATH")
		os.Unsetenv("CURRENT_VERSION")
		os.Unsetenv("CHANGELOG_PREVIEW")
	}()

	ghCmd1 := newGHCommandWithParent(fs, ghClient, "testorg", "testrepo")
	ghCmd1.SetArgs([]string{"pr", "open", "--mapping-file", mappingFile})
	err := ghCmd1.Execute()
	require.NoError(t, err)

	os.Setenv("PROJECT", "api")
	os.Setenv("PROJECT_PATH", "/workspace/api")
	os.Setenv("CURRENT_VERSION", "1.0.0")
	os.Setenv("CHANGELOG_PREVIEW", "## Minor Changes\n\n- API changes")

	ghCmd2 := newGHCommandWithParent(fs, ghClient, "testorg", "testrepo")
	ghCmd2.SetArgs([]string{"pr", "open", "--mapping-file", mappingFile})
	err = ghCmd2.Execute()
	require.NoError(t, err)

	os.Setenv("PROJECT", "shared")
	os.Setenv("PROJECT_PATH", "/workspace/shared")
	os.Setenv("CURRENT_VERSION", "1.0.0")
	os.Setenv("CHANGELOG_PREVIEW", "## Minor Changes\n\n- Shared changes")

	ghCmd3 := newGHCommandWithParent(fs, ghClient, "testorg", "testrepo")
	ghCmd3.SetArgs([]string{"pr", "open", "--mapping-file", mappingFile})
	err = ghCmd3.Execute()
	require.NoError(t, err)

	prs := ghClient.GetAllPullRequests("testorg", "testrepo")
	require.Len(t, prs, 3)

	os.WriteFile(treeFile, treeJSON, 0644)

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
	fs := filesystem.NewMockFileSystem()
	ghClient := github.NewMockClient()

	ctx := &models.ProjectContext{
		Project:          "auth",
		ProjectPath:      "/workspace/auth",
		CurrentVersion:   "1.0.0",
		ChangelogPreview: "",
	}

	fs.AddDir("/workspace")
	fs.AddDir("/workspace/auth")
	fs.AddFile("/workspace/auth/version.txt", []byte("1.0.0"))

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

	os.Setenv("PROJECT", ctx.Project)
	os.Setenv("PROJECT_PATH", ctx.ProjectPath)
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
	fs := filesystem.NewMockFileSystem()
	ghClient := github.NewMockClient()

	existingPR := &github.PullRequest{
		Number:  42,
		Title:   "🚀 Release auth v1.0.0",
		Body:    "Original changelog",
		Head:    "changeset-release/auth",
		Base:    "main",
		State:   "open",
		HTMLURL: "https://github.com/testorg/testrepo/pull/42",
	}
	ghClient.AddPullRequestByHead("testorg", "testrepo", "changeset-release/auth", existingPR)

	fs.AddDir("/workspace")
	fs.AddDir("/workspace/auth")
	fs.AddFile("/workspace/auth/version.txt", []byte("1.1.0"))

	ghCmd := newGHCommandWithParent(fs, ghClient, "testorg", "testrepo")
	ghCmd.SetArgs([]string{"pr", "open"})

	os.Setenv("PROJECT", "auth")
	os.Setenv("PROJECT_PATH", "/workspace/auth")
	os.Setenv("CURRENT_VERSION", "1.1.0")
	os.Setenv("CHANGELOG_PREVIEW", "## Minor Changes\n\n- Add new feature")
	defer func() {
		os.Unsetenv("PROJECT")
		os.Unsetenv("PROJECT_PATH")
		os.Unsetenv("CURRENT_VERSION")
		os.Unsetenv("CHANGELOG_PREVIEW")
	}()

	err := ghCmd.Execute()
	require.NoError(t, err)

	pr, err := ghClient.GetPullRequest(nilContext, "testorg", "testrepo", 42)
	require.NoError(t, err)
	require.Equal(t, "🚀 Release auth v1.1.0", pr.Title)
	require.Contains(t, pr.Body, "Add new feature")
	require.NotContains(t, pr.Body, "Original changelog")
}
