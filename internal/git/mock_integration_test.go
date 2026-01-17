//go:build integration

package git_test

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/jakoblorz/go-changesets/internal/git"
	"github.com/stretchr/testify/require"
)

// TestComparison_BasicTagOperations compares mock and OS git for basic tag operations
func TestComparison_BasicTagOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping comparison test in short mode")
	}

	// Setup OS git client
	osClient, repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	originalDir, _ := os.Getwd()
	os.Chdir(repoPath)
	defer os.Chdir(originalDir)

	// Setup mock git client
	mockClient := git.NewMockGitClient()

	// Test 1: Create tag
	osErr1 := osClient.CreateTag("backend@v1.0.0", "Release 1.0.0")
	mockErr1 := mockClient.CreateTag("backend@v1.0.0", "Release 1.0.0")

	if (osErr1 == nil) != (mockErr1 == nil) {
		t.Errorf("CreateTag error mismatch: OS=%v, Mock=%v", osErr1, mockErr1)
	}

	// Test 2: TagExists
	osExists, osErr2 := osClient.TagExists("backend@v1.0.0")
	mockExists, mockErr2 := mockClient.TagExists("backend@v1.0.0")

	if osExists != mockExists {
		t.Errorf("TagExists mismatch: OS=%v, Mock=%v", osExists, mockExists)
	}
	if (osErr2 == nil) != (mockErr2 == nil) {
		t.Errorf("TagExists error mismatch: OS=%v, Mock=%v", osErr2, mockErr2)
	}

	// Test 3: Create second tag
	createCommit(t, repoPath, "Work")
	mockClient.CreateCommit("Work")

	osClient.CreateTag("backend@v1.1.0", "Release 1.1.0")
	mockClient.CreateTag("backend@v1.1.0", "Release 1.1.0")

	// Test 4: GetLatestTag
	osTag, osErr3 := osClient.GetLatestTag("backend")
	mockTag, mockErr3 := mockClient.GetLatestTag("backend")

	if osTag != mockTag {
		t.Errorf("GetLatestTag mismatch: OS=%s, Mock=%s", osTag, mockTag)
	}
	if (osErr3 == nil) != (mockErr3 == nil) {
		t.Errorf("GetLatestTag error mismatch: OS=%v, Mock=%v", osErr3, mockErr3)
	}

	// Test 5: GetTagsWithPrefix
	osTags, osErr4 := osClient.GetTagsWithPrefix("backend@v*")
	mockTags, mockErr4 := mockClient.GetTagsWithPrefix("backend@v*")

	if len(osTags) != len(mockTags) {
		t.Errorf("GetTagsWithPrefix length mismatch: OS=%d, Mock=%d", len(osTags), len(mockTags))
	}

	for i := 0; i < len(osTags) && i < len(mockTags); i++ {
		if osTags[i] != mockTags[i] {
			t.Errorf("GetTagsWithPrefix[%d] mismatch: OS=%s, Mock=%s", i, osTags[i], mockTags[i])
		}
	}

	if (osErr4 == nil) != (mockErr4 == nil) {
		t.Errorf("GetTagsWithPrefix error mismatch: OS=%v, Mock=%v", osErr4, mockErr4)
	}
}

func TestComparison_WildcardPatterns(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping comparison test in short mode")
	}

	osClient, repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	originalDir, _ := os.Getwd()
	os.Chdir(repoPath)
	defer os.Chdir(originalDir)

	mockClient := git.NewMockGitClient()

	// Create identical tags on both
	tags := []struct {
		name    string
		message string
	}{
		{"backend@v1.0.0", "backend 1.0.0"},
		{"backend@v1.1.0", "backend 1.1.0"},
		{"backend@v1.2.0", "backend 1.2.0"},
		{"www@v2.0.0", "WWW 2.0.0"},
		{"api@v0.5.0", "API 0.5.0"},
	}

	for i, tag := range tags {
		if i > 0 {
			createCommit(t, repoPath, "Commit "+tag.name)
			mockClient.CreateCommit("Commit " + tag.name)
		}
		osClient.CreateTag(tag.name, tag.message)
		mockClient.CreateTag(tag.name, tag.message)
	}

	// Test pattern matching for each project
	patterns := []string{
		"backend@v*",
		"www@v*",
		"api@v*",
	}

	for _, pattern := range patterns {
		osTags, _ := osClient.GetTagsWithPrefix(pattern)
		mockTags, _ := mockClient.GetTagsWithPrefix(pattern)

		if len(osTags) != len(mockTags) {
			t.Errorf("Pattern %s: length mismatch OS=%d, Mock=%d", pattern, len(osTags), len(mockTags))
			continue
		}

		for i := 0; i < len(osTags); i++ {
			if osTags[i] != mockTags[i] {
				t.Errorf("Pattern %s[%d]: OS=%s, Mock=%s", pattern, i, osTags[i], mockTags[i])
			}
		}
	}
}

