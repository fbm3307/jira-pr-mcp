package tools

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	gh "github.com/codeready-toolchain/jira-pr-mcp/pkg/github"
	"github.com/codeready-toolchain/jira-pr-mcp/pkg/jira"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func setupTestServer(t *testing.T, tools []Tool) *mcp.Server {
	t.Helper()
	s := mcp.NewServer(
		&mcp.Implementation{Name: "test", Version: "0.0.1"},
		nil,
	)
	for _, tool := range tools {
		tool.RegisterWith(s)
	}
	return s
}

func testDeps(t *testing.T, jiraHandler http.HandlerFunc) *Deps {
	t.Helper()
	srv := httptest.NewServer(jiraHandler)
	t.Cleanup(srv.Close)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	return &Deps{
		Jira:   jira.NewClient(srv.URL, "test@example.com", "test-token", "SANDBOX", 0, logger),
		GitHub: gh.NewClient("test-github-token", logger),
	}
}

func TestCreateJiraIssue_MissingEpicKey(t *testing.T) {
	deps := testDeps(t, func(w http.ResponseWriter, _ *http.Request) {})
	tool := NewCreateJiraIssue(deps).(*CreateJiraIssue)

	result, _, err := tool.handle(context.Background(), nil, createJiraIssueArgs{
		Summary: "Test issue",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result for missing epic_key")
	}
	text := result.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(text, "epic_key is required") {
		t.Errorf("expected error about epic_key, got: %s", text)
	}
}

func TestCreateJiraIssue_InvalidStoryPoints(t *testing.T) {
	deps := testDeps(t, func(w http.ResponseWriter, _ *http.Request) {})
	tool := NewCreateJiraIssue(deps).(*CreateJiraIssue)

	result, _, err := tool.handle(context.Background(), nil, createJiraIssueArgs{
		Summary:     "Test issue",
		EpicKey:     "SANDBOX-100",
		StoryPoints: 4,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result for invalid story points")
	}
	text := result.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(text, "Fibonacci") {
		t.Errorf("expected Fibonacci error, got: %s", text)
	}
}

func TestCreateJiraIssue_ProjectScopeViolation(t *testing.T) {
	deps := testDeps(t, func(w http.ResponseWriter, _ *http.Request) {})
	tool := NewCreateJiraIssue(deps).(*CreateJiraIssue)

	result, _, err := tool.handle(context.Background(), nil, createJiraIssueArgs{
		Project: "OTHER",
		Summary: "Test issue",
		EpicKey: "OTHER-100",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result for project scope violation")
	}
	text := result.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(text, "Project scope violation") {
		t.Errorf("expected scope violation error, got: %s", text)
	}
}

func TestCreateJiraEpic_MissingFeatureKey(t *testing.T) {
	deps := testDeps(t, func(w http.ResponseWriter, _ *http.Request) {})
	tool := NewCreateJiraEpic(deps).(*CreateJiraEpic)

	result, _, err := tool.handle(context.Background(), nil, createJiraEpicArgs{
		Summary: "Test epic",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result for missing feature_key")
	}
	text := result.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(text, "feature_key is required") {
		t.Errorf("expected error about feature_key, got: %s", text)
	}
}

func TestCreateJiraEpic_WrongParentType(t *testing.T) {
	deps := testDeps(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/rest/api/3/issue/SANDBOX-100") {
			resp := jira.Issue{
				Key: "SANDBOX-100",
				Fields: jira.IssueFields{
					Summary:   "An Epic",
					IssueType: jira.IssueType{Name: "Epic"},
				},
			}
			json.NewEncoder(w).Encode(resp)
		}
	})
	tool := NewCreateJiraEpic(deps).(*CreateJiraEpic)

	result, _, err := tool.handle(context.Background(), nil, createJiraEpicArgs{
		Summary:    "New epic",
		FeatureKey: "SANDBOX-100",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error for wrong parent type")
	}
	text := result.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(text, "not a Feature") {
		t.Errorf("expected wrong-type error, got: %s", text)
	}
}

func TestCreateJiraIssue_DuplicateDetection(t *testing.T) {
	deps := testDeps(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/api/3/issue/SANDBOX-100" {
			resp := jira.Issue{
				Key: "SANDBOX-100",
				Fields: jira.IssueFields{
					Summary:   "Parent Epic",
					IssueType: jira.IssueType{Name: "Epic"},
				},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}
		if r.URL.Path == "/rest/api/3/search/jql" {
			resp := jira.SearchResult{
				Total: 1,
				Issues: []jira.Issue{
					{
						Key: "SANDBOX-789",
						Fields: jira.IssueFields{
							Summary:   "Fix login bug",
							IssueType: jira.IssueType{Name: "Task"},
						},
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}
	})
	tool := NewCreateJiraIssue(deps).(*CreateJiraIssue)

	result, _, err := tool.handle(context.Background(), nil, createJiraIssueArgs{
		Summary: "Fix login bug",
		EpicKey: "SANDBOX-100",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Error("duplicate detection should not be an error result")
	}
	text := result.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(text, "duplicate_found") {
		t.Errorf("expected duplicate_found status, got: %s", text)
	}
	if !strings.Contains(text, "SANDBOX-789") {
		t.Errorf("expected existing issue key in output, got: %s", text)
	}
}

func TestLinkJiraIssueToPR_ProjectScope(t *testing.T) {
	deps := testDeps(t, func(w http.ResponseWriter, _ *http.Request) {})
	tool := NewLinkJiraIssueToPR(deps).(*LinkJiraIssueToPR)

	result, _, err := tool.handle(context.Background(), nil, linkJiraIssueToPRArgs{
		IssueKey: "OTHER-123",
		PRURL:    "https://github.com/org/repo/pull/1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error for cross-project link")
	}
}

func TestAddJiraComment_MissingFields(t *testing.T) {
	deps := testDeps(t, func(w http.ResponseWriter, _ *http.Request) {})
	tool := NewAddJiraComment(deps).(*AddJiraComment)

	result, _, err := tool.handle(context.Background(), nil, addJiraCommentArgs{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error for missing fields")
	}
}
