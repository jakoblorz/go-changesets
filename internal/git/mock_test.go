package git_test

import (
	"fmt"
	"testing"

	"github.com/jakoblorz/go-changesets/internal/git"
)

func TestMockGitClient_TagOperations(t *testing.T) {
	mock := git.NewMockGitClient()

	// Test adding tags
	mock.AddTag("auth", "1.0.0", "Initial release")
	mock.AddTag("auth", "1.1.0", "Add feature")
	mock.AddTag("api", "0.1.0", "First release")

	// Test GetLatestTag
	tag, err := mock.GetLatestTag("auth")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if tag != "auth@v1.1.0" {
		t.Errorf("expected auth@v1.1.0, got %s", tag)
	}

	// Test GetLatestTag for different project
	tag, err = mock.GetLatestTag("api")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if tag != "api@v0.1.0" {
		t.Errorf("expected api@v0.1.0, got %s", tag)
	}

	// Test GetLatestTag for non-existent project
	_, err = mock.GetLatestTag("nonexistent")
	if err == nil {
		t.Error("expected error for non-existent project, got nil")
	}
}

func TestMockGitClient_CreateAndPushTag(t *testing.T) {
	mock := git.NewMockGitClient()

	// Create a tag
	err := mock.CreateTag("auth@v1.0.0", "Release 1.0.0")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify tag exists
	exists, err := mock.TagExists("auth@v1.0.0")
	if err != nil {
		t.Fatalf("expected no error checking existence, got %v", err)
	}
	if !exists {
		t.Error("expected tag to exist")
	}

	// Try to create duplicate tag
	err = mock.CreateTag("auth@v1.0.0", "Duplicate")
	if err == nil {
		t.Error("expected error for duplicate tag, got nil")
	}

	// Push the tag
	err = mock.PushTag("auth@v1.0.0")
	if err != nil {
		t.Fatalf("expected no error pushing tag, got %v", err)
	}

	// Verify tag is pushed
	tags := mock.GetAllTags()
	if !tags["auth@v1.0.0"].IsPushed {
		t.Error("expected tag to be marked as pushed")
	}
}

func TestMockGitClient_PushedTags(t *testing.T) {
	mock := git.NewMockGitClient()

	// Add local and pushed tags
	mock.AddTag("auth", "1.0.0", "Local only")
	mock.AddPushedTag("auth", "1.1.0", "Pushed")
	mock.AddPushedTag("auth", "1.2.0", "Pushed latest")

	// GetLatestTag should return all tags (pushed and unpushed)
	tag, err := mock.GetLatestTag("auth")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if tag != "auth@v1.2.0" {
		t.Errorf("expected auth@v1.2.0, got %s", tag)
	}

	// Verify IsPushed flag is set correctly
	allTags := mock.GetAllTags()
	if !allTags["auth@v1.1.0"].IsPushed {
		t.Error("expected auth@v1.1.0 to be marked as pushed")
	}
	if !allTags["auth@v1.2.0"].IsPushed {
		t.Error("expected auth@v1.2.0 to be marked as pushed")
	}
	if allTags["auth@v1.0.0"].IsPushed {
		t.Error("expected auth@v1.0.0 to NOT be marked as pushed")
	}
}

func TestMockGitClient_TagAnnotation(t *testing.T) {
	mock := git.NewMockGitClient()

	mock.AddTag("auth", "1.0.0", "This is the release message")

	annotation, err := mock.GetTagAnnotation("auth@v1.0.0")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if annotation != "This is the release message" {
		t.Errorf("expected 'This is the release message', got %s", annotation)
	}

	// Test non-existent tag
	_, err = mock.GetTagAnnotation("nonexistent@v1.0.0")
	if err == nil {
		t.Error("expected error for non-existent tag, got nil")
	}
}

