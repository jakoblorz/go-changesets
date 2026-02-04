package versioning

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/jakoblorz/go-changesets/internal/filesystem"
	"github.com/jakoblorz/go-changesets/internal/models"
)

// VersionStore abstracts reading/writing project versions across project types.
type VersionStore interface {
	Read(projectRoot string) (*models.Version, error)
	Write(projectRoot string, version *models.Version) error
	IsEnabled(projectRoot string) bool
}

// NewVersionStore returns the appropriate store for the project type.
func NewVersionStore(fs filesystem.FileSystem, projectType models.ProjectType) VersionStore {
	switch projectType {
	case models.ProjectTypeNode:
		return NewPackageJSONVersionStore(fs)
	default:
		return NewVersionFile(fs)
	}
}

var _ VersionStore = (*PackageJSONVersionStore)(nil)

var versionRegex = regexp.MustCompile(`"version"\s*:\s*"[^"]*"`)

// PackageJSONVersionStore reads/writes the version field in package.json.
type PackageJSONVersionStore struct {
	fs filesystem.FileSystem
}

// NewPackageJSONVersionStore creates a new store for Node projects.
func NewPackageJSONVersionStore(fs filesystem.FileSystem) *PackageJSONVersionStore {
	return &PackageJSONVersionStore{fs: fs}
}

// Read reads the version from package.json; missing or empty defaults to 0.0.0.
func (p *PackageJSONVersionStore) Read(projectRoot string) (*models.Version, error) {
	pkgPath := filepath.Join(projectRoot, "package.json")
	if !p.fs.Exists(pkgPath) {
		return models.ParseVersion("0.0.0")
	}

	data, err := p.fs.ReadFile(pkgPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read package.json: %w", err)
	}

	var pkg map[string]interface{}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, fmt.Errorf("failed to parse package.json: %w", err)
	}

	versionStr, _ := pkg["version"].(string)
	if strings.TrimSpace(versionStr) == "" {
		versionStr = "0.0.0"
	}

	version, err := models.ParseVersion(versionStr)
	if err != nil {
		return nil, fmt.Errorf("invalid version in package.json: %w", err)
	}

	return version, nil
}

// Write writes the version to the version field in package.json without reformatting the file.
func (p *PackageJSONVersionStore) Write(projectRoot string, version *models.Version) error {
	pkgPath := filepath.Join(projectRoot, "package.json")
	data, err := p.fs.ReadFile(pkgPath)
	if err != nil {
		return fmt.Errorf("failed to read package.json: %w", err)
	}

	newField := fmt.Sprintf(`"version": "%s"`, version.String())

	if versionRegex.Match(data) {
		data = versionRegex.ReplaceAll(data, []byte(newField))
	} else {
		insert := "\n  " + newField + ","
		idx := strings.Index(string(data), "{")
		if idx == -1 {
			return fmt.Errorf("invalid package.json: missing '{'")
		}
		idx++
		data = append(data[:idx], append([]byte(insert), data[idx:]...)...)
	}

	if err := p.fs.WriteFile(pkgPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write package.json: %w", err)
	}

	return nil
}

// IsEnabled always returns true for Node projects (no disable flag).
func (p *PackageJSONVersionStore) IsEnabled(string) bool {
	return true
}
