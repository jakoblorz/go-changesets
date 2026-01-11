package changeset

import (
	"context"
	"fmt"

	"github.com/jakoblorz/go-changesets/internal/git"
	"github.com/jakoblorz/go-changesets/internal/github"
	"github.com/jakoblorz/go-changesets/internal/models"
)

type PREnricher struct {
	git git.GitClient
	gh  github.GitHubClient
}

type PREnrichmentResult struct {
	Enriched int
	Warnings []error
}

func NewPREnricher(gitClient git.GitClient, ghClient github.GitHubClient) *PREnricher {
	return &PREnricher{git: gitClient, gh: ghClient}
}

func (e *PREnricher) Enrich(ctx context.Context, changesets []*models.Changeset, owner, repo string) (PREnrichmentResult, error) {
	if e.gh == nil {
		return PREnrichmentResult{}, fmt.Errorf("GitHub client not available")
	}
	if e.git == nil {
		return PREnrichmentResult{}, fmt.Errorf("git client not available")
	}

	result := PREnrichmentResult{}

	for _, cs := range changesets {
		if cs.FilePath == "" {
			continue
		}

		commitSHA, err := e.git.GetFileCreationCommit(cs.FilePath)
		if err != nil || commitSHA == "" {
			continue
		}

		prs, err := e.gh.ListPullRequestsByCommit(ctx, owner, repo, commitSHA)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Errorf("failed to lookup PRs for %s: %w", cs.ID, err))
			continue
		}

		pr := selectBestPR(prs, commitSHA)
		if pr == nil {
			continue
		}

		cs.PR = &models.PullRequest{
			Number: pr.Number,
			Title:  pr.Title,
			URL:    pr.HTMLURL,
			Author: pr.Author,
		}
		result.Enriched++
	}

	return result, nil
}

func selectBestPR(prs []*github.PullRequest, commitSHA string) *github.PullRequest {
	if len(prs) == 0 {
		return nil
	}

	for _, pr := range prs {
		if pr.Merged && pr.MergeCommitSHA == commitSHA {
			return pr
		}
	}

	for _, pr := range prs {
		if pr.Merged {
			return pr
		}
	}

	for _, pr := range prs {
		if pr.State == "closed" {
			return pr
		}
	}

	return prs[0]
}
