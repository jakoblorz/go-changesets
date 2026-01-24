package github

import (
	"context"
	"fmt"
	"os"

	"github.com/google/go-github/v57/github"
	"golang.org/x/oauth2"
)

// Client implements GitHubClient using the real GitHub API
type Client struct {
	client *github.Client
}

// NewClient creates a new GitHub API client
func NewClient(token string) *Client {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)

	return &Client{
		client: github.NewClient(tc),
	}
}

var (
	ErrGitHubTokenNotFound = fmt.Errorf("GITHUB_TOKEN or GH_TOKEN environment variable not found")
)

// NewClientFromEnv creates a GitHub client using the token from environment variables
func NewClientFromEnv() (*Client, error) {
	token := os.Getenv("GH_TOKEN")
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
	}
	if token == "" {
		return nil, ErrGitHubTokenNotFound
	}

	return NewClient(token), nil
}

// NewClientWithoutAuth creates a GitHub client without authentication (for public operations)
func NewClientWithoutAuth() *Client {
	return &Client{
		client: github.NewClient(nil),
	}
}

func (c *Client) GetLatestRelease(ctx context.Context, owner, repo string) (*Release, error) {
	release, _, err := c.client.Repositories.GetLatestRelease(ctx, owner, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest release: %w", err)
	}
	return convertRelease(release), nil
}

func (c *Client) GetReleaseByTag(ctx context.Context, owner, repo, tag string) (*Release, error) {
	release, _, err := c.client.Repositories.GetReleaseByTag(ctx, owner, repo, tag)
	if err != nil {
		return nil, fmt.Errorf("failed to get release by tag %s: %w", tag, err)
	}
	return convertRelease(release), nil
}

func (c *Client) CreateRelease(ctx context.Context, owner, repo string, req *CreateReleaseRequest) (*Release, error) {
	ghRelease := &github.RepositoryRelease{
		TagName:         &req.TagName,
		Name:            &req.Name,
		Body:            &req.Body,
		Draft:           &req.Draft,
		Prerelease:      &req.Prerelease,
		TargetCommitish: &req.TargetCommitish,
	}

	release, _, err := c.client.Repositories.CreateRelease(ctx, owner, repo, ghRelease)
	if err != nil {
		return nil, fmt.Errorf("failed to create release: %w", err)
	}
	return convertRelease(release), nil
}

func (c *Client) GetRepository(ctx context.Context, owner, repo string) (*Repository, error) {
	repository, _, err := c.client.Repositories.Get(ctx, owner, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}
	return convertRepository(repository), nil
}

func convertRelease(r *github.RepositoryRelease) *Release {
	release := &Release{
		ID:         r.GetID(),
		TagName:    r.GetTagName(),
		Name:       r.GetName(),
		Body:       r.GetBody(),
		Draft:      r.GetDraft(),
		Prerelease: r.GetPrerelease(),
	}

	if !r.GetCreatedAt().IsZero() {
		release.CreatedAt = r.GetCreatedAt().Time
	}
	if !r.GetPublishedAt().IsZero() {
		release.PublishedAt = r.GetPublishedAt().Time
	}

	return release
}

func convertRepository(r *github.Repository) *Repository {
	return &Repository{
		Owner:         r.GetOwner().GetLogin(),
		Name:          r.GetName(),
		FullName:      r.GetFullName(),
		URL:           r.GetHTMLURL(),
		DefaultBranch: r.GetDefaultBranch(),
	}
}

func (c *Client) GetPullRequest(ctx context.Context, owner, repo string, number int) (*PullRequest, error) {
	pr, _, err := c.client.PullRequests.Get(ctx, owner, repo, number)
	if err != nil {
		return nil, fmt.Errorf("failed to get pull request #%d: %w", number, err)
	}
	return convertPullRequest(pr), nil
}

