package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type TransitionJiraIssue struct {
	deps *Deps
}

func NewTransitionJiraIssue(deps *Deps) Tool {
	return &TransitionJiraIssue{deps: deps}
}

type transitionJiraIssueArgs struct {
	IssueKey       string `json:"issue_key"`
	TransitionName string `json:"transition_name"`
}

func (t *TransitionJiraIssue) RegisterWith(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name: "transition_jira_issue",
		Description: `Move a Jira issue to a new workflow state (e.g., "In Progress", "In Review", "Done").
Provide the transition_name (case-insensitive). If the name doesn't match any available transition, the tool returns the list of valid transitions so you can retry.`,
	}, t.handle)
}

func (t *TransitionJiraIssue) handle(ctx context.Context, _ *mcp.CallToolRequest, args transitionJiraIssueArgs) (*mcp.CallToolResult, any, error) {
	if args.IssueKey == "" {
		return errText("issue_key is required"), nil, nil
	}
	if args.TransitionName == "" {
		return errText("transition_name is required"), nil, nil
	}

	issue, err := t.deps.Jira.GetIssue(ctx, args.IssueKey)
	if err != nil {
		return errText("Failed to get issue %s: %v", args.IssueKey, err), nil, nil
	}
	projectKey := strings.Split(issue.Key, "-")[0]
	if !strings.EqualFold(projectKey, t.deps.Jira.Project()) {
		return errText("Project scope violation: this server is configured for project %s, cannot transition %s", t.deps.Jira.Project(), args.IssueKey), nil, nil
	}

	transitions, err := t.deps.Jira.GetTransitions(ctx, args.IssueKey)
	if err != nil {
		return errText("Failed to get transitions: %v", err), nil, nil
	}

	var matchedID string
	for _, tr := range transitions {
		if strings.EqualFold(tr.Name, args.TransitionName) {
			matchedID = tr.ID
			break
		}
	}

	if matchedID == "" {
		available := make([]string, len(transitions))
		for i, tr := range transitions {
			available[i] = fmt.Sprintf("%s (→ %s)", tr.Name, tr.To.Name)
		}
		output := map[string]interface{}{
			"status":                "no_match",
			"message":              fmt.Sprintf("No transition matching %q found for %s", args.TransitionName, args.IssueKey),
			"current_status":       issue.Fields.Status.Name,
			"available_transitions": available,
		}
		data, _ := json.MarshalIndent(output, "", "  ")
		return okText(string(data)), nil, nil
	}

	if err := t.deps.Jira.TransitionIssue(ctx, args.IssueKey, matchedID); err != nil {
		return errText("Failed to transition issue: %v", err), nil, nil
	}

	output := map[string]interface{}{
		"key":         args.IssueKey,
		"status":      "transitioned",
		"from":        issue.Fields.Status.Name,
		"to":          args.TransitionName,
		"url":         fmt.Sprintf("%s/browse/%s", t.deps.Jira.BaseURL(), args.IssueKey),
	}
	data, _ := json.MarshalIndent(output, "", "  ")
	return okText(string(data)), nil, nil
}
