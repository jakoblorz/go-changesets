package github

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPRMapping_ReadWrite(t *testing.T) {
	mapping := NewPRMapping()
	mapping.Set("auth", PREntry{PRNumber: 123, Branch: "changeset-release/auth", Version: "1.2.0"})
	mapping.Set("api", PREntry{PRNumber: 124, Branch: "changeset-release/api", Version: "2.1.0"})

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "pr-mapping.json")

	err := mapping.Write(path)
	require.NoError(t, err)

	// Read it back
	readMapping, err := ReadPRMapping(path)
	require.NoError(t, err)
	require.NotNil(t, readMapping)

	entry, ok := readMapping.Get("auth")
	require.True(t, ok)
	require.Equal(t, 123, entry.PRNumber)
	require.Equal(t, "changeset-release/auth", entry.Branch)
	require.Equal(t, "1.2.0", entry.Version)

	entry2, ok := readMapping.Get("api")
	require.True(t, ok)
	require.Equal(t, 124, entry2.PRNumber)
}

func TestPRMapping_Remove(t *testing.T) {
	mapping := NewPRMapping()
	mapping.Set("auth", PREntry{PRNumber: 123, Branch: "changeset-release/auth"})
	mapping.Set("api", PREntry{PRNumber: 124, Branch: "changeset-release/api"})

	mapping.Remove("auth")

	_, ok := mapping.Get("auth")
	require.False(t, ok)

	_, ok = mapping.Get("api")
	require.True(t, ok)
}

func TestPRMapping_ReadNonExistent(t *testing.T) {
	mapping, err := ReadPRMapping("/non/existent/path.json")
	require.NoError(t, err)
	require.NotNil(t, mapping)
	require.True(t, mapping.IsEmpty())
}

func TestPRMapping_ReadEmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "empty.json")

	err := os.WriteFile(path, []byte(""), 0644)
	require.NoError(t, err)

	mapping, err := ReadPRMapping(path)
	require.NoError(t, err)
	require.NotNil(t, mapping)
	require.True(t, mapping.IsEmpty())
}

func TestTemplateData(t *testing.T) {
	data := TemplateData{
		Project:          "auth",
		Version:          "1.2.0",
		CurrentVersion:   "1.1.0",
		ChangelogPreview: "## Minor Changes\n- Add OAuth2 support",
		CommitSHA:        "abc123def456",
		RelatedPRs: []RelatedPRInfo{
			{Number: 123, Project: "auth", Version: "1.2.0"},
			{Number: 124, Project: "api", Version: "2.1.0"},
		},
	}

	require.Equal(t, "auth", data.Project)
	require.Equal(t, "1.2.0", data.Version)
	require.Len(t, data.RelatedPRs, 2)
}

func TestExecuteDefaultTemplate(t *testing.T) {
	data := TemplateData{
		Project: "auth",
		Version: "1.2.0",
	}

	title, err := ExecuteDefaultTemplate("title", data)
	require.NoError(t, err)
	require.Equal(t, "🚀 Release auth v1.2.0", title)

	body, err := ExecuteDefaultTemplate("body", data)
	require.NoError(t, err)
	require.Contains(t, body, "auth")
	require.Contains(t, body, "1.2.0")
	require.Contains(t, body, "RELATED_PRS_PLACEHOLDER")
}

func TestBuildRelatedPRsSection(t *testing.T) {
	relatedPRs := []RelatedPRInfo{
		{Number: 123, Project: "auth", Version: "1.2.0"},
		{Number: 124, Project: "api", Version: "2.1.0"},
		{Number: 125, Project: "shared", Version: "0.5.0"},
	}

	section := BuildRelatedPRsSection("abc123def456", relatedPRs, "auth")
	require.Contains(t, section, "## 🔗 Related Release PRs")
	require.Contains(t, section, "abc123def456")
	require.Contains(t, section, "#123 Release auth v1.2.0")
	require.Contains(t, section, "← **You are here**")
	require.Contains(t, section, "#124 Release api v2.1.0")
	require.Contains(t, section, "#125 Release shared v0.5.0")
}

func TestBuildRelatedPRsSection_Single(t *testing.T) {
	relatedPRs := []RelatedPRInfo{
		{Number: 123, Project: "auth", Version: "1.2.0"},
	}

	section := BuildRelatedPRsSection("abc123def456", relatedPRs, "auth")
	require.Empty(t, section)
}

func TestReplaceRelatedPRsPlaceholder(t *testing.T) {
	body := "This is a test\n\n<!-- RELATED_PRS_PLACEHOLDER -->\n\nFooter"
	replacement := "## Related PRs\n- #123"

	result := ReplaceRelatedPRsPlaceholder(body, replacement)
	require.Contains(t, result, "## Related PRs")
	require.Contains(t, result, "#123")
}

func TestReplaceRelatedPRsPlaceholder_NoPlaceholder(t *testing.T) {
	body := "This is a test without placeholder"
	replacement := "## Related PRs\n- #123"

	result := ReplaceRelatedPRsPlaceholder(body, replacement)
	require.Equal(t, body+"\n\n"+replacement, result)
}

func TestParseTemplateFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.tmpl")

	content := "# Release {{.Project}} v{{.Version}}"
	err := os.WriteFile(path, []byte(content), 0644)
	require.NoError(t, err)

	tmpl, err := ParseTemplateFile(path)
	require.NoError(t, err)
	require.NotNil(t, tmpl)

	data := TemplateData{Project: "auth", Version: "1.2.0"}
	result, err := ExecuteTemplate(tmpl, "pr-body", data)
	require.NoError(t, err)
	require.Equal(t, "# Release auth v1.2.0", result)
}
