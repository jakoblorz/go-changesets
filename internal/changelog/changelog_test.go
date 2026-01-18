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

func formatProjectPreview(t *testing.T, ws *workspace.Workspace, fs *filesystem.MockFileSystem, projectName string) (string, []*models.Changeset) {
	t.Helper()

	csManager := changeset.NewManager(fs, ws.ChangesetDir())
	projectChangesets, err := csManager.ReadAllOfProject(projectName)
	require.NoError(t, err)

	cl := NewChangelog(fs)
	project, err := ws.GetProject(projectName)
	require.NoError(t, err)

	preview, err := cl.FormatEntry(projectChangesets, projectName, project.RootPath)
	require.NoError(t, err)

	return preview, projectChangesets
}

func TestChangelog_FormatEntry(t *testing.T) {
	fs := filesystem.NewMockFileSystem()
	cl := NewChangelog(fs)

	tests := []struct {
		name        string
		changesets  []*models.Changeset
		projectName string
	}{
		{
			name:        "no changesets",
			changesets:  []*models.Changeset{},
			projectName: "auth",
		},
		{
			name: "single bump types",
			changesets: []*models.Changeset{
				{ID: "patch", Message: "Fix memory leak", Projects: map[string]models.BumpType{"auth": models.BumpPatch}},
				{ID: "minor", Message: "Add OAuth2 support", Projects: map[string]models.BumpType{"auth": models.BumpMinor}},
				{ID: "major", Message: "Breaking API change", Projects: map[string]models.BumpType{"auth": models.BumpMajor}},
			},
			projectName: "auth",
		},
		{
			name: "multiple changesets grouped by type",
			changesets: []*models.Changeset{
				{ID: "patch1", Message: "Fix bug 1", Projects: map[string]models.BumpType{"auth": models.BumpPatch}},
				{ID: "minor1", Message: "Add feature 1", Projects: map[string]models.BumpType{"auth": models.BumpMinor}},
				{ID: "patch2", Message: "Fix bug 2", Projects: map[string]models.BumpType{"auth": models.BumpPatch}},
				{ID: "minor2", Message: "Add feature 2", Projects: map[string]models.BumpType{"auth": models.BumpMinor}},
			},
			projectName: "auth",
		},
		{
			name: "filters by project name",
			changesets: []*models.Changeset{
				{ID: "auth-change", Message: "Auth change", Projects: map[string]models.BumpType{"auth": models.BumpMinor}},
				{ID: "api-change", Message: "API change", Projects: map[string]models.BumpType{"api": models.BumpMinor}},
			},
			projectName: "auth",
		},
		{
			name: "empty project name includes all",
			changesets: []*models.Changeset{
				{ID: "change1", Message: "Change 1", Projects: map[string]models.BumpType{"auth": models.BumpMinor}},
				{ID: "change2", Message: "Change 2", Projects: map[string]models.BumpType{"api": models.BumpPatch}},
			},
			projectName: "",
		},
		{
			name: "multiline message with mixed bump types",
			changesets: []*models.Changeset{
				{ID: "multiline", Message: "Add new feature\n\nThis is a longer description\n with multiple lines", Projects: map[string]models.BumpType{"auth": models.BumpMinor}},
				{ID: "patch-fix", Message: "Fix minor bug", Projects: map[string]models.BumpType{"auth": models.BumpPatch}},
				{ID: "patch-fix-2", Message: "Fix another minor bug", Projects: map[string]models.BumpType{"auth": models.BumpPatch}},
			},
			projectName: "auth",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := cl.FormatEntry(tt.changesets, tt.projectName, "/workspace")
			require.NoError(t, err)
			snaps.MatchSnapshot(t, result)
		})
	}
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
	ws, fs := buildWorkspace(t, func(wb *workspace.WorkspaceBuilder) {
		wb.AddProject("auth", "packages/auth", "github.com/test/auth")
		wb.AddProject("api", "packages/api", "github.com/test/api")
		wb.AddChangeset("auth-minor", "auth", "minor", "Add OAuth2 authentication support")
		wb.AddChangeset("auth-patch", "auth", "patch", "Fix memory leak in session handler")
		wb.AddChangeset("api-minor", "api", "minor", "Add GraphQL endpoint")
	})

	preview, _ := formatProjectPreview(t, ws, fs, "auth")

	require.Contains(t, preview, "### Minor Changes")
	require.Contains(t, preview, "### Patch Changes")
	require.Contains(t, preview, "Add OAuth2 authentication support")
	require.Contains(t, preview, "Fix memory leak in session handler")
}

