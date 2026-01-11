package git

import (
	"context"
)

// GitClient provides an abstraction over git operations for testability
//
// IMPORTANT: All tag operations are branch-aware and only return tags
// that are reachable from the current HEAD (using git's --merged HEAD).
//
// This means:
//   - On 'main': returns all tags merged into main
//   - On 'canary': only returns tags in canary's ancestry
//   - After merge: returns tags from both branches
//
// This branch-awareness is critical for snapshot/RC workflows where
// different branches may have different sets of published versions.
type GitClient interface {
	// Tag operations (all branch-aware via --merged HEAD)
	GetLatestTag(projectName string) (string, error)
	GetTagsWithPrefix(prefix string) ([]string, error)
	CreateTag(tagName, message string) error
	PushTag(tagName string) error
	TagExists(tagName string) (bool, error)
	GetTagAnnotation(tagName string) (string, error)

	// Repository operations
	IsGitRepo() (bool, error)
	GetCurrentBranch() (string, error)

	// RC tag operations
	ExtractRCNumber(tag string) (int, error)

	// File history operations
	GetFileCreationCommit(filePath string) (string, error)
	GetCommitMessage(commitSHA string) (string, error)

	// Context support for network operations
	WithContext(ctx context.Context) GitClient
}
