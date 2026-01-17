package changelog

import (
	"path/filepath"
	"testing"
	"text/template"
	"time"

	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/jakoblorz/go-changesets/internal/filesystem"
	"github.com/jakoblorz/go-changesets/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChangelog_FormatEntry(t *testing.T) {
	fs := filesystem.NewMockFileSystem()
	cl := NewChangelog(fs)

	tests := []struct {
		name        string
		changesets  []*models.Changeset
		projectName string
		want        string
	}{
		{
			name:        "no changesets",
			changesets:  []*models.Changeset{},
			projectName: "auth",
		},
		{
			name: "single patch changeset",
			changesets: []*models.Changeset{
				{
					ID:      "abc123",
					Message: "Fix memory leak",
					Projects: map[string]models.BumpType{
						"auth": models.BumpPatch,
					},
				},
			},
			projectName: "auth",
		},
		{
			name: "single minor changeset",
			changesets: []*models.Changeset{
				{
					ID:      "def456",
					Message: "Add OAuth2 support",
					Projects: map[string]models.BumpType{
						"auth": models.BumpMinor,
					},
				},
			},
			projectName: "auth",
		},
		{
			name: "single major changeset",
			changesets: []*models.Changeset{
				{
					ID:      "ghi789",
					Message: "Breaking API change",
					Projects: map[string]models.BumpType{
						"auth": models.BumpMajor,
					},
				},
			},
			projectName: "auth",
		},
		{
			name: "multiple changesets grouped by type",
			changesets: []*models.Changeset{
				{
					ID:      "patch1",
					Message: "Fix bug 1",
					Projects: map[string]models.BumpType{
						"auth": models.BumpPatch,
					},
				},
				{
					ID:      "minor1",
					Message: "Add feature 1",
					Projects: map[string]models.BumpType{
						"auth": models.BumpMinor,
					},
				},
				{
					ID:      "patch2",
					Message: "Fix bug 2",
					Projects: map[string]models.BumpType{
						"auth": models.BumpPatch,
					},
				},
				{
					ID:      "minor2",
					Message: "Add feature 2",
					Projects: map[string]models.BumpType{
						"auth": models.BumpMinor,
					},
				},
			},
			projectName: "auth",
		},
		{
			name: "filters by project name",
			changesets: []*models.Changeset{
				{
					ID:      "auth-change",
					Message: "Auth change",
					Projects: map[string]models.BumpType{
						"auth": models.BumpMinor,
					},
				},
				{
					ID:      "api-change",
					Message: "API change",
					Projects: map[string]models.BumpType{
						"api": models.BumpMinor,
					},
				},
			},
			projectName: "auth",
		},
		{
			name: "empty project name includes all",
			changesets: []*models.Changeset{
				{
					ID:      "change1",
					Message: "Change 1",
					Projects: map[string]models.BumpType{
						"auth": models.BumpMinor,
					},
				},
				{
					ID:      "change2",
					Message: "Change 2",
					Projects: map[string]models.BumpType{
						"api": models.BumpPatch,
					},
				},
			},
			projectName: "",
		},
		{
			name: "multiline message",
			changesets: []*models.Changeset{
				{
					ID:      "multiline",
					Message: "Add new feature\n\nThis is a longer description\n with multiple lines",
					Projects: map[string]models.BumpType{
						"auth": models.BumpMinor,
					},
				},
			},
			projectName: "auth",
		},
		{
			name: "multiline message with additional bump types",
			changesets: []*models.Changeset{
				{
					ID:      "multiline",
					Message: "Add new feature\n\nThis is a longer description\n with multiple lines",
					Projects: map[string]models.BumpType{
						"auth": models.BumpMinor,
					},
				},
				{
					ID:      "patch-fix",
					Message: "Fix minor bug",
					Projects: map[string]models.BumpType{
						"auth": models.BumpPatch,
					},
				},
				{
					ID:      "patch-fix-2",
					Message: "Fix another minor bug",
					Projects: map[string]models.BumpType{
						"auth": models.BumpPatch,
					},
				},
			},
			projectName: "auth",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := cl.FormatEntry(tt.changesets, tt.projectName, "/workspace")
			assert.NoError(t, err, "FormatEntry() should not return an error")
			snaps.MatchSnapshot(t, got)
		})
	}
}

