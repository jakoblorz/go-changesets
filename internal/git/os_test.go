package git_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/jakoblorz/go-changesets/internal/git"
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
	if err != nil {
		t.Fatalf("git %v failed: %v\nOutput: %s", args, err, output)
	}
}

// writeFile writes content to a file
func writeFile(t *testing.T, dir, filename, content string) {
	t.Helper()
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write file %s: %v", path, err)
	}
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

	// Create a tag
	err := client.CreateTag("backend@v1.0.0", "Release 1.0.0")
	if err != nil {
		t.Fatalf("CreateTag failed: %v", err)
	}

	// Verify tag exists
	exists, err := client.TagExists("backend@v1.0.0")
	if err != nil {
		t.Fatalf("TagExists failed: %v", err)
	}
	if !exists {
		t.Error("Expected tag to exist")
	}

	// Verify non-existent tag
	exists, err = client.TagExists("nonexistent@v1.0.0")
	if err != nil {
		t.Fatalf("TagExists failed: %v", err)
	}
	if exists {
		t.Error("Expected tag not to exist")
	}

	// Create second tag
	err = client.CreateTag("backend@v1.1.0", "Release 1.1.0")
	if err != nil {
		t.Fatalf("CreateTag failed: %v", err)
	}

	// Get latest tag
	tag, err := client.GetLatestTag("backend")
	if err != nil {
		t.Fatalf("GetLatestTag failed: %v", err)
	}
	if tag != "backend@v1.1.0" {
		t.Errorf("Expected backend@v1.1.0, got %s", tag)
	}

	// Try to create duplicate tag (should fail)
	err = client.CreateTag("backend@v1.1.0", "Duplicate")
	if err == nil {
		t.Error("Expected error when creating duplicate tag")
	}
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

	// Get all backend tags with wildcard
	tags, err := client.GetTagsWithPrefix("backend@v*")
	if err != nil {
		t.Fatalf("GetTagsWithPrefix failed: %v", err)
	}

	// Should get all backend tags, sorted by version (descending)
	if len(tags) != 3 {
		t.Errorf("Expected 3 backend tags, got %d: %v", len(tags), tags)
	}

	// Verify correct tags
	expectedTags := []string{"backend@v1.2.0", "backend@v1.1.0", "backend@v1.0.0"}
	for i, expected := range expectedTags {
		if i >= len(tags) || tags[i] != expected {
			t.Errorf("Expected tags[%d] = %s, got %v", i, expected, tags)
			break
		}
	}

	// Verify www tags not included
	for _, tag := range tags {
		if tag == "www@v2.0.0" {
			t.Error("Expected www tag not to be included in backend results")
		}
	}

	// Get www tags
	wwwTags, err := client.GetTagsWithPrefix("www@v*")
	if err != nil {
		t.Fatalf("GetTagsWithPrefix failed: %v", err)
	}
	if len(wwwTags) != 1 || wwwTags[0] != "www@v2.0.0" {
		t.Errorf("Expected [www@v2.0.0], got %v", wwwTags)
	}
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
		{"backend@v1.2.0", -1, false},    // Not an RC tag
		{"www@v2.0.0", -1, false},        // Not an RC tag
		{"backend@v1.2.0-rc", -1, true},  // Invalid (no number)
		{"backend@v1.2.0-rcX", -1, true}, // Invalid (not a number)
	}

	for _, tt := range tests {
		t.Run(tt.tag, func(t *testing.T) {
			result, err := client.ExtractRCNumber(tt.tag)

			if tt.hasError {
				if err == nil {
					t.Errorf("Expected error for %s, got nil", tt.tag)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for %s: %v", tt.tag, err)
				}
				if result != tt.expected {
					t.Errorf("Expected %d for %s, got %d", tt.expected, tt.tag, result)
				}
			}
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
	if err != nil {
		t.Fatalf("GetTagsWithPrefix failed: %v", err)
	}

	if len(tags) != 1 {
		t.Errorf("Expected 1 tag from canary, got %d: %v", len(tags), tags)
	}

	if len(tags) > 0 && tags[0] != "backend@v1.0.0" {
		t.Errorf("Expected backend@v1.0.0 from canary, got %s", tags[0])
	}

	// Verify v1.1.0 is NOT in the list
	for _, tag := range tags {
		if tag == "backend@v1.1.0" {
			t.Error("canary branch should NOT see backend@v1.1.0 (not an ancestor)")
		}
	}

	// Switch back to main and verify it sees both tags
	checkoutBranch(t, repoPath, "main")
	tags, err = client.GetTagsWithPrefix("backend@v*")
	if err != nil {
		t.Fatalf("GetTagsWithPrefix failed: %v", err)
	}

	if len(tags) != 2 {
		t.Errorf("Expected 2 tags from main, got %d: %v", len(tags), tags)
	}

	// Both tags should be visible from main
	foundV1_0 := false
	foundV1_1 := false
	for _, tag := range tags {
		if tag == "backend@v1.0.0" {
			foundV1_0 = true
		}
		if tag == "backend@v1.1.0" {
			foundV1_1 = true
		}
	}

	if !foundV1_0 || !foundV1_1 {
		t.Errorf("main should see both tags, got: %v", tags)
	}
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

	// Get all tags
	allTags, err := client.GetTagsWithPrefix("backend@v*")
	if err != nil {
		t.Fatalf("GetTagsWithPrefix failed: %v", err)
	}

	if len(allTags) != 4 {
		t.Errorf("Expected 4 total tags, got %d: %v", len(allTags), allTags)
	}

	// Filter out RC tags (simulating what publish command does)
	var nonRCTags []string
	for _, tag := range allTags {
		rcNum, _ := client.ExtractRCNumber(tag)
		if rcNum < 0 { // Not an RC tag
			nonRCTags = append(nonRCTags, tag)
		}
	}

	// Should only have the final release
	if len(nonRCTags) != 1 {
		t.Errorf("Expected 1 non-RC tag, got %d: %v", len(nonRCTags), nonRCTags)
	}

	if len(nonRCTags) > 0 && nonRCTags[0] != "backend@v1.2.0" {
		t.Errorf("Expected backend@v1.2.0, got %s", nonRCTags[0])
	}
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
	if len(allTags) != 1 {
		t.Errorf("Expected 1 RC tag, got %d", len(allTags))
	}

	// Find highest RC number (simulating snapshot command logic)
	highestRC := -1
	for _, tag := range allTags {
		rcNum, _ := client.ExtractRCNumber(tag)
		if rcNum > highestRC {
			highestRC = rcNum
		}
	}

	if highestRC != 0 {
		t.Errorf("Expected highest RC to be 0, got %d", highestRC)
	}

	// Create second RC (increment)
	nextRC := highestRC + 1
	client.CreateTag("backend@v1.3.0-rc1", "Second RC")

	// Verify we now have 2 RCs
	allTags, _ = client.GetTagsWithPrefix("backend@v1.3.0-rc*")
	if len(allTags) != 2 {
		t.Errorf("Expected 2 RC tags, got %d: %v", len(allTags), allTags)
	}

	// Find highest RC again
	highestRC = -1
	for _, tag := range allTags {
		rcNum, _ := client.ExtractRCNumber(tag)
		if rcNum > highestRC {
			highestRC = rcNum
		}
	}

	if highestRC != nextRC {
		t.Errorf("Expected highest RC to be %d, got %d", nextRC, highestRC)
	}

	// Create third RC
	client.CreateTag("backend@v1.3.0-rc2", "Third RC")

	allTags, _ = client.GetTagsWithPrefix("backend@v1.3.0-rc*")
	if len(allTags) != 3 {
		t.Errorf("Expected 3 RC tags, got %d", len(allTags))
	}
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
	if len(tags) != 2 {
		t.Errorf("Canary should see 2 tags, got %d: %v", len(tags), tags)
	}

	// Step 5: More changes on canary, create second snapshot
	createCommit(t, repoPath, "canary_fix")
	client.CreateTag("backend@v1.1.0-rc1", "Second RC")

	tags, _ = client.GetTagsWithPrefix("backend@v*")
	if len(tags) != 3 {
		t.Errorf("Canary should see 3 tags, got %d: %v", len(tags), tags)
	}

	// Step 6: Meanwhile, main continues
	checkoutBranch(t, repoPath, "main")
	createCommit(t, repoPath, "main_other_feature")
	client.CreateTag("backend@v1.1.0", "Final release on main")

	// Step 7: Canary doesn't see main's v1.1.0 yet
	checkoutBranch(t, repoPath, "canary")
	tags, _ = client.GetTagsWithPrefix("backend@v*")
	foundFinalRelease := false
	for _, tag := range tags {
		if tag == "backend@v1.1.0" {
			foundFinalRelease = true
		}
	}
	if foundFinalRelease {
		t.Error("Canary should NOT see main's v1.1.0 (not merged yet)")
	}

	// Step 8: Merge main into canary
	mergeBranch(t, repoPath, "main")

	// Step 9: Now canary sees all tags
	tags, _ = client.GetTagsWithPrefix("backend@v*")
	if len(tags) != 4 {
		t.Errorf("After merge, canary should see 4 tags, got %d: %v", len(tags), tags)
	}

	// Verify all expected tags
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
		if !found {
			t.Errorf("Expected to find tag %s after merge, but didn't", tag)
		}
	}

	// Step 10: Test filtering logic (publish should ignore RCs)
	var nonRCTags []string
	for _, tag := range tags {
		rcNum, _ := client.ExtractRCNumber(tag)
		if rcNum < 0 {
			nonRCTags = append(nonRCTags, tag)
		}
	}

	// Should have v1.0.0 and v1.1.0
	if len(nonRCTags) != 2 {
		t.Errorf("Expected 2 non-RC tags, got %d: %v", len(nonRCTags), nonRCTags)
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
	if len(tagsBefore) != 1 || tagsBefore[0] != "backend@v1.0.0" {
		t.Errorf("Before merge, canary should see [backend@v1.0.0], got %v", tagsBefore)
	}

	// Merge main into canary
	mergeBranch(t, repoPath, "main")

	// Canary after merge: should see both v1.0.0 and v1.1.0
	tagsAfter, _ := client.GetTagsWithPrefix("backend@v*")
	if len(tagsAfter) != 2 {
		t.Errorf("After merge, canary should see 2 tags, got %d: %v", len(tagsAfter), tagsAfter)
	}

	found_v1_0 := false
	found_v1_1 := false
	for _, tag := range tagsAfter {
		if tag == "backend@v1.0.0" {
			found_v1_0 = true
		}
		if tag == "backend@v1.1.0" {
			found_v1_1 = true
		}
	}

	if !found_v1_0 || !found_v1_1 {
		t.Errorf("After merge, expected to find both v1.0.0 and v1.1.0, got: %v", tagsAfter)
	}
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
	err := client.CreateTag("backend@v1.0.0", message)
	if err != nil {
		t.Fatalf("CreateTag failed: %v", err)
	}

	// Retrieve annotation
	annotation, err := client.GetTagAnnotation("backend@v1.0.0")
	if err != nil {
		t.Fatalf("GetTagAnnotation failed: %v", err)
	}

	if annotation != message {
		t.Errorf("Expected message %q, got %q", message, annotation)
	}
}
