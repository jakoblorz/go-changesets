package git

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMockGitClient_TagOperations(t *testing.T) {
	mock := NewMockGitClient()

	// Test adding tags
	mock.AddTag("auth", "1.0.0", "Initial release")
	mock.AddTag("auth", "1.1.0", "Add feature")
	mock.AddTag("api", "0.1.0", "First release")

	// Test GetLatestTag
	tag, err := mock.GetLatestTag("auth")
	require.NoError(t, err)
	require.Equal(t, "auth@v1.1.0", tag)

	tag, err = mock.GetLatestTag("api")
	require.NoError(t, err)
	require.Equal(t, "api@v0.1.0", tag)

	_, err = mock.GetLatestTag("nonexistent")
	require.Error(t, err)
}

func TestMockGitClient_CreateAndPushTag(t *testing.T) {
	mock := NewMockGitClient()

	err := mock.CreateTag("auth@v1.0.0", "Release 1.0.0")
	require.NoError(t, err)

	exists, err := mock.TagExists("auth@v1.0.0")
	require.NoError(t, err)
	require.True(t, exists)

	err = mock.CreateTag("auth@v1.0.0", "Duplicate")
	require.Error(t, err)

	err = mock.PushTag("auth@v1.0.0")
	require.NoError(t, err)

	tags := mock.GetAllTags()
	require.True(t, tags["auth@v1.0.0"].IsPushed)
}

func TestMockGitClient_PushedTags(t *testing.T) {
	mock := NewMockGitClient()

	mock.AddTag("auth", "1.0.0", "Local only")
	mock.AddPushedTag("auth", "1.1.0", "Pushed")
	mock.AddPushedTag("auth", "1.2.0", "Pushed latest")

	tag, err := mock.GetLatestTag("auth")
	require.NoError(t, err)
	require.Equal(t, "auth@v1.2.0", tag)

	allTags := mock.GetAllTags()
	require.True(t, allTags["auth@v1.1.0"].IsPushed)
	require.True(t, allTags["auth@v1.2.0"].IsPushed)
	require.False(t, allTags["auth@v1.0.0"].IsPushed)
}

func TestMockGitClient_TagAnnotation(t *testing.T) {
	mock := NewMockGitClient()

	mock.AddTag("auth", "1.0.0", "This is the release message")

	annotation, err := mock.GetTagAnnotation("auth@v1.0.0")
	require.NoError(t, err)
	require.Equal(t, "This is the release message", annotation)

	_, err = mock.GetTagAnnotation("nonexistent@v1.0.0")
	require.Error(t, err)
}

func TestMockGitClient_RepoOperations(t *testing.T) {
	mock := NewMockGitClient()

	// Default is a git repo
	isRepo, err := mock.IsGitRepo()
	require.NoError(t, err)
	require.True(t, isRepo)

	mock.SetIsRepo(false)
	isRepo, err = mock.IsGitRepo()
	require.NoError(t, err)
	require.False(t, isRepo)

	branch, err := mock.GetCurrentBranch()
	require.NoError(t, err)
	require.Equal(t, "main", branch)

	mock.SetBranch("develop")
	branch, err = mock.GetCurrentBranch()
	require.NoError(t, err)
	require.Equal(t, "develop", branch)
}

func TestMockGitClient_Reset(t *testing.T) {
	mock := NewMockGitClient()

	// Add some data
	mock.AddTag("auth", "1.0.0", "Release")
	mock.SetIsRepo(false)
	mock.SetBranch("feature")

	mock.Reset()

	_, err := mock.GetLatestTag("auth")
	require.Error(t, err)

	isRepo, _ := mock.IsGitRepo()
	require.True(t, isRepo)

	branch, _ := mock.GetCurrentBranch()
	require.Equal(t, "main", branch)
}

func TestMockGitClient_ErrorScenarios(t *testing.T) {
	mock := NewMockGitClient()

	// Set error hooks
	mock.GetLatestTagError = fmt.Errorf("simulated error")

	_, err := mock.GetLatestTag("auth")
	require.Error(t, err)
	require.EqualError(t, err, "simulated error")

	mock.GetLatestTagError = nil
	mock.CreateTagError = fmt.Errorf("create failed")

	err = mock.CreateTag("auth@v1.0.0", "Release")
	require.Error(t, err)
}

// Tests matching OS implementation behavior