func TestMockGitClient_RepoOperations(t *testing.T) {
	mock := git.NewMockGitClient()

	// Default is a git repo
	isRepo, err := mock.IsGitRepo()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !isRepo {
		t.Error("expected to be a git repo by default")
	}

	// Set as not a repo
	mock.SetIsRepo(false)
	isRepo, err = mock.IsGitRepo()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if isRepo {
		t.Error("expected not to be a git repo")
	}

	// Test branch
	branch, err := mock.GetCurrentBranch()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if branch != "main" {
		t.Errorf("expected 'main', got %s", branch)
	}

	mock.SetBranch("develop")
	branch, err = mock.GetCurrentBranch()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if branch != "develop" {
		t.Errorf("expected 'develop', got %s", branch)
	}
}

func TestMockGitClient_Reset(t *testing.T) {
	mock := git.NewMockGitClient()

	// Add some data
	mock.AddTag("auth", "1.0.0", "Release")
	mock.SetIsRepo(false)
	mock.SetBranch("feature")

	// Reset
	mock.Reset()

	// Verify reset
	_, err := mock.GetLatestTag("auth")
	if err == nil {
		t.Error("expected error after reset, got nil")
	}

	isRepo, _ := mock.IsGitRepo()
	if !isRepo {
		t.Error("expected isRepo to be reset to true")
	}

	branch, _ := mock.GetCurrentBranch()
	if branch != "main" {
		t.Errorf("expected branch to be reset to 'main', got %s", branch)
	}
}

func TestMockGitClient_ErrorScenarios(t *testing.T) {
	mock := git.NewMockGitClient()

	// Set error hooks
	mock.GetLatestTagError = fmt.Errorf("simulated error")

	_, err := mock.GetLatestTag("auth")
	if err == nil {
		t.Error("expected error, got nil")
	}
	if err.Error() != "simulated error" {
		t.Errorf("expected 'simulated error', got %v", err)
	}

	// Test CreateTag error
	mock.GetLatestTagError = nil
	mock.CreateTagError = fmt.Errorf("create failed")

	err = mock.CreateTag("auth@v1.0.0", "Release")
	if err == nil {
		t.Error("expected error, got nil")
	}
}

// Tests matching OS implementation behavior

func TestMockGit_BasicTagOperationsWithGraph(t *testing.T) {
	mock := git.NewMockGitClient()

	// Create a tag at initial commit
	err := mock.CreateTag("backend@v1.0.0", "Release 1.0.0")
	if err != nil {
		t.Fatalf("CreateTag failed: %v", err)
	}

	// Verify tag exists
	exists, err := mock.TagExists("backend@v1.0.0")
	if err != nil {
		t.Fatalf("TagExists failed: %v", err)
	}
	if !exists {
		t.Error("Expected tag to exist")
	}

	// Create a commit, then another tag
	mock.CreateCommit("Some work")
	err = mock.CreateTag("backend@v1.1.0", "Release 1.1.0")
	if err != nil {
		t.Fatalf("CreateTag failed: %v", err)
	}

	// Get latest tag
	tag, err := mock.GetLatestTag("backend")
	if err != nil {
		t.Fatalf("GetLatestTag failed: %v", err)
	}
	if tag != "backend@v1.1.0" {
		t.Errorf("Expected backend@v1.1.0, got %s", tag)
	}

	// Both tags should be reachable
	tags, err := mock.GetTagsWithPrefix("backend@v*")
	if err != nil {
		t.Fatalf("GetTagsWithPrefix failed: %v", err)
	}
	if len(tags) != 2 {
		t.Errorf("Expected 2 reachable tags, got %d: %v", len(tags), tags)
	}
}

