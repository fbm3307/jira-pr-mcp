package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type GetJiraIssue struct {
	deps *Deps
}

func NewGetJiraIssue(deps *Deps) Tool {
	return &GetJiraIssue{deps: deps}
}

type getJiraIssueArgs struct {
	IssueKey string `json:"issue_key"`
}

type issueDetail struct {
	Key         string            `json:"key"`
	Summary     string            `json:"summary"`
	Description string            `json:"description"`
	Status      string            `json:"status"`
	Type        string            `json:"type"`
	Assignee    string            `json:"assignee,omitempty"`
	Reporter    string            `json:"reporter,omitempty"`
	Labels      []string          `json:"labels,omitempty"`
	StoryPoints float64           `json:"story_points,omitempty"`
	EpicLink    string            `json:"epic_link,omitempty"`
	Parent      string            `json:"parent,omitempty"`
	RemoteLinks []remoteLinkShort `json:"remote_links,omitempty"`
	URL         string            `json:"url"`
}

type remoteLinkShort struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

func (t *GetJiraIssue) RegisterWith(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_jira_issue",
		Description: "Get full details of a specific Jira issue by key (e.g. SANDBOX-456). Returns summary, description, status, type, assignee, labels, story points, epic link, parent, and linked issues.",
	}, t.handle)
}

func (t *GetJiraIssue) handle(ctx context.Context, _ *mcp.CallToolRequest, args getJiraIssueArgs) (*mcp.CallToolResult, any, error) {
	if args.IssueKey == "" {
		return errText("issue_key is required"), nil, nil
	}

	issue, err := t.deps.Jira.GetIssue(ctx, args.IssueKey)
	if err != nil {
		return errText("Failed to get issue: %v", err), nil, nil
	}

	detail := issueDetail{
		Key:         issue.Key,
		Summary:     issue.Fields.Summary,
		Description: issue.Fields.Description,
		Status:      issue.Fields.Status.Name,
		Type:        issue.Fields.IssueType.Name,
		Labels:      issue.Fields.Labels,
		StoryPoints: issue.Fields.StoryPoints,
		EpicLink:    issue.Fields.EpicLink,
		URL:         fmt.Sprintf("%s/browse/%s", t.deps.Jira.BaseURL(), issue.Key),
	}

	if issue.Fields.Assignee != nil {
		detail.Assignee = issue.Fields.Assignee.DisplayName
	}
	if issue.Fields.Reporter != nil {
		detail.Reporter = issue.Fields.Reporter.DisplayName
	}
	if issue.Fields.Parent != nil {
		detail.Parent = fmt.Sprintf("%s (%s)", issue.Fields.Parent.Key, issue.Fields.Parent.Fields.Summary)
	}

	remoteLinks, err := t.deps.Jira.GetRemoteLinks(ctx, args.IssueKey)
	if err == nil {
		for _, rl := range remoteLinks {
			detail.RemoteLinks = append(detail.RemoteLinks, remoteLinkShort{
				Title: rl.Object.Title,
				URL:   rl.Object.URL,
			})
		}
	}

	data, _ := json.MarshalIndent(detail, "", "  ")
	return okText(string(data)), nil, nil
}
