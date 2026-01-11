package git

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// OSGitClient implements GitClient using real git commands
type OSGitClient struct {
	ctx context.Context
}

// NewOSGitClient creates a new OSGitClient
func NewOSGitClient() *OSGitClient {
	return &OSGitClient{
		ctx: context.Background(),
	}
}

// WithContext returns a new client with the given context
func (g *OSGitClient) WithContext(ctx context.Context) GitClient {
	return &OSGitClient{
		ctx: ctx,
	}
}

// GetLatestTag returns the latest tag for a project on the current branch
// Tags are expected to be in the format: {projectName}@v{version}
func (g *OSGitClient) GetLatestTag(projectName string) (string, error) {
	// Get all tags matching the project pattern, sorted by version, merged into current branch
	pattern := fmt.Sprintf("%s@v*", projectName)
	cmd := exec.CommandContext(g.ctx, "git", "tag", "-l", pattern, "--sort=-version:refname", "--merged", "HEAD")

	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to list tags: %w", err)
	}

	// Get the first line (latest version)
	output := strings.TrimSpace(out.String())
	if output == "" {
		return "", fmt.Errorf("no tags found for project %s", projectName)
	}

	lines := strings.Split(output, "\n")
	return strings.TrimSpace(lines[0]), nil
}

// CreateTag creates an annotated tag
func (g *OSGitClient) CreateTag(tagName, message string) error {
	cmd := exec.CommandContext(g.ctx, "git", "tag", "-a", tagName, "-m", message)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create tag %s: %w: %s", tagName, err, stderr.String())
	}

	return nil
}

// PushTag pushes a tag to the remote
func (g *OSGitClient) PushTag(tagName string) error {
	cmd := exec.CommandContext(g.ctx, "git", "push", "origin", tagName)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to push tag %s: %w: %s", tagName, err, stderr.String())
	}

	return nil
}

// TagExists checks if a tag exists locally
func (g *OSGitClient) TagExists(tagName string) (bool, error) {
	cmd := exec.CommandContext(g.ctx, "git", "tag", "-l", tagName)

	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return false, fmt.Errorf("failed to check tag: %w", err)
	}

	return strings.TrimSpace(out.String()) != "", nil
}

// GetTagAnnotation returns the annotation message of a tag
func (g *OSGitClient) GetTagAnnotation(tagName string) (string, error) {
	cmd := exec.CommandContext(g.ctx, "git", "tag", "-l", "--format=%(contents)", tagName)

	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get tag annotation: %w", err)
	}

	return strings.TrimSpace(out.String()), nil
}

// GetTagsWithPrefix returns all tags matching the prefix that are reachable from HEAD
// Uses --merged HEAD to ensure only tags in current branch's ancestry are returned
func (g *OSGitClient) GetTagsWithPrefix(prefix string) ([]string, error) {
	cmd := exec.CommandContext(g.ctx, "git", "tag", "-l", prefix, "--sort=-version:refname", "--merged", "HEAD")

	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to list reachable tags: %w", err)
	}

	output := strings.TrimSpace(out.String())
	if output == "" {
		return []string{}, nil
	}

	tags := strings.Split(output, "\n")
	var result []string
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag != "" {
			result = append(result, tag)
		}
	}

	return result, nil
}

// ExtractRCNumber extracts the RC number from a tag
// Examples:
//   - "backend@v1.3.0-rc5" -> 5, nil
//   - "backend@v1.3.0" -> -1, nil (not an RC)
//   - "backend@v1.3.0-rc" -> -1, error
func (g *OSGitClient) ExtractRCNumber(tag string) (int, error) {
	// Find the -rc prefix
	rcIdx := strings.Index(tag, "-rc")
	if rcIdx == -1 {
		return -1, nil // Not an RC tag
	}

	// Extract everything after "-rc"
	rcSuffix := tag[rcIdx+3:]
	if rcSuffix == "" {
		return -1, fmt.Errorf("invalid RC tag format: %s (expected -rc{number})", tag)
	}

	// Parse the number
	num, err := strconv.Atoi(rcSuffix)
	if err != nil {
		return -1, fmt.Errorf("invalid RC number in tag %s: %w", tag, err)
	}

	return num, nil
}

// IsGitRepo checks if the current directory is a git repository
func (g *OSGitClient) IsGitRepo() (bool, error) {
	cmd := exec.CommandContext(g.ctx, "git", "rev-parse", "--git-dir")

	if err := cmd.Run(); err != nil {
		// Not a git repo
		return false, nil
	}

	return true, nil
}

// GetCurrentBranch returns the current git branch name
func (g *OSGitClient) GetCurrentBranch() (string, error) {
	cmd := exec.CommandContext(g.ctx, "git", "branch", "--show-current")

	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}

	return strings.TrimSpace(out.String()), nil
}

// GetFileCreationCommit returns the commit SHA that added a file
// Returns empty string if file doesn't exist in git history
func (g *OSGitClient) GetFileCreationCommit(filePath string) (string, error) {
	// git log --follow --diff-filter=A --pretty=format:"%H" -1 {file}
	// --follow: track file renames
	// --diff-filter=A: only show when file was Added
	// -1: only first (creation) commit
	cmd := exec.CommandContext(g.ctx, "git", "log", "--follow", "--diff-filter=A",
		"--pretty=format:%H", "-1", filePath)

	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		// File not in git history or other error
		return "", nil
	}

	return strings.TrimSpace(out.String()), nil
}

// GetCommitMessage returns the commit message for a given SHA
func (g *OSGitClient) GetCommitMessage(commitSHA string) (string, error) {
	if commitSHA == "" {
		return "", fmt.Errorf("commit SHA cannot be empty")
	}

	// git log --format=%s -n 1 {sha}
	// %s = subject (first line of commit message)
	cmd := exec.CommandContext(g.ctx, "git", "log", "--format=%s", "-n", "1", commitSHA)

	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get commit message: %w", err)
	}

	return strings.TrimSpace(out.String()), nil
}
