package github

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MockClient implements GitHubClient for testing
type MockClient struct {
	mu           sync.RWMutex
	releases     map[string][]*Release     // key: "owner/repo"
	repositories map[string]*Repository    // key: "owner/repo"
	pullRequests map[string][]*PullRequest // key: "owner/repo"
	commitPRs    map[string][]*PullRequest // key: "owner/repo/sha"

	// Hooks for testing error scenarios
	GetLatestReleaseError         error
	GetReleaseByTagError          error
	CreateReleaseError            error
	GetRepositoryError            error
	GetPullRequestError           error
	ListPullRequestsByCommitError error
}

// NewMockClient creates a new MockClient
func NewMockClient() *MockClient {
	return &MockClient{
		releases:     make(map[string][]*Release),
		repositories: make(map[string]*Repository),
		pullRequests: make(map[string][]*PullRequest),
		commitPRs:    make(map[string][]*PullRequest),
	}
}

// SetupRepository adds a repository to the mock
func (m *MockClient) SetupRepository(owner, repo string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s/%s", owner, repo)
	m.repositories[key] = &Repository{
		Owner:         owner,
		Name:          repo,
		FullName:      key,
		URL:           fmt.Sprintf("https://github.com/%s/%s", owner, repo),
		DefaultBranch: "main",
	}
}

// AddRelease adds a release to the mock
func (m *MockClient) AddRelease(owner, repo string, release *Release) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s/%s", owner, repo)
	m.releases[key] = append(m.releases[key], release)
}

func (m *MockClient) GetLatestRelease(ctx context.Context, owner, repo string) (*Release, error) {
	if m.GetLatestReleaseError != nil {
		return nil, m.GetLatestReleaseError
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	key := fmt.Sprintf("%s/%s", owner, repo)
	releases, exists := m.releases[key]
	if !exists || len(releases) == 0 {
		return nil, fmt.Errorf("no releases found for %s", key)
	}

	// Return the most recent release
	return releases[len(releases)-1], nil
}

func (m *MockClient) GetReleaseByTag(ctx context.Context, owner, repo, tag string) (*Release, error) {
	if m.GetReleaseByTagError != nil {
		return nil, m.GetReleaseByTagError
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	key := fmt.Sprintf("%s/%s", owner, repo)
	releases, exists := m.releases[key]
	if !exists {
		return nil, fmt.Errorf("no releases found for %s", key)
	}

	for _, r := range releases {
		if r.TagName == tag {
			return r, nil
		}
	}

	return nil, fmt.Errorf("release with tag %s not found", tag)
}

func (m *MockClient) CreateRelease(ctx context.Context, owner, repo string, req *CreateReleaseRequest) (*Release, error) {
	if m.CreateReleaseError != nil {
		return nil, m.CreateReleaseError
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s/%s", owner, repo)

	// Check if tag already exists
	if releases, exists := m.releases[key]; exists {
		for _, r := range releases {
			if r.TagName == req.TagName {
				return nil, fmt.Errorf("release with tag %s already exists", req.TagName)
			}
		}
	}

	release := &Release{
		ID:          int64(len(m.releases[key]) + 1),
		TagName:     req.TagName,
		Name:        req.Name,
		Body:        req.Body,
		Draft:       req.Draft,
		Prerelease:  req.Prerelease,
		CreatedAt:   time.Now(),
		PublishedAt: time.Now(),
	}

	m.releases[key] = append(m.releases[key], release)
	return release, nil
}

func (m *MockClient) GetRepository(ctx context.Context, owner, repo string) (*Repository, error) {
	if m.GetRepositoryError != nil {
		return nil, m.GetRepositoryError
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	key := fmt.Sprintf("%s/%s", owner, repo)
	repository, exists := m.repositories[key]
	if !exists {
		return nil, fmt.Errorf("repository %s not found", key)
	}

	return repository, nil
}

// GetAllReleases returns all releases for a repository (helper for testing)
func (m *MockClient) GetAllReleases(owner, repo string) []*Release {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := fmt.Sprintf("%s/%s", owner, repo)
	return m.releases[key]
}

// Reset clears all data from the mock (helper for testing)
func (m *MockClient) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.releases = make(map[string][]*Release)
	m.repositories = make(map[string]*Repository)
	m.pullRequests = make(map[string][]*PullRequest)
	m.commitPRs = make(map[string][]*PullRequest)
	m.GetLatestReleaseError = nil
	m.GetReleaseByTagError = nil
	m.CreateReleaseError = nil
	m.GetRepositoryError = nil
	m.GetPullRequestError = nil
	m.ListPullRequestsByCommitError = nil
}

// AddPullRequest adds a pull request to the mock
func (m *MockClient) AddPullRequest(owner, repo string, pr *PullRequest) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s/%s", owner, repo)
	m.pullRequests[key] = append(m.pullRequests[key], pr)
}

// AddPullRequestForCommit associates a pull request with a commit SHA
func (m *MockClient) AddPullRequestForCommit(owner, repo, sha string, pr *PullRequest) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s/%s/%s", owner, repo, sha)
	m.commitPRs[key] = append(m.commitPRs[key], pr)

	// Also add to pullRequests map for GetPullRequest lookups
	repoKey := fmt.Sprintf("%s/%s", owner, repo)
	m.pullRequests[repoKey] = append(m.pullRequests[repoKey], pr)
}

func (m *MockClient) GetPullRequest(ctx context.Context, owner, repo string, number int) (*PullRequest, error) {
	if m.GetPullRequestError != nil {
		return nil, m.GetPullRequestError
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	key := fmt.Sprintf("%s/%s", owner, repo)
	prs, exists := m.pullRequests[key]
	if !exists {
		return nil, fmt.Errorf("no pull requests found for %s", key)
	}

	for _, pr := range prs {
		if pr.Number == number {
			return pr, nil
		}
	}

	return nil, fmt.Errorf("pull request #%d not found", number)
}

func (m *MockClient) ListPullRequestsByCommit(ctx context.Context, owner, repo, sha string) ([]*PullRequest, error) {
	if m.ListPullRequestsByCommitError != nil {
		return nil, m.ListPullRequestsByCommitError
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	key := fmt.Sprintf("%s/%s/%s", owner, repo, sha)
	prs, exists := m.commitPRs[key]
	if !exists {
		return []*PullRequest{}, nil
	}

	return prs, nil
}
