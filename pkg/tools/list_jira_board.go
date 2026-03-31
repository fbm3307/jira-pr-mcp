package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type ListJiraBoard struct {
	deps *Deps
}

func NewListJiraBoard(deps *Deps) Tool {
	return &ListJiraBoard{deps: deps}
}

type listJiraBoardArgs struct {
	BoardID    int    `json:"board_id"`
	Sprint     string `json:"sprint"`
	MaxResults int    `json:"max_results"`
}

func (t *ListJiraBoard) RegisterWith(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_jira_board",
		Description: "List issues on a Jira board's sprint. Specify board_id and sprint state (active/future/closed). Use to browse current work in a sprint for context.",
	}, t.handle)
}

func (t *ListJiraBoard) handle(ctx context.Context, _ *mcp.CallToolRequest, args listJiraBoardArgs) (*mcp.CallToolResult, any, error) {
	if args.BoardID == 0 {
		return errText("board_id is required"), nil, nil
	}

	sprintState := args.Sprint
	if sprintState == "" {
		sprintState = "active"
	}

	maxResults := args.MaxResults
	if maxResults <= 0 {
		maxResults = 50
	}

	sprints, err := t.deps.Jira.ListBoardSprints(ctx, args.BoardID, sprintState)
	if err != nil {
		return errText("Failed to list sprints: %v", err), nil, nil
	}

	if len(sprints.Values) == 0 {
		return okText(fmt.Sprintf("No %s sprints found for board %d", sprintState, args.BoardID)), nil, nil
	}

	sprint := sprints.Values[0]
	issues, err := t.deps.Jira.GetSprintIssues(ctx, sprint.ID, maxResults)
	if err != nil {
		return errText("Failed to get sprint issues: %v", err), nil, nil
	}

	type boardIssue struct {
		Key      string `json:"key"`
		Summary  string `json:"summary"`
		Status   string `json:"status"`
		Type     string `json:"type"`
		Assignee string `json:"assignee,omitempty"`
		Parent   string `json:"parent,omitempty"`
	}

	output := struct {
		SprintName string       `json:"sprint_name"`
		SprintID   int          `json:"sprint_id"`
		Total      int          `json:"total"`
		Issues     []boardIssue `json:"issues"`
	}{
		SprintName: sprint.Name,
		SprintID:   sprint.ID,
		Total:      issues.Total,
	}

	for _, issue := range issues.Issues {
		item := boardIssue{
			Key:     issue.Key,
			Summary: issue.Fields.Summary,
			Status:  issue.Fields.Status.Name,
			Type:    issue.Fields.IssueType.Name,
		}
		if issue.Fields.Assignee != nil {
			item.Assignee = issue.Fields.Assignee.DisplayName
		}
		if issue.Fields.Parent != nil {
			item.Parent = issue.Fields.Parent.Key
		}
		output.Issues = append(output.Issues, item)
	}

	data, _ := json.MarshalIndent(output, "", "  ")
	return okText(string(data)), nil, nil
}
