package filesystem

import (
	"io/fs"
	"os"
	"path/filepath"
)

// OSFileSystem implements FileSystem using real OS operations
type OSFileSystem struct{}

// NewOSFileSystem creates a new OSFileSystem
func NewOSFileSystem() *OSFileSystem {
	return &OSFileSystem{}
}

func (osfs *OSFileSystem) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (osfs *OSFileSystem) WriteFile(path string, data []byte, perm fs.FileMode) error {
	return os.WriteFile(path, data, perm)
}

func (osfs *OSFileSystem) Remove(path string) error {
	return os.Remove(path)
}

func (osfs *OSFileSystem) ReadDir(path string) ([]fs.DirEntry, error) {
	return os.ReadDir(path)
}

func (osfs *OSFileSystem) MkdirAll(path string, perm fs.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (osfs *OSFileSystem) Stat(path string) (fs.FileInfo, error) {
	return os.Stat(path)
}

func (osfs *OSFileSystem) Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (osfs *OSFileSystem) Getwd() (string, error) {
	return os.Getwd()
}

func (osfs *OSFileSystem) WalkDir(root string, fn fs.WalkDirFunc) error {
	return filepath.WalkDir(root, fn)
}

func (osfs *OSFileSystem) Glob(pattern string) ([]string, error) {
	return filepath.Glob(pattern)
}
