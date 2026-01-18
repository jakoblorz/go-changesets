package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/jakoblorz/go-changesets/internal/filesystem"
	"github.com/jakoblorz/go-changesets/internal/github"
	"github.com/jakoblorz/go-changesets/internal/models"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

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
	fs := filesystem.NewMockFileSystem()
	ghClient := github.NewMockClient()

	ctx := &models.ProjectContext{
		Project:          "auth",
		ProjectPath:      "/workspace/auth",
		CurrentVersion:   "1.0.0",
		ChangelogPreview: "## Minor Changes\n\n- Add OAuth2 support",
	}

	fs.AddDir("/workspace")
	fs.AddDir("/workspace/auth")
	fs.AddFile("/workspace/auth/version.txt", []byte("1.0.0"))

	ghCmd := newGHOpenCommandWithParent(fs, ghClient, "testorg", "testrepo")
	ghCmd.SetArgs([]string{"pr", "open"})

	os.Setenv("PROJECT", ctx.Project)
	os.Setenv("PROJECT_PATH", ctx.ProjectPath)
	os.Setenv("CURRENT_VERSION", ctx.CurrentVersion)
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
	require.Contains(t, pr.Body, "<!-- RELATED_PRS_PLACEHOLDER -->")
	require.Equal(t, "changeset-release/auth", pr.Head)
	require.Equal(t, "main", pr.Base)
}

func TestGHOpen_UpdateExistingPR(t *testing.T) {
	fs := filesystem.NewMockFileSystem()
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

	ctx := &models.ProjectContext{
		Project:          "auth",
		ProjectPath:      "/workspace/auth",
		CurrentVersion:   "1.1.0",
		ChangelogPreview: "## Minor Changes\n\n- Add new feature",
	}

	fs.AddDir("/workspace")
	fs.AddDir("/workspace/auth")
	fs.AddFile("/workspace/auth/version.txt", []byte("1.1.0"))

	ghCmd := newGHOpenCommandWithParent(fs, ghClient, "testorg", "testrepo")
	ghCmd.SetArgs([]string{"pr", "open"})

	os.Setenv("PROJECT", ctx.Project)
	os.Setenv("PROJECT_PATH", ctx.ProjectPath)
	os.Setenv("CURRENT_VERSION", ctx.CurrentVersion)
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
	require.Equal(t, 42, pr.Number)
	require.Equal(t, "🚀 Release auth v1.1.0", pr.Title)
	require.Contains(t, pr.Body, "Add new feature")
}

func TestGHOpen_DefaultTemplate(t *testing.T) {
	fs := filesystem.NewMockFileSystem()
	ghClient := github.NewMockClient()

	ctx := &models.ProjectContext{
		Project:          "api",
		ProjectPath:      "/workspace/api",
		CurrentVersion:   "2.0.0",
		ChangelogPreview: "## Major Changes\n\n- Breaking API change",
	}

	fs.AddDir("/workspace")
	fs.AddDir("/workspace/api")
	fs.AddFile("/workspace/api/version.txt", []byte("2.0.0"))

	ghCmd := newGHOpenCommandWithParent(fs, ghClient, "testorg", "testrepo")
	ghCmd.SetArgs([]string{"pr", "open"})

	os.Setenv("PROJECT", ctx.Project)
	os.Setenv("PROJECT_PATH", ctx.ProjectPath)
	os.Setenv("CURRENT_VERSION", ctx.CurrentVersion)
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
	snaps.MatchSnapshot(t, pr.Title)
	snaps.MatchSnapshot(t, pr.Body)
}

func TestGHOpen_MappingFileUpdated(t *testing.T) {
	fs := filesystem.NewMockFileSystem()
	ghClient := github.NewMockClient()

	ctx := &models.ProjectContext{
		Project:          "auth",
		ProjectPath:      "/workspace/auth",
		CurrentVersion:   "1.0.0",
		ChangelogPreview: "- Test",
	}

	fs.AddDir("/workspace")
	fs.AddDir("/workspace/auth")
	fs.AddFile("/workspace/auth/version.txt", []byte("1.0.0"))

	mappingFile := filepath.Join(t.TempDir(), "pr-mapping.json")

	ghCmd := newGHOpenCommandWithParent(fs, ghClient, "testorg", "testrepo")
	ghCmd.SetArgs([]string{"pr", "open", "--mapping-file", mappingFile})

	os.Setenv("PROJECT", ctx.Project)
	os.Setenv("PROJECT_PATH", ctx.ProjectPath)
	os.Setenv("CURRENT_VERSION", ctx.CurrentVersion)
	os.Setenv("CHANGELOG_PREVIEW", ctx.ChangelogPreview)
	defer func() {
		os.Unsetenv("PROJECT")
		os.Unsetenv("PROJECT_PATH")
		os.Unsetenv("CURRENT_VERSION")
		os.Unsetenv("CHANGELOG_PREVIEW")
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
	fs := filesystem.NewMockFileSystem()
	ghClient := github.NewMockClient()

	ctx := &models.ProjectContext{
		Project:          "auth",
		ProjectPath:      "/workspace/auth",
		CurrentVersion:   "0.0.0",
		ChangelogPreview: "- Version bump",
	}

	fs.AddDir("/workspace")
	fs.AddDir("/workspace/auth")
	fs.AddFile("/workspace/auth/version.txt", []byte("3.2.1"))

	ghCmd := newGHOpenCommandWithParent(fs, ghClient, "testorg", "testrepo")
	ghCmd.SetArgs([]string{"pr", "open"})

	os.Setenv("PROJECT", ctx.Project)
	os.Setenv("PROJECT_PATH", ctx.ProjectPath)
	os.Setenv("CURRENT_VERSION", ctx.CurrentVersion)
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
	require.Equal(t, "🚀 Release auth v3.2.1", pr.Title)
}

func TestGHOpen_EnvVarContext(t *testing.T) {
	fs := filesystem.NewMockFileSystem()
	ghClient := github.NewMockClient()

	ctx := &models.ProjectContext{
		Project:          "api",
		ProjectPath:      "/workspace/api",
		CurrentVersion:   "1.0.0",
		ChangelogPreview: "- API change",
	}

	fs.AddDir("/workspace")
	fs.AddDir("/workspace/api")
	fs.AddFile("/workspace/api/version.txt", []byte("1.0.0"))

	ghCmd := newGHOpenCommandWithParent(fs, ghClient, "testorg", "testrepo")
	ghCmd.SetArgs([]string{"pr", "open"})

	os.Setenv("PROJECT", ctx.Project)
	os.Setenv("PROJECT_PATH", ctx.ProjectPath)
	os.Setenv("CURRENT_VERSION", ctx.CurrentVersion)
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