func TestComparison_RCExtraction(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping comparison test in short mode")
	}

	osClient, _, cleanup := setupTestRepo(t)
	defer cleanup()

	mockClient := git.NewMockGitClient()

	testCases := []string{
		"backend@v1.2.0-rc0",
		"backend@v1.2.0-rc5",
		"backend@v1.2.0-rc123",
		"backend@v1.2.0",
		"www@v2.0.0",
		"backend@v1.2.0-rc",
		"backend@v1.2.0-rcX",
	}

	for _, tag := range testCases {
		osNum, osErr := osClient.ExtractRCNumber(tag)
		mockNum, mockErr := mockClient.ExtractRCNumber(tag)

		if osNum != mockNum {
			t.Errorf("ExtractRCNumber(%s): OS=%d, Mock=%d", tag, osNum, mockNum)
		}

		if (osErr == nil) != (mockErr == nil) {
			t.Errorf("ExtractRCNumber(%s) error mismatch: OS=%v, Mock=%v", tag, osErr, mockErr)
		}
	}
}

func TestComparison_TagSorting(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping comparison test in short mode")
	}

	osClient, repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	originalDir, _ := os.Getwd()
	os.Chdir(repoPath)
	defer os.Chdir(originalDir)

	mockClient := git.NewMockGitClient()

	// Create tags in random order
	tagOrder := []string{
		"backend@v1.5.0",
		"backend@v1.0.0",
		"backend@v1.2.0",
		"backend@v1.10.0",
		"backend@v1.1.0",
	}

	for i, tag := range tagOrder {
		if i > 0 {
			createCommit(t, repoPath, "Commit for "+tag)
			mockClient.CreateCommit("Commit for " + tag)
		}
		osClient.CreateTag(tag, "Release "+tag)
		mockClient.CreateTag(tag, "Release "+tag)
	}

	// Get tags - should be sorted the same way
	osTags, _ := osClient.GetTagsWithPrefix("backend@v*")
	mockTags, _ := mockClient.GetTagsWithPrefix("backend@v*")

	require.Lenf(t, osTags, len(mockTags), "Tag count mismatch: OS=%d, Mock=%d", len(osTags), len(mockTags))

	for i := 0; i < len(osTags); i++ {
		if osTags[i] != mockTags[i] {
			t.Errorf("Sort order[%d]: OS=%s, Mock=%s", i, osTags[i], mockTags[i])
			t.Logf("Full OS order: %v", osTags)
			t.Logf("Full Mock order: %v", mockTags)
			break
		}
	}
}

func TestComparison_Annotations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping comparison test in short mode")
	}

	osClient, repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	originalDir, _ := os.Getwd()
	os.Chdir(repoPath)
	defer os.Chdir(originalDir)

	mockClient := git.NewMockGitClient()

	// Create tags with multiline messages
	message := "Release 1.0.0\n\nChangelog:\n- Feature A\n- Feature B"

	osClient.CreateTag("backend@v1.0.0", message)
	mockClient.CreateTag("backend@v1.0.0", message)

	// Retrieve annotations
	osAnnotation, osErr := osClient.GetTagAnnotation("backend@v1.0.0")
	mockAnnotation, mockErr := mockClient.GetTagAnnotation("backend@v1.0.0")

	if osAnnotation != mockAnnotation {
		t.Errorf("GetTagAnnotation mismatch:\nOS=%q\nMock=%q", osAnnotation, mockAnnotation)
	}

	if (osErr == nil) != (mockErr == nil) {
		t.Errorf("GetTagAnnotation error mismatch: OS=%v, Mock=%v", osErr, mockErr)
	}
}

