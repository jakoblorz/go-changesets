package versioning_test

import (
	"testing"

	"github.com/jakoblorz/go-changesets/internal/filesystem"
	"github.com/jakoblorz/go-changesets/internal/versioning"
)

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
		{"no file", "", false, true}, // Missing file = enabled by default
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := filesystem.NewMockFileSystem()
			fs.AddDir("project")

			if tt.hasFile {
				fs.AddFile("project/version.txt", []byte(tt.content))
			}

			vf := versioning.NewVersionFile(fs)
			result := vf.IsEnabled("project")

			if result != tt.expected {
				t.Errorf("IsEnabled() = %v, want %v (content: %q)", result, tt.expected, tt.content)
			}
		})
	}
}

func TestVersionFile_Read_WithDisabledProject(t *testing.T) {
	fs := filesystem.NewMockFileSystem()
	fs.AddDir("project")
	fs.AddFile("project/version.txt", []byte("false"))

	vf := versioning.NewVersionFile(fs)

	// IsEnabled should return false
	if vf.IsEnabled("project") {
		t.Error("Expected IsEnabled() to return false")
	}

	// Read should still work (returns error about invalid version, not about being disabled)
	// This is because Read() doesn't check IsEnabled - workspace filters before calling Read
	_, err := vf.Read("project")
	if err == nil {
		t.Error("Expected error when reading 'false' as version")
	}
}

func TestVersionFile_Read_WithEnabledProject(t *testing.T) {
	fs := filesystem.NewMockFileSystem()
	fs.AddDir("project")
	fs.AddFile("project/version.txt", []byte("1.2.3"))

	vf := versioning.NewVersionFile(fs)

	// IsEnabled should return true
	if !vf.IsEnabled("project") {
		t.Error("Expected IsEnabled() to return true")
	}

	// Read should work
	version, err := vf.Read("project")
	if err != nil {
		t.Fatalf("Read() failed: %v", err)
	}

	if version.String() != "1.2.3" {
		t.Errorf("Expected version 1.2.3, got %s", version.String())
	}
}

func TestVersionFile_IsEnabled_NoFile(t *testing.T) {
	fs := filesystem.NewMockFileSystem()

	vf := versioning.NewVersionFile(fs)

	// No version.txt = enabled by default
	if !vf.IsEnabled("project") {
		t.Error("Expected IsEnabled() to return true when file doesn't exist")
	}
}
