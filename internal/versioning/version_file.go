package versioning

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/jakoblorz/go-changesets/internal/filesystem"
	"github.com/jakoblorz/go-changesets/internal/models"
)

const versionFileName = "version.txt"

var _ VersionStore = (*VersionFile)(nil)

// VersionFile handles reading and writing version.txt files
type VersionFile struct {
	fs filesystem.FileSystem
}

// NewVersionFile creates a new VersionFile instance
func NewVersionFile(fs filesystem.FileSystem) *VersionFile {
	return &VersionFile{fs: fs}
}

// Read reads the version from version.txt in the project root
func (vf *VersionFile) Read(projectRoot string) (*models.Version, error) {
	versionPath := filepath.Join(projectRoot, versionFileName)

	if !vf.fs.Exists(versionPath) {
		// Default to 0.0.0 if version.txt doesn't exist
		return models.ParseVersion("0.0.0")
	}

	data, err := vf.fs.ReadFile(versionPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read version file: %w", err)
	}

	versionStr := strings.TrimSpace(string(data))
	version, err := models.ParseVersion(versionStr)
	if err != nil {
		return nil, fmt.Errorf("invalid version in %s: %w", versionPath, err)
	}

	return version, nil
}

// IsEnabled checks if a project is enabled for versioning
// Returns false if version.txt contains "false" (case-insensitive)
// Returns true if version.txt doesn't exist or contains a valid version
func (vf *VersionFile) IsEnabled(projectRoot string) bool {
	versionPath := filepath.Join(projectRoot, versionFileName)

	if !vf.fs.Exists(versionPath) {
		return true // No version.txt = enabled by default
	}

	data, err := vf.fs.ReadFile(versionPath)
	if err != nil {
		return true // Can't read = assume enabled
	}

	content := strings.TrimSpace(strings.ToLower(string(data)))

	// Check if explicitly disabled
	if content == "false" {
		return false
	}

	return true
}

// Write writes the version to version.txt in the project root
func (vf *VersionFile) Write(projectRoot string, version *models.Version) error {
	versionPath := filepath.Join(projectRoot, versionFileName)

	content := version.String() + "\n"
	if err := vf.fs.WriteFile(versionPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write version file: %w", err)
	}

	return nil
}
