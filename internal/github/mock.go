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
	headPRs      map[string]*PullRequest   // key: "owner/repo/head-branch"
	branches     map[string]bool           // key: "owner/repo/branch"

	// Hooks for testing error scenarios
	GetLatestReleaseError         error
	GetReleaseByTagError          error
	CreateReleaseError            error
	GetRepositoryError            error
	GetPullRequestError           error
	GetPullRequestByHeadError     error
	ListPullRequestsByCommitError error
	CreatePullRequestError        error
	UpdatePullRequestError        error
	ClosePullRequestError         error
	DeleteBranchError             error
}

// NewMockClient creates a new MockClient
func NewMockClient() *MockClient {
	return &MockClient{
		releases:     make(map[string][]*Release),
		repositories: make(map[string]*Repository),
		pullRequests: make(map[string][]*PullRequest),
		commitPRs:    make(map[string][]*PullRequest),
		headPRs:      make(map[string]*PullRequest),
		branches:     make(map[string]bool),
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

func (m *MockClient) GetPullRequestByHead(ctx context.Context, owner, repo, headBranch string) (*PullRequest, error) {
	if m.GetPullRequestByHeadError != nil {
		return nil, m.GetPullRequestByHeadError
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	key := fmt.Sprintf("%s/%s/%s", owner, repo, headBranch)
	pr, exists := m.headPRs[key]
	if !exists {
		return nil, nil
	}

	return pr, nil
}

func (m *MockClient) CreatePullRequest(ctx context.Context, owner, repo string, req *CreatePullRequestRequest) (*PullRequest, error) {
	if m.CreatePullRequestError != nil {
		return nil, m.CreatePullRequestError
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	pr := &PullRequest{
		Number:  len(m.pullRequests[fmt.Sprintf("%s/%s", owner, repo)]) + 1,
		Title:   req.Title,
		Body:    req.Body,
		Head:    req.Head,
		Base:    req.Base,
		State:   "open",
		HTMLURL: fmt.Sprintf("https://github.com/%s/%s/pull/%d", owner, repo, len(m.pullRequests[fmt.Sprintf("%s/%s", owner, repo)])+1),
	}

	key := fmt.Sprintf("%s/%s", owner, repo)
	m.pullRequests[key] = append(m.pullRequests[key], pr)

	// Index by head branch for GetPullRequestByHead lookups
	m.headPRs[fmt.Sprintf("%s/%s/%s", owner, repo, req.Head)] = pr

	// Index by commit PRs
	m.commitPRs[fmt.Sprintf("%s/%s", owner, repo)] = append(m.commitPRs[fmt.Sprintf("%s/%s", owner, repo)], pr)

	return pr, nil
}

func (m *MockClient) UpdatePullRequest(ctx context.Context, owner, repo string, number int, req *UpdatePullRequestRequest) (*PullRequest, error) {
	if m.UpdatePullRequestError != nil {
		return nil, m.UpdatePullRequestError
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s/%s", owner, repo)
	for i, pr := range m.pullRequests[key] {
		if pr.Number == number {
			m.pullRequests[key][i].Title = req.Title
			m.pullRequests[key][i].Body = req.Body
			return m.pullRequests[key][i], nil
		}
	}

	return nil, fmt.Errorf("pull request #%d not found", number)
}

func (m *MockClient) ClosePullRequest(ctx context.Context, owner, repo string, number int) error {
	if m.ClosePullRequestError != nil {
		return m.ClosePullRequestError
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s/%s", owner, repo)
	for i, pr := range m.pullRequests[key] {
		if pr.Number == number {
			m.pullRequests[key][i].State = "closed"
			return nil
		}
	}

	return fmt.Errorf("pull request #%d not found", number)
}

func (m *MockClient) DeleteBranch(ctx context.Context, owner, repo, branch string) error {
	if m.DeleteBranchError != nil {
		return m.DeleteBranchError
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s/%s/%s", owner, repo, branch)
	if _, exists := m.branches[key]; !exists {
		return fmt.Errorf("branch %s not found", branch)
	}
	delete(m.branches, key)
	return nil
}

// AddBranch adds a branch to the mock
func (m *MockClient) AddBranch(owner, repo, branch string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.branches[fmt.Sprintf("%s/%s/%s", owner, repo, branch)] = true
}

// AddPullRequestByHead adds a PR and indexes it by head branch
func (m *MockClient) AddPullRequestByHead(owner, repo, headBranch string, pr *PullRequest) {
	m.mu.Lock()
	defer m.mu.Unlock()

	pr.Head = headBranch
	key := fmt.Sprintf("%s/%s", owner, repo)
	m.pullRequests[key] = append(m.pullRequests[key], pr)
	m.headPRs[fmt.Sprintf("%s/%s/%s", owner, repo, headBranch)] = pr
}

// Reset clears all data from the mock (helper for testing)
func (m *MockClient) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.releases = make(map[string][]*Release)
	m.repositories = make(map[string]*Repository)
	m.pullRequests = make(map[string][]*PullRequest)
	m.commitPRs = make(map[string][]*PullRequest)
	m.headPRs = make(map[string]*PullRequest)
	m.branches = make(map[string]bool)
	m.GetLatestReleaseError = nil
	m.GetReleaseByTagError = nil
	m.CreateReleaseError = nil
	m.GetRepositoryError = nil
	m.GetPullRequestError = nil
	m.GetPullRequestByHeadError = nil
	m.ListPullRequestsByCommitError = nil
	m.CreatePullRequestError = nil
	m.UpdatePullRequestError = nil
	m.ClosePullRequestError = nil
	m.DeleteBranchError = nil
}