func TestMockGit_WildcardPatternMatchingWithGraph(t *testing.T) {
	mock := git.NewMockGitClient()

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

	// Get all backend tags
	tags, err := mock.GetTagsWithPrefix("backend@v*")
	if err != nil {
		t.Fatalf("GetTagsWithPrefix failed: %v", err)
	}

	if len(tags) != 3 {
		t.Errorf("Expected 3 backend tags, got %d: %v", len(tags), tags)
	}

	// Verify sorted correctly (descending)
	expectedTags := []string{"backend@v1.2.0", "backend@v1.1.0", "backend@v1.0.0"}
	for i, expected := range expectedTags {
		if i >= len(tags) || tags[i] != expected {
			t.Errorf("Expected tags[%d] = %s, got %v", i, expected, tags)
			break
		}
	}

	// Get www tags
	wwwTags, _ := mock.GetTagsWithPrefix("www@v*")
	if len(wwwTags) != 1 || wwwTags[0] != "www@v2.0.0" {
		t.Errorf("Expected [www@v2.0.0], got %v", wwwTags)
	}
}

func TestMockGit_BranchDivergence(t *testing.T) {
	mock := git.NewMockGitClient()

	// Main: Initial → A → B
	mock.CreateCommit("commit_A")
	mock.CreateCommit("commit_B")
	mock.CreateTag("backend@v1.0.0", "Release 1.0.0")

	// Create canary branch from current HEAD
	err := mock.CreateBranch("canary")
	if err != nil {
		t.Fatalf("CreateBranch failed: %v", err)
	}

	// Canary: B → C → D
	err = mock.CheckoutBranch("canary")
	if err != nil {
		t.Fatalf("CheckoutBranch failed: %v", err)
	}
	mock.CreateCommit("commit_C")
	mock.CreateCommit("commit_D")

	// Switch to main
	err = mock.CheckoutBranch("main")
	if err != nil {
		t.Fatalf("CheckoutBranch main failed: %v", err)
	}

	// Main: B → E → F
	mock.CreateCommit("commit_E")
	mock.CreateCommit("commit_F")
	mock.CreateTag("backend@v1.1.0", "Release 1.1.0")

	// From canary: should only see v1.0.0
	mock.CheckoutBranch("canary")
	tags, err := mock.GetTagsWithPrefix("backend@v*")
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

	// From main: should see both tags
	mock.CheckoutBranch("main")
	tags, err = mock.GetTagsWithPrefix("backend@v*")
	if err != nil {
		t.Fatalf("GetTagsWithPrefix failed: %v", err)
	}

	if len(tags) != 2 {
		t.Errorf("Expected 2 tags from main, got %d: %v", len(tags), tags)
	}
}

func TestMockGit_RCTagFilteringWithGraph(t *testing.T) {
	mock := git.NewMockGitClient()

	// Create RC tags and final release
	mock.CreateTag("backend@v1.2.0-rc0", "RC 0")
	mock.CreateCommit("Fix 1")
	mock.CreateTag("backend@v1.2.0-rc1", "RC 1")
	mock.CreateCommit("Fix 2")
	mock.CreateTag("backend@v1.2.0-rc2", "RC 2")
	mock.CreateCommit("Final work")
	mock.CreateTag("backend@v1.2.0", "Final Release")

	// All tags should be reachable
	allTags, err := mock.GetTagsWithPrefix("backend@v*")
	if err != nil {
		t.Fatalf("GetTagsWithPrefix failed: %v", err)
	}

	if len(allTags) != 4 {
		t.Errorf("Expected 4 total tags, got %d: %v", len(allTags), allTags)
	}

	// Filter out RC tags (simulating publish logic)
	var nonRCTags []string
	for _, tag := range allTags {
		rcNum, _ := mock.ExtractRCNumber(tag)
		if rcNum < 0 {
			nonRCTags = append(nonRCTags, tag)
		}
	}

	if len(nonRCTags) != 1 {
		t.Errorf("Expected 1 non-RC tag, got %d: %v", len(nonRCTags), nonRCTags)
	}

	if len(nonRCTags) > 0 && nonRCTags[0] != "backend@v1.2.0" {
		t.Errorf("Expected backend@v1.2.0, got %s", nonRCTags[0])
	}
}

