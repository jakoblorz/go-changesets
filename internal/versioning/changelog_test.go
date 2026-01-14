package versioning

import (
	"strings"
	"testing"
	"time"

	"github.com/jakoblorz/go-changesets/internal/filesystem"
	"github.com/jakoblorz/go-changesets/internal/models"
	"github.com/stretchr/testify/assert"
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
			want:        "",
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
			want:        "### Patch Changes\n\n- Fix memory leak\n\n",
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
			want:        "### Minor Changes\n\n- Add OAuth2 support\n\n",
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
			want:        "### Major Changes\n\n- Breaking API change\n\n",
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
			want: `### Minor Changes

- Add feature 1
- Add feature 2

### Patch Changes

- Fix bug 1
- Fix bug 2

`,
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
			want:        "### Minor Changes\n\n- Auth change\n\n",
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
			want: `### Minor Changes

- Change 1

### Patch Changes

- Change 2

`,
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
			want:        "### Minor Changes\n\n- Add new feature\n    This is a longer description\n    with multiple lines\n\n",
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
			want:        "### Minor Changes\n\n- Add new feature\n    This is a longer description\n    with multiple lines\n\n### Patch Changes\n\n- Fix minor bug\n- Fix another minor bug\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := cl.FormatEntry(tt.changesets, tt.projectName, "/workspace")
			assert.NoError(t, err, "FormatEntry() should not return an error")
			assert.Equal(t, tt.want, got, "FormatEntry() output mismatch")
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
	if err != nil {
		t.Fatalf("FormatEntry() error = %v", err)
	}

	// Verify order: Major, Minor, Patch
	majorIdx := strings.Index(result, "### Major Changes")
	minorIdx := strings.Index(result, "### Minor Changes")
	patchIdx := strings.Index(result, "### Patch Changes")

	if majorIdx == -1 || minorIdx == -1 || patchIdx == -1 {
		t.Fatalf("Missing sections in output:\n%s", result)
	}

	if !(majorIdx < minorIdx && minorIdx < patchIdx) {
		t.Errorf("Sections not in correct order (Major, Minor, Patch):\n%s", result)
	}

	// Verify content
	if !strings.Contains(result, "- Breaking change") {
		t.Errorf("Missing major change in output")
	}
	if !strings.Contains(result, "- New feature") {
		t.Errorf("Missing minor change in output")
	}
	if !strings.Contains(result, "- Bug fix") {
		t.Errorf("Missing patch change in output")
	}
}

func TestChangelog_CustomTemplateOverride(t *testing.T) {
	fs := filesystem.NewMockFileSystem()
	templateContent := "Project: {{.Project}} Version: {{.Version}} Sections: {{len .Sections}}"
	fs.AddFile("/workspace/.changeset/changelog.tmpl", []byte(templateContent))

	cl := NewChangelog(fs)
	entry := &ChangelogEntry{
		Version:    mustParseVersion("1.2.3"),
		Date:       time.Date(2024, 12, 6, 0, 0, 0, 0, time.UTC),
		Changesets: []*models.Changeset{},
	}

	output, err := cl.formatEntry(entry, "auth", "/workspace")
	if err != nil {
		t.Fatalf("formatEntry() error = %v", err)
	}
	if output != "Project: auth Version: 1.2.3 Sections: 0" {
		t.Fatalf("unexpected template output: %s", output)
	}
}

func mustParseVersion(s string) *models.Version {
	v, err := models.ParseVersion(s)
	if err != nil {
		panic(err)
	}
	return v
}