func TestChangelog_FormatEntry_AllBumpTypes(t *testing.T) {
	fs := filesystem.NewMockFileSystem()
	cl := NewChangelog(fs)

	changesets := []*models.Changeset{
		{
			ID:      "major1",
			Message: "Breaking change",
			Projects: map[string]models.BumpType{
				"auth": models.BumpMajor,
			},
		},
		{
			ID:      "minor1",
			Message: "New feature",
			Projects: map[string]models.BumpType{
				"auth": models.BumpMinor,
			},
		},
		{
			ID:      "patch1",
			Message: "Bug fix",
			Projects: map[string]models.BumpType{
				"auth": models.BumpPatch,
			},
		},
	}

	result, err := cl.FormatEntry(changesets, "auth", "/workspace")
	require.NoError(t, err, "FormatEntry() should not return an error")
	snaps.MatchSnapshot(t, result)
}

func TestChangelog_FormatEntry_RawItems(t *testing.T) {
	t.Cleanup(func() {
		resetChangelogTemplateCache()
	})

	fs := filesystem.NewMockFileSystem()

	templateContent := "{{- range .Items}}\n- {{.FirstLine}}{{if .PR}} ([#{{.PR.Number}}]({{.PR.URL}}) by @{{.PR.Author}}){{end}}{{end}}"
	fs.AddFile("/workspace/.changeset/changelog.tmpl", []byte(templateContent))

	cl := NewChangelog(fs)
	changesets := []*models.Changeset{
		{
			ID:      "major1",
			Message: "Breaking change",
			Projects: map[string]models.BumpType{
				"auth": models.BumpMajor,
			},
			PR: &models.PullRequest{
				Number: 123,
				URL:    "https://github.com/org/repo/pull/123",
				Author: "alice",
			},
		},
		{
			ID:      "minor1",
			Message: "New feature",
			Projects: map[string]models.BumpType{
				"auth": models.BumpMinor,
			},
			PR: &models.PullRequest{
				Number: 123,
				URL:    "https://github.com/org/repo/pull/123",
				Author: "alice",
			},
		},
		{
			ID:      "patch1",
			Message: "Bug fix",
			Projects: map[string]models.BumpType{
				"auth": models.BumpPatch,
			},
			PR: &models.PullRequest{
				Number: 123,
				URL:    "https://github.com/org/repo/pull/123",
				Author: "alice",
			},
		},
	}

	result, err := cl.FormatEntry(changesets, "auth", "/workspace")
	require.NoError(t, err, "FormatEntry() should not return an error")
	snaps.MatchSnapshot(t, result)
}

func TestChangelog_CustomTemplateOverride(t *testing.T) {
	t.Cleanup(func() {
		resetChangelogTemplateCache()
	})

	fs := filesystem.NewMockFileSystem()
	templateContent := "Project: {{.Project}} Version: {{.Version}} Sections: {{len .Sections}}"
	fs.AddFile("/workspace/.changeset/changelog.tmpl", []byte(templateContent))

	cl := NewChangelog(fs)
	entry := &Entry{
		Version:    &models.Version{Major: 1, Minor: 2, Patch: 3},
		Date:       time.Date(2024, 12, 6, 0, 0, 0, 0, time.UTC),
		Changesets: []*models.Changeset{},
	}

	output, err := cl.formatEntry(entry, "auth", "/workspace")
	require.NoError(t, err, "formatEntry() should not return an error")
	snaps.MatchSnapshot(t, output)
}