func TestMockGit_MultipleRCIncrementsWithGraph(t *testing.T) {
	mock := git.NewMockGitClient()

	// Create first RC
	mock.CreateTag("backend@v1.3.0-rc0", "First RC")

	// Get RC tags
	allTags, _ := mock.GetTagsWithPrefix("backend@v1.3.0-rc*")
	if len(allTags) != 1 {
		t.Errorf("Expected 1 RC tag, got %d", len(allTags))
	}

	// Find highest RC
	highestRC := -1
	for _, tag := range allTags {
		rcNum, _ := mock.ExtractRCNumber(tag)
		if rcNum > highestRC {
			highestRC = rcNum
		}
	}

	if highestRC != 0 {
		t.Errorf("Expected highest RC = 0, got %d", highestRC)
	}

	// Create second RC
	mock.CreateCommit("Fix")
	mock.CreateTag("backend@v1.3.0-rc1", "Second RC")

	allTags, _ = mock.GetTagsWithPrefix("backend@v1.3.0-rc*")
	if len(allTags) != 2 {
		t.Errorf("Expected 2 RC tags, got %d: %v", len(allTags), allTags)
	}

	// Find highest RC again
	highestRC = -1
	for _, tag := range allTags {
		rcNum, _ := mock.ExtractRCNumber(tag)
		if rcNum > highestRC {
			highestRC = rcNum
		}
	}

	if highestRC != 1 {
		t.Errorf("Expected highest RC = 1, got %d", highestRC)
	}
}

func TestMockGit_CanaryWorkflowComplete(t *testing.T) {
	mock := git.NewMockGitClient()

	// Main: Initial → v1.0.0
	mock.CreateCommit("main_v1.0.0")
	mock.CreateTag("backend@v1.0.0", "Release 1.0.0")

	// Create canary from this point
	mock.CreateBranch("canary")
	mock.CheckoutBranch("canary")

	// Canary: Add features
	mock.CreateCommit("canary_feature_1")
	mock.CreateCommit("canary_feature_2")

	// Create RC snapshots on canary
	mock.CreateTag("backend@v1.1.0-rc0", "First RC")

	tags, _ := mock.GetTagsWithPrefix("backend@v*")
	if len(tags) != 2 { // v1.0.0 and v1.1.0-rc0
		t.Errorf("Canary should see 2 tags, got %d: %v", len(tags), tags)
	}

	// More work on canary
	mock.CreateCommit("canary_fix")
	mock.CreateTag("backend@v1.1.0-rc1", "Second RC")

	tags, _ = mock.GetTagsWithPrefix("backend@v*")
	if len(tags) != 3 { // v1.0.0, rc0, rc1
		t.Errorf("Canary should see 3 tags, got %d: %v", len(tags), tags)
	}

	// Main continues in parallel
	mock.CheckoutBranch("main")
	mock.CreateCommit("main_other_feature")
	mock.CreateTag("backend@v1.1.0", "Final release on main")

	// Canary shouldn't see main's v1.1.0 yet
	mock.CheckoutBranch("canary")
	tags, _ = mock.GetTagsWithPrefix("backend@v*")

	foundFinal := false
	for _, tag := range tags {
		if tag == "backend@v1.1.0" {
			foundFinal = true
		}
	}
	if foundFinal {
		t.Error("Canary should NOT see main's v1.1.0 before merge")
	}

	// Merge main into canary
	_, err := mock.MergeBranch("main")
	if err != nil {
		t.Fatalf("MergeBranch failed: %v", err)
	}

	// After merge: canary sees all tags
	tags, _ = mock.GetTagsWithPrefix("backend@v*")
	if len(tags) != 4 { // v1.0.0, rc0, rc1, v1.1.0
		t.Errorf("After merge, expected 4 tags, got %d: %v", len(tags), tags)
	}

	// Verify all expected tags present
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
			t.Errorf("Expected tag %s not found after merge", tag)
		}
	}
}

