package git

import (
	"os"
	"os/exec"
	"strings"
	"testing"

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
	mockClient := NewMockGitClient()

	// Test 1: Create tag
	osErr1 := osClient.CreateTag("backend@v1.0.0", "Release 1.0.0")
	mockErr1 := mockClient.CreateTag("backend@v1.0.0", "Release 1.0.0")

	require.Equal(t, osErr1 == nil, mockErr1 == nil, "CreateTag error mismatch: OS=%v, Mock=%v", osErr1, mockErr1)

	// Test 2: TagExists
	osExists, osErr2 := osClient.TagExists("backend@v1.0.0")
	mockExists, mockErr2 := mockClient.TagExists("backend@v1.0.0")

	require.Equal(t, osExists, mockExists, "TagExists mismatch: OS=%v, Mock=%v", osExists, mockExists)
	require.Equal(t, osErr2 == nil, mockErr2 == nil, "TagExists error mismatch: OS=%v, Mock=%v", osErr2, mockErr2)

	// Test 3: Create second tag
	createCommit(t, repoPath, "Work")
	mockClient.CreateCommit("Work")

	osClient.CreateTag("backend@v1.1.0", "Release 1.1.0")
	mockClient.CreateTag("backend@v1.1.0", "Release 1.1.0")

	// Test 4: GetLatestTag
	osTag, osErr3 := osClient.GetLatestTag("backend")
	mockTag, mockErr3 := mockClient.GetLatestTag("backend")

	require.Equal(t, osTag, mockTag, "GetLatestTag mismatch: OS=%s, Mock=%s", osTag, mockTag)
	require.Equal(t, osErr3 == nil, mockErr3 == nil, "GetLatestTag error mismatch: OS=%v, Mock=%v", osErr3, mockErr3)

	// Test 5: GetTagsWithPrefix
	osTags, osErr4 := osClient.GetTagsWithPrefix("backend@v*")
	mockTags, mockErr4 := mockClient.GetTagsWithPrefix("backend@v*")

	require.Lenf(t, osTags, len(mockTags), "GetTagsWithPrefix length mismatch: OS=%d, Mock=%d", len(osTags), len(mockTags))

	for i := 0; i < len(osTags) && i < len(mockTags); i++ {
		require.Equal(t, osTags[i], mockTags[i], "GetTagsWithPrefix[%d] mismatch: OS=%s, Mock=%s", i, osTags[i], mockTags[i])
	}

	require.Equal(t, osErr4 == nil, mockErr4 == nil, "GetTagsWithPrefix error mismatch: OS=%v, Mock=%v", osErr4, mockErr4)
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

	mockClient := NewMockGitClient()

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

		require.Lenf(t, osTags, len(mockTags), "Pattern %s: length mismatch OS=%d, Mock=%d", pattern, len(osTags), len(mockTags))

		for i := 0; i < len(osTags); i++ {
			require.Equal(t, osTags[i], mockTags[i], "Pattern %s[%d]: OS=%s, Mock=%s", pattern, i, osTags[i], mockTags[i])
		}
	}
}

