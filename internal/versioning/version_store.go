package versioning

import (
	"github.com/jakoblorz/go-changesets/internal/filesystem"
	"github.com/jakoblorz/go-changesets/internal/models"
)

// VersionStore abstracts reading/writing project versions.
type VersionStore interface {
	Read(projectRoot string) (*models.Version, error)
	Write(projectRoot string, version *models.Version) error
	IsEnabled(projectRoot string) bool
}

// NewVersionStore returns the version store for the given project type.
//
// Currently go-changeset supports Go projects only and uses version.txt.
func NewVersionStore(fs filesystem.FileSystem, _ models.ProjectType) VersionStore {
	return NewVersionFile(fs)
}
