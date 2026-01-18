package cli

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/jakoblorz/go-changesets/internal/filesystem"
	"github.com/jakoblorz/go-changesets/internal/github"
	"github.com/jakoblorz/go-changesets/internal/workspace"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

var nilContext = context.Background()

func newGHCloseCommandWithParent(fs filesystem.FileSystem, ghClient github.GitHubClient, owner, repo string) *cobra.Command {
	ghCmd := &cobra.Command{
		Use: "gh",
	}

	ghCmd.PersistentFlags().String("owner", owner, "GitHub repository owner (required)")
	ghCmd.PersistentFlags().String("repo", repo, "GitHub repository name (required)")

	prCmd := &cobra.Command{
		Use: "pr",
	}
	prCmd.AddCommand(NewGHCloseCommand(fs, ghClient))

	ghCmd.AddCommand(prCmd)

	return ghCmd
}

func TestGHClose_FindsAndClosesPR(t *testing.T) {
	ws, fs := buildWorkspaceForGH(t, func(wb *workspace.WorkspaceBuilder) {
		wb.AddProject("auth", "packages/auth", "github.com/test/auth")
		wb.SetVersion("auth", "1.0.0")
	})

	ghClient := github.NewMockClient()

	ghClient.AddBranch("testorg", "testrepo", "changeset-release/auth")
	ghClient.AddPullRequestByHead("testorg", "testrepo", "changeset-release/auth", &github.PullRequest{
		Number:  42,
		Title:   "🚀 Release auth v1.0.0",
		Body:    "Release body",
		Head:    "changeset-release/auth",
		Base:    "main",
		State:   "open",
		HTMLURL: "https://github.com/testorg/testrepo/pull/42",
	})

	os.Setenv("PROJECT", "auth")
	os.Setenv("PROJECT_PATH", ws.Projects[0].RootPath)
	defer func() {
		os.Unsetenv("PROJECT")
		os.Unsetenv("PROJECT_PATH")
	}()

	ghCmd := newGHCloseCommandWithParent(fs, ghClient, "testorg", "testrepo")
	ghCmd.SetArgs([]string{"pr", "close"})

	err := ghCmd.Execute()
	require.NoError(t, err)

	pr, err := ghClient.GetPullRequestByHead(nilContext, "testorg", "testrepo", "changeset-release/auth")
	require.NoError(t, err)
	require.NotNil(t, pr)
	require.Equal(t, "closed", pr.State)
}

func TestGHClose_NoPRExists(t *testing.T) {
	ws, fs := buildWorkspaceForGH(t, func(wb *workspace.WorkspaceBuilder) {
		wb.AddProject("auth", "packages/auth", "github.com/test/auth")
		wb.SetVersion("auth", "1.0.0")
	})

	ghClient := github.NewMockClient()

	os.Setenv("PROJECT", "auth")
	os.Setenv("PROJECT_PATH", ws.Projects[0].RootPath)
	defer func() {
		os.Unsetenv("PROJECT")
		os.Unsetenv("PROJECT_PATH")
	}()

	ghCmd := newGHCloseCommandWithParent(fs, ghClient, "testorg", "testrepo")
	ghCmd.SetArgs([]string{"pr", "close"})

	err := ghCmd.Execute()
	require.NoError(t, err)
}

func TestGHClose_AlreadyClosed(t *testing.T) {
	ws, fs := buildWorkspaceForGH(t, func(wb *workspace.WorkspaceBuilder) {
		wb.AddProject("auth", "packages/auth", "github.com/test/auth")
		wb.SetVersion("auth", "1.0.0")
	})

	ghClient := github.NewMockClient()

	ghClient.AddPullRequestByHead("testorg", "testrepo", "changeset-release/auth", &github.PullRequest{
		Number:  42,
		Title:   "🚀 Release auth v1.0.0",
		Body:    "Release body",
		Head:    "changeset-release/auth",
		Base:    "main",
		State:   "closed",
		HTMLURL: "https://github.com/testorg/testrepo/pull/42",
	})

	os.Setenv("PROJECT", "auth")
	os.Setenv("PROJECT_PATH", ws.Projects[0].RootPath)
	defer func() {
		os.Unsetenv("PROJECT")
		os.Unsetenv("PROJECT_PATH")
	}()

	ghCmd := newGHCloseCommandWithParent(fs, ghClient, "testorg", "testrepo")
	ghCmd.SetArgs([]string{"pr", "close"})

	err := ghCmd.Execute()
	require.NoError(t, err)
}

