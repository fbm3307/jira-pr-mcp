package tools

import (
	"github.com/codeready-toolchain/jira-pr-mcp/pkg/github"
	"github.com/codeready-toolchain/jira-pr-mcp/pkg/jira"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Tool is the interface every MCP tool in this server must implement.
type Tool interface {
	RegisterWith(s *mcp.Server)
}

// Deps holds shared dependencies injected into each tool.
type Deps struct {
	Jira   *jira.Client
	GitHub *github.Client
}
