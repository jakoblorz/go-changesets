package workspace

import (
	"path/filepath"

	"github.com/jakoblorz/go-changesets/internal/filesystem"
)

func findFileUp(fs filesystem.FileSystem, startDir, filename string) (string, bool, error) {
	dir := filepath.Clean(startDir)

	for {
		candidate := filepath.Join(dir, filename)
		if fs.Exists(candidate) {
			return candidate, true, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false, nil
		}
		dir = parent
	}
}