func TestComparison_RCExtraction(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping comparison test in short mode")
	}

	osClient, _, cleanup := setupTestRepo(t)
	defer cleanup()

	mockClient := NewMockGitClient()

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

		require.Equal(t, osNum, mockNum, "ExtractRCNumber(%s): OS=%d, Mock=%d", tag, osNum, mockNum)

		require.Equal(t, osErr == nil, mockErr == nil, "ExtractRCNumber(%s) error mismatch: OS=%v, Mock=%v", tag, osErr, mockErr)
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

	mockClient := NewMockGitClient()

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
		require.Equal(t, osTags[i], mockTags[i], "Sort order[%d]: OS=%s, Mock=%s", i, osTags[i], mockTags[i])
		t.Logf("Full OS order: %v", osTags)
		t.Logf("Full Mock order: %v", mockTags)
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

	mockClient := NewMockGitClient()

	// Create tags with multiline messages
	message := "Release 1.0.0\n\nChangelog:\n- Feature A\n- Feature B"

	osClient.CreateTag("backend@v1.0.0", message)
	mockClient.CreateTag("backend@v1.0.0", message)

	// Retrieve annotations
	osAnnotation, osErr := osClient.GetTagAnnotation("backend@v1.0.0")
	mockAnnotation, mockErr := mockClient.GetTagAnnotation("backend@v1.0.0")

	require.Equal(t, osAnnotation, mockAnnotation, "GetTagAnnotation mismatch:\nOS=%q\nMock=%q", osAnnotation, mockAnnotation)

	require.Equal(t, osErr == nil, mockErr == nil, "GetTagAnnotation error mismatch: OS=%v, Mock=%v", osErr, mockErr)
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

	mockClient := NewMockGitClient()

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

		require.Equal(t, osErr == nil, mockErr == nil, "Project %s error mismatch: OS=%v, Mock=%v", project, osErr, mockErr)

		require.Lenf(t, osTags, len(mockTags), "Project %s count mismatch: OS=%d, Mock=%d", project, len(osTags), len(mockTags))

		for i := 0; i < len(osTags); i++ {
			require.Equal(t, osTags[i], mockTags[i], "Project %s[%d]: OS=%s, Mock=%s", project, i, osTags[i], mockTags[i])
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

	mockClient := NewMockGitClient()

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

	require.Equal(t, osCommit1, initialCommit, "OS GetFileCreationCommit returned wrong commit: got=%s, want=%s", osCommit1, initialCommit)

	require.Equal(t, mockCommit1Retrieved, mockCommit1, "Mock GetFileCreationCommit: got=%s, want=%s", mockCommit1Retrieved, mockCommit1)

	require.Equal(t, osErr1 == nil, mockErr1 == nil, "GetFileCreationCommit error mismatch: OS=%v, Mock=%v", osErr1, mockErr1)

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

	require.Equal(t, osCommit2, secondCommit, "OS GetFileCreationCommit for file2: got=%s, want=%s", osCommit2, secondCommit)

	require.Equal(t, mockCommit2Retrieved, mockCommit2, "Mock GetFileCreationCommit for file2: got=%s, want=%s", mockCommit2Retrieved, mockCommit2)

	// Test: Verify files have different creation commits
	require.NotEqual(t, osCommit1, osCommit2, "Expected files to have different creation commits")

	require.NotEqual(t, mockCommit1, mockCommit2, "Mock: Expected files to have different creation commits")

	// Test: Non-existent file
	osCommit3, osErr3 := osClient.GetFileCreationCommit("nonexistent.md")
	mockCommit3, mockErr3 := mockClient.GetFileCreationCommit("nonexistent.md")

	require.Empty(t, osCommit3, "OS should return empty string for nonexistent file, got: %s", osCommit3)

	require.Empty(t, mockCommit3, "Mock should return empty string for nonexistent file, got: %s", mockCommit3)

	require.Equal(t, osErr3 == nil, mockErr3 == nil, "Nonexistent file error mismatch: OS=%v, Mock=%v", osErr3, mockErr3)
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

	mockClient := NewMockGitClient()

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

	require.Equal(t, osMsg1, "feat: add OAuth2 support", "OS GetCommitMessage for commit1: got=%q, want=%q", osMsg1, "feat: add OAuth2 support")

	require.Equal(t, mockMsg1, "feat: add OAuth2 support", "Mock GetCommitMessage for commit1: got=%q, want=%q", mockMsg1, "feat: add OAuth2 support")

	require.Equal(t, osErr1 == nil, mockErr1 == nil, "GetCommitMessage error mismatch for commit1: OS=%v, Mock=%v", osErr1, mockErr1)

	osMsg2, osErr2 := osClient.GetCommitMessage(commit2)
	mockMsg2, mockErr2 := mockClient.GetCommitMessage(mockCommit2)

	require.Equal(t, osMsg2, "fix: memory leak in handler", "OS GetCommitMessage for commit2: got=%q, want=%q", osMsg2, "fix: memory leak in handler")

	require.Equal(t, mockMsg2, "fix: memory leak in handler", "Mock GetCommitMessage for commit2: got=%q, want=%q", mockMsg2, "fix: memory leak in handler")

	require.Equal(t, osErr2 == nil, mockErr2 == nil, "GetCommitMessage error mismatch for commit2: OS=%v, Mock=%v", osErr2, mockErr2)

	// Test: Empty commit SHA (error case)
	_, osErr3 := osClient.GetCommitMessage("")
	_, mockErr3 := mockClient.GetCommitMessage("")

	require.Error(t, osErr3, "OS should return error for empty commit SHA")

	require.Error(t, mockErr3, "Mock should return error for empty commit SHA")

	// Test: Invalid commit SHA
	_, osErr4 := osClient.GetCommitMessage("invalidsha123")
	_, mockErr4 := mockClient.GetCommitMessage("invalidsha123")

	require.Equal(t, osErr4 == nil, mockErr4 == nil, "Invalid SHA error mismatch: OS=%v, Mock=%v", osErr4, mockErr4)
}

// Helper: Get current commit SHA
func getCurrentCommit(t *testing.T, repoPath string) string {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	require.NoError(t, err, "Failed to get current commit")
	return strings.TrimSpace(string(output))
}
