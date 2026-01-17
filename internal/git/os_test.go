package git_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/jakoblorz/go-changesets/internal/git"
	"github.com/stretchr/testify/require"
)

// setupTestRepo creates a temporary git repository for testing
func setupTestRepo(t *testing.T) (*git.OSGitClient, string, func()) {
	t.Helper()

	// Check if git is available
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available in PATH")
	}

	// Create temp directory
	tmpDir := t.TempDir()

	// Initialize git repo with main as default branch
	runGitCmd(t, tmpDir, "init", "-b", "main")
	runGitCmd(t, tmpDir, "config", "user.name", "Test User")
	runGitCmd(t, tmpDir, "config", "user.email", "test@example.com")

	// Create initial commit (empty repos can't have branches)
	writeFile(t, tmpDir, "README.md", "# Test Repo")
	runGitCmd(t, tmpDir, "add", ".")
	runGitCmd(t, tmpDir, "commit", "-m", "Initial commit")

	// Create client
	client := git.NewOSGitClient()

	cleanup := func() {
		// Temp directory is automatically cleaned up by t.TempDir()
	}

	return client, tmpDir, cleanup
}

// runGitCmd runs a git command in the specified directory
func runGitCmd(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "git %v failed\nOutput: %s", args, output)
}

// writeFile writes content to a file
func writeFile(t *testing.T, dir, filename, content string) {
	t.Helper()
	path := filepath.Join(dir, filename)
	require.NoErrorf(t, os.WriteFile(path, []byte(content), 0644), "failed to write file %s", path)
}

// createCommit creates a new commit in the repo
func createCommit(t *testing.T, repoPath, message string) {
	t.Helper()
	filename := "file_" + message + ".txt"
	writeFile(t, repoPath, filename, "content for "+message)
	runGitCmd(t, repoPath, "add", ".")
	runGitCmd(t, repoPath, "commit", "-m", message)
}

// createBranch creates and checks out a new branch
func createBranch(t *testing.T, repoPath, branchName string) {
	t.Helper()
	runGitCmd(t, repoPath, "checkout", "-b", branchName)
}

// checkoutBranch switches to an existing branch
func checkoutBranch(t *testing.T, repoPath, branchName string) {
	t.Helper()
	runGitCmd(t, repoPath, "checkout", branchName)
}

// mergeBranch merges a branch into current branch
func mergeBranch(t *testing.T, repoPath, branchName string) {
	t.Helper()
	runGitCmd(t, repoPath, "merge", branchName, "--no-ff", "-m", "Merge "+branchName)
}

// TestOSGit_BasicTagOperations tests basic tag create/list/exists operations
func TestOSGit_BasicTagOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client, repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	// Change to repo directory for git operations
	originalDir, _ := os.Getwd()
	os.Chdir(repoPath)
	defer os.Chdir(originalDir)

	err := client.CreateTag("backend@v1.0.0", "Release 1.0.0")
	require.NoError(t, err)

	exists, err := client.TagExists("backend@v1.0.0")
	require.NoError(t, err)
	require.True(t, exists)

	exists, err = client.TagExists("nonexistent@v1.0.0")
	require.NoError(t, err)
	require.False(t, exists)

	err = client.CreateTag("backend@v1.1.0", "Release 1.1.0")
	require.NoError(t, err)

	tag, err := client.GetLatestTag("backend")
	require.NoError(t, err)
	require.Equal(t, "backend@v1.1.0", tag)

	err = client.CreateTag("backend@v1.1.0", "Duplicate")
	require.Error(t, err)
}

// TestOSGit_WildcardPatternMatching tests GetTagsWithPrefix with wildcards
func TestOSGit_WildcardPatternMatching(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client, repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	originalDir, _ := os.Getwd()
	os.Chdir(repoPath)
	defer os.Chdir(originalDir)

	// Create multiple tags for different projects
	client.CreateTag("backend@v1.0.0", "backend 1.0.0")
	client.CreateTag("backend@v1.1.0", "backend 1.1.0")
	client.CreateTag("backend@v1.2.0", "backend 1.2.0")
	client.CreateTag("www@v2.0.0", "WWW 2.0.0")
	client.CreateTag("api@v0.5.0", "API 0.5.0")

	tags, err := client.GetTagsWithPrefix("backend@v*")
	require.NoError(t, err)
	require.Equal(t, []string{"backend@v1.2.0", "backend@v1.1.0", "backend@v1.0.0"}, tags)

	for _, tag := range tags {
		require.NotEqual(t, "www@v2.0.0", tag)
	}

	wwwTags, err := client.GetTagsWithPrefix("www@v*")
	require.NoError(t, err)
	require.Equal(t, []string{"www@v2.0.0"}, wwwTags)
}

