package changelog

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/jakoblorz/go-changesets/internal/changeset"
	"github.com/jakoblorz/go-changesets/internal/filesystem"
	"github.com/jakoblorz/go-changesets/internal/models"
	"github.com/jakoblorz/go-changesets/internal/workspace"
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

func TestChangelog_FormatEntry_InProjectContext(t *testing.T) {
	// Setup mock workspace with multiple projects
	wb := workspace.NewWorkspaceBuilder("/test-workspace")
	wb.AddProject("auth", "packages/auth", "github.com/test/auth")
	wb.AddProject("api", "packages/api", "github.com/test/api")

	// Add changesets
	wb.AddChangeset("auth-minor", "auth", "minor", "Add OAuth2 authentication support")
	wb.AddChangeset("auth-patch", "auth", "patch", "Fix memory leak in session handler")
	wb.AddChangeset("api-minor", "api", "minor", "Add GraphQL endpoint")

	fs := wb.Build()

	// Detect workspace
	ws := workspace.New(fs)
	require.NoError(t, ws.Detect())

	csManager := changeset.NewManager(fs, ws.ChangesetDir())
	allChangesets, err := csManager.ReadAll()
	require.NoError(t, err)
	require.Len(t, allChangesets, 3)

	authChangesets := changeset.FilterByProject(allChangesets, "auth")
	cl := NewChangelog(fs)
	authProject, _ := ws.GetProject("auth")
	preview, err := cl.FormatEntry(authChangesets, "auth", authProject.RootPath)
	require.NoError(t, err)

	require.Contains(t, preview, "### Minor Changes")
	require.Contains(t, preview, "### Patch Changes")
	require.Contains(t, preview, "Add OAuth2 authentication support")
	require.Contains(t, preview, "Fix memory leak in session handler")
}

func TestChangelog_FormatEntry_MultilineMessages(t *testing.T) {
	// Setup workspace
	wb := workspace.NewWorkspaceBuilder("/test-workspace")
	wb.AddProject("auth", "packages/auth", "github.com/test/auth")

	// Add changeset with multiline message
	multilineMsg := `Add comprehensive OAuth2 support

This change includes:
- Google OAuth2 provider
- GitHub OAuth2 provider
- Token refresh mechanism`

	wb.AddChangeset("multiline", "auth", "minor", multilineMsg)
	fs := wb.Build()

	ws := workspace.New(fs)
	require.NoError(t, ws.Detect())

	csManager := changeset.NewManager(fs, ws.ChangesetDir())
	allChangesets, err := csManager.ReadAll()
	require.NoError(t, err)

	authChangesets := changeset.FilterByProject(allChangesets, "auth")
	cl := NewChangelog(fs)
	authProject, _ := ws.GetProject("auth")
	preview, err := cl.FormatEntry(authChangesets, "auth", authProject.RootPath)
	require.NoError(t, err)

	require.True(t, strings.Contains(preview, "Add comprehensive OAuth2 support"), "Expected first line in preview")
	require.True(t, strings.Contains(preview, "Google OAuth2 provider"), "Expected detail line in preview")
	lines := strings.Split(preview, "\n")
	foundIndented := false
	for _, line := range lines {
		if strings.HasPrefix(line, "  ") && strings.Contains(line, "Google") {
			foundIndented = true
			break
		}
	}
	require.True(t, foundIndented, "Expected indented lines for multiline message")
}

func TestChangelog_FormatEntry_ContextIntegration(t *testing.T) {
	// Test that ProjectContext would include changelog preview
	// This simulates what the 'each' command does
	wb := workspace.NewWorkspaceBuilder("/test-workspace")
	wb.AddProject("auth", "packages/auth", "github.com/test/auth")
	wb.SetVersion("auth", "0.1.0")
	wb.AddChangeset("test", "auth", "minor", "Add new feature")
	fs := wb.Build()

	ws := workspace.New(fs)
	require.NoError(t, ws.Detect())

	csManager := changeset.NewManager(fs, ws.ChangesetDir())
	allChangesets, err := csManager.ReadAll()
	require.NoError(t, err)

	// Build context for auth project (simulating what each command does)
	project := ws.Projects[0]
	projectChangesets := changeset.FilterByProject(allChangesets, project.Name)

	// Generate changelog preview
	var changelogPreview string
	if len(projectChangesets) > 0 {
		cl := NewChangelog(fs)
		changelogPreview, err = cl.FormatEntry(projectChangesets, project.Name, project.RootPath)
		require.NoError(t, err)
	}

	// Create context
	ctx := &models.ProjectContext{
		Project:          project.Name,
		ProjectPath:      project.RootPath,
		ModulePath:       project.ModulePath,
		HasChangesets:    len(projectChangesets) > 0,
		ChangelogPreview: changelogPreview,
	}

	// Verify context has changelog preview
	require.NotEmpty(t, ctx.ChangelogPreview, "Expected ChangelogPreview to be populated in context")

	require.Contains(t, ctx.ChangelogPreview, "Add new feature")

	// Verify it can be marshaled to JSON
	jsonData, err := json.Marshal(ctx)
	require.NoError(t, err)

	// Verify JSON contains changelogPreview field
	jsonStr := string(jsonData)
	require.Contains(t, jsonStr, "changelogPreview")

	// Unmarshal and verify
	var ctxCopy models.ProjectContext
	require.NoError(t, json.Unmarshal(jsonData, &ctxCopy))

	require.Equal(t, ctx.ChangelogPreview, ctxCopy.ChangelogPreview, "ChangelogPreview not preserved through JSON round-trip")
}

func TestChangelog_FormatEntry_OrderByBumpType(t *testing.T) {
	// Verify that changesets are ordered by bump type (Major, Minor, Patch)
	wb := workspace.NewWorkspaceBuilder("/test-workspace")
	wb.AddProject("auth", "packages/auth", "github.com/test/auth")

	// Add in random order
	wb.AddChangeset("patch1", "auth", "patch", "Fix bug 1")
	wb.AddChangeset("major1", "auth", "major", "Breaking change")
	wb.AddChangeset("minor1", "auth", "minor", "New feature")
	wb.AddChangeset("patch2", "auth", "patch", "Fix bug 2")

	fs := wb.Build()
	ws := workspace.New(fs)
	require.NoError(t, ws.Detect())

	csManager := changeset.NewManager(fs, ws.ChangesetDir())
	allChangesets, err := csManager.ReadAll()
	require.NoError(t, err)

	authChangesets := changeset.FilterByProject(allChangesets, "auth")
	cl := NewChangelog(fs)
	authProject, _ := ws.GetProject("auth")
	preview, err := cl.FormatEntry(authChangesets, "auth", authProject.RootPath)
	require.NoError(t, err)

	// Find positions of each section
	majorIdx := strings.Index(preview, "### Major Changes")
	minorIdx := strings.Index(preview, "### Minor Changes")
	patchIdx := strings.Index(preview, "### Patch Changes")

	require.NotEqual(t, -1, majorIdx, "Missing Major section in preview:\n%s", preview)
	require.NotEqual(t, -1, minorIdx, "Missing Minor section in preview:\n%s", preview)
	require.NotEqual(t, -1, patchIdx, "Missing Patch section in preview:\n%s", preview)

	// Verify order: Major before Minor before Patch
	require.Less(t, majorIdx, minorIdx, "Major Changes should come before Minor Changes")
	require.Less(t, minorIdx, patchIdx, "Minor Changes should come before Patch Changes")
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
		require.NoError(t, err, "FormatEntry failed: %v", err)
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
