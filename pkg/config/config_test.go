package config

import (
	"testing"
)

func TestConfig_Validate_MissingJiraEmail(t *testing.T) {
	cfg := &Config{
		JiraEmail:   "",
		JiraToken:   "jira-token",
		GitHubToken: "gh-token",
		Transport:   "stdio",
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for missing jira email")
	}
	ce, ok := err.(*ConfigError)
	if !ok {
		t.Fatalf("expected ConfigError, got %T", err)
	}
	if ce.Field != "JIRA_EMAIL" {
		t.Errorf("field = %q, want JIRA_EMAIL", ce.Field)
	}
}

func TestConfig_Validate_MissingJiraToken(t *testing.T) {
	cfg := &Config{
		JiraEmail:   "user@example.com",
		JiraToken:   "",
		GitHubToken: "gh-token",
		Transport:   "stdio",
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for missing jira token")
	}
}

func TestConfig_Validate_MissingGitHubToken(t *testing.T) {
	cfg := &Config{
		JiraEmail:   "user@example.com",
		JiraToken:   "jira-token",
		GitHubToken: "",
		Transport:   "stdio",
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for missing github token")
	}
}

func TestConfig_Validate_InvalidTransport(t *testing.T) {
	cfg := &Config{
		JiraEmail:   "user@example.com",
		JiraToken:   "jira-token",
		GitHubToken: "gh-token",
		Transport:   "websocket",
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for invalid transport")
	}
}

func TestConfig_Validate_Valid(t *testing.T) {
	cfg := &Config{
		JiraEmail:   "user@example.com",
		JiraToken:   "jira-token",
		GitHubToken: "gh-token",
		Transport:   "stdio",
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestConfig_ApplyDefaults(t *testing.T) {
	cfg := &Config{}
	cfg.ApplyDefaults()

	if cfg.JiraURL != "https://redhat.atlassian.net" {
		t.Errorf("JiraURL = %q, want https://redhat.atlassian.net", cfg.JiraURL)
	}
	if cfg.JiraProject != "SANDBOX" {
		t.Errorf("JiraProject = %q, want SANDBOX", cfg.JiraProject)
	}
	if cfg.Transport != "stdio" {
		t.Errorf("Transport = %q, want stdio", cfg.Transport)
	}
	if cfg.Address != ":8080" {
		t.Errorf("Address = %q, want :8080", cfg.Address)
	}
}

func TestConfig_ApplyDefaults_FlagOverride(t *testing.T) {
	cfg := &Config{
		JiraURL:   "https://custom.jira.com",
		Transport: "http",
	}
	cfg.ApplyDefaults()

	if cfg.JiraURL != "https://custom.jira.com" {
		t.Errorf("JiraURL = %q, want custom value", cfg.JiraURL)
	}
	if cfg.Transport != "http" {
		t.Errorf("Transport = %q, want http", cfg.Transport)
	}
}