// TestOSGit_ExtractRCNumber tests RC number extraction from tags
func TestOSGit_ExtractRCNumber(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client, _, cleanup := setupTestRepo(t)
	defer cleanup()

	tests := []struct {
		tag      string
		expected int
		hasError bool
	}{
		{"backend@v1.2.0-rc0", 0, false},
		{"backend@v1.2.0-rc1", 1, false},
		{"backend@v1.2.0-rc5", 5, false},
		{"backend@v1.2.0-rc123", 123, false},
		{"backend@v1.2.0", -1, false},
		{"www@v2.0.0", -1, false},
		{"backend@v1.2.0-rc", -1, true},
		{"backend@v1.2.0-rcX", -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.tag, func(t *testing.T) {
			result, err := client.ExtractRCNumber(tt.tag)
			if tt.hasError {
				require.Error(t, err, "expected error for %s", tt.tag)
				return
			}
			require.NoError(t, err)
			require.Equalf(t, tt.expected, result, "tag %s", tt.tag)
		})
	}
}

// TestOSGit_TagAncestry_BranchDivergence tests that tags are only visible on their branch
func TestOSGit_TagAncestry_BranchDivergence(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client, repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	originalDir, _ := os.Getwd()
	os.Chdir(repoPath)
	defer os.Chdir(originalDir)

	// Main branch: Initial → A → B
	createCommit(t, repoPath, "commit_A")
	createCommit(t, repoPath, "commit_B")

	// Tag at B on main
	client.CreateTag("backend@v1.0.0", "Release 1.0.0")

	// Create canary branch from B
	createBranch(t, repoPath, "canary")

	// Canary: B → C → D
	createCommit(t, repoPath, "commit_C")
	createCommit(t, repoPath, "commit_D")

	// Switch back to main
	checkoutBranch(t, repoPath, "main")

	// Main: B → E → F
	createCommit(t, repoPath, "commit_E")
	createCommit(t, repoPath, "commit_F")

	// Tag at F on main (new tag not in canary's history)
	client.CreateTag("backend@v1.1.0", "Release 1.1.0")

	// Switch to canary
	checkoutBranch(t, repoPath, "canary")

	// From canary: should only see v1.0.0 (not v1.1.0)
	tags, err := client.GetTagsWithPrefix("backend@v*")
	require.NoError(t, err)

	require.Len(t, tags, 1)
	require.Equal(t, "backend@v1.0.0", tags[0])

	for _, tag := range tags {
		require.NotEqual(t, "backend@v1.1.0", tag, "canary branch should not see v1.1.0")
	}

	checkoutBranch(t, repoPath, "main")
	tags, err = client.GetTagsWithPrefix("backend@v*")
	require.NoError(t, err)

	require.Len(t, tags, 2)
	require.Contains(t, tags, "backend@v1.0.0")
	require.Contains(t, tags, "backend@v1.1.0")
}

// TestOSGit_RCTagFiltering tests that publish can filter out RC tags
func TestOSGit_RCTagFiltering(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client, repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	originalDir, _ := os.Getwd()
	os.Chdir(repoPath)
	defer os.Chdir(originalDir)

	// Create a mix of RC and final release tags
	client.CreateTag("backend@v1.2.0-rc0", "RC 0")
	client.CreateTag("backend@v1.2.0-rc1", "RC 1")
	client.CreateTag("backend@v1.2.0-rc2", "RC 2")
	client.CreateTag("backend@v1.2.0", "Final Release")

	allTags, err := client.GetTagsWithPrefix("backend@v*")
	require.NoError(t, err)

	require.Len(t, allTags, 4)

	var nonRCTags []string
	for _, tag := range allTags {
		rcNum, _ := client.ExtractRCNumber(tag)
		if rcNum < 0 {
			nonRCTags = append(nonRCTags, tag)
		}
	}

	require.Equal(t, []string{"backend@v1.2.0"}, nonRCTags)
}

// TestOSGit_MultipleRCIncrements tests creating multiple RC tags
func TestOSGit_MultipleRCIncrements(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client, repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	originalDir, _ := os.Getwd()
	os.Chdir(repoPath)
	defer os.Chdir(originalDir)

	// Create first RC
	client.CreateTag("backend@v1.3.0-rc0", "First RC")

	// Get RC tags for v1.3.0
	allTags, _ := client.GetTagsWithPrefix("backend@v1.3.0-rc*")
	require.Len(t, allTags, 1)

	highestRC := -1
	for _, tag := range allTags {
		rcNum, _ := client.ExtractRCNumber(tag)
		if rcNum > highestRC {
			highestRC = rcNum
		}
	}
	require.Equal(t, 0, highestRC)

	// Create second RC (increment)
	nextRC := highestRC + 1
	client.CreateTag("backend@v1.3.0-rc1", "Second RC")

	// Verify we now have 2 RCs
	allTags, _ = client.GetTagsWithPrefix("backend@v1.3.0-rc*")
	require.Len(t, allTags, 2)

	highestRC = -1
	for _, tag := range allTags {
		rcNum, _ := client.ExtractRCNumber(tag)
		if rcNum > highestRC {
			highestRC = rcNum
		}
	}
	require.Equal(t, nextRC, highestRC)

	client.CreateTag("backend@v1.3.0-rc2", "Third RC")

	allTags, _ = client.GetTagsWithPrefix("backend@v1.3.0-rc*")
	require.Len(t, allTags, 3)
}

