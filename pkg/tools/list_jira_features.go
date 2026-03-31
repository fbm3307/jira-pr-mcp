package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/codeready-toolchain/jira-pr-mcp/pkg/jira"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type ListJiraFeatures struct {
	deps *Deps
}

func NewListJiraFeatures(deps *Deps) Tool {
	return &ListJiraFeatures{deps: deps}
}

type listJiraFeaturesArgs struct {
	Project    string `json:"project"`
	Text       string `json:"text"`
	Status     string `json:"status"`
	StartAt    *int   `json:"start_at,omitempty"`
	MaxResults int    `json:"max_results"`
}

func (t *ListJiraFeatures) RegisterWith(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_jira_features",
		Description: "List or search Features (Initiatives) in the Jira project. Use to find a Feature to link a new Epic to. Features are read-only -- the agent cannot create Features. Results are cached for 5 minutes.",
	}, t.handle)
}

func (t *ListJiraFeatures) handle(ctx context.Context, _ *mcp.CallToolRequest, args listJiraFeaturesArgs) (*mcp.CallToolResult, any, error) {
	maxResults := args.MaxResults
	if maxResults <= 0 {
		maxResults = 30
	}

	startAt := 0
	if args.StartAt != nil {
		startAt = *args.StartAt
	}

	cacheKey := fmt.Sprintf("features:%s:%s:%s:%d:%d", args.Project, args.Text, args.Status, startAt, maxResults)
	if cached, ok := t.deps.Jira.GetCache().Get(cacheKey); ok {
		data, _ := json.MarshalIndent(cached, "", "  ")
		return okText(string(data)), nil, nil
	}

	result, err := t.deps.Jira.SearchByType(ctx, "Feature", args.Text, args.Status, startAt, maxResults)
	if err != nil {
		return errText("Failed to list features: %v", err), nil, nil
	}

	output := formatFeatureResult(result, t.deps.Jira.BaseURL())
	t.deps.Jira.GetCache().Set(cacheKey, output)

	data, _ := json.MarshalIndent(output, "", "  ")
	return okText(string(data)), nil, nil
}

type featureOutput struct {
	Total      int            `json:"total"`
	StartAt    int            `json:"start_at"`
	MaxResults int            `json:"max_results"`
	Features   []featureShort `json:"features"`
}

type featureShort struct {
	Key         string `json:"key"`
	Summary     string `json:"summary"`
	Description string `json:"description,omitempty"`
	Status      string `json:"status"`
	URL         string `json:"url"`
}

func formatFeatureResult(sr *jira.SearchResult, baseURL string) *featureOutput {
	out := &featureOutput{Total: sr.Total, StartAt: sr.StartAt, MaxResults: sr.MaxResults}
	for _, issue := range sr.Issues {
		item := featureShort{
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
		out.Features = append(out.Features, item)
	}
	return out
}
