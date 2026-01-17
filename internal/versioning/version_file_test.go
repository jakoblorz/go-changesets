package versioning_test

import (
	"path/filepath"
	"testing"

	"github.com/jakoblorz/go-changesets/internal/filesystem"
	"github.com/jakoblorz/go-changesets/internal/versioning"
	"github.com/jakoblorz/go-changesets/internal/workspace"
	"github.com/stretchr/testify/require"
)

const versionTestRoot = "/version-test-workspace"

func setupVersionTestFS(t *testing.T, content string, hasFile bool) (*filesystem.MockFileSystem, string) {
	t.Helper()
	wb := workspace.NewWorkspaceBuilder(versionTestRoot)
	wb.AddProject("pkg", "packages/pkg", "github.com/test/pkg")
	fs := wb.Build()

	projectPath := filepath.Join(versionTestRoot, "packages/pkg")
	if hasFile {
		fs.AddFile(filepath.Join(projectPath, "version.txt"), []byte(content))
	}

	return fs, projectPath
}

func TestVersionFile_IsEnabled(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		hasFile  bool
		expected bool
	}{
		{"disabled lowercase", "false", true, false},
		{"disabled uppercase", "FALSE", true, false},
		{"disabled mixed case", "False", true, false},
		{"disabled with whitespace", "  false  \n", true, false},
		{"disabled with newline", "false\n", true, false},
		{"enabled with version", "1.2.3", true, true},
		{"enabled with zero version", "0.0.0", true, true},
		{"enabled empty", "", true, true},
		{"enabled invalid content", "not-a-version", true, true},
		{"enabled with prerelease", "1.2.3-rc0", true, true},
		{"no file", "", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs, projectPath := setupVersionTestFS(t, tt.content, tt.hasFile)
			vf := versioning.NewVersionFile(fs)
			result := vf.IsEnabled(projectPath)
			require.Equalf(t, tt.expected, result, "content: %q", tt.content)
		})
	}
}

func TestVersionFile_Read_WithDisabledProject(t *testing.T) {
	fs, projectPath := setupVersionTestFS(t, "false", true)
	vf := versioning.NewVersionFile(fs)
	require.False(t, vf.IsEnabled(projectPath))

	_, err := vf.Read(projectPath)
	require.Error(t, err)
}

func TestVersionFile_Read_WithEnabledProject(t *testing.T) {
	fs, projectPath := setupVersionTestFS(t, "1.2.3", true)
	vf := versioning.NewVersionFile(fs)
	require.True(t, vf.IsEnabled(projectPath))

	version, err := vf.Read(projectPath)
	require.NoError(t, err)
	require.Equal(t, "1.2.3", version.String())
}

func TestVersionFile_IsEnabled_NoFile(t *testing.T) {
	fs, projectPath := setupVersionTestFS(t, "", false)
	vf := versioning.NewVersionFile(fs)
	require.True(t, vf.IsEnabled(projectPath))
}
