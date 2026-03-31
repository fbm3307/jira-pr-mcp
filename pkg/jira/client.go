package jira

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	baseURL    string
	email      string
	token      string
	project    string
	httpClient *http.Client
	logger     *slog.Logger
	cache      *Cache
}

// NewClient creates a Jira Cloud client using Basic auth (email + API token).
func NewClient(baseURL, email, token, project string, cacheTTL time.Duration, logger *slog.Logger) *Client {
	if cacheTTL <= 0 {
		cacheTTL = 5 * time.Minute
	}
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		email:   email,
		token:   token,
		project: project,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
		cache:  NewCache(cacheTTL),
	}
}

func (c *Client) ValidateCredentials(ctx context.Context) (*MySelfResponse, error) {
	var result MySelfResponse
	if err := c.doRequest(ctx, http.MethodGet, "/rest/api/3/myself", nil, &result); err != nil {
		return nil, fmt.Errorf("jira authentication failed -- check your JIRA_EMAIL and JIRA_API_TOKEN: %w", err)
	}
	return &result, nil
}

func (c *Client) GetIssue(ctx context.Context, issueKey string) (*Issue, error) {
	path := fmt.Sprintf("/rest/api/3/issue/%s?fields=*all", url.PathEscape(issueKey))
	var rawResp json.RawMessage
	if err := c.doRequest(ctx, http.MethodGet, path, nil, &rawResp); err != nil {
		return nil, fmt.Errorf("failed to get issue %s: %w", issueKey, err)
	}
	var issue Issue
	if err := json.Unmarshal(rawResp, &issue); err != nil {
		return nil, fmt.Errorf("failed to parse issue %s: %w", issueKey, err)
	}
	var wrapper struct {
		Fields json.RawMessage `json:"fields"`
	}
	if err := json.Unmarshal(rawResp, &wrapper); err == nil {
		issue.Fields.RawFields = wrapper.Fields
	}
	issue.Fields.ParseDescription()
	c.extractCustomFields(&issue)
	return &issue, nil
}

func (c *Client) SearchIssues(ctx context.Context, jql string, startAt, maxResults int) (*SearchResult, error) {
	body := map[string]interface{}{
		"jql":        jql,
		"maxResults": maxResults,
		"fields":     []string{"summary", "description", "status", "issuetype", "assignee", "reporter", "labels", "created", "updated", "issuelinks", "parent", "customfield_12310243", "customfield_12311140"},
	}
	if startAt > 0 {
		body["startAt"] = startAt
	}
	var rawResp json.RawMessage
	if err := c.doRequest(ctx, http.MethodPost, "/rest/api/3/search/jql", body, &rawResp); err != nil {
		return nil, fmt.Errorf("jira search failed: %w", err)
	}
	var result SearchResult
	if err := json.Unmarshal(rawResp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse search results: %w", err)
	}
	var rawWrapper struct {
		Issues []struct {
			Fields json.RawMessage `json:"fields"`
		} `json:"issues"`
	}
	if json.Unmarshal(rawResp, &rawWrapper) == nil {
		for i := range result.Issues {
			if i < len(rawWrapper.Issues) {
				result.Issues[i].Fields.RawFields = rawWrapper.Issues[i].Fields
			}
		}
	}
	for i := range result.Issues {
		result.Issues[i].Fields.ParseDescription()
		c.extractCustomFields(&result.Issues[i])
	}
	return &result, nil
}

// SearchByType searches for issues of a specific type in the configured project.
func (c *Client) SearchByType(ctx context.Context, issueType, text, status string, startAt, maxResults int) (*SearchResult, error) {
	jql := fmt.Sprintf("project = %s AND issuetype = %s", c.project, issueType)
	if text != "" {
		jql += fmt.Sprintf(` AND text ~ %q`, text)
	}
	if status != "" {
		if strings.EqualFold(status, "open") {
			jql += " AND status != Done"
		} else {
			jql += fmt.Sprintf(" AND status = %q", status)
		}
	}
	jql += " ORDER BY updated DESC"
	return c.SearchIssues(ctx, jql, startAt, maxResults)
}

