package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/codeready-toolchain/jira-pr-mcp/pkg/jira"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type SearchJiraIssues struct {
	deps *Deps
}

func NewSearchJiraIssues(deps *Deps) Tool {
	return &SearchJiraIssues{deps: deps}
}

type searchJiraIssuesArgs struct {
	JQL        string   `json:"jql"`
	Text       string   `json:"text"`
	Status     string   `json:"status"`
	Labels     []string `json:"labels"`
	StartAt    *int     `json:"start_at,omitempty"`
	MaxResults int      `json:"max_results"`
}

func (t *SearchJiraIssues) RegisterWith(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "search_jira_issues",
		Description: "Search Jira Tasks/Stories/Bugs using JQL or text search. Returns matching issues with key, summary, status, type, and epic link. Use this to find if a relevant Jira issue already exists for a PR.",
	}, t.handle)
}

func (t *SearchJiraIssues) handle(ctx context.Context, _ *mcp.CallToolRequest, args searchJiraIssuesArgs) (*mcp.CallToolResult, any, error) {
	maxResults := args.MaxResults
	if maxResults <= 0 {
		maxResults = 20
	}

	var searchResult *jira.SearchResult
	var err error

	startAt := 0
	if args.StartAt != nil {
		startAt = *args.StartAt
	}

	if args.JQL != "" {
		searchResult, err = t.deps.Jira.SearchIssues(ctx, args.JQL, startAt, maxResults)
	} else {
		searchResult, err = t.deps.Jira.SearchTasksStoriesBugs(ctx, args.Text, args.Status, args.Labels, startAt, maxResults)
	}

	if err != nil {
		return errText("Jira search failed: %v", err), nil, nil
	}

	output := formatSearchResult(searchResult, t.deps.Jira.BaseURL())
	data, _ := json.MarshalIndent(output, "", "  ")
	return okText(string(data)), nil, nil
}

type searchOutput struct {
	Total      int          `json:"total"`
	StartAt    int          `json:"start_at"`
	MaxResults int          `json:"max_results"`
	Issues     []issueShort `json:"issues"`
}

type issueShort struct {
	Key         string   `json:"key"`
	Summary     string   `json:"summary"`
	Description string   `json:"description,omitempty"`
	Status      string   `json:"status"`
	Type        string   `json:"type"`
	Assignee    string   `json:"assignee,omitempty"`
	Labels      []string `json:"labels,omitempty"`
	EpicLink    string   `json:"epic_link,omitempty"`
	Parent      string   `json:"parent,omitempty"`
	Created     string   `json:"created"`
	Updated     string   `json:"updated"`
	URL         string   `json:"url"`
}

func formatSearchResult(sr *jira.SearchResult, baseURL string) *searchOutput {
	out := &searchOutput{Total: sr.Total, StartAt: sr.StartAt, MaxResults: sr.MaxResults}
	for _, issue := range sr.Issues {
		item := issueShort{
			Key:     issue.Key,
			Summary: issue.Fields.Summary,
			Status:  issue.Fields.Status.Name,
			Type:    issue.Fields.IssueType.Name,
			Labels:  issue.Fields.Labels,
			Created: issue.Fields.Created,
			Updated: issue.Fields.Updated,
			URL:     fmt.Sprintf("%s/browse/%s", baseURL, issue.Key),
		}
		if desc := issue.Fields.Description; len(desc) > 200 {
			item.Description = desc[:200] + "..."
		} else {
			item.Description = desc
		}
		if issue.Fields.Assignee != nil {
			item.Assignee = issue.Fields.Assignee.DisplayName
		}
		if issue.Fields.EpicLink != "" {
			item.EpicLink = issue.Fields.EpicLink
		}
		if issue.Fields.Parent != nil {
			item.Parent = issue.Fields.Parent.Key
		}
		out.Issues = append(out.Issues, item)
	}
	return out
}
