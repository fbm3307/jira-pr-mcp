package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/codeready-toolchain/jira-pr-mcp/pkg/jira"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type UpdateJiraIssue struct {
	deps *Deps
}

func NewUpdateJiraIssue(deps *Deps) Tool {
	return &UpdateJiraIssue{deps: deps}
}

type updateJiraIssueArgs struct {
	IssueKey    string   `json:"issue_key"`
	Summary     string   `json:"summary"`
	Description string   `json:"description"`
	Labels      []string `json:"labels"`
	StoryPoints float64  `json:"story_points"`
	Assignee    string   `json:"assignee"`
}

func (t *UpdateJiraIssue) RegisterWith(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name: "update_jira_issue",
		Description: `Update fields on an existing Jira issue. All fields are optional -- only provided fields are updated.
Supports: summary, description, labels, story_points (Fibonacci: 1,2,3,5,8,13,21), and assignee (email or display name).
For assignee, provide an email address or display name -- the tool resolves it to a Jira account ID automatically.`,
	}, t.handle)
}

func (t *UpdateJiraIssue) handle(ctx context.Context, _ *mcp.CallToolRequest, args updateJiraIssueArgs) (*mcp.CallToolResult, any, error) {
	if args.IssueKey == "" {
		return errText("issue_key is required"), nil, nil
	}

	issue, err := t.deps.Jira.GetIssue(ctx, args.IssueKey)
	if err != nil {
		return errText("Failed to get issue %s: %v", args.IssueKey, err), nil, nil
	}

	projectKey := strings.Split(issue.Key, "-")[0]
	if !strings.EqualFold(projectKey, t.deps.Jira.Project()) {
		return errText("Project scope violation: this server is configured for project %s, cannot update %s", t.deps.Jira.Project(), args.IssueKey), nil, nil
	}

	fields := make(map[string]interface{})
	updated := []string{}

	if args.Summary != "" {
		fields["summary"] = args.Summary
		updated = append(updated, "summary")
	}

	if args.Description != "" {
		fields["description"] = jira.NewADFDescription(args.Description)
		updated = append(updated, "description")
	}

	if args.Labels != nil {
		fields["labels"] = args.Labels
		updated = append(updated, "labels")
	}

	if args.StoryPoints > 0 {
		if !validFibonacci[args.StoryPoints] {
			return errText("story_points must be a Fibonacci number (1, 2, 3, 5, 8, 13, 21), got %.0f", args.StoryPoints), nil, nil
		}
		fields["customfield_12310243"] = args.StoryPoints
		updated = append(updated, "story_points")
	}

	if len(fields) > 0 {
		if err := t.deps.Jira.UpdateIssue(ctx, args.IssueKey, fields); err != nil {
			return errText("Failed to update issue: %v", err), nil, nil
		}
	}

	if args.Assignee != "" {
		users, err := t.deps.Jira.SearchUsers(ctx, args.Assignee)
		if err != nil {
			return errText("Failed to search for user %q: %v", args.Assignee, err), nil, nil
		}
		if len(users) == 0 {
			return errText("No Jira user found matching %q", args.Assignee), nil, nil
		}

		matched := matchUser(users, args.Assignee)
		if matched == nil {
			matched = &users[0]
		}

		if err := t.deps.Jira.AssignIssue(ctx, args.IssueKey, matched.AccountID); err != nil {
			return errText("Failed to assign issue: %v", err), nil, nil
		}
		updated = append(updated, fmt.Sprintf("assignee → %s", matched.DisplayName))
	}

	if len(updated) == 0 {
		return errText("No fields provided to update"), nil, nil
	}

	output := map[string]interface{}{
		"key":     args.IssueKey,
		"status":  "updated",
		"updated": updated,
		"url":     fmt.Sprintf("%s/browse/%s", t.deps.Jira.BaseURL(), args.IssueKey),
	}
	data, _ := json.MarshalIndent(output, "", "  ")
	return okText(string(data)), nil, nil
}

func matchUser(users []jira.UserSearchResult, query string) *jira.UserSearchResult {
	q := strings.ToLower(query)
	for i, u := range users {
		if strings.EqualFold(u.EmailAddr, query) || strings.EqualFold(u.DisplayName, query) {
			return &users[i]
		}
		if strings.Contains(strings.ToLower(u.DisplayName), q) {
			return &users[i]
		}
	}
	return nil
}
