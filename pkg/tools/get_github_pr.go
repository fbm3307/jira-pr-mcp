package tools

import (
	"context"
	"encoding/json"
	"fmt"

	gh "github.com/codeready-toolchain/jira-pr-mcp/pkg/github"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type GetGitHubPR struct {
	deps *Deps
}

func NewGetGitHubPR(deps *Deps) Tool {
	return &GetGitHubPR{deps: deps}
}

type getGitHubPRArgs struct {
	PRURL    string `json:"pr_url"`
	Owner    string `json:"owner"`
	Repo     string `json:"repo"`
	PRNumber int    `json:"pr_number"`
}

func (t *GetGitHubPR) RegisterWith(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_github_pr",
		Description: "Fetch GitHub PR metadata including title, description, branch, commits, and changed files. Provide either pr_url (full GitHub PR URL) or owner+repo+pr_number.",
	}, t.handle)
}

func (t *GetGitHubPR) handle(ctx context.Context, _ *mcp.CallToolRequest, args getGitHubPRArgs) (*mcp.CallToolResult, any, error) {
	owner := args.Owner
	repo := args.Repo
	number := args.PRNumber

	if args.PRURL != "" {
		var err error
		owner, repo, number, err = gh.ParsePRURL(args.PRURL)
		if err != nil {
			return errText("Invalid PR URL: %v", err), nil, nil
		}
	}

	if owner == "" || repo == "" || number == 0 {
		return errText("Provide either pr_url or all of owner, repo, and pr_number"), nil, nil
	}

	pr, err := t.deps.GitHub.GetPR(ctx, owner, repo, number)
	if err != nil {
		return errText("Failed to fetch PR: %v", err), nil, nil
	}

	data, _ := json.MarshalIndent(pr, "", "  ")
	return okText(string(data)), nil, nil
}

func errText(format string, a ...interface{}) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf(format, a...)}},
		IsError: true,
	}
}

func okText(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}
}