func TestChangelog_Format_Append_withPRDetails(t *testing.T) {
	changesets := []*models.Changeset{
		{
			ID:      "minor-with-pr",
			Message: "Add OAuth2 support\n\nIncludes Google + GitHub providers",
			Projects: map[string]models.BumpType{
				"auth": models.BumpMinor,
			},
			PR: &models.PullRequest{
				Number: 123,
				URL:    "https://github.com/org/repo/pull/123",
				Author: "alice",
			},
		},
		{
			ID:      "patch",
			Message: "Fix memory leak",
			Projects: map[string]models.BumpType{
				"auth": models.BumpPatch,
			},
		},
		{
			ID:      "major",
			Message: "Breaking API change",
			Projects: map[string]models.BumpType{
				"auth": models.BumpMajor,
			},
		},
	}

	t.Run("FormatEntry", func(t *testing.T) {
		fs := filesystem.NewMockFileSystem()
		changelog := NewChangelog(fs)

		preview, err := changelog.FormatEntry(changesets, "auth", "/workspace")
		if err != nil {
			t.Fatalf("FormatEntry failed: %v", err)
		}
		snaps.MatchSnapshot(t, preview)
	})

	t.Run("Append", func(t *testing.T) {
		fs := filesystem.NewMockFileSystem()
		changelog := NewChangelog(fs)

		rootEntry := &Entry{
			Version:    &models.Version{Major: 2, Minor: 0, Patch: 0},
			Date:       time.Date(2024, 12, 6, 0, 0, 0, 0, time.UTC),
			Changesets: changesets,
		}
		err := changelog.Append(".", "", rootEntry)
		assert.NoError(t, err, "Append should not return an error")

		content, err := fs.ReadFile("./CHANGELOG.md")
		assert.NoError(t, err, "ReadFile should not return an error")

		snaps.MatchSnapshot(t, string(content))
	})

	t.Run("AppendWithProject", func(t *testing.T) {
		fs := filesystem.NewMockFileSystem()
		changelog := NewChangelog(fs)

		rootEntry := &Entry{
			Version:    &models.Version{Major: 2, Minor: 0, Patch: 0},
			Date:       time.Date(2024, 12, 6, 0, 0, 0, 0, time.UTC),
			Changesets: changesets,
		}
		err := changelog.Append(".", "auth", rootEntry)
		assert.NoError(t, err, "Append should not return an error")

		content, err := fs.ReadFile("./CHANGELOG.md")
		assert.NoError(t, err, "ReadFile should not return an error")

		snaps.MatchSnapshot(t, string(content))
	})

}

func TestChangelog_Append(t *testing.T) {
	runTestCase := func(t *testing.T, projectName string) {
		fs := filesystem.NewMockFileSystem()
		cl := NewChangelog(fs)

		projectRoot := "/test/project"
		changelogPath := filepath.Join(projectRoot, "CHANGELOG.md")

		fs.AddDir(projectRoot)

		renderChangelogEntry := func(entry *Entry) string {
			err := cl.Append(projectRoot, projectName, entry)
			require.NoError(t, err, "append should not error")

			data, err := fs.ReadFile(changelogPath)
			require.NoError(t, err, "read should not error")

			return string(data)
		}

		// First append - should add header
		snaps.MatchSnapshot(t, renderChangelogEntry(&Entry{
			Version: &models.Version{Major: 1, Minor: 0, Patch: 0},
			Date:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			Changesets: []*models.Changeset{
				{
					ID:      "first",
					Message: "First change",
					Projects: map[string]models.BumpType{
						"test": models.BumpMinor,
					},
				},
			},
		}))

		// Second append - should preserve header
		snaps.MatchSnapshot(t, renderChangelogEntry(&Entry{
			Version: &models.Version{Major: 1, Minor: 1, Patch: 0},
			Date:    time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
			Changesets: []*models.Changeset{
				{
					ID:      "second",
					Message: "Second change",
					Projects: map[string]models.BumpType{
						"test": models.BumpMinor,
					},
				},
			},
		}))

		// Third append - should contain project name in version header
		snaps.MatchSnapshot(t, renderChangelogEntry(&Entry{
			Version: &models.Version{Major: 1, Minor: 1, Patch: 1},
			Date:    time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
			Changesets: []*models.Changeset{
				{
					ID:      "third",
					Message: "Third change",
					Projects: map[string]models.BumpType{
						"test": models.BumpMinor,
					},
				},
			},
		}))
	}

	t.Run("should continously insert entries below the header", func(t *testing.T) {
		runTestCase(t, "")
	})

	t.Run("should include project name in version headers", func(t *testing.T) {
		runTestCase(t, "test")
	})

}

func resetChangelogTemplateCache() {
	templateCacheLock.Lock()
	defer templateCacheLock.Unlock()
	templateCache = make(map[string]*template.Template)
}
