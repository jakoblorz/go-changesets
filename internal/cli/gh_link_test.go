package cli

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/jakoblorz/go-changesets/internal/filesystem"
	"github.com/jakoblorz/go-changesets/internal/github"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func newGHLinkCommandWithParent(fs filesystem.FileSystem, ghClient github.GitHubClient, owner, repo string) *cobra.Command {
	ghCmd := &cobra.Command{
		Use: "gh",
	}

	ghCmd.PersistentFlags().String("owner", owner, "GitHub repository owner (required)")
	ghCmd.PersistentFlags().String("repo", repo, "GitHub repository name (required)")

	prCmd := &cobra.Command{
		Use: "pr",
	}
	prCmd.AddCommand(NewGHLinkCommand(fs, ghClient))

	ghCmd.AddCommand(prCmd)

	return ghCmd
}

func TestGHLink_LinksRelatedPRs(t *testing.T) {
	fs := filesystem.NewMockFileSystem()
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

	mapping := github.NewPRMapping()
	mapping.Set("auth", github.PREntry{
		PRNumber:         1,
		Branch:           "changeset-release/auth",
		Version:          "1.0.0",
		ChangelogPreview: "## Minor Changes\n- Auth feature",
	})
	mapping.Set("api", github.PREntry{
		PRNumber:         2,
		Branch:           "changeset-release/api",
		Version:          "1.0.0",
		ChangelogPreview: "## Minor Changes\n- API feature",
	})
	mapping.Set("shared", github.PREntry{
		PRNumber:         3,
		Branch:           "changeset-release/shared",
		Version:          "1.0.0",
		ChangelogPreview: "## Minor Changes\n- Shared feature",
	})
	mappingJSON, _ := json.MarshalIndent(mapping, "", "  ")
	os.WriteFile(mappingFile, mappingJSON, 0644)

	ghClient.AddPullRequestByHead("testorg", "testrepo", "changeset-release/auth", &github.PullRequest{
		Number: 1,
		Title:  "🚀 Release auth v1.0.0",
		Body:   "This PR was automatically generated for **auth**.\n\n## 📋 Changes\n\n## Minor Changes\n- Auth feature\n\n## 📦 What happens when you merge?\n- Version bumped to **1.0.0**\n- Changelog updated\n- Changesets consumed\n- Publish workflow creates GitHub release: `auth@1.0.0`",
		Head:   "changeset-release/auth",
		Base:   "main",
		State:  "open",
	})
	ghClient.AddPullRequestByHead("testorg", "testrepo", "changeset-release/api", &github.PullRequest{
		Number: 2,
		Title:  "🚀 Release api v1.0.0",
		Body:   "This PR was automatically generated for **api**.\n\n## 📋 Changes\n\n## Minor Changes\n- API feature\n\n## 📦 What happens when you merge?\n- Version bumped to **1.0.0**\n- Changelog updated\n- Changesets consumed\n- Publish workflow creates GitHub release: `api@1.0.0`",
		Head:   "changeset-release/api",
		Base:   "main",
		State:  "open",
	})
	ghClient.AddPullRequestByHead("testorg", "testrepo", "changeset-release/shared", &github.PullRequest{
		Number: 3,
		Title:  "🚀 Release shared v1.0.0",
		Body:   "This PR was automatically generated for **shared**.\n\n## 📋 Changes\n\n## Minor Changes\n- Shared feature\n\n## 📦 What happens when you merge?\n- Version bumped to **1.0.0**\n- Changelog updated\n- Changesets consumed\n- Publish workflow creates GitHub release: `shared@1.0.0`",
		Head:   "changeset-release/shared",
		Base:   "main",
		State:  "open",
	})

	ghCmd := newGHLinkCommandWithParent(fs, ghClient, "testorg", "testrepo")
	ghCmd.SetArgs([]string{
		"pr", "link",
		"--tree-file", treeFile,
		"--mapping-file", mappingFile,
	})

	err := ghCmd.Execute()
	require.NoError(t, err)

	pr1, err := ghClient.GetPullRequest(nilContext, "testorg", "testrepo", 1)
	require.NoError(t, err)
	snaps.MatchSnapshot(t, pr1.Body)

	pr2, err := ghClient.GetPullRequest(nilContext, "testorg", "testrepo", 2)
	require.NoError(t, err)
	snaps.MatchSnapshot(t, pr2.Body)

	pr3, err := ghClient.GetPullRequest(nilContext, "testorg", "testrepo", 3)
	require.NoError(t, err)
	snaps.MatchSnapshot(t, pr3.Body)
}