func TestChangelog_FormatEntry_MultilineMessages(t *testing.T) {
	multilineMsg := `Add comprehensive OAuth2 support

This change includes:
- Google OAuth2 provider
- GitHub OAuth2 provider
- Token refresh mechanism`

	ws, fs := buildWorkspace(t, func(wb *workspace.WorkspaceBuilder) {
		wb.AddProject("auth", "packages/auth", "github.com/test/auth")
		wb.AddChangeset("multiline", "auth", "minor", multilineMsg)
	})

	preview, _ := formatProjectPreview(t, ws, fs, "auth")

	require.Contains(t, preview, "Add comprehensive OAuth2 support")
	require.Contains(t, preview, "Google OAuth2 provider")

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
	ws, fs := buildWorkspace(t, func(wb *workspace.WorkspaceBuilder) {
		wb.AddProject("auth", "packages/auth", "github.com/test/auth")
		wb.SetVersion("auth", "0.1.0")
		wb.AddChangeset("test", "auth", "minor", "Add new feature")
	})

	preview, changesets := formatProjectPreview(t, ws, fs, "auth")

	project := ws.Projects[0]
	ctx := &models.ProjectContext{
		Project:          project.Name,
		ProjectPath:      project.RootPath,
		ModulePath:       project.ModulePath,
		HasChangesets:    len(changesets) > 0,
		ChangelogPreview: preview,
	}

	require.NotEmpty(t, ctx.ChangelogPreview)
	require.Contains(t, ctx.ChangelogPreview, "Add new feature")

	jsonData, err := json.Marshal(ctx)
	require.NoError(t, err)

	jsonStr := string(jsonData)
	require.Contains(t, jsonStr, "changelogPreview")

	var ctxCopy models.ProjectContext
	require.NoError(t, json.Unmarshal(jsonData, &ctxCopy))

	require.Equal(t, ctx.ChangelogPreview, ctxCopy.ChangelogPreview)
}

func TestChangelog_FormatEntry_OrderByBumpType(t *testing.T) {
	ws, fs := buildWorkspace(t, func(wb *workspace.WorkspaceBuilder) {
		wb.AddProject("auth", "packages/auth", "github.com/test/auth")
		wb.AddChangeset("patch1", "auth", "patch", "Fix bug 1")
		wb.AddChangeset("major1", "auth", "major", "Breaking change")
		wb.AddChangeset("minor1", "auth", "minor", "New feature")
		wb.AddChangeset("patch2", "auth", "patch", "Fix bug 2")
	})

	preview, _ := formatProjectPreview(t, ws, fs, "auth")

	majorIdx := strings.Index(preview, "### Major Changes")
	minorIdx := strings.Index(preview, "### Minor Changes")
	patchIdx := strings.Index(preview, "### Patch Changes")

	require.NotEqual(t, -1, majorIdx, "Missing Major section in preview:\n%s", preview)
	require.NotEqual(t, -1, minorIdx, "Missing Minor section in preview:\n%s", preview)
	require.NotEqual(t, -1, patchIdx, "Missing Patch section in preview:\n%s", preview)

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

func TestChangelog_FormatEntry_withPRDetails(t *testing.T) {
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

	fs := filesystem.NewMockFileSystem()
	changelog := NewChangelog(fs)

	preview, err := changelog.FormatEntry(changesets, "auth", "/workspace")
	require.NoError(t, err)
	snaps.MatchSnapshot(t, preview)

}

func TestChangelog_Append_withPRDetails(t *testing.T) {
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

	appendCases := []struct {
		name        string
		projectName string
	}{
		{name: "Append", projectName: ""},
		{name: "AppendWithProject", projectName: "auth"},
	}

	for _, tc := range appendCases {
		t.Run(tc.name, func(t *testing.T) {
			fs := filesystem.NewMockFileSystem()
			changelog := NewChangelog(fs)

			rootEntry := &Entry{
				Version:    &models.Version{Major: 2, Minor: 0, Patch: 0},
				Date:       time.Date(2024, 12, 6, 0, 0, 0, 0, time.UTC),
				Changesets: changesets,
			}
			require.NoError(t, changelog.Append(".", tc.projectName, rootEntry))

			content, err := fs.ReadFile("./CHANGELOG.md")
			require.NoError(t, err)

			snaps.MatchSnapshot(t, string(content))
		})
	}
}

func TestChangelog_Append(t *testing.T) {
	testCases := []struct {
		name        string
		projectName string
	}{
		{name: "global changelog", projectName: ""},
		{name: "project changelog", projectName: "test"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fs := filesystem.NewMockFileSystem()
			cl := NewChangelog(fs)

			projectRoot := "/test/project"
			changelogPath := filepath.Join(projectRoot, "CHANGELOG.md")

			fs.AddDir(projectRoot)

			renderChangelogEntry := func(entry *Entry) string {
				t.Helper()

				require.NoError(t, cl.Append(projectRoot, tc.projectName, entry))

				data, err := fs.ReadFile(changelogPath)
				require.NoError(t, err)
				return string(data)
			}

			first := renderChangelogEntry(&Entry{
				Version: &models.Version{Major: 1, Minor: 0, Patch: 0},
				Date:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				Changesets: []*models.Changeset{
					{ID: "first", Message: "First change", Projects: map[string]models.BumpType{"test": models.BumpMinor}},
				},
			})
			snaps.MatchSnapshot(t, first)

			second := renderChangelogEntry(&Entry{
				Version: &models.Version{Major: 1, Minor: 1, Patch: 0},
				Date:    time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
				Changesets: []*models.Changeset{
					{ID: "second", Message: "Second change", Projects: map[string]models.BumpType{"test": models.BumpMinor}},
				},
			})
			snaps.MatchSnapshot(t, second)

			third := renderChangelogEntry(&Entry{
				Version: &models.Version{Major: 1, Minor: 1, Patch: 1},
				Date:    time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
				Changesets: []*models.Changeset{
					{ID: "third", Message: "Third change", Projects: map[string]models.BumpType{"test": models.BumpMinor}},
				},
			})
			snaps.MatchSnapshot(t, third)
		})
	}
}

func resetChangelogTemplateCache() {
	templateCacheLock.Lock()
	defer templateCacheLock.Unlock()
	templateCache = make(map[string]*template.Template)
}