// TestOSGit_CanaryWorkflow_Complete tests the full canary workflow
func TestOSGit_CanaryWorkflow_Complete(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client, repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	originalDir, _ := os.Getwd()
	os.Chdir(repoPath)
	defer os.Chdir(originalDir)

	// Step 1: Main branch has v1.0.0
	createCommit(t, repoPath, "main_v1.0.0")
	client.CreateTag("backend@v1.0.0", "Release 1.0.0")

	// Step 2: Create canary branch from this point
	createBranch(t, repoPath, "canary")

	// Step 3: Canary makes changes
	createCommit(t, repoPath, "canary_feature_1")
	createCommit(t, repoPath, "canary_feature_2")

	// Step 4: Create first snapshot on canary
	// (In real workflow, this would be calculated from changesets)
	client.CreateTag("backend@v1.1.0-rc0", "First RC")

	// Verify canary sees v1.0.0 and rc0
	tags, _ := client.GetTagsWithPrefix("backend@v*")
	require.Len(t, tags, 2)

	createCommit(t, repoPath, "canary_fix")
	client.CreateTag("backend@v1.1.0-rc1", "Second RC")

	tags, _ = client.GetTagsWithPrefix("backend@v*")
	require.Len(t, tags, 3)

	checkoutBranch(t, repoPath, "main")
	createCommit(t, repoPath, "main_other_feature")
	client.CreateTag("backend@v1.1.0", "Final release on main")

	checkoutBranch(t, repoPath, "canary")
	tags, _ = client.GetTagsWithPrefix("backend@v*")
	for _, tag := range tags {
		require.NotEqual(t, "backend@v1.1.0", tag, "canary should not see final release yet")
	}

	mergeBranch(t, repoPath, "main")

	tags, _ = client.GetTagsWithPrefix("backend@v*")
	require.Len(t, tags, 4)
	expectedTags := map[string]bool{
		"backend@v1.0.0":     false,
		"backend@v1.1.0-rc0": false,
		"backend@v1.1.0-rc1": false,
		"backend@v1.1.0":     false,
	}
	for _, tag := range tags {
		if _, exists := expectedTags[tag]; exists {
			expectedTags[tag] = true
		}
	}
	for tag, found := range expectedTags {
		require.Truef(t, found, "expected tag %s after merge", tag)
	}
}

// TestOSGit_MergeMainIntoCanary tests tag visibility after merge
func TestOSGit_MergeMainIntoCanary(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client, repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	originalDir, _ := os.Getwd()
	os.Chdir(repoPath)
	defer os.Chdir(originalDir)

	// Main: Initial → A (tag v1.0.0)
	createCommit(t, repoPath, "commit_A")
	client.CreateTag("backend@v1.0.0", "Release 1.0.0")

	// Create canary from A
	createBranch(t, repoPath, "canary")
	createCommit(t, repoPath, "canary_work")

	// Main continues: A → B (tag v1.1.0)
	checkoutBranch(t, repoPath, "main")
	createCommit(t, repoPath, "commit_B")
	client.CreateTag("backend@v1.1.0", "Release 1.1.0")

	// Canary before merge: should only see v1.0.0
	checkoutBranch(t, repoPath, "canary")
	tagsBefore, _ := client.GetTagsWithPrefix("backend@v*")
	require.Equal(t, []string{"backend@v1.0.0"}, tagsBefore)

	mergeBranch(t, repoPath, "main")

	tagsAfter, _ := client.GetTagsWithPrefix("backend@v*")
	require.Len(t, tagsAfter, 2)
	require.Contains(t, tagsAfter, "backend@v1.0.0")
	require.Contains(t, tagsAfter, "backend@v1.1.0")
}

// TestOSGit_TagAnnotations tests that tag messages are stored correctly
func TestOSGit_TagAnnotations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client, repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	originalDir, _ := os.Getwd()
	os.Chdir(repoPath)
	defer os.Chdir(originalDir)

	// Create annotated tag with message
	message := "Release 1.0.0\n\nThis is a test release with\nmultiple lines"
	require.NoError(t, client.CreateTag("backend@v1.0.0", message))

	annotation, err := client.GetTagAnnotation("backend@v1.0.0")
	require.NoError(t, err)
	require.Equal(t, message, annotation)
}