func TestGHLink_SingleProjectGroup(t *testing.T) {
	fs := filesystem.NewMockFileSystem()
	ghClient := github.NewMockClient()

	treeFile := filepath.Join(t.TempDir(), "tree.json")
	mappingFile := filepath.Join(t.TempDir(), "pr-mapping.json")

	treeData := TreeOutput{
		Groups: []ChangesetGroup{
			{
				Commit: "abc123def456",
				Projects: []ProjectChangesetsInfo{
					{Name: "auth"},
				},
			},
		},
	}
	treeJSON, _ := json.Marshal(treeData)
	os.WriteFile(treeFile, treeJSON, 0644)

	mapping := github.NewPRMapping()
	mapping.Set("auth", github.PREntry{PRNumber: 1, Branch: "changeset-release/auth", Version: "1.0.0"})
	mappingJSON, _ := json.MarshalIndent(mapping, "", "  ")
	os.WriteFile(mappingFile, mappingJSON, 0644)

	ghClient.AddPullRequestByHead("testorg", "testrepo", "changeset-release/auth", &github.PullRequest{
		Number: 1,
		Title:  "🚀 Release auth v1.0.0",
		Body:   "Auth release",
		Head:   "changeset-release/auth",
		Base:   "main",
		State:  "open",
	})

	ghCmd := newGHLinkCommandWithParent(fs, ghClient, "testorg", "testrepo")
	ghCmd.SetArgs([]string{
		"pr", "link",
		"--tree-file", treeFile,
		"--mapping-file", mappingFile,
	})

	err := ghCmd.Execute()
	require.NoError(t, err)
}

func TestGHLink_MissingMappingEntry(t *testing.T) {
	fs := filesystem.NewMockFileSystem()
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
				},
			},
		},
	}
	treeJSON, _ := json.Marshal(treeData)
	os.WriteFile(treeFile, treeJSON, 0644)

	mapping := github.NewPRMapping()
	mapping.Set("auth", github.PREntry{PRNumber: 1, Branch: "changeset-release/auth", Version: "1.0.0"})
	mappingJSON, _ := json.MarshalIndent(mapping, "", "  ")
	os.WriteFile(mappingFile, mappingJSON, 0644)

	ghClient.AddPullRequestByHead("testorg", "testrepo", "changeset-release/auth", &github.PullRequest{
		Number: 1,
		Title:  "🚀 Release auth v1.0.0",
		Body:   "This PR was automatically generated for **auth**.\n\n## 📋 Changes\n\n## Minor Changes\n- Auth feature\n\n## 📦 What happens when you merge?\n- Version bumped to **1.0.0**\n- Changelog updated\n- Changesets consumed\n- Publish workflow creates GitHub release: `auth@1.0.0`",
		Head:   "changeset-release/auth",
		Base:   "main",
		State:  "open",
	})
	ghClient.AddPullRequestByHead("testorg", "testrepo", "changeset-release/api", &github.PullRequest{
		Number: 2,
		Title:  "🚀 Release api v1.0.0",
		Body:   "This PR was automatically generated for **api**.\n\n## 📋 Changes\n\n## Minor Changes\n- API feature\n\n## 📦 What happens when you merge?\n- Version bumped to **1.0.0**\n- Changelog updated\n- Changesets consumed\n- Publish workflow creates GitHub release: `api@1.0.0`",
		Head:   "changeset-release/api",
		Base:   "main",
		State:  "open",
	})

	ghCmd := newGHLinkCommandWithParent(fs, ghClient, "testorg", "testrepo")
	ghCmd.SetArgs([]string{
		"pr", "link",
		"--tree-file", treeFile,
		"--mapping-file", mappingFile,
	})

	err := ghCmd.Execute()
	require.NoError(t, err)
}

func TestGHLink_MissingTreeFile(t *testing.T) {
	fs := filesystem.NewMockFileSystem()
	ghClient := github.NewMockClient()

	mappingFile := filepath.Join(t.TempDir(), "pr-mapping.json")
	mapping := github.NewPRMapping()
	mappingJSON, _ := json.MarshalIndent(mapping, "", "  ")
	os.WriteFile(mappingFile, mappingJSON, 0644)

	ghCmd := newGHLinkCommandWithParent(fs, ghClient, "testorg", "testrepo")
	ghCmd.SetArgs([]string{
		"pr", "link",
		"--tree-file", "/nonexistent/tree.json",
		"--mapping-file", mappingFile,
	})

	err := ghCmd.Execute()
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to read tree file")
}

