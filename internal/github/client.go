package github

import (
	"context"
	"fmt"

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
		HTMLURL: pr.GetHTMLURL(),
		State:   pr.GetState(),
		Merged:  pr.GetMerged(),
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

func extractLabels(labels []*github.Label) []string {
	result := make([]string, 0, len(labels))
	for _, label := range labels {
		if label != nil {
			result = append(result, label.GetName())
		}
	}
	return result
}