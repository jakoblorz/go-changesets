package filesystem

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// MockFileSystem provides in-memory filesystem for testing
type MockFileSystem struct {
	files      map[string]*MockFile
	currentDir string
}

// MockFile represents a file in the mock filesystem
type MockFile struct {
	Content []byte
	Mode    fs.FileMode
	ModTime time.Time
	IsDir   bool
}

// mockFileInfo implements fs.FileInfo
type mockFileInfo struct {
	name    string
	size    int64
	mode    fs.FileMode
	modTime time.Time
	isDir   bool
}

func (m *mockFileInfo) Name() string       { return m.name }
func (m *mockFileInfo) Size() int64        { return m.size }
func (m *mockFileInfo) Mode() fs.FileMode  { return m.mode }
func (m *mockFileInfo) ModTime() time.Time { return m.modTime }
func (m *mockFileInfo) IsDir() bool        { return m.isDir }
func (m *mockFileInfo) Sys() interface{}   { return nil }

// mockDirEntry implements fs.DirEntry
type mockDirEntry struct {
	info fs.FileInfo
}

func (m *mockDirEntry) Name() string               { return m.info.Name() }
func (m *mockDirEntry) IsDir() bool                { return m.info.IsDir() }
func (m *mockDirEntry) Type() fs.FileMode          { return m.info.Mode().Type() }
func (m *mockDirEntry) Info() (fs.FileInfo, error) { return m.info, nil }

// NewMockFileSystem creates a new MockFileSystem
func NewMockFileSystem() *MockFileSystem {
	return &MockFileSystem{
		files:      make(map[string]*MockFile),
		currentDir: "/workspace",
	}
}

// AddFile adds a file to the mock filesystem
func (mfs *MockFileSystem) AddFile(path string, content []byte) {
	cleanPath := filepath.Clean(path)
	mfs.files[cleanPath] = &MockFile{
		Content: content,
		Mode:    0644,
		ModTime: time.Now(),
		IsDir:   false,
	}

	// Ensure parent directories exist
	dir := filepath.Dir(cleanPath)
	for dir != "." && dir != "/" && dir != cleanPath {
		if _, exists := mfs.files[dir]; !exists {
			mfs.AddDir(dir)
		}
		dir = filepath.Dir(dir)
	}
}

// AddDir adds a directory to the mock filesystem
func (mfs *MockFileSystem) AddDir(path string) {
	cleanPath := filepath.Clean(path)
	if _, exists := mfs.files[cleanPath]; !exists {
		mfs.files[cleanPath] = &MockFile{
			Mode:    0755 | fs.ModeDir,
			ModTime: time.Now(),
			IsDir:   true,
		}
	}

	// Ensure parent directories exist
	dir := filepath.Dir(cleanPath)
	for dir != "." && dir != "/" && dir != cleanPath {
		if _, exists := mfs.files[dir]; !exists {
			mfs.AddDir(dir)
		}
		dir = filepath.Dir(dir)
	}
}

func (mfs *MockFileSystem) ReadFile(path string) ([]byte, error) {
	file, exists := mfs.files[filepath.Clean(path)]
	if !exists {
		return nil, fs.ErrNotExist
	}
	if file.IsDir {
		return nil, errors.New("is a directory")
	}
	return file.Content, nil
}

func (mfs *MockFileSystem) WriteFile(path string, data []byte, perm fs.FileMode) error {
	cleanPath := filepath.Clean(path)

	// Ensure parent directory exists
	dir := filepath.Dir(cleanPath)
	if dir != "." && dir != "/" {
		if _, exists := mfs.files[dir]; !exists {
			return &fs.PathError{Op: "open", Path: path, Err: fs.ErrNotExist}
		}
	}

	mfs.files[cleanPath] = &MockFile{
		Content: data,
		Mode:    perm,
		ModTime: time.Now(),
		IsDir:   false,
	}
	return nil
}

func (mfs *MockFileSystem) Remove(path string) error {
	cleanPath := filepath.Clean(path)
	if _, exists := mfs.files[cleanPath]; !exists {
		return fs.ErrNotExist
	}
	delete(mfs.files, cleanPath)
	return nil
}

