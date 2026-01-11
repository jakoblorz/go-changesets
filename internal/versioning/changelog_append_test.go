package versioning

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jakoblorz/go-changesets/internal/filesystem"
	"github.com/jakoblorz/go-changesets/internal/models"
)

func TestChangelog_Append_PreservesHeader(t *testing.T) {
	fs := filesystem.NewMockFileSystem()
	cl := NewChangelog(fs)
	projectRoot := "/test/project"

	fs.AddDir(projectRoot)

	// First append - should add header
	entry1 := &ChangelogEntry{
		Version: &models.Version{Major: 1, Minor: 0, Patch: 0},
		Date:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Changesets: []*models.Changeset{
			{
				ID:      "first",
				Message: "First change",
				Projects: map[string]models.BumpType{
					"test": models.BumpMinor,
				},
			},
		},
	}

	if err := cl.Append(projectRoot, entry1); err != nil {
		t.Fatalf("First append failed: %v", err)
	}

	// Read and verify first version has header
	changelogPath := filepath.Join(projectRoot, "CHANGELOG.md")
	data, err := fs.ReadFile(changelogPath)
	if err != nil {
		t.Fatalf("Failed to read changelog: %v", err)
	}

	content := string(data)

	if !strings.Contains(content, "# Changelog") {
		t.Error("First append should include '# Changelog' header")
	}
	if !strings.Contains(content, "All notable changes to this project will be documented in this file.") {
		t.Error("First append should include description line")
	}
	if !strings.Contains(content, "## 1.0.0 (2024-01-01)") {
		t.Error("First append should include version 1.0.0")
	}
	if !strings.Contains(content, "First change") {
		t.Error("First append should include first change")
	}

	// Second append - should preserve header
	entry2 := &ChangelogEntry{
		Version: &models.Version{Major: 1, Minor: 1, Patch: 0},
		Date:    time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
		Changesets: []*models.Changeset{
			{
				ID:      "second",
				Message: "Second change",
				Projects: map[string]models.BumpType{
					"test": models.BumpMinor,
				},
			},
		},
	}

	if err := cl.Append(projectRoot, entry2); err != nil {
		t.Fatalf("Second append failed: %v", err)
	}

	// Read and verify second version still has header
	data, err = fs.ReadFile(changelogPath)
	if err != nil {
		t.Fatalf("Failed to read changelog after second append: %v", err)
	}

	content = string(data)

	// Verify header is still present
	if !strings.Contains(content, "# Changelog") {
		t.Error("Second append should preserve '# Changelog' header")
	}
	if !strings.Contains(content, "All notable changes to this project will be documented in this file.") {
		t.Error("Second append should preserve description line")
	}

	// Verify both versions are present
	if !strings.Contains(content, "## 1.1.0 (2024-02-01)") {
		t.Error("Second append should include version 1.1.0")
	}
	if !strings.Contains(content, "## 1.0.0 (2024-01-01)") {
		t.Error("Second append should preserve version 1.0.0")
	}
	if !strings.Contains(content, "Second change") {
		t.Error("Second append should include second change")
	}
	if !strings.Contains(content, "First change") {
		t.Error("Second append should preserve first change")
	}

	// Verify header appears only once
	headerCount := strings.Count(content, "# Changelog")
	if headerCount != 1 {
		t.Errorf("Expected header to appear exactly once, found %d times", headerCount)
	}

	// Verify order: Header, then newest entry, then older entry
	headerIdx := strings.Index(content, "# Changelog")
	v110Idx := strings.Index(content, "## 1.1.0")
	v100Idx := strings.Index(content, "## 1.0.0")

	if headerIdx == -1 || v110Idx == -1 || v100Idx == -1 {
		t.Fatal("Could not find expected sections in changelog")
	}

	if !(headerIdx < v110Idx && v110Idx < v100Idx) {
		t.Error("Changelog sections not in correct order (Header -> Newest -> Oldest)")
	}
}

func TestChangelog_Append_ThirdEntry(t *testing.T) {
	// Test that third and subsequent appends also preserve header
	fs := filesystem.NewMockFileSystem()
	cl := NewChangelog(fs)
	projectRoot := "/test/project"

	fs.AddDir(projectRoot)

	// Add three entries
	for i := 0; i < 3; i++ {
		entry := &ChangelogEntry{
			Version: &models.Version{Major: 1, Minor: i, Patch: 0},
			Date:    time.Date(2024, time.Month(i+1), 1, 0, 0, 0, 0, time.UTC),
			Changesets: []*models.Changeset{
				{
					ID:      "change" + string(rune(i)),
					Message: "Change number " + string(rune('0'+i)),
					Projects: map[string]models.BumpType{
						"test": models.BumpMinor,
					},
				},
			},
		}

		if err := cl.Append(projectRoot, entry); err != nil {
			t.Fatalf("Append %d failed: %v", i, err)
		}
	}

	// Read final changelog
	changelogPath := filepath.Join(projectRoot, "CHANGELOG.md")
	data, err := fs.ReadFile(changelogPath)
	if err != nil {
		t.Fatalf("Failed to read changelog: %v", err)
	}

	content := string(data)

	// Header should appear exactly once
	headerCount := strings.Count(content, "# Changelog")
	if headerCount != 1 {
		t.Errorf("Expected header exactly once after 3 appends, found %d times", headerCount)
	}

	// All three versions should be present
	if !strings.Contains(content, "## 1.0.0") {
		t.Error("Should contain version 1.0.0")
	}
	if !strings.Contains(content, "## 1.1.0") {
		t.Error("Should contain version 1.1.0")
	}
	if !strings.Contains(content, "## 1.2.0") {
		t.Error("Should contain version 1.2.0")
	}
}

func TestChangelog_Append_NoExistingFile(t *testing.T) {
	// Test first append when no file exists
	fs := filesystem.NewMockFileSystem()
	cl := NewChangelog(fs)
	projectRoot := "/test/project"

	fs.AddDir(projectRoot)

	entry := &ChangelogEntry{
		Version: &models.Version{Major: 0, Minor: 1, Patch: 0},
		Date:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Changesets: []*models.Changeset{
			{
				ID:      "initial",
				Message: "Initial release",
				Projects: map[string]models.BumpType{
					"test": models.BumpMinor,
				},
			},
		},
	}

	if err := cl.Append(projectRoot, entry); err != nil {
		t.Fatalf("Append failed: %v", err)
	}

	changelogPath := filepath.Join(projectRoot, "CHANGELOG.md")
	data, err := fs.ReadFile(changelogPath)
	if err != nil {
		t.Fatalf("Failed to read changelog: %v", err)
	}

	content := string(data)

	// Should have header
	if !strings.HasPrefix(content, "# Changelog\n\n") {
		t.Error("Changelog should start with header")
	}

	// Should have version entry
	if !strings.Contains(content, "## 0.1.0 (2024-01-01)") {
		t.Error("Should contain version entry")
	}
}