func TestComparison_MultipleProjects(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping comparison test in short mode")
	}

	osClient, repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	originalDir, _ := os.Getwd()
	os.Chdir(repoPath)
	defer os.Chdir(originalDir)

	mockClient := git.NewMockGitClient()

	// Create tags for multiple projects
	projects := []struct {
		project string
		version string
	}{
		{"backend", "1.0.0"},
		{"backend", "1.1.0"},
		{"www", "2.0.0"},
		{"www", "2.1.0"},
		{"api", "0.5.0"},
	}

	for i, p := range projects {
		if i > 0 {
			createCommit(t, repoPath, p.project+"-"+p.version)
			mockClient.CreateCommit(p.project + "-" + p.version)
		}
		tagName := p.project + "@v" + p.version
		osClient.CreateTag(tagName, "Release "+p.version)
		mockClient.CreateTag(tagName, "Release "+p.version)
	}

	// Verify each project returns same tags
	for _, project := range []string{"backend", "www", "api"} {
		pattern := project + "@v*"

		osTags, osErr := osClient.GetTagsWithPrefix(pattern)
		mockTags, mockErr := mockClient.GetTagsWithPrefix(pattern)

		if (osErr == nil) != (mockErr == nil) {
			t.Errorf("Project %s error mismatch: OS=%v, Mock=%v", project, osErr, mockErr)
			continue
		}

		if len(osTags) != len(mockTags) {
			t.Errorf("Project %s count mismatch: OS=%d, Mock=%d", project, len(osTags), len(mockTags))
			continue
		}

		for i := 0; i < len(osTags); i++ {
			if osTags[i] != mockTags[i] {
				t.Errorf("Project %s[%d]: OS=%s, Mock=%s", project, i, osTags[i], mockTags[i])
			}
		}
	}
}

func TestComparison_FileCreationCommit(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping comparison test in short mode")
	}

	osClient, repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	originalDir, _ := os.Getwd()
	os.Chdir(repoPath)
	defer os.Chdir(originalDir)

	mockClient := git.NewMockGitClient()

	// Create .changeset directory
	os.MkdirAll(repoPath+"/.changeset", 0755)

	// Create a file in initial commit
	filePath1 := ".changeset/test1.md"
	writeFile(t, repoPath, filePath1, "Test changeset 1")
	createCommit(t, repoPath, "Add test1 changeset")
	initialCommit := getCurrentCommit(t, repoPath)

	// Track in mock
	mockCommit1 := mockClient.CreateCommit("Add test1 changeset")
	mockClient.SetFileCreationCommit(filePath1, mockCommit1)

	// Test: Get creation commit for first file
	osCommit1, osErr1 := osClient.GetFileCreationCommit(filePath1)
	mockCommit1Retrieved, mockErr1 := mockClient.GetFileCreationCommit(filePath1)

	if osCommit1 != initialCommit {
		t.Errorf("OS GetFileCreationCommit returned wrong commit: got=%s, want=%s", osCommit1, initialCommit)
	}

	if mockCommit1Retrieved != mockCommit1 {
		t.Errorf("Mock GetFileCreationCommit: got=%s, want=%s", mockCommit1Retrieved, mockCommit1)
	}

	if (osErr1 == nil) != (mockErr1 == nil) {
		t.Errorf("GetFileCreationCommit error mismatch: OS=%v, Mock=%v", osErr1, mockErr1)
	}

	// Create another file in new commit
	filePath2 := ".changeset/test2.md"
	writeFile(t, repoPath, filePath2, "Test changeset 2")
	createCommit(t, repoPath, "Add test2 changeset")
	secondCommit := getCurrentCommit(t, repoPath)

	mockCommit2 := mockClient.CreateCommit("Add test2 changeset")
	mockClient.SetFileCreationCommit(filePath2, mockCommit2)

	// Test: Get creation commit for second file
	osCommit2, _ := osClient.GetFileCreationCommit(filePath2)
	mockCommit2Retrieved, _ := mockClient.GetFileCreationCommit(filePath2)

	if osCommit2 != secondCommit {
		t.Errorf("OS GetFileCreationCommit for file2: got=%s, want=%s", osCommit2, secondCommit)
	}

	if mockCommit2Retrieved != mockCommit2 {
		t.Errorf("Mock GetFileCreationCommit for file2: got=%s, want=%s", mockCommit2Retrieved, mockCommit2)
	}

	// Test: Verify files have different creation commits
	if osCommit1 == osCommit2 {
		t.Error("Expected files to have different creation commits")
	}

	if mockCommit1 == mockCommit2 {
		t.Error("Mock: Expected files to have different creation commits")
	}

	// Test: Non-existent file
	osCommit3, osErr3 := osClient.GetFileCreationCommit("nonexistent.md")
	mockCommit3, mockErr3 := mockClient.GetFileCreationCommit("nonexistent.md")

	if osCommit3 != "" {
		t.Errorf("OS should return empty string for nonexistent file, got: %s", osCommit3)
	}

	if mockCommit3 != "" {
		t.Errorf("Mock should return empty string for nonexistent file, got: %s", mockCommit3)
	}

	if (osErr3 == nil) != (mockErr3 == nil) {
		t.Errorf("Nonexistent file error mismatch: OS=%v, Mock=%v", osErr3, mockErr3)
	}
}