func TestMockGit_MergeMainIntoCanary(t *testing.T) {
	mock := git.NewMockGitClient()

	// Main: Initial → A (tag v1.0.0)
	mock.CreateCommit("commit_A")
	mock.CreateTag("backend@v1.0.0", "Release 1.0.0")

	// Create canary from A
	mock.CreateBranch("canary")
	mock.CheckoutBranch("canary")
	mock.CreateCommit("canary_work")

	// Main continues: A → B (tag v1.1.0)
	mock.CheckoutBranch("main")
	mock.CreateCommit("commit_B")
	mock.CreateTag("backend@v1.1.0", "Release 1.1.0")

	// Before merge: canary sees v1.0.0 only
	mock.CheckoutBranch("canary")
	tagsBefore, _ := mock.GetTagsWithPrefix("backend@v*")
	if len(tagsBefore) != 1 || tagsBefore[0] != "backend@v1.0.0" {
		t.Errorf("Before merge, expected [backend@v1.0.0], got %v", tagsBefore)
	}

	// Merge main into canary
	_, err := mock.MergeBranch("main")
	if err != nil {
		t.Fatalf("MergeBranch failed: %v", err)
	}

	// After merge: canary sees both
	tagsAfter, _ := mock.GetTagsWithPrefix("backend@v*")
	if len(tagsAfter) != 2 {
		t.Errorf("After merge, expected 2 tags, got %d: %v", len(tagsAfter), tagsAfter)
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
		t.Errorf("After merge, expected both v1.0.0 and v1.1.0, got: %v", tagsAfter)
	}
}

func TestMockGit_CommitGraphOperations(t *testing.T) {
	mock := git.NewMockGitClient()

	// Test CreateCommit returns unique hashes
	hash1 := mock.CreateCommit("Commit 1")
	hash2 := mock.CreateCommit("Commit 2")
	hash3 := mock.CreateCommit("Commit 3")

	if hash1 == hash2 || hash2 == hash3 || hash1 == hash3 {
		t.Error("CreateCommit should return unique hashes")
	}

	// Test branch operations
	err := mock.CreateBranch("feature")
	if err != nil {
		t.Fatalf("CreateBranch failed: %v", err)
	}

	// Try to create duplicate branch
	err = mock.CreateBranch("feature")
	if err == nil {
		t.Error("Expected error when creating duplicate branch")
	}

	// Checkout branch
	err = mock.CheckoutBranch("feature")
	if err != nil {
		t.Fatalf("CheckoutBranch failed: %v", err)
	}

	// Create commit on feature branch
	hash4 := mock.CreateCommit("Feature work")

	// Switch back to main
	mock.CheckoutBranch("main")

	// Tag on feature should not be reachable from main
	mock.CheckoutBranch("feature")
	mock.CreateTag("backend@v2.0.0", "Feature release")

	mock.CheckoutBranch("main")
	tags, _ := mock.GetTagsWithPrefix("backend@v*")

	// Should not see v2.0.0 from main
	for _, tag := range tags {
		if tag == "backend@v2.0.0" {
			t.Error("main should NOT see feature branch tag")
		}
	}

	// Merge feature into main
	_, err = mock.MergeBranch("feature")
	if err != nil {
		t.Fatalf("MergeBranch failed: %v", err)
	}

	// Now main should see v2.0.0
	tags, _ = mock.GetTagsWithPrefix("backend@v*")
	found := false
	for _, tag := range tags {
		if tag == "backend@v2.0.0" {
			found = true
		}
	}
	if !found {
		t.Error("main should see feature tag after merge")
	}

	_ = hash4
}

func TestMockGit_TagAnnotationsWithGraph(t *testing.T) {
	mock := git.NewMockGitClient()

	message := "Release 1.0.0\n\nThis is a test release\nwith multiple lines"
	mock.CreateTag("backend@v1.0.0", message)

	annotation, err := mock.GetTagAnnotation("backend@v1.0.0")
	if err != nil {
		t.Fatalf("GetTagAnnotation failed: %v", err)
	}

	if annotation != message {
		t.Errorf("Expected message %q, got %q", message, annotation)
	}
}