func TestGHClose_CustomComment(t *testing.T) {
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

	os.Setenv("PROJECT", "auth")
	os.Setenv("PROJECT_PATH", ws.Projects[0].RootPath)
	defer func() {
		os.Unsetenv("PROJECT")
		os.Unsetenv("PROJECT_PATH")
	}()

	ghCmd := newGHCloseCommandWithParent(fs, ghClient, "testorg", "testrepo")
	ghCmd.SetArgs([]string{"pr", "close", "--comment", "Custom close message"})

	err := ghCmd.Execute()
	require.NoError(t, err)
}

func TestGHClose_ExtractsProjectName(t *testing.T) {
	ws, fs := buildWorkspaceForGH(t, func(wb *workspace.WorkspaceBuilder) {
		wb.AddProject("my-service", "packages/my-service", "github.com/test/my-service")
		wb.SetVersion("my-service", "1.0.0")
	})

	ghClient := github.NewMockClient()

	existingPR := &github.PullRequest{
		Number:  42,
		Title:   "🚀 Release my-service v1.0.0",
		Body:    "Release body",
		Head:    "changeset-release/my-service",
		Base:    "main",
		State:   "open",
		HTMLURL: "https://github.com/testorg/testrepo/pull/42",
	}
	ghClient.AddPullRequestByHead("testorg", "testrepo", "changeset-release/my-service", existingPR)
	ghClient.AddBranch("testorg", "testrepo", "changeset-release/my-service")

	os.Setenv("PROJECT_PATH", ws.Projects[0].RootPath)
	defer func() {
		os.Unsetenv("PROJECT_PATH")
	}()

	ghCmd := newGHCloseCommandWithParent(fs, ghClient, "testorg", "testrepo")
	ghCmd.SetArgs([]string{"pr", "close"})

	err := ghCmd.Execute()
	require.NoError(t, err)

	pr, err := ghClient.GetPullRequestByHead(nilContext, "testorg", "testrepo", "changeset-release/my-service")
	require.NoError(t, err)
	require.NotNil(t, pr)
	require.Equal(t, "closed", pr.State)
}

func TestGHClose_MissingProjectContext(t *testing.T) {
	fs := filesystem.NewMockFileSystem()
	ghClient := github.NewMockClient()

	ghCmd := newGHCloseCommandWithParent(fs, ghClient, "testorg", "testrepo")
	ghCmd.SetArgs([]string{"pr", "close"})

	err := ghCmd.Execute()
	require.Error(t, err)
	require.Contains(t, err.Error(), "no project context available")
}

func TestGHClose_BranchDeletionFails(t *testing.T) {
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

	ghClient.DeleteBranchError = errors.New("branch not found")

	os.Setenv("PROJECT", "auth")
	os.Setenv("PROJECT_PATH", ws.Projects[0].RootPath)
	defer func() {
		os.Unsetenv("PROJECT")
		os.Unsetenv("PROJECT_PATH")
	}()

	ghCmd := newGHCloseCommandWithParent(fs, ghClient, "testorg", "testrepo")
	ghCmd.SetArgs([]string{"pr", "close"})

	err := ghCmd.Execute()
	require.NoError(t, err)

	pr, err := ghClient.GetPullRequestByHead(nilContext, "testorg", "testrepo", "changeset-release/auth")
	require.NoError(t, err)
	require.NotNil(t, pr)
	require.Equal(t, "closed", pr.State)
}

func TestGHClose_CommandRegistration(t *testing.T) {
	fs := filesystem.NewMockFileSystem()
	var ghClient github.GitHubClient = github.NewMockClient()

	cobraCmd := NewGHCloseCommand(fs, ghClient)
	require.NotNil(t, cobraCmd)
	require.Equal(t, "close", cobraCmd.Name())

	flags := cobraCmd.Flags()
	require.NotNil(t, flags.Lookup("comment"))
}