func (c *Client) SearchTasksStoriesBugs(ctx context.Context, text, status string, labels []string, startAt, maxResults int) (*SearchResult, error) {
	jql := fmt.Sprintf("project = %s AND issuetype in (Task, Story, Bug)", c.project)
	if text != "" {
		jql += fmt.Sprintf(` AND text ~ %q`, text)
	}
	if status != "" {
		if strings.EqualFold(status, "open") {
			jql += " AND status != Done"
		} else {
			jql += fmt.Sprintf(" AND status = %q", status)
		}
	}
	for _, label := range labels {
		jql += fmt.Sprintf(" AND labels = %q", label)
	}
	jql += " ORDER BY updated DESC"
	return c.SearchIssues(ctx, jql, startAt, maxResults)
}

// FindDuplicate searches for an existing Task/Story/Bug with the same summary across all Epics.
func (c *Client) FindDuplicate(ctx context.Context, summary string) (*Issue, error) {
	jql := fmt.Sprintf(`project = %s AND issuetype in (Task, Story, Bug) AND summary ~ %q`, c.project, summary)
	result, err := c.SearchIssues(ctx, jql, 0, 5)
	if err != nil {
		return nil, err
	}
	for _, issue := range result.Issues {
		if strings.EqualFold(strings.TrimSpace(issue.Fields.Summary), strings.TrimSpace(summary)) {
			return &issue, nil
		}
	}
	return nil, nil
}

func (c *Client) CreateIssue(ctx context.Context, req *CreateIssueRequest) (*CreateIssueResponse, error) {
	var result CreateIssueResponse
	if err := c.doRequest(ctx, http.MethodPost, "/rest/api/3/issue", req, &result); err != nil {
		return nil, fmt.Errorf("failed to create issue: %w", err)
	}
	c.cache.Invalidate()
	return &result, nil
}

func (c *Client) GetRemoteLinks(ctx context.Context, issueKey string) ([]RemoteLink, error) {
	path := fmt.Sprintf("/rest/api/3/issue/%s/remotelink", url.PathEscape(issueKey))
	var links []RemoteLink
	if err := c.doRequest(ctx, http.MethodGet, path, nil, &links); err != nil {
		return nil, fmt.Errorf("failed to get remote links for %s: %w", issueKey, err)
	}
	return links, nil
}

func (c *Client) UpdateIssue(ctx context.Context, issueKey string, fields map[string]interface{}) error {
	path := fmt.Sprintf("/rest/api/3/issue/%s", url.PathEscape(issueKey))
	body := map[string]interface{}{"fields": fields}
	if err := c.doRequest(ctx, http.MethodPut, path, body, nil); err != nil {
		return fmt.Errorf("failed to update issue %s: %w", issueKey, err)
	}
	c.cache.Invalidate()
	return nil
}

func (c *Client) GetTransitions(ctx context.Context, issueKey string) ([]Transition, error) {
	path := fmt.Sprintf("/rest/api/3/issue/%s/transitions", url.PathEscape(issueKey))
	var resp TransitionsResponse
	if err := c.doRequest(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return nil, fmt.Errorf("failed to get transitions for %s: %w", issueKey, err)
	}
	return resp.Transitions, nil
}

func (c *Client) TransitionIssue(ctx context.Context, issueKey, transitionID string) error {
	path := fmt.Sprintf("/rest/api/3/issue/%s/transitions", url.PathEscape(issueKey))
	body := map[string]interface{}{
		"transition": map[string]string{"id": transitionID},
	}
	if err := c.doRequest(ctx, http.MethodPost, path, body, nil); err != nil {
		return fmt.Errorf("failed to transition issue %s: %w", issueKey, err)
	}
	return nil
}

func (c *Client) AssignIssue(ctx context.Context, issueKey, accountID string) error {
	path := fmt.Sprintf("/rest/api/3/issue/%s/assignee", url.PathEscape(issueKey))
	body := map[string]string{"accountId": accountID}
	if err := c.doRequest(ctx, http.MethodPut, path, body, nil); err != nil {
		return fmt.Errorf("failed to assign issue %s: %w", issueKey, err)
	}
	return nil
}

func (c *Client) SearchUsers(ctx context.Context, query string) ([]UserSearchResult, error) {
	path := fmt.Sprintf("/rest/api/3/user/search?query=%s&maxResults=10", url.QueryEscape(query))
	var users []UserSearchResult
	if err := c.doRequest(ctx, http.MethodGet, path, nil, &users); err != nil {
		return nil, fmt.Errorf("failed to search users: %w", err)
	}
	return users, nil
}