func TestMockGit_BasicTagOperationsWithGraph(t *testing.T) {
	mock := NewMockGitClient()

	err := mock.CreateTag("backend@v1.0.0", "Release 1.0.0")
	require.NoError(t, err)

	exists, err := mock.TagExists("backend@v1.0.0")
	require.NoError(t, err)
	require.True(t, exists)

	mock.CreateCommit("Some work")
	err = mock.CreateTag("backend@v1.1.0", "Release 1.1.0")
	require.NoError(t, err)

	tag, err := mock.GetLatestTag("backend")
	require.NoError(t, err)
	require.Equal(t, "backend@v1.1.0", tag)

	tags, err := mock.GetTagsWithPrefix("backend@v*")
	require.NoError(t, err)
	require.Equal(t, []string{"backend@v1.1.0", "backend@v1.0.0"}, tags)
}

func TestMockGit_WildcardPatternMatchingWithGraph(t *testing.T) {
	mock := NewMockGitClient()

	// Create tags for different projects
	mock.CreateTag("backend@v1.0.0", "backend 1.0.0")
	mock.CreateCommit("Work")
	mock.CreateTag("backend@v1.1.0", "backend 1.1.0")
	mock.CreateCommit("More work")
	mock.CreateTag("backend@v1.2.0", "backend 1.2.0")
	mock.CreateCommit("Other")
	mock.CreateTag("www@v2.0.0", "WWW 2.0.0")
	mock.CreateCommit("API work")
	mock.CreateTag("api@v0.5.0", "API 0.5.0")

	tags, err := mock.GetTagsWithPrefix("backend@v*")
	require.NoError(t, err)
	require.Len(t, tags, 3)

	expectedTags := []string{"backend@v1.2.0", "backend@v1.1.0", "backend@v1.0.0"}
	for i, expected := range expectedTags {
		require.Less(t, i, len(tags))
		require.Equal(t, expected, tags[i])
	}

	wwwTags, err := mock.GetTagsWithPrefix("www@v*")
	require.NoError(t, err)
	require.Equal(t, []string{"www@v2.0.0"}, wwwTags)
}

func TestMockGit_BranchDivergence(t *testing.T) {
	mock := NewMockGitClient()

	mock.CreateCommit("commit_A")
	mock.CreateCommit("commit_B")
	mock.CreateTag("backend@v1.0.0", "Release 1.0.0")

	require.NoError(t, mock.CreateBranch("canary"))
	require.NoError(t, mock.CheckoutBranch("canary"))
	mock.CreateCommit("commit_C")
	mock.CreateCommit("commit_D")

	require.NoError(t, mock.CheckoutBranch("main"))
	mock.CreateCommit("commit_E")
	mock.CreateCommit("commit_F")
	mock.CreateTag("backend@v1.1.0", "Release 1.1.0")

	require.NoError(t, mock.CheckoutBranch("canary"))
	tags, err := mock.GetTagsWithPrefix("backend@v*")
	require.NoError(t, err)
	require.Len(t, tags, 1)
	require.Equal(t, "backend@v1.0.0", tags[0])

	for _, tag := range tags {
		require.NotEqual(t, "backend@v1.1.0", tag, "canary should not see main-only tag")
	}

	require.NoError(t, mock.CheckoutBranch("main"))
	tags, err = mock.GetTagsWithPrefix("backend@v*")
	require.NoError(t, err)
	require.Len(t, tags, 2)
}

func TestMockGit_RCTagFilteringWithGraph(t *testing.T) {
	mock := NewMockGitClient()

	mock.CreateTag("backend@v1.2.0-rc0", "RC 0")
	mock.CreateCommit("Fix 1")
	mock.CreateTag("backend@v1.2.0-rc1", "RC 1")
	mock.CreateCommit("Fix 2")
	mock.CreateTag("backend@v1.2.0-rc2", "RC 2")
	mock.CreateCommit("Final work")
	mock.CreateTag("backend@v1.2.0", "Final Release")

	allTags, err := mock.GetTagsWithPrefix("backend@v*")
	require.NoError(t, err)
	require.Len(t, allTags, 4)

	var nonRCTags []string
	for _, tag := range allTags {
		rcNum, _ := mock.ExtractRCNumber(tag)
		if rcNum < 0 {
			nonRCTags = append(nonRCTags, tag)
		}
	}

	require.Len(t, nonRCTags, 1)
	require.Equal(t, "backend@v1.2.0", nonRCTags[0])
}

func TestMockGit_MultipleRCIncrementsWithGraph(t *testing.T) {
	mock := NewMockGitClient()

	mock.CreateTag("backend@v1.3.0-rc0", "First RC")

	allTags, _ := mock.GetTagsWithPrefix("backend@v1.3.0-rc*")
	require.Len(t, allTags, 1)

	highestRC := -1
	for _, tag := range allTags {
		rcNum, _ := mock.ExtractRCNumber(tag)
		if rcNum > highestRC {
			highestRC = rcNum
		}
	}
	require.Equal(t, 0, highestRC)

	mock.CreateCommit("Fix")
	mock.CreateTag("backend@v1.3.0-rc1", "Second RC")

	allTags, _ = mock.GetTagsWithPrefix("backend@v1.3.0-rc*")
	require.Len(t, allTags, 2)

	highestRC = -1
	for _, tag := range allTags {
		rcNum, _ := mock.ExtractRCNumber(tag)
		if rcNum > highestRC {
			highestRC = rcNum
		}
	}
	require.Equal(t, 1, highestRC)
}