func TestComparison_CommitMessage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping comparison test in short mode")
	}

	osClient, repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	originalDir, _ := os.Getwd()
	os.Chdir(repoPath)
	defer os.Chdir(originalDir)

	mockClient := git.NewMockGitClient()

	// Create commits with specific messages
	createCommit(t, repoPath, "feat: add OAuth2 support")
	commit1 := getCurrentCommit(t, repoPath)
	mockCommit1 := mockClient.CreateCommit("feat: add OAuth2 support")

	createCommit(t, repoPath, "fix: memory leak in handler")
	commit2 := getCurrentCommit(t, repoPath)
	mockCommit2 := mockClient.CreateCommit("fix: memory leak in handler")

	// Test: Get commit messages
	osMsg1, osErr1 := osClient.GetCommitMessage(commit1)
	mockMsg1, mockErr1 := mockClient.GetCommitMessage(mockCommit1)

	if osMsg1 != "feat: add OAuth2 support" {
		t.Errorf("OS GetCommitMessage for commit1: got=%q, want=%q", osMsg1, "feat: add OAuth2 support")
	}

	if mockMsg1 != "feat: add OAuth2 support" {
		t.Errorf("Mock GetCommitMessage for commit1: got=%q, want=%q", mockMsg1, "feat: add OAuth2 support")
	}

	if (osErr1 == nil) != (mockErr1 == nil) {
		t.Errorf("GetCommitMessage error mismatch for commit1: OS=%v, Mock=%v", osErr1, mockErr1)
	}

	osMsg2, osErr2 := osClient.GetCommitMessage(commit2)
	mockMsg2, mockErr2 := mockClient.GetCommitMessage(mockCommit2)

	if osMsg2 != "fix: memory leak in handler" {
		t.Errorf("OS GetCommitMessage for commit2: got=%q, want=%q", osMsg2, "fix: memory leak in handler")
	}

	if mockMsg2 != "fix: memory leak in handler" {
		t.Errorf("Mock GetCommitMessage for commit2: got=%q, want=%q", mockMsg2, "fix: memory leak in handler")
	}

	if (osErr2 == nil) != (mockErr2 == nil) {
		t.Errorf("GetCommitMessage error mismatch for commit2: OS=%v, Mock=%v", osErr2, mockErr2)
	}

	// Test: Empty commit SHA (error case)
	_, osErr3 := osClient.GetCommitMessage("")
	_, mockErr3 := mockClient.GetCommitMessage("")

	if osErr3 == nil {
		t.Error("OS should return error for empty commit SHA")
	}

	if mockErr3 == nil {
		t.Error("Mock should return error for empty commit SHA")
	}

	// Test: Invalid commit SHA
	_, osErr4 := osClient.GetCommitMessage("invalidsha123")
	_, mockErr4 := mockClient.GetCommitMessage("invalidsha123")

	if (osErr4 == nil) != (mockErr4 == nil) {
		t.Errorf("Invalid SHA error mismatch: OS=%v, Mock=%v", osErr4, mockErr4)
	}
}

// Helper: Get current commit SHA
func getCurrentCommit(t *testing.T, repoPath string) string {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	require.NoError(t, err, "Failed to get current commit")
	return strings.TrimSpace(string(output))
}
