package models

import (
	"fmt"
)

// BumpType represents the type of version bump
type BumpType string

const (
	BumpPatch BumpType = "patch"
	BumpMinor BumpType = "minor"
	BumpMajor BumpType = "major"
)

// IsValid checks if the bump type is valid
func (b BumpType) IsValid() bool {
	switch b {
	case BumpPatch, BumpMinor, BumpMajor:
		return true
	default:
		return false
	}
}

// String returns the string representation of BumpType
func (b BumpType) String() string {
	return string(b)
}

// ParseBumpType parses a string into a BumpType
func ParseBumpType(s string) (BumpType, error) {
	bt := BumpType(s)
	if !bt.IsValid() {
		return "", fmt.Errorf("invalid bump type: %s (must be patch, minor, or major)", s)
	}
	return bt, nil
}

// PullRequest represents GitHub pull request metadata
type PullRequest struct {
	// Number is the PR number (e.g., 123)
	Number int

	// Title is the PR title
	Title string

	// URL is the full URL to the PR (e.g., https://github.com/owner/repo/pull/123)
	URL string

	// Author is the GitHub username of the PR author
	Author string

	// Labels are the labels assigned to the PR
	Labels []string
}

// Changeset represents a changeset file with its metadata
type Changeset struct {
	// ID is the unique identifier for this changeset (filename without extension)
	ID string

	// Projects maps project names to their bump types
	Projects map[string]BumpType

	// Message is the markdown content describing the change
	Message string

	// FilePath is the path to the changeset file
	FilePath string

	// PR contains optional pull request metadata (populated via GitHub API)
	PR *PullRequest
}

// NewChangeset creates a new Changeset instance
func NewChangeset(id string, projects map[string]BumpType, message string) *Changeset {
	return &Changeset{
		ID:       id,
		Projects: projects,
		Message:  message,
	}
}

// GetBumpForProject returns the bump type for a specific project
func (c *Changeset) GetBumpForProject(projectName string) (BumpType, bool) {
	bump, exists := c.Projects[projectName]
	return bump, exists
}

// AffectsProject checks if this changeset affects a specific project
func (c *Changeset) AffectsProject(projectName string) bool {
	_, exists := c.Projects[projectName]
	return exists
}
