package models

import (
	"testing"
)

func TestChangeset_FormatPRSuffix(t *testing.T) {
	tests := []struct {
		name        string
		pr          *PullRequest
		includeLink bool
		expected    string
	}{
		{
			name:        "nil PR returns empty",
			pr:          nil,
			includeLink: false,
			expected:    "",
		},
		{
			name:        "nil PR with link returns empty",
			pr:          nil,
			includeLink: true,
			expected:    "",
		},
		{
			name: "PR without link",
			pr: &PullRequest{
				Number: 123,
				Title:  "Fix bug",
				URL:    "https://github.com/owner/repo/pull/123",
				Author: "alice",
			},
			includeLink: false,
			expected:    "(#123 by @alice)",
		},
		{
			name: "PR with link",
			pr: &PullRequest{
				Number: 456,
				Title:  "Add feature",
				URL:    "https://github.com/owner/repo/pull/456",
				Author: "bob",
			},
			includeLink: true,
			expected:    "[#456](https://github.com/owner/repo/pull/456) by @bob",
		},
		{
			name: "PR with link but empty URL falls back to plain format",
			pr: &PullRequest{
				Number: 789,
				Title:  "Update docs",
				URL:    "",
				Author: "charlie",
			},
			includeLink: true,
			expected:    "(#789 by @charlie)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := &Changeset{
				ID:       "test-changeset",
				Message:  "Test message",
				Projects: map[string]BumpType{"test": BumpPatch},
				PR:       tt.pr,
			}

			result := cs.FormatPRSuffix(tt.includeLink)
			if result != tt.expected {
				t.Errorf("FormatPRSuffix(%v) = %q, want %q", tt.includeLink, result, tt.expected)
			}
		})
	}
}

func TestChangeset_AffectsProject(t *testing.T) {
	cs := &Changeset{
		ID:      "test",
		Message: "Test",
		Projects: map[string]BumpType{
			"project-a": BumpMajor,
			"project-b": BumpPatch,
		},
	}

	if !cs.AffectsProject("project-a") {
		t.Error("AffectsProject(project-a) = false, want true")
	}

	if !cs.AffectsProject("project-b") {
		t.Error("AffectsProject(project-b) = false, want true")
	}

	if cs.AffectsProject("project-c") {
		t.Error("AffectsProject(project-c) = true, want false")
	}
}

func TestChangeset_GetBumpForProject(t *testing.T) {
	cs := &Changeset{
		ID:      "test",
		Message: "Test",
		Projects: map[string]BumpType{
			"project-a": BumpMajor,
			"project-b": BumpMinor,
		},
	}

	bump, exists := cs.GetBumpForProject("project-a")
	if !exists {
		t.Error("GetBumpForProject(project-a) exists = false, want true")
	}
	if bump != BumpMajor {
		t.Errorf("GetBumpForProject(project-a) = %s, want %s", bump, BumpMajor)
	}

	bump, exists = cs.GetBumpForProject("project-b")
	if !exists {
		t.Error("GetBumpForProject(project-b) exists = false, want true")
	}
	if bump != BumpMinor {
		t.Errorf("GetBumpForProject(project-b) = %s, want %s", bump, BumpMinor)
	}

	_, exists = cs.GetBumpForProject("nonexistent")
	if exists {
		t.Error("GetBumpForProject(nonexistent) exists = true, want false")
	}
}