func (c *Client) AddRemoteLink(ctx context.Context, issueKey string, req *RemoteLinkRequest) (*RemoteLinkResponse, error) {
	path := fmt.Sprintf("/rest/api/3/issue/%s/remotelink", url.PathEscape(issueKey))
	var result RemoteLinkResponse
	if err := c.doRequest(ctx, http.MethodPost, path, req, &result); err != nil {
		return nil, fmt.Errorf("failed to add remote link to %s: %w", issueKey, err)
	}
	return &result, nil
}

func (c *Client) AddComment(ctx context.Context, issueKey string, body string) (*CommentResponse, error) {
	path := fmt.Sprintf("/rest/api/3/issue/%s/comment", url.PathEscape(issueKey))
	req := &CommentRequest{Body: NewADFDescription(body)}
	var result CommentResponse
	if err := c.doRequest(ctx, http.MethodPost, path, req, &result); err != nil {
		return nil, fmt.Errorf("failed to add comment to %s: %w", issueKey, err)
	}
	return &result, nil
}

func (c *Client) ListBoardSprints(ctx context.Context, boardID int, state string) (*SprintList, error) {
	path := fmt.Sprintf("/rest/agile/1.0/board/%d/sprint?state=%s", boardID, url.QueryEscape(state))
	var result SprintList
	if err := c.doRequest(ctx, http.MethodGet, path, nil, &result); err != nil {
		return nil, fmt.Errorf("failed to list sprints for board %d: %w", boardID, err)
	}
	return &result, nil
}

func (c *Client) GetSprintIssues(ctx context.Context, sprintID, maxResults int) (*SprintIssues, error) {
	path := fmt.Sprintf("/rest/agile/1.0/sprint/%d/issue?maxResults=%d", sprintID, maxResults)
	var result SprintIssues
	if err := c.doRequest(ctx, http.MethodGet, path, nil, &result); err != nil {
		return nil, fmt.Errorf("failed to get issues for sprint %d: %w", sprintID, err)
	}
	return &result, nil
}

func (c *Client) Project() string {
	return c.project
}

func (c *Client) BaseURL() string {
	return c.baseURL
}

func (c *Client) GetCache() *Cache {
	return c.cache
}

const maxRetries = 3

func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	var bodyData []byte
	if body != nil {
		var err error
		bodyData, err = json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
	}

	reqURL := c.baseURL + path
	creds := base64.StdEncoding.EncodeToString([]byte(c.email + ":" + c.token))

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			wait := time.Duration(1<<uint(attempt-1)) * 500 * time.Millisecond
			c.logger.Debug("retrying request", "attempt", attempt+1, "wait", wait, "url", reqURL)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(wait):
			}
		}

		var bodyReader io.Reader
		if bodyData != nil {
			bodyReader = bytes.NewReader(bodyData)
		}

		req, err := http.NewRequestWithContext(ctx, method, reqURL, bodyReader)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Authorization", "Basic "+creds)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")

		c.logger.Debug("jira request", "method", method, "url", reqURL, "attempt", attempt+1)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			continue
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("failed to read response body: %w", err)
			continue
		}

		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			return fmt.Errorf("authentication failed (HTTP %d) -- check your JIRA_EMAIL and JIRA_API_TOKEN", resp.StatusCode)
		}

		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			retryAfter := resp.Header.Get("Retry-After")
			lastErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, truncate(string(respBody), 500))
			if retryAfter != "" {
				if d, parseErr := time.ParseDuration(retryAfter + "s"); parseErr == nil {
					select {
					case <-ctx.Done():
						return ctx.Err()
					case <-time.After(d):
					}
				}
			}
			continue
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return fmt.Errorf("HTTP %d: %s", resp.StatusCode, truncate(string(respBody), 500))
		}

		if result != nil && len(respBody) > 0 {
			if err := json.Unmarshal(respBody, result); err != nil {
				return fmt.Errorf("failed to parse response: %w", err)
			}
		}
		return nil
	}
	return fmt.Errorf("request failed after %d attempts: %w", maxRetries, lastErr)
}

func (c *Client) extractCustomFields(issue *Issue) {
	if issue.Fields.RawFields == nil {
		return
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(issue.Fields.RawFields, &raw); err != nil {
		return
	}
	if sp, ok := raw["customfield_12310243"]; ok && string(sp) != "null" {
		var pts float64
		if json.Unmarshal(sp, &pts) == nil {
			issue.Fields.StoryPoints = pts
		}
	}
	if el, ok := raw["customfield_12311140"]; ok && string(el) != "null" {
		var link string
		if json.Unmarshal(el, &link) == nil {
			issue.Fields.EpicLink = link
		}
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