func TestGHLink_MissingMappingFile(t *testing.T) {
	fs := filesystem.NewMockFileSystem()
	ghClient := github.NewMockClient()

	treeFile := filepath.Join(t.TempDir(), "tree.json")
	mappingFile := filepath.Join(t.TempDir(), "pr-mapping.json")

	treeData := TreeOutput{
		Groups: []ChangesetGroup{
			{
				Commit: "abc123def456",
				Projects: []ProjectChangesetsInfo{
					{Name: "auth"},
				},
			},
		},
	}
	treeJSON, _ := json.Marshal(treeData)
	os.WriteFile(treeFile, treeJSON, 0644)

	mapping := github.NewPRMapping()
	mapping.Set("auth", github.PREntry{PRNumber: 1, Branch: "changeset-release/auth", Version: "1.0.0"})
	mappingJSON, _ := json.MarshalIndent(mapping, "", "  ")
	os.WriteFile(mappingFile, mappingJSON, 0644)

	os.Remove(mappingFile)

	ghCmd := newGHLinkCommandWithParent(fs, ghClient, "testorg", "testrepo")
	ghCmd.SetArgs([]string{
		"pr", "link",
		"--tree-file", treeFile,
		"--mapping-file", mappingFile,
	})

	err := ghCmd.Execute()
	require.NoError(t, err)
}

func TestGHLink_NoRelatedPRs(t *testing.T) {
	fs := filesystem.NewMockFileSystem()
	ghClient := github.NewMockClient()

	treeFile := filepath.Join(t.TempDir(), "tree.json")
	mappingFile := filepath.Join(t.TempDir(), "pr-mapping.json")

	treeData := TreeOutput{
		Groups: []ChangesetGroup{
			{
				Commit: "abc123def456",
				Projects: []ProjectChangesetsInfo{
					{Name: "auth"},
				},
			},
			{
				Commit: "def789abc123",
				Projects: []ProjectChangesetsInfo{
					{Name: "api"},
				},
			},
		},
	}
	treeJSON, _ := json.Marshal(treeData)
	os.WriteFile(treeFile, treeJSON, 0644)

	mapping := github.NewPRMapping()
	mapping.Set("auth", github.PREntry{PRNumber: 1, Branch: "changeset-release/auth", Version: "1.0.0"})
	mapping.Set("api", github.PREntry{PRNumber: 2, Branch: "changeset-release/api", Version: "1.0.0"})
	mappingJSON, _ := json.MarshalIndent(mapping, "", "  ")
	os.WriteFile(mappingFile, mappingJSON, 0644)

	ghCmd := newGHLinkCommandWithParent(fs, ghClient, "testorg", "testrepo")
	ghCmd.SetArgs([]string{
		"pr", "link",
		"--tree-file", treeFile,
		"--mapping-file", mappingFile,
	})

	err := ghCmd.Execute()
	require.NoError(t, err)
}

func TestGHLink_GetPRFails(t *testing.T) {
	fs := filesystem.NewMockFileSystem()
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
				},
			},
		},
	}
	treeJSON, _ := json.Marshal(treeData)
	os.WriteFile(treeFile, treeJSON, 0644)

	mapping := github.NewPRMapping()
	mapping.Set("auth", github.PREntry{PRNumber: 1, Branch: "changeset-release/auth", Version: "1.0.0"})
	mapping.Set("api", github.PREntry{PRNumber: 2, Branch: "changeset-release/api", Version: "1.0.0"})
	mappingJSON, _ := json.MarshalIndent(mapping, "", "  ")
	os.WriteFile(mappingFile, mappingJSON, 0644)

	ghClient.AddPullRequestByHead("testorg", "testrepo", "changeset-release/auth", &github.PullRequest{
		Number: 1,
		Title:  "🚀 Release auth v1.0.0",
		Body:   "This PR was automatically generated for **auth**.\n\n## 📋 Changes\n\n## Minor Changes\n- Auth feature\n\n## 📦 What happens when you merge?\n- Version bumped to **1.0.0**\n- Changelog updated\n- Changesets consumed\n- Publish workflow creates GitHub release: `auth@1.0.0`",
		Head:   "changeset-release/auth",
		Base:   "main",
		State:  "open",
	})

	ghClient.GetPullRequestError = errors.New("API error")

	ghCmd := newGHLinkCommandWithParent(fs, ghClient, "testorg", "testrepo")
	ghCmd.SetArgs([]string{
		"pr", "link",
		"--tree-file", treeFile,
		"--mapping-file", mappingFile,
	})

	err := ghCmd.Execute()
	require.NoError(t, err)
}

func TestGHLink_CommandRegistration(t *testing.T) {
	fs := filesystem.NewMockFileSystem()
	var ghClient github.GitHubClient = github.NewMockClient()

	cobraCmd := NewGHLinkCommand(fs, ghClient)
	require.NotNil(t, cobraCmd)
	require.Equal(t, "link", cobraCmd.Name())

	flags := cobraCmd.Flags()
	require.NotNil(t, flags.Lookup("tree-file"))
	require.NotNil(t, flags.Lookup("mapping-file"))
}
