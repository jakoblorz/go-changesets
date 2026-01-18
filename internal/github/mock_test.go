package github

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMockClient_CreatePullRequest(t *testing.T) {
	m := NewMockClient()

	pr, err := m.CreatePullRequest(context.Background(), "owner", "repo", &CreatePullRequestRequest{
		Title: "Test PR",
		Body:  "Test body",
		Head:  "feature-branch",
		Base:  "main",
		Draft: false,
	})

	require.NoError(t, err)
	require.NotNil(t, pr)
	require.Equal(t, 1, pr.Number)
	require.Equal(t, "Test PR", pr.Title)
	require.Equal(t, "Test body", pr.Body)
	require.Equal(t, "feature-branch", pr.Head)
	require.Equal(t, "main", pr.Base)
	require.Equal(t, "open", pr.State)
	// Author is empty since no user was set
	require.Empty(t, pr.Author)
}

func TestMockClient_GetPullRequestByHead(t *testing.T) {
	m := NewMockClient()

	// First create a PR
	createdPR, err := m.CreatePullRequest(context.Background(), "owner", "repo", &CreatePullRequestRequest{
		Title: "Test PR",
		Head:  "feature-branch",
		Base:  "main",
	})
	require.NoError(t, err)
	require.NotNil(t, createdPR)

	// Now find it by head branch
	pr, err := m.GetPullRequestByHead(context.Background(), "owner", "repo", "feature-branch")
	require.NoError(t, err)
	require.NotNil(t, pr)
	require.Equal(t, createdPR.Number, pr.Number)
	require.Equal(t, "Test PR", pr.Title)

	// Non-existent branch returns nil
	pr, err = m.GetPullRequestByHead(context.Background(), "owner", "repo", "non-existent")
	require.NoError(t, err)
	require.Nil(t, pr)
}

func TestMockClient_UpdatePullRequest(t *testing.T) {
	m := NewMockClient()

	// Create a PR
	_, err := m.CreatePullRequest(context.Background(), "owner", "repo", &CreatePullRequestRequest{
		Title: "Original Title",
		Body:  "Original body",
		Head:  "feature-branch",
		Base:  "main",
	})
	require.NoError(t, err)

	// Update it
	updatedPR, err := m.UpdatePullRequest(context.Background(), "owner", "repo", 1, &UpdatePullRequestRequest{
		Title: "Updated Title",
		Body:  "Updated body",
	})
	require.NoError(t, err)
	require.Equal(t, "Updated Title", updatedPR.Title)
	require.Equal(t, "Updated body", updatedPR.Body)

	// Verify the change persisted
	pr, err := m.GetPullRequest(context.Background(), "owner", "repo", 1)
	require.NoError(t, err)
	require.Equal(t, "Updated Title", pr.Title)
	require.Equal(t, "Updated body", pr.Body)
}

func TestMockClient_ClosePullRequest(t *testing.T) {
	m := NewMockClient()

	// Create a PR
	_, err := m.CreatePullRequest(context.Background(), "owner", "repo", &CreatePullRequestRequest{
		Title: "Test PR",
		Head:  "feature-branch",
		Base:  "main",
	})
	require.NoError(t, err)

	// Close it
	err = m.ClosePullRequest(context.Background(), "owner", "repo", 1)
	require.NoError(t, err)

	// Verify it's closed
	pr, err := m.GetPullRequest(context.Background(), "owner", "repo", 1)
	require.NoError(t, err)
	require.Equal(t, "closed", pr.State)
}

func TestMockClient_DeleteBranch(t *testing.T) {
	m := NewMockClient()

	// Add a branch first
	m.AddBranch("owner", "repo", "feature-branch")

	// Delete it
	err := m.DeleteBranch(context.Background(), "owner", "repo", "feature-branch")
	require.NoError(t, err)

	// Non-existent branch should fail
	err = m.DeleteBranch(context.Background(), "owner", "repo", "non-existent")
	require.Error(t, err)
}

func TestMockClient_AddPullRequestByHead(t *testing.T) {
	m := NewMockClient()

	m.AddPullRequestByHead("owner", "repo", "custom-branch", &PullRequest{
		Number: 42,
		Title:  "Custom PR",
	})

	pr, err := m.GetPullRequestByHead(context.Background(), "owner", "repo", "custom-branch")
	require.NoError(t, err)
	require.NotNil(t, pr)
	require.Equal(t, 42, pr.Number)
	require.Equal(t, "custom-branch", pr.Head)
}

func TestMockClient_Reset(t *testing.T) {
	m := NewMockClient()

	// Add some data
	m.AddRelease("owner", "repo", &Release{TagName: "v1.0.0"})
	m.SetupRepository("owner", "repo")
	m.AddBranch("owner", "repo", "feature-branch")

	// Reset
	m.Reset()

	// Verify all data is cleared
	releases := m.GetAllReleases("owner", "repo")
	require.Empty(t, releases)

	pr, err := m.GetPullRequestByHead(context.Background(), "owner", "repo", "feature-branch")
	require.NoError(t, err)
	require.Nil(t, pr)
}
