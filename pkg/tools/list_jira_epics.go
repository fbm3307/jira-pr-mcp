package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/codeready-toolchain/jira-pr-mcp/pkg/jira"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type ListJiraEpics struct {
	deps *Deps
}

func NewListJiraEpics(deps *Deps) Tool {
	return &ListJiraEpics{deps: deps}
}

type listJiraEpicsArgs struct {
	Text       string `json:"text"`
	Status     string `json:"status"`
	StartAt    *int   `json:"start_at,omitempty"`
	MaxResults int    `json:"max_results"`
}

func (t *ListJiraEpics) RegisterWith(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_jira_epics",
		Description: "List or search Epics in the Jira project. Use to find a relevant Epic to link a new Task to. Returns each Epic's key, summary, status, parent Feature, and URL. Results are cached for 5 minutes.",
	}, t.handle)
}

func (t *ListJiraEpics) handle(ctx context.Context, _ *mcp.CallToolRequest, args listJiraEpicsArgs) (*mcp.CallToolResult, any, error) {
	maxResults := args.MaxResults
	if maxResults <= 0 {
		maxResults = 30
	}

	startAt := 0
	if args.StartAt != nil {
		startAt = *args.StartAt
	}

	cacheKey := fmt.Sprintf("epics:%s:%s:%d:%d", args.Text, args.Status, startAt, maxResults)
	if cached, ok := t.deps.Jira.GetCache().Get(cacheKey); ok {
		data, _ := json.MarshalIndent(cached, "", "  ")
		return okText(string(data)), nil, nil
	}

	result, err := t.deps.Jira.SearchByType(ctx, "Epic", args.Text, args.Status, startAt, maxResults)
	if err != nil {
		return errText("Failed to list epics: %v", err), nil, nil
	}

	output := formatEpicResult(result, t.deps.Jira.BaseURL())
	t.deps.Jira.GetCache().Set(cacheKey, output)

	data, _ := json.MarshalIndent(output, "", "  ")
	return okText(string(data)), nil, nil
}

type epicOutput struct {
	Total      int         `json:"total"`
	StartAt    int         `json:"start_at"`
	MaxResults int         `json:"max_results"`
	Epics      []epicShort `json:"epics"`
}

type epicShort struct {
	Key           string `json:"key"`
	Summary       string `json:"summary"`
	Description   string `json:"description,omitempty"`
	Status        string `json:"status"`
	ParentFeature string `json:"parent_feature,omitempty"`
	URL           string `json:"url"`
}

func formatEpicResult(sr *jira.SearchResult, baseURL string) *epicOutput {
	out := &epicOutput{Total: sr.Total, StartAt: sr.StartAt, MaxResults: sr.MaxResults}
	for _, issue := range sr.Issues {
		item := epicShort{
			Key:     issue.Key,
			Summary: issue.Fields.Summary,
			Status:  issue.Fields.Status.Name,
			URL:     fmt.Sprintf("%s/browse/%s", baseURL, issue.Key),
		}
		if desc := issue.Fields.Description; len(desc) > 200 {
			item.Description = desc[:200] + "..."
		} else {
			item.Description = desc
		}
		if issue.Fields.Parent != nil {
			item.ParentFeature = fmt.Sprintf("%s (%s)", issue.Fields.Parent.Key, issue.Fields.Parent.Fields.Summary)
		}
		out.Epics = append(out.Epics, item)
	}
	return out
}
