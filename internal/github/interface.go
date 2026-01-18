package github

import (
	"context"
	"time"
)

// GitHubClient provides an abstraction over GitHub API operations
type GitHubClient interface {
	// Release operations
	GetLatestRelease(ctx context.Context, owner, repo string) (*Release, error)
	GetReleaseByTag(ctx context.Context, owner, repo, tag string) (*Release, error)
	CreateRelease(ctx context.Context, owner, repo string, release *CreateReleaseRequest) (*Release, error)

	// Repository operations
	GetRepository(ctx context.Context, owner, repo string) (*Repository, error)

	// Pull request operations
	GetPullRequest(ctx context.Context, owner, repo string, number int) (*PullRequest, error)
	GetPullRequestByHead(ctx context.Context, owner, repo, headBranch string) (*PullRequest, error)
	ListPullRequestsByCommit(ctx context.Context, owner, repo, sha string) ([]*PullRequest, error)
	CreatePullRequest(ctx context.Context, owner, repo string, req *CreatePullRequestRequest) (*PullRequest, error)
	UpdatePullRequest(ctx context.Context, owner, repo string, number int, req *UpdatePullRequestRequest) (*PullRequest, error)
	ClosePullRequest(ctx context.Context, owner, repo string, number int) error
	DeleteBranch(ctx context.Context, owner, repo, branch string) error
}

// CreatePullRequestRequest represents a request to create a pull request
type CreatePullRequestRequest struct {
	Title string
	Body  string
	Head  string
	Base  string
	Draft bool
}

// UpdatePullRequestRequest represents a request to update a pull request
type UpdatePullRequestRequest struct {
	Title string
	Body  string
}

// Release represents a GitHub release
type Release struct {
	ID          int64
	TagName     string
	Name        string
	Body        string
	Draft       bool
	Prerelease  bool
	CreatedAt   time.Time
	PublishedAt time.Time
}

// CreateReleaseRequest represents a request to create a release
type CreateReleaseRequest struct {
	TagName         string
	Name            string
	Body            string
	Draft           bool
	Prerelease      bool
	TargetCommitish string
}

// Repository represents a GitHub repository
type Repository struct {
	Owner         string
	Name          string
	FullName      string
	URL           string
	DefaultBranch string
}

// PullRequest represents a GitHub pull request
type PullRequest struct {
	Number         int
	Title          string
	Body           string
	HTMLURL        string
	Author         string
	State          string
	Merged         bool
	MergeCommitSHA string
	Head           string
	Base           string
	Labels         []string
}
