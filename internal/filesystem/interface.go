package filesystem

import (
	"io/fs"
)

// FileSystem provides an abstraction over file operations for testability
type FileSystem interface {
	// File operations
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte, perm fs.FileMode) error
	Remove(path string) error

	// Directory operations
	ReadDir(path string) ([]fs.DirEntry, error)
	MkdirAll(path string, perm fs.FileMode) error

	// Path operations
	Stat(path string) (fs.FileInfo, error)
	Exists(path string) bool
	Getwd() (string, error)

	// File walking
	WalkDir(root string, fn fs.WalkDirFunc) error

	// Glob patterns
	Glob(pattern string) ([]string, error)
}
