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
	require.NotContains(t, body, "RELATED_PRS_PLACEHOLDER")
}

func TestExecuteDefaultTemplate_WithRelatedPRs(t *testing.T) {
	data := TemplateData{
		Project:          "auth",
		Version:          "1.2.0",
		ChangelogPreview: "## Minor Changes\n- Add OAuth2 support",
		RelatedPRs: []RelatedPRInfo{
			{Number: 123, Project: "auth", Version: "1.2.0"},
			{Number: 124, Project: "api", Version: "2.1.0"},
		},
	}

	body, err := ExecuteDefaultTemplate("body", data)
	require.NoError(t, err)
	require.Contains(t, body, "## 🔗 Related Release PRs")
	require.Contains(t, body, "#123 Release auth v1.2.0")
	require.Contains(t, body, "#124 Release api v2.1.0")
}

func TestExecuteDefaultTemplate_NoRelatedPRs(t *testing.T) {
	data := TemplateData{
		Project:          "auth",
		Version:          "1.2.0",
		ChangelogPreview: "## Minor Changes\n- Add OAuth2 support",
		RelatedPRs:       []RelatedPRInfo{},
	}

	body, err := ExecuteDefaultTemplate("body", data)
	require.NoError(t, err)
	require.NotContains(t, body, "## 🔗 Related Release PRs")
	require.Contains(t, body, "## 📋 Changes")
	require.Contains(t, body, "Add OAuth2 support")
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
