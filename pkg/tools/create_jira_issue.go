package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/codeready-toolchain/jira-pr-mcp/pkg/jira"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type CreateJiraIssue struct {
	deps *Deps
}

func NewCreateJiraIssue(deps *Deps) Tool {
	return &CreateJiraIssue{deps: deps}
}

type createJiraIssueArgs struct {
	Project     string   `json:"project"`
	Summary     string   `json:"summary"`
	Description string   `json:"description"`
	IssueType   string   `json:"issue_type"`
	Labels      []string `json:"labels"`
	StoryPoints float64  `json:"story_points"`
	EpicKey     string   `json:"epic_key"`
}

var validFibonacci = map[float64]bool{1: true, 2: true, 3: true, 5: true, 8: true, 13: true, 21: true}

func (t *CreateJiraIssue) RegisterWith(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name: "create_jira_issue",
		Description: `Create a new Task or Story linked to an Epic. epic_key is REQUIRED -- every Task/Story must belong to an Epic.
Story points must be a Fibonacci number (1, 2, 3, 5, 8, 13, 21).
IMPORTANT: Write concise, human-style summaries and descriptions. Do NOT write verbose AI-generated prose. Before calling this tool, review existing issues (via search_jira_issues or get_jira_issue) to match the tone and brevity of human-written ones.`,
	}, t.handle)
}

func (t *CreateJiraIssue) handle(ctx context.Context, _ *mcp.CallToolRequest, args createJiraIssueArgs) (*mcp.CallToolResult, any, error) {
	if args.EpicKey == "" {
		return errText("epic_key is required -- every Task/Story must be linked to an Epic"), nil, nil
	}
	if args.Summary == "" {
		return errText("summary is required"), nil, nil
	}

	issueType := args.IssueType
	if issueType == "" {
		issueType = "Task"
	}

	project := args.Project
	if project == "" {
		project = t.deps.Jira.Project()
	}

	if !strings.EqualFold(project, t.deps.Jira.Project()) {
		return errText("Project scope violation: this server is configured for project %s, cannot create in %s", t.deps.Jira.Project(), project), nil, nil
	}

	if args.StoryPoints > 0 && !validFibonacci[args.StoryPoints] {
		return errText("story_points must be a Fibonacci number (1, 2, 3, 5, 8, 13, 21), got %.0f", args.StoryPoints), nil, nil
	}

	parent, err := t.deps.Jira.GetIssue(ctx, args.EpicKey)
	if err != nil {
		return errText("Failed to verify Epic %s: %v", args.EpicKey, err), nil, nil
	}
	if !strings.EqualFold(parent.Fields.IssueType.Name, "Epic") {
		return errText("%s is not an Epic (it's a %s) -- epic_key must point to an Epic", args.EpicKey, parent.Fields.IssueType.Name), nil, nil
	}

	dup, err := t.deps.Jira.FindDuplicate(ctx, args.Summary)
	if err != nil {
		return errText("Duplicate check failed: %v", err), nil, nil
	}
	if dup != nil {
		dupInfo := map[string]string{
			"status":  "duplicate_found",
			"message": fmt.Sprintf("An existing issue with a similar summary was found: %s (%s). Use the existing issue or change the summary to proceed.", dup.Key, dup.Fields.Summary),
			"key":     dup.Key,
			"summary": dup.Fields.Summary,
			"url":     fmt.Sprintf("%s/browse/%s", t.deps.Jira.BaseURL(), dup.Key),
		}
		if dup.Fields.Parent != nil {
			dupInfo["epic"] = dup.Fields.Parent.Key
		}
		data, _ := json.MarshalIndent(dupInfo, "", "  ")
		return okText(string(data)), nil, nil
	}

	req := &jira.CreateIssueRequest{
		Fields: jira.CreateIssueFields{
			Project:     jira.ProjectRef{Key: project},
			IssueType:   jira.IssueTypeRef{Name: issueType},
			Summary:     args.Summary,
			Description: jira.NewADFDescription(args.Description),
			Labels:      args.Labels,
			Parent:      &jira.ParentKeyRef{Key: args.EpicKey},
		},
	}
	if args.StoryPoints > 0 {
		req.Fields.StoryPoints = &args.StoryPoints
	}

	result, err := t.deps.Jira.CreateIssue(ctx, req)
	if err != nil {
		return errText("Failed to create issue: %v", err), nil, nil
	}

	output := map[string]interface{}{
		"key":         result.Key,
		"url":         fmt.Sprintf("%s/browse/%s", t.deps.Jira.BaseURL(), result.Key),
		"parent_epic": args.EpicKey,
		"type":        issueType,
		"status":      "created",
	}
	if args.StoryPoints > 0 {
		output["story_points"] = args.StoryPoints
	}
	data, _ := json.MarshalIndent(output, "", "  ")
	return okText(string(data)), nil, nil
}
