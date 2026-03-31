package config

import (
	"os"
	"time"
)

type Config struct {
	JiraURL     string
	JiraEmail   string
	JiraToken   string
	JiraProject string
	GitHubToken string
	Transport   string
	Address     string
	CacheTTL    time.Duration
	Debug       bool
}

func (c *Config) ApplyDefaults() {
	applyEnvDefault(&c.JiraURL, "JIRA_URL", "https://redhat.atlassian.net")
	applyEnvDefault(&c.JiraEmail, "JIRA_EMAIL", "")
	applyEnvDefault(&c.JiraToken, "JIRA_API_TOKEN", "")
	applyEnvDefault(&c.JiraProject, "JIRA_PROJECT", "SANDBOX")
	applyEnvDefault(&c.GitHubToken, "GITHUB_TOKEN", "")
	applyEnvDefault(&c.Transport, "MCP_TRANSPORT", "stdio")
	applyEnvDefault(&c.Address, "MCP_ADDRESS", ":8080")
	if c.CacheTTL == 0 {
		if v := os.Getenv("CACHE_TTL"); v != "" {
			if d, err := time.ParseDuration(v); err == nil {
				c.CacheTTL = d
			}
		}
		if c.CacheTTL == 0 {
			c.CacheTTL = 5 * time.Minute
		}
	}
}

func (c *Config) Validate() error {
	if c.JiraEmail == "" {
		return &ConfigError{Field: "JIRA_EMAIL", Message: "Jira email is required for Jira Cloud auth. Set JIRA_EMAIL env var or use --jira-email flag."}
	}
	if c.JiraToken == "" {
		return &ConfigError{Field: "JIRA_API_TOKEN", Message: "Jira API token is required. Create one at https://id.atlassian.com/manage-profile/security/api-tokens. Set JIRA_API_TOKEN env var or use --jira-token flag."}
	}
	if c.GitHubToken == "" {
		return &ConfigError{Field: "GITHUB_TOKEN", Message: "GitHub token is required. Set GITHUB_TOKEN env var or use --github-token flag."}
	}
	if c.Transport != "stdio" && c.Transport != "http" {
		return &ConfigError{Field: "MCP_TRANSPORT", Message: "Transport must be 'stdio' or 'http'."}
	}
	return nil
}

type ConfigError struct {
	Field   string
	Message string
}

func (e *ConfigError) Error() string {
	return e.Message
}

func applyEnvDefault(field *string, envKey, defaultVal string) {
	if *field != "" {
		return
	}
	if v := os.Getenv(envKey); v != "" {
		*field = v
		return
	}
	*field = defaultVal
}
