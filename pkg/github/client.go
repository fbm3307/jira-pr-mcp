package github

import (
	"context"
	"fmt"
	"log/slog"

	gh "github.com/google/go-github/v68/github"
	"golang.org/x/oauth2"
)

type Client struct {
	client *gh.Client
	logger *slog.Logger
}

func NewClient(token string, logger *slog.Logger) *Client {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(context.Background(), ts)
	return &Client{
		client: gh.NewClient(tc),
		logger: logger,
	}
}

func (c *Client) ValidateCredentials(ctx context.Context) error {
	_, _, err := c.client.Users.Get(ctx, "")
	if err != nil {
		return fmt.Errorf("github authentication failed -- check your GITHUB_TOKEN: %w", err)
	}
	return nil
}

func (c *Client) GetPR(ctx context.Context, owner, repo string, number int) (*PRDetails, error) {
	c.logger.Debug("fetching PR", "owner", owner, "repo", repo, "number", number)

	pr, _, err := c.client.PullRequests.Get(ctx, owner, repo, number)
	if err != nil {
		return nil, fmt.Errorf("failed to get PR %s/%s#%d: %w", owner, repo, number, err)
	}

	details := &PRDetails{
		Owner:      owner,
		Repo:       repo,
		Number:     number,
		Title:      pr.GetTitle(),
		Body:       pr.GetBody(),
		State:      pr.GetState(),
		HeadBranch: pr.GetHead().GetRef(),
		BaseBranch: pr.GetBase().GetRef(),
		URL:        pr.GetHTMLURL(),
	}

	for _, l := range pr.Labels {
		details.Labels = append(details.Labels, l.GetName())
	}

	commits, err := c.listCommits(ctx, owner, repo, number)
	if err != nil {
		c.logger.Warn("failed to list commits, continuing without them", "error", err)
	} else {
		details.Commits = commits
	}

	files, err := c.listFiles(ctx, owner, repo, number)
	if err != nil {
		c.logger.Warn("failed to list changed files, continuing without them", "error", err)
	} else {
		details.ChangedFiles = files
	}

	return details, nil
}

func (c *Client) listCommits(ctx context.Context, owner, repo string, number int) ([]CommitInfo, error) {
	commits, _, err := c.client.PullRequests.ListCommits(ctx, owner, repo, number, &gh.ListOptions{PerPage: 30})
	if err != nil {
		return nil, err
	}

	var result []CommitInfo
	for _, commit := range commits {
		result = append(result, CommitInfo{
			SHA:     commit.GetSHA()[:7],
			Message: commit.GetCommit().GetMessage(),
		})
	}
	return result, nil
}

func (c *Client) listFiles(ctx context.Context, owner, repo string, number int) ([]FileChange, error) {
	files, _, err := c.client.PullRequests.ListFiles(ctx, owner, repo, number, &gh.ListOptions{PerPage: 100})
	if err != nil {
		return nil, err
	}

	var result []FileChange
	for _, f := range files {
		result = append(result, FileChange{
			Filename:  f.GetFilename(),
			Additions: f.GetAdditions(),
			Deletions: f.GetDeletions(),
			Status:    f.GetStatus(),
		})
	}
	return result, nil
}
