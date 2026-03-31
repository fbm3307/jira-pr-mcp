package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/codeready-toolchain/jira-pr-mcp/pkg/jira"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type CreateJiraEpic struct {
	deps *Deps
}

func NewCreateJiraEpic(deps *Deps) Tool {
	return &CreateJiraEpic{deps: deps}
}

type createJiraEpicArgs struct {
	Project     string   `json:"project"`
	Summary     string   `json:"summary"`
	Description string   `json:"description"`
	Labels      []string `json:"labels"`
	FeatureKey  string   `json:"feature_key"`
}

func (t *CreateJiraEpic) RegisterWith(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name: "create_jira_epic",
		Description: `Create a new Epic linked to a Feature. feature_key is REQUIRED -- every Epic must belong to a Feature.
IMPORTANT: Write concise, human-style summaries and descriptions. Do NOT write verbose AI-generated prose. Before calling this tool, review existing Epics (via list_jira_epics) to match the tone and brevity of human-written ones.`,
	}, t.handle)
}

func (t *CreateJiraEpic) handle(ctx context.Context, _ *mcp.CallToolRequest, args createJiraEpicArgs) (*mcp.CallToolResult, any, error) {
	if args.FeatureKey == "" {
		return errText("feature_key is required -- every Epic must be linked to a Feature"), nil, nil
	}
	if args.Summary == "" {
		return errText("summary is required"), nil, nil
	}

	project := args.Project
	if project == "" {
		project = t.deps.Jira.Project()
	}

	if !strings.EqualFold(project, t.deps.Jira.Project()) {
		return errText("Project scope violation: this server is configured for project %s, cannot create in %s", t.deps.Jira.Project(), project), nil, nil
	}

	parent, err := t.deps.Jira.GetIssue(ctx, args.FeatureKey)
	if err != nil {
		return errText("Failed to verify Feature %s: %v", args.FeatureKey, err), nil, nil
	}
	if !strings.EqualFold(parent.Fields.IssueType.Name, "Feature") {
		return errText("%s is not a Feature (it's a %s) -- feature_key must point to a Feature", args.FeatureKey, parent.Fields.IssueType.Name), nil, nil
	}

	req := &jira.CreateIssueRequest{
		Fields: jira.CreateIssueFields{
			Project:     jira.ProjectRef{Key: project},
			IssueType:   jira.IssueTypeRef{Name: "Epic"},
			Summary:     args.Summary,
			Description: jira.NewADFDescription(args.Description),
			Labels:      args.Labels,
			Parent:      &jira.ParentKeyRef{Key: args.FeatureKey},
		},
	}

	result, err := t.deps.Jira.CreateIssue(ctx, req)
	if err != nil {
		return errText("Failed to create Epic: %v", err), nil, nil
	}

	output := map[string]string{
		"key":            result.Key,
		"url":            fmt.Sprintf("%s/browse/%s", t.deps.Jira.BaseURL(), result.Key),
		"parent_feature": args.FeatureKey,
		"status":         "created",
	}
	data, _ := json.MarshalIndent(output, "", "  ")
	return okText(string(data)), nil, nil
}