func (mfs *MockFileSystem) ReadDir(path string) ([]fs.DirEntry, error) {
	cleanPath := filepath.Clean(path)

	file, exists := mfs.files[cleanPath]
	if !exists {
		return nil, fs.ErrNotExist
	}
	if !file.IsDir {
		return nil, errors.New("not a directory")
	}

	var entries []fs.DirEntry
	for p, f := range mfs.files {
		dir := filepath.Dir(p)
		if dir == cleanPath {
			name := filepath.Base(p)
			info := &mockFileInfo{
				name:    name,
				size:    int64(len(f.Content)),
				mode:    f.Mode,
				modTime: f.ModTime,
				isDir:   f.IsDir,
			}
			entries = append(entries, &mockDirEntry{info: info})
		}
	}

	// Sort entries by name for consistent ordering
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	return entries, nil
}

func (mfs *MockFileSystem) MkdirAll(path string, perm fs.FileMode) error {
	cleanPath := filepath.Clean(path)
	parts := strings.Split(cleanPath, string(filepath.Separator))

	current := ""
	for _, part := range parts {
		if part == "" {
			continue
		}
		if current == "" {
			current = string(filepath.Separator) + part
		} else {
			current = filepath.Join(current, part)
		}

		if _, exists := mfs.files[current]; !exists {
			mfs.files[current] = &MockFile{
				Mode:    perm | fs.ModeDir,
				ModTime: time.Now(),
				IsDir:   true,
			}
		}
	}
	return nil
}

func (mfs *MockFileSystem) Stat(path string) (fs.FileInfo, error) {
	file, exists := mfs.files[filepath.Clean(path)]
	if !exists {
		return nil, fs.ErrNotExist
	}

	return &mockFileInfo{
		name:    filepath.Base(path),
		size:    int64(len(file.Content)),
		mode:    file.Mode,
		modTime: file.ModTime,
		isDir:   file.IsDir,
	}, nil
}

func (mfs *MockFileSystem) Exists(path string) bool {
	_, exists := mfs.files[filepath.Clean(path)]
	return exists
}

func (mfs *MockFileSystem) Getwd() (string, error) {
	return mfs.currentDir, nil
}

func (mfs *MockFileSystem) WalkDir(root string, fn fs.WalkDirFunc) error {
	cleanRoot := filepath.Clean(root)

	if _, exists := mfs.files[cleanRoot]; !exists {
		return fs.ErrNotExist
	}

	// Collect all paths that are under root
	var paths []string
	for p := range mfs.files {
		if p == cleanRoot || strings.HasPrefix(p, cleanRoot+string(filepath.Separator)) {
			paths = append(paths, p)
		}
	}

	// Sort paths for consistent ordering
	sort.Strings(paths)

	for _, p := range paths {
		file := mfs.files[p]
		info := &mockFileInfo{
			name:    filepath.Base(p),
			size:    int64(len(file.Content)),
			mode:    file.Mode,
			modTime: file.ModTime,
			isDir:   file.IsDir,
		}

		entry := &mockDirEntry{info: info}

		if err := fn(p, entry, nil); err != nil {
			if err == filepath.SkipDir && file.IsDir {
				continue
			}
			return err
		}
	}

	return nil
}

func (mfs *MockFileSystem) Glob(pattern string) ([]string, error) {
	var matches []string

	for p := range mfs.files {
		matched, err := filepath.Match(pattern, p)
		if err != nil {
			return nil, err
		}
		if matched {
			matches = append(matches, p)
		}
	}

	sort.Strings(matches)
	return matches, nil
}

// SetCurrentDir sets the current working directory for the mock
func (mfs *MockFileSystem) SetCurrentDir(dir string) {
	mfs.currentDir = dir
}

// GetFiles returns all files in the mock filesystem (for debugging)
func (mfs *MockFileSystem) GetFiles() map[string]*MockFile {
	return mfs.files
}

// PrintTree prints the filesystem tree (for debugging)
func (mfs *MockFileSystem) PrintTree() {
	var paths []string
	for p := range mfs.files {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	for _, p := range paths {
		file := mfs.files[p]
		marker := "📄"
		if file.IsDir {
			marker = "📁"
		}
		fmt.Printf("%s %s\n", marker, p)
	}
}
