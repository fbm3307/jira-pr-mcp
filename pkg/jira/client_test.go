package jira

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func testClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	return NewClient(srv.URL, "test@example.com", "test-token", "SANDBOX", 0, logger)
}

func TestValidateCredentials(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/3/myself" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		auth := r.Header.Get("Authorization")
		if auth == "" || !strings.HasPrefix(auth, "Basic ") {
			t.Error("missing or wrong authorization header, expected Basic auth")
		}
		json.NewEncoder(w).Encode(MySelfResponse{Name: "testuser", DisplayName: "Test User"})
	})

	result, err := client.ValidateCredentials(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.DisplayName != "Test User" {
		t.Errorf("got %q, want %q", result.DisplayName, "Test User")
	}
}

func TestValidateCredentials_Unauthorized(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"errorMessages":["not authorized"]}`))
	})

	_, err := client.ValidateCredentials(context.Background())
	if err == nil {
		t.Fatal("expected error for unauthorized, got nil")
	}
}

func TestGetIssue(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/3/issue/SANDBOX-123" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		resp := Issue{
			Key: "SANDBOX-123",
			Fields: IssueFields{
				Summary:   "Fix login bug",
				Status:    Status{Name: "In Progress"},
				IssueType: IssueType{Name: "Task"},
				Assignee:  &User{DisplayName: "John"},
				Labels:    []string{"sre", "support"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})

	issue, err := client.GetIssue(context.Background(), "SANDBOX-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if issue.Key != "SANDBOX-123" {
		t.Errorf("key = %q, want SANDBOX-123", issue.Key)
	}
	if issue.Fields.Summary != "Fix login bug" {
		t.Errorf("summary = %q, want 'Fix login bug'", issue.Fields.Summary)
	}
}

func TestSearchIssues(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/3/search/jql" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		resp := SearchResult{
			Total:      1,
			StartAt:    0,
			MaxResults: 20,
			Issues: []Issue{
				{
					Key: "SANDBOX-456",
					Fields: IssueFields{
						Summary:   "Update observability",
						Status:    Status{Name: "Open"},
						IssueType: IssueType{Name: "Story"},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})

	result, err := client.SearchIssues(context.Background(), "project = SANDBOX", 0, 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 1 {
		t.Errorf("total = %d, want 1", result.Total)
	}
	if result.Issues[0].Key != "SANDBOX-456" {
		t.Errorf("issue key = %q, want SANDBOX-456", result.Issues[0].Key)
	}
}

func TestFindDuplicate_Found(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		resp := SearchResult{
			Total: 1,
			Issues: []Issue{
				{
					Key: "SANDBOX-789",
					Fields: IssueFields{
						Summary:   "Fix login bug",
						IssueType: IssueType{Name: "Task"},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})

	dup, err := client.FindDuplicate(context.Background(), "Fix login bug")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dup == nil {
		t.Fatal("expected duplicate, got nil")
	}
	if dup.Key != "SANDBOX-789" {
		t.Errorf("duplicate key = %q, want SANDBOX-789", dup.Key)
	}
}

func TestFindDuplicate_NotFound(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, _ *http.Request) {
		resp := SearchResult{Total: 0, Issues: []Issue{}}
		json.NewEncoder(w).Encode(resp)
	})

	dup, err := client.FindDuplicate(context.Background(), "Something unique")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dup != nil {
		t.Errorf("expected nil, got %v", dup)
	}
}

func TestAddRemoteLink(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/3/issue/SANDBOX-123/remotelink" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		resp := RemoteLinkResponse{ID: 42}
		json.NewEncoder(w).Encode(resp)
	})

	req := &RemoteLinkRequest{
		GlobalID: "github-pr-test",
		Object:   RemoteLinkObject{URL: "https://github.com/org/repo/pull/1", Title: "PR #1"},
	}
	result, err := client.AddRemoteLink(context.Background(), "SANDBOX-123", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != 42 {
		t.Errorf("link ID = %d, want 42", result.ID)
	}
}

func TestAddComment(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/3/issue/SANDBOX-123/comment" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		resp := CommentResponse{ID: "10001", Created: "2026-03-24T10:00:00.000+0000"}
		json.NewEncoder(w).Encode(resp)
	})

	result, err := client.AddComment(context.Background(), "SANDBOX-123", "Test comment")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "10001" {
		t.Errorf("comment ID = %q, want 10001", result.ID)
	}
}

func TestCreateIssue(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/3/issue" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		resp := CreateIssueResponse{ID: "12345", Key: "SANDBOX-999"}
		json.NewEncoder(w).Encode(resp)
	})

	req := &CreateIssueRequest{
		Fields: CreateIssueFields{
			Project:   ProjectRef{Key: "SANDBOX"},
			IssueType: IssueTypeRef{Name: "Task"},
			Summary:   "New task",
		},
	}
	result, err := client.CreateIssue(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Key != "SANDBOX-999" {
		t.Errorf("key = %q, want SANDBOX-999", result.Key)
	}
}
