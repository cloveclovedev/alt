package planning

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// GitHubClient reads public repositories without authentication.
type GitHubClient struct{ client *http.Client }

func NewGitHubClient() *GitHubClient {
	return &GitHubClient{client: &http.Client{Timeout: 20 * time.Second}}
}

// ValidatePublicRepository verifies the repository is reachable without credentials.
func (c *GitHubClient) ValidatePublicRepository(ctx context.Context, owner, name string) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/repos/"+url.PathEscape(owner)+"/"+url.PathEscape(name), nil)
	if err != nil {
		return fmt.Errorf("create GitHub repository request: %w", err)
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("X-GitHub-Api-Version", "2026-03-10")
	response, err := c.client.Do(request)
	if err != nil {
		return fmt.Errorf("send GitHub repository request: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("GitHub repository is not publicly reachable (%s)", response.Status)
	}
	var payload struct {
		Private bool `json:"private"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 128<<10)).Decode(&payload); err != nil {
		return fmt.Errorf("decode GitHub repository response: %w", err)
	}
	if payload.Private {
		return fmt.Errorf("GitHub repository is private")
	}
	return nil
}

// ListOpenIssues follows GitHub pagination and filters pull-request resources.
func (c *GitHubClient) ListOpenIssues(ctx context.Context, repository GitHubRepository) ([]GitHubIssue, error) {
	nextURL := "https://api.github.com/repos/" + url.PathEscape(repository.Owner) + "/" + url.PathEscape(repository.Name) + "/issues?state=open&per_page=100"
	var issues []GitHubIssue
	for nextURL != "" {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, nextURL, nil)
		if err != nil {
			return nil, fmt.Errorf("create GitHub issues request: %w", err)
		}
		request.Header.Set("Accept", "application/vnd.github+json")
		request.Header.Set("X-GitHub-Api-Version", "2026-03-10")
		response, err := c.client.Do(request)
		if err != nil {
			return nil, fmt.Errorf("send GitHub issues request: %w", err)
		}
		if response.StatusCode != http.StatusOK {
			response.Body.Close()
			return nil, fmt.Errorf("GitHub issues request returned %s", response.Status)
		}
		var page []struct {
			Number  int    `json:"number"`
			Title   string `json:"title"`
			HTMLURL string `json:"html_url"`
			Labels  []struct {
				Name string `json:"name"`
			} `json:"labels"`
			Milestone *struct {
				Title string `json:"title"`
			} `json:"milestone"`
			Assignees []struct {
				Login string `json:"login"`
			} `json:"assignees"`
			UpdatedAt   time.Time       `json:"updated_at"`
			PullRequest json.RawMessage `json:"pull_request"`
		}
		err = json.NewDecoder(io.LimitReader(response.Body, 4<<20)).Decode(&page)
		response.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("decode GitHub issues response: %w", err)
		}
		for _, item := range page {
			if len(item.PullRequest) > 0 && string(item.PullRequest) != "null" {
				continue
			}
			issue := GitHubIssue{RepositoryOwner: repository.Owner, RepositoryName: repository.Name, Number: item.Number, Title: item.Title, HTMLURL: item.HTMLURL, UpdatedAt: item.UpdatedAt}
			for _, label := range item.Labels {
				issue.Labels = append(issue.Labels, label.Name)
			}
			if item.Milestone != nil {
				issue.Milestone = item.Milestone.Title
			}
			for _, assignee := range item.Assignees {
				issue.Assignees = append(issue.Assignees, assignee.Login)
			}
			issues = append(issues, issue)
		}
		nextURL = githubNextPage(response.Header.Get("Link"))
	}
	return issues, nil
}

func githubNextPage(header string) string {
	for _, part := range strings.Split(header, ",") {
		sections := strings.Split(strings.TrimSpace(part), ";")
		if len(sections) != 2 || !strings.Contains(sections[1], "rel=\"next\"") {
			continue
		}
		value := strings.TrimSpace(sections[0])
		if strings.HasPrefix(value, "<") && strings.HasSuffix(value, ">") {
			return value[1 : len(value)-1]
		}
	}
	return ""
}

func parseGitHubRateLimit(response *http.Response) string {
	if response.StatusCode != http.StatusForbidden {
		return ""
	}
	if remaining := response.Header.Get("X-RateLimit-Remaining"); remaining == "0" {
		if reset, err := strconv.ParseInt(response.Header.Get("X-RateLimit-Reset"), 10, 64); err == nil {
			return time.Unix(reset, 0).UTC().Format(time.RFC3339)
		}
		return "unknown"
	}
	return ""
}
