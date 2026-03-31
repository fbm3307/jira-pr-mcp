package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/codeready-toolchain/jira-pr-mcp/pkg/config"
	gh "github.com/codeready-toolchain/jira-pr-mcp/pkg/github"
	"github.com/codeready-toolchain/jira-pr-mcp/pkg/jira"
	"github.com/codeready-toolchain/jira-pr-mcp/pkg/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func Run(ctx context.Context, cfg *config.Config, logger *slog.Logger) error {
	if err := cfg.Validate(); err != nil {
		return err
	}

	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	jiraClient := jira.NewClient(cfg.JiraURL, cfg.JiraEmail, cfg.JiraToken, cfg.JiraProject, cfg.CacheTTL, logger)
	ghClient := gh.NewClient(cfg.GitHubToken, logger)

	logger.Info("validating credentials...")

	myself, err := jiraClient.ValidateCredentials(ctx)
	if err != nil {
		return fmt.Errorf("startup check failed: %w", err)
	}
	logger.Info("jira credentials valid", "user", myself.DisplayName)

	if err := ghClient.ValidateCredentials(ctx); err != nil {
		return fmt.Errorf("startup check failed: %w", err)
	}
	logger.Info("github credentials valid")

	deps := &tools.Deps{
		Jira:   jiraClient,
		GitHub: ghClient,
	}

	s := mcp.NewServer(
		&mcp.Implementation{
			Name:    "jira-pr-mcp",
			Version: "0.1.0",
		},
		&mcp.ServerOptions{
			Logger: logger,
		},
	)

	allTools := []tools.Tool{
		tools.NewGetGitHubPR(deps),
		tools.NewSearchJiraIssues(deps),
		tools.NewGetJiraIssue(deps),
		tools.NewListJiraEpics(deps),
		tools.NewListJiraFeatures(deps),
		tools.NewCreateJiraEpic(deps),
		tools.NewCreateJiraIssue(deps),
		tools.NewLinkJiraIssueToPR(deps),
		tools.NewAddJiraComment(deps),
		tools.NewListJiraBoard(deps),
		tools.NewListJiraIssueChildren(deps),
		tools.NewUpdateJiraIssue(deps),
		tools.NewTransitionJiraIssue(deps),
	}

	for _, tool := range allTools {
		tool.RegisterWith(s)
	}

	logger.Info("starting server", "transport", cfg.Transport, "tools", len(allTools), "project", cfg.JiraProject, "jira_url", cfg.JiraURL)

	switch cfg.Transport {
	case "stdio":
		return s.Run(ctx, &mcp.StdioTransport{})
	case "http":
		return runHTTP(ctx, s, cfg, len(allTools), logger)
	default:
		return fmt.Errorf("unknown transport: %s", cfg.Transport)
	}
}

func runHTTP(ctx context.Context, s *mcp.Server, cfg *config.Config, toolCount int, logger *slog.Logger) error {
	handler := mcp.NewStreamableHTTPHandler(func(_ *http.Request) *mcp.Server {
		return s
	}, nil)

	healthJSON := fmt.Sprintf(`{"status":"ok","project":%q,"jira_url":%q,"transport":%q,"tools_registered":%d,"version":"0.1.0"}`,
		cfg.JiraProject, cfg.JiraURL, cfg.Transport, toolCount)

	mux := http.NewServeMux()
	mux.Handle("/mcp", handler)
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, healthJSON)
	})

	srv := &http.Server{
		Addr:    cfg.Address,
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		logger.Info("shutting down HTTP server...")
		srv.Close()
	}()

	logger.Info("HTTP server listening", "address", cfg.Address)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("HTTP server error: %w", err)
	}
	return nil
}
