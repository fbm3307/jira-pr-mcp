package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/codeready-toolchain/jira-pr-mcp/pkg/jira"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type LinkJiraIssueToPR struct {
	deps *Deps
}

func NewLinkJiraIssueToPR(deps *Deps) Tool {
	return &LinkJiraIssueToPR{deps: deps}
}

type linkJiraIssueToPRArgs struct {
	IssueKey string `json:"issue_key"`
	PRURL    string `json:"pr_url"`
	PRTitle  string `json:"pr_title"`
}

func (t *LinkJiraIssueToPR) RegisterWith(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "link_jira_issue_to_pr",
		Description: "Add a GitHub PR as a remote link on a Jira issue. This closes the loop between Jira and GitHub. Always call this after finding or creating a Jira issue for a PR.",
	}, t.handle)
}

func (t *LinkJiraIssueToPR) handle(ctx context.Context, _ *mcp.CallToolRequest, args linkJiraIssueToPRArgs) (*mcp.CallToolResult, any, error) {
	if args.IssueKey == "" {
		return errText("issue_key is required"), nil, nil
	}
	if args.PRURL == "" {
		return errText("pr_url is required"), nil, nil
	}

	issueProject := strings.Split(args.IssueKey, "-")[0]
	if !strings.EqualFold(issueProject, t.deps.Jira.Project()) {
		return errText("Project scope violation: this server is configured for project %s, cannot modify %s", t.deps.Jira.Project(), args.IssueKey), nil, nil
	}

	title := args.PRTitle
	if title == "" {
		title = args.PRURL
	}

	req := &jira.RemoteLinkRequest{
		GlobalID: fmt.Sprintf("github-pr-%s", args.PRURL),
		Object: jira.RemoteLinkObject{
			URL:   args.PRURL,
			Title: title,
		},
	}

	result, err := t.deps.Jira.AddRemoteLink(ctx, args.IssueKey, req)
	if err != nil {
		return errText("Failed to add remote link: %v", err), nil, nil
	}

	output := map[string]interface{}{
		"status":    "linked",
		"link_id":   result.ID,
		"issue_key": args.IssueKey,
		"pr_url":    args.PRURL,
	}
	data, _ := json.MarshalIndent(output, "", "  ")
	return okText(string(data)), nil, nil
}
