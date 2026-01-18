package changeset

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/adrg/frontmatter"
	"github.com/jakoblorz/go-changesets/internal/filesystem"
	"github.com/jakoblorz/go-changesets/internal/models"
)

// Manager handles changeset operations
type Manager struct {
	fs           filesystem.FileSystem
	changesetDir string
}

// NewManager creates a new changeset manager
func NewManager(fs filesystem.FileSystem, changesetDir string) *Manager {
	return &Manager{
		fs:           fs,
		changesetDir: changesetDir,
	}
}

// GenerateID generates a unique, human-friendly ID for a changeset
// Format: adjective_animal_nanoid (e.g., "dazzling_mouse_V1StGXR8")
func (m *Manager) GenerateID() (string, error) {
	id, err := generateHumanFriendlyID()
	if err != nil {
		return "", fmt.Errorf("failed to generate ID: %w", err)
	}
	return id, nil
}

// ReadAll reads all changeset files from the .changeset directory
func (m *Manager) ReadAll() ([]*models.Changeset, error) {
	if !m.fs.Exists(m.changesetDir) {
		return []*models.Changeset{}, nil
	}

	entries, err := m.fs.ReadDir(m.changesetDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read changeset directory: %w", err)
	}

	var changesets []*models.Changeset
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Only process .md files
		if !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		filePath := filepath.Join(m.changesetDir, entry.Name())
		changeset, err := m.Read(filePath)
		if err != nil {
			// Log warning but continue processing other files
			fmt.Printf("Warning: failed to read changeset %s: %v\n", entry.Name(), err)
			continue
		}

		changesets = append(changesets, changeset)
	}

	return changesets, nil
}

// ReadAllOfProject reads all changesets and filters for a specific project
func (m *Manager) ReadAllOfProject(projectName string) ([]*models.Changeset, error) {
	all, err := m.ReadAll()
	if err != nil {
		return nil, err
	}

	return FilterByProject(all, projectName), nil
}

// Read reads a single changeset file
func (m *Manager) Read(filePath string) (*models.Changeset, error) {
	data, err := m.fs.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return m.Parse(filePath, data)
}

// Parse parses changeset data from bytes
func (m *Manager) Parse(filePath string, data []byte) (*models.Changeset, error) {
	var matter map[string]string
	var body bytes.Buffer

	rest, err := frontmatter.Parse(bytes.NewReader(data), &matter)
	if err != nil {
		return nil, fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	// Copy remaining content to body
	body.Write(rest)

	// Parse projects and bump types from frontmatter
	projects := make(map[string]models.BumpType)
	for projectName, bumpStr := range matter {
		bump, err := models.ParseBumpType(bumpStr)
		if err != nil {
			return nil, fmt.Errorf("invalid bump type for project %s: %w", projectName, err)
		}
		projects[projectName] = bump
	}

	if len(projects) == 0 {
		return nil, fmt.Errorf("no projects found in changeset frontmatter")
	}

	// Extract ID from filename
	filename := filepath.Base(filePath)
	id := strings.TrimSuffix(filename, ".md")

	changeset := models.NewChangeset(id, projects, strings.TrimSpace(body.String()))
	changeset.FilePath = filePath

	return changeset, nil
}

// Write creates a new changeset file
func (m *Manager) Write(changeset *models.Changeset) error {
	// Ensure .changeset directory exists
	if !m.fs.Exists(m.changesetDir) {
		if err := m.fs.MkdirAll(m.changesetDir, 0755); err != nil {
			return fmt.Errorf("failed to create changeset directory: %w", err)
		}
	}

	// Generate content with frontmatter
	var buf bytes.Buffer

	// Write frontmatter
	buf.WriteString("---\n")
	for projectName, bump := range changeset.Projects {
		buf.WriteString(fmt.Sprintf("%s: %s\n", projectName, bump))
	}
	buf.WriteString("---\n\n")

	// Write message
	buf.WriteString(changeset.Message)
	buf.WriteString("\n")

	// Write to file
	filePath := filepath.Join(m.changesetDir, changeset.ID+".md")
	if err := m.fs.WriteFile(filePath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write changeset file: %w", err)
	}

	changeset.FilePath = filePath
	return nil
}

// Delete removes a changeset file
func (m *Manager) Delete(changeset *models.Changeset) error {
	if changeset.FilePath == "" {
		return fmt.Errorf("changeset has no file path")
	}

	if err := m.fs.Remove(changeset.FilePath); err != nil {
		return fmt.Errorf("failed to delete changeset: %w", err)
	}

	return nil
}

// GetHighestBump determines the highest bump type from multiple changesets
func (m *Manager) GetHighestBump(changesets []*models.Changeset, projectName string) models.BumpType {
	highest := models.BumpPatch

	for _, cs := range changesets {
		if bump, exists := cs.GetBumpForProject(projectName); exists {
			if bump == models.BumpMajor {
				return models.BumpMajor
			}
			if bump == models.BumpMinor && highest == models.BumpPatch {
				highest = models.BumpMinor
			}
		}
	}

	return highest
}

// FilterByProject returns changesets that affect a specific project
func FilterByProject(changesets []*models.Changeset, projectName string) []*models.Changeset {
	var relevant []*models.Changeset
	for _, cs := range changesets {
		if cs.AffectsProject(projectName) {
			relevant = append(relevant, cs)
		}
	}
	return relevant
}