func TestMockGit_CanaryWorkflowComplete(t *testing.T) {
	mock := NewMockGitClient()

	mock.CreateCommit("main_v1.0.0")
	mock.CreateTag("backend@v1.0.0", "Release 1.0.0")

	mock.CreateBranch("canary")
	mock.CheckoutBranch("canary")
	mock.CreateCommit("canary_feature_1")
	mock.CreateCommit("canary_feature_2")
	mock.CreateTag("backend@v1.1.0-rc0", "First RC")

	tags, _ := mock.GetTagsWithPrefix("backend@v*")
	require.Len(t, tags, 2)

	mock.CreateCommit("canary_fix")
	mock.CreateTag("backend@v1.1.0-rc1", "Second RC")
	tags, _ = mock.GetTagsWithPrefix("backend@v*")
	require.Len(t, tags, 3)

	mock.CheckoutBranch("main")
	mock.CreateCommit("main_other_feature")
	mock.CreateTag("backend@v1.1.0", "Final release on main")

	mock.CheckoutBranch("canary")
	tags, _ = mock.GetTagsWithPrefix("backend@v*")
	for _, tag := range tags {
		require.NotEqual(t, "backend@v1.1.0", tag, "canary should not see final release yet")
	}

	_, err := mock.MergeBranch("main")
	require.NoError(t, err)

	tags, _ = mock.GetTagsWithPrefix("backend@v*")
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

func TestMockGit_MergeMainIntoCanary(t *testing.T) {
	mock := NewMockGitClient()

	mock.CreateCommit("commit_A")
	mock.CreateTag("backend@v1.0.0", "Release 1.0.0")

	mock.CreateBranch("canary")
	mock.CheckoutBranch("canary")
	mock.CreateCommit("canary_work")

	mock.CheckoutBranch("main")
	mock.CreateCommit("commit_B")
	mock.CreateTag("backend@v1.1.0", "Release 1.1.0")

	mock.CheckoutBranch("canary")
	tagsBefore, err := mock.GetTagsWithPrefix("backend@v*")
	require.NoError(t, err)
	require.Equal(t, []string{"backend@v1.0.0"}, tagsBefore)

	_, err = mock.MergeBranch("main")
	require.NoError(t, err)

	tagsAfter, err := mock.GetTagsWithPrefix("backend@v*")
	require.NoError(t, err)
	require.Len(t, tagsAfter, 2)
	for _, expected := range []string{"backend@v1.0.0", "backend@v1.1.0"} {
		require.Contains(t, tagsAfter, expected)
	}
}

func TestMockGit_CommitGraphOperations(t *testing.T) {
	mock := NewMockGitClient()

	hash1 := mock.CreateCommit("Commit 1")
	hash2 := mock.CreateCommit("Commit 2")
	hash3 := mock.CreateCommit("Commit 3")
	require.NotEqual(t, hash1, hash2)
	require.NotEqual(t, hash2, hash3)
	require.NotEqual(t, hash1, hash3)

	err := mock.CreateBranch("feature")
	require.NoError(t, err)

	err = mock.CreateBranch("feature")
	require.Error(t, err)

	err = mock.CheckoutBranch("feature")
	require.NoError(t, err)

	hash4 := mock.CreateCommit("Feature work")

	mock.CheckoutBranch("main")
	mock.CheckoutBranch("feature")
	mock.CreateTag("backend@v2.0.0", "Feature release")
	mock.CheckoutBranch("main")
	tags, _ := mock.GetTagsWithPrefix("backend@v*")
	for _, tag := range tags {
		require.NotEqual(t, "backend@v2.0.0", tag, "main should not see feature tag before merge")
	}

	_, err = mock.MergeBranch("feature")
	require.NoError(t, err)

	tags, _ = mock.GetTagsWithPrefix("backend@v*")
	require.Contains(t, tags, "backend@v2.0.0")

	_ = hash4
}

func TestMockGit_TagAnnotationsWithGraph(t *testing.T) {
	mock := NewMockGitClient()

	message := "Release 1.0.0\n\nThis is a test release\nwith multiple lines"
	mock.CreateTag("backend@v1.0.0", message)

	annotation, err := mock.GetTagAnnotation("backend@v1.0.0")
	require.NoError(t, err)
	require.Equal(t, message, annotation)
}
