package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/codeready-toolchain/jira-pr-mcp/pkg/config"
	"github.com/codeready-toolchain/jira-pr-mcp/pkg/server"
	"github.com/spf13/cobra"
)

func main() {
	cfg := &config.Config{}

	rootCmd := &cobra.Command{
		Use:   "jira-pr-mcp",
		Short: "MCP server for linking GitHub PRs to Jira issues",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg.ApplyDefaults()

			level := slog.LevelInfo
			if cfg.Debug {
				level = slog.LevelDebug
			}
			logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))

			return server.Run(cmd.Context(), cfg, logger)
		},
	}

	rootCmd.Flags().StringVar(&cfg.JiraURL, "jira-url", "", "Jira instance URL (env: JIRA_URL)")
	rootCmd.Flags().StringVar(&cfg.JiraEmail, "jira-email", "", "Jira account email for API auth (env: JIRA_EMAIL)")
	rootCmd.Flags().StringVar(&cfg.JiraToken, "jira-token", "", "Jira API token (env: JIRA_API_TOKEN)")
	rootCmd.Flags().StringVar(&cfg.JiraProject, "jira-project", "", "Default Jira project key (env: JIRA_PROJECT)")
	rootCmd.Flags().StringVar(&cfg.GitHubToken, "github-token", "", "GitHub API token (env: GITHUB_TOKEN)")
	rootCmd.Flags().StringVar(&cfg.Transport, "transport", "", "Transport: stdio or http (env: MCP_TRANSPORT)")
	rootCmd.Flags().StringVar(&cfg.Address, "address", "", "HTTP listen address (env: MCP_ADDRESS)")
	rootCmd.Flags().DurationVar(&cfg.CacheTTL, "cache-ttl", 0, "Cache TTL for Jira queries (e.g., 5m, 10m) (env: CACHE_TTL)")
	rootCmd.Flags().BoolVar(&cfg.Debug, "debug", false, "Enable debug logging")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
