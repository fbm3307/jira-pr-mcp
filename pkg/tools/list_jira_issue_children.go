package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type ListJiraIssueChildren struct {
	deps *Deps
}

func NewListJiraIssueChildren(deps *Deps) Tool {
	return &ListJiraIssueChildren{deps: deps}
}

type listJiraIssueChildrenArgs struct {
	IssueKey   string `json:"issue_key"`
	Status     string `json:"status,omitempty"`
	StartAt    *int   `json:"start_at,omitempty"`
	MaxResults *int   `json:"max_results,omitempty"`
}

func (t *ListJiraIssueChildren) RegisterWith(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_jira_issue_children",
		Description: "List all child issues (Tasks, Stories, Bugs) under a parent issue (typically an Epic). Use this before creating a new Task to check for duplicates. Supports pagination via start_at and max_results.",
	}, t.handle)
}

func (t *ListJiraIssueChildren) handle(ctx context.Context, _ *mcp.CallToolRequest, args listJiraIssueChildrenArgs) (*mcp.CallToolResult, any, error) {
	if args.IssueKey == "" {
		return errText("issue_key is required"), nil, nil
	}

	maxResults := 50
	if args.MaxResults != nil && *args.MaxResults > 0 {
		maxResults = *args.MaxResults
	}
	startAt := 0
	if args.StartAt != nil {
		startAt = *args.StartAt
	}

	jql := fmt.Sprintf("parent = %s", args.IssueKey)
	if args.Status != "" {
		if args.Status == "open" {
			jql += " AND status != Done AND status != Closed"
		} else {
			jql += fmt.Sprintf(" AND status = %q", args.Status)
		}
	}
	jql += " ORDER BY created DESC"

	result, err := t.deps.Jira.SearchIssues(ctx, jql, startAt, maxResults)
	if err != nil {
		return errText("Failed to list children of %s: %v", args.IssueKey, err), nil, nil
	}

	output := childrenOutput{
		Parent:     args.IssueKey,
		Total:      result.Total,
		StartAt:    result.StartAt,
		MaxResults: result.MaxResults,
	}
	for _, issue := range result.Issues {
		child := childShort{
			Key:     issue.Key,
			Summary: issue.Fields.Summary,
			Status:  issue.Fields.Status.Name,
			Type:    issue.Fields.IssueType.Name,
			URL:     fmt.Sprintf("%s/browse/%s", t.deps.Jira.BaseURL(), issue.Key),
		}
		if issue.Fields.Assignee != nil {
			child.Assignee = issue.Fields.Assignee.DisplayName
		}
		if issue.Fields.Labels != nil {
			child.Labels = issue.Fields.Labels
		}
		output.Children = append(output.Children, child)
	}

	data, _ := json.MarshalIndent(output, "", "  ")
	return okText(string(data)), nil, nil
}

type childrenOutput struct {
	Parent     string       `json:"parent"`
	Total      int          `json:"total"`
	StartAt    int          `json:"start_at"`
	MaxResults int          `json:"max_results"`
	Children   []childShort `json:"children"`
}

type childShort struct {
	Key      string   `json:"key"`
	Summary  string   `json:"summary"`
	Status   string   `json:"status"`
	Type     string   `json:"type"`
	Assignee string   `json:"assignee,omitempty"`
	Labels   []string `json:"labels,omitempty"`
	URL      string   `json:"url"`
}
