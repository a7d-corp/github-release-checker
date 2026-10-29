// Package github finds the latest release of a repository and counts the
// commits on its default branch since that release.
package github

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	gh "github.com/google/go-github/v93/github"

	"github.com/a7d-corp/github-release-checker/internal/config"
)

const requestTimeout = 30 * time.Second

// Result is the release status of one repository.
type Result struct {
	Repo         config.Repo
	Tag          string
	PublishedAt  time.Time
	CommitsSince int
	NoRelease    bool
	Err          error
}

// Checker queries the GitHub API.
type Checker struct {
	client *gh.Client
}

// NewChecker returns a Checker that authenticates with token. An empty
// baseURL uses the public GitHub API.
func NewChecker(token, baseURL string) (*Checker, error) {
	opts := []gh.ClientOptionsFunc{
		gh.WithAuthToken(token),
		gh.WithTimeout(requestTimeout),
	}
	if baseURL != "" {
		opts = append(opts, gh.WithURLs(&baseURL, nil))
	}

	client, err := gh.NewClient(opts...)
	if err != nil {
		return nil, fmt.Errorf("create GitHub client: %w", err)
	}
	return &Checker{client: client}, nil
}

// Check returns the release status of repo. Errors go into Result.Err.
func (c *Checker) Check(ctx context.Context, repo config.Repo) Result {
	result := Result{Repo: repo}

	r, _, err := c.client.Repositories.Get(ctx, repo.Owner, repo.Name)
	if err != nil {
		result.Err = fmt.Errorf("get repository: %w", err)
		return result
	}

	release, _, err := c.client.Repositories.GetLatestRelease(ctx, repo.Owner, repo.Name)
	var errResp *gh.ErrorResponse
	if errors.As(err, &errResp) && errResp.Response != nil && errResp.Response.StatusCode == http.StatusNotFound {
		result.NoRelease = true
		return result
	}
	if err != nil {
		result.Err = fmt.Errorf("get latest release: %w", err)
		return result
	}
	result.Tag = release.GetTagName()
	result.PublishedAt = release.GetPublishedAt().Time

	comparison, _, err := c.client.Repositories.CompareCommits(ctx, repo.Owner, repo.Name, result.Tag, r.GetDefaultBranch(), nil)
	if err != nil {
		result.Err = fmt.Errorf("compare %s...%s: %w", result.Tag, r.GetDefaultBranch(), err)
		return result
	}
	result.CommitsSince = comparison.GetAheadBy()

	return result
}
