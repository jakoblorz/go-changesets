package versioning

import (
	"testing"
	"time"

	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/jakoblorz/go-changesets/internal/filesystem"
	"github.com/jakoblorz/go-changesets/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestFormattingSnapshots(t *testing.T) {
	changesets := []*models.Changeset{
		{
			ID:      "minor-with-pr",
			Message: "Add OAuth2 support\n\nIncludes Google + GitHub providers",
			Projects: map[string]models.BumpType{
				"auth": models.BumpMinor,
			},
			PR: &models.PullRequest{
				Number: 123,
				URL:    "https://github.com/org/repo/pull/123",
				Author: "alice",
			},
		},
		{
			ID:      "patch",
			Message: "Fix memory leak",
			Projects: map[string]models.BumpType{
				"auth": models.BumpPatch,
			},
		},
		{
			ID:      "major",
			Message: "Breaking API change",
			Projects: map[string]models.BumpType{
				"auth": models.BumpMajor,
			},
		},
	}

	t.Run("changelog preview", func(t *testing.T) {
		fs := filesystem.NewMockFileSystem()
		changelog := NewChangelog(fs)

		preview, err := changelog.FormatEntry(changesets, "auth", "/workspace")
		if err != nil {
			t.Fatalf("FormatEntry failed: %v", err)
		}
		snaps.MatchSnapshot(t, preview)
	})

	t.Run("changelog with project name", func(t *testing.T) {
		fs := filesystem.NewMockFileSystem()
		changelog := NewChangelog(fs)

		rootEntry := &ChangelogEntry{
			Version:    &models.Version{Major: 2, Minor: 0, Patch: 0},
			Date:       time.Date(2024, 12, 6, 0, 0, 0, 0, time.UTC),
			Changesets: changesets,
		}
		err := changelog.Append(".", "auth", rootEntry)
		assert.NoError(t, err, "Append should not return an error")

		content, err := fs.ReadFile("./CHANGELOG.md")
		assert.NoError(t, err, "ReadFile should not return an error")

		snaps.MatchSnapshot(t, string(content))
	})
}