func (c *Client) ListPullRequestsByCommit(ctx context.Context, owner, repo, sha string) ([]*PullRequest, error) {
	prs, _, err := c.client.PullRequests.ListPullRequestsWithCommit(ctx, owner, repo, sha, &github.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list pull requests for commit %s: %w", sha, err)
	}

	result := make([]*PullRequest, 0, len(prs))
	for _, pr := range prs {
		result = append(result, convertPullRequest(pr))
	}
	return result, nil
}

func convertPullRequest(pr *github.PullRequest) *PullRequest {
	result := &PullRequest{
		Number:  pr.GetNumber(),
		Title:   pr.GetTitle(),
		Body:    pr.GetBody(),
		HTMLURL: pr.GetHTMLURL(),
		State:   pr.GetState(),
		Merged:  pr.GetMerged(),
		Head:    pr.GetHead().GetRef(),
		Base:    pr.GetBase().GetRef(),
		Labels:  extractLabels(pr.Labels),
	}

	if pr.GetUser() != nil {
		result.Author = pr.GetUser().GetLogin()
	}

	if pr.GetMergeCommitSHA() != "" {
		result.MergeCommitSHA = pr.GetMergeCommitSHA()
	}

	return result
}

func filterSlices[T any](input []T, predicate func(T) bool) []T {
	result := make([]T, 0, len(input))
	for _, item := range input {
		if predicate(item) {
			result = append(result, item)
		}
	}
	return result
}

func (c *Client) GetPullRequestByHead(ctx context.Context, owner, repo, headBranch string) (*PullRequest, error) {
	prs, _, err := c.client.PullRequests.List(ctx, owner, repo, &github.PullRequestListOptions{
		Head: headBranch,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list pull requests with head %s: %w", headBranch, err)
	}

	prs = filterSlices(prs, func(pr *github.PullRequest) bool {
		return pr.GetHead().GetRef() == headBranch && pr.GetState() != "closed"
	})

	if len(prs) == 0 {
		return nil, nil
	}

	return convertPullRequest(prs[0]), nil
}

func (c *Client) CreatePullRequest(ctx context.Context, owner, repo string, req *CreatePullRequestRequest) (*PullRequest, error) {
	title := req.Title
	body := req.Body
	head := req.Head
	base := req.Base
	draft := req.Draft
	newPR := &github.NewPullRequest{
		Title: &title,
		Body:  &body,
		Head:  &head,
		Base:  &base,
		Draft: &draft,
	}

	pr, _, err := c.client.PullRequests.Create(ctx, owner, repo, newPR)
	if err != nil {
		return nil, fmt.Errorf("failed to create pull request: %w", err)
	}

	return convertPullRequest(pr), nil
}

func (c *Client) UpdatePullRequest(ctx context.Context, owner, repo string, number int, req *UpdatePullRequestRequest) (*PullRequest, error) {
	title := req.Title
	body := req.Body
	pr, _, err := c.client.PullRequests.Edit(ctx, owner, repo, number, &github.PullRequest{
		Title: &title,
		Body:  &body,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update pull request #%d: %w", number, err)
	}

	return convertPullRequest(pr), nil
}

func (c *Client) ClosePullRequest(ctx context.Context, owner, repo string, number int) error {
	_, _, err := c.client.PullRequests.Merge(ctx, owner, repo, number, "", &github.PullRequestOptions{
		MergeMethod: "close",
	})
	if err != nil {
		return fmt.Errorf("failed to close pull request #%d: %w", number, err)
	}
	return nil
}

func (c *Client) DeleteBranch(ctx context.Context, owner, repo, branch string) error {
	_, err := c.client.Git.DeleteRef(ctx, owner, repo, "heads/"+branch)
	if err != nil {
		return fmt.Errorf("failed to delete branch %s: %w", branch, err)
	}
	return nil
}

func extractLabels(labels []*github.Label) []string {
	result := make([]string, 0, len(labels))
	for _, label := range labels {
		if label != nil {
			result = append(result, label.GetName())
		}
	}
	return result
}
