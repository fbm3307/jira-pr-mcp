package tools

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type AddJiraComment struct {
	deps *Deps
}

func NewAddJiraComment(deps *Deps) Tool {
	return &AddJiraComment{deps: deps}
}

type addJiraCommentArgs struct {
	IssueKey string `json:"issue_key"`
	Body     string `json:"body"`
}

func (t *AddJiraComment) RegisterWith(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "add_jira_comment",
		Description: "Post a comment on a Jira issue. Useful for adding context like 'Linked from PR #123 -- adds feature X'.",
	}, t.handle)
}

func (t *AddJiraComment) handle(ctx context.Context, _ *mcp.CallToolRequest, args addJiraCommentArgs) (*mcp.CallToolResult, any, error) {
	if args.IssueKey == "" {
		return errText("issue_key is required"), nil, nil
	}
	if args.Body == "" {
		return errText("body is required"), nil, nil
	}

	issueProject := strings.Split(args.IssueKey, "-")[0]
	if !strings.EqualFold(issueProject, t.deps.Jira.Project()) {
		return errText("Project scope violation: this server is configured for project %s, cannot modify %s", t.deps.Jira.Project(), args.IssueKey), nil, nil
	}

	result, err := t.deps.Jira.AddComment(ctx, args.IssueKey, args.Body)
	if err != nil {
		return errText("Failed to add comment: %v", err), nil, nil
	}

	output := map[string]string{
		"status":     "commented",
		"comment_id": result.ID,
		"created":    result.Created,
		"issue_key":  args.IssueKey,
	}
	data, _ := json.MarshalIndent(output, "", "  ")
	return okText(string(data)), nil, nil
}
