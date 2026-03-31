# jira-pr-mcp

MCP server for managing Jira issues in the context of GitHub PRs. Finds existing Jira issues related to a PR, creates new ones when needed (respecting the Feature → Epic → Task hierarchy), and links PRs back to Jira — with built-in duplicate prevention.

## Tools

| Tool | Description |
|------|-------------|
| `get_github_pr` | Fetch PR metadata (title, commits, changed files) |
| `search_jira_issues` | Search Tasks/Stories/Bugs by text or JQL |
| `get_jira_issue` | Get full issue details including remote links |
| `list_jira_epics` | Search Epics in the project |
| `list_jira_features` | Search Features (read-only, cannot create) |
| `list_jira_issue_children` | List all children under a parent (Epic → Tasks) |
| `create_jira_epic` | Create an Epic linked to a Feature |
| `create_jira_issue` | Create a Task/Story linked to an Epic |
| `update_jira_issue` | Update fields (summary, description, labels, story points, assignee) |
| `transition_jira_issue` | Move issue through Kanban states (To Do → In Progress → In Review → Done) |
| `link_jira_issue_to_pr` | Add PR as a remote link on a Jira issue |
| `add_jira_comment` | Post a comment on a Jira issue |
| `list_jira_board` | View sprint/board issues |

## Key Behaviors

- **Hierarchy enforcement**: Task → Epic → Feature. Every level must be linked.
- **Duplicate prevention**: Searches by keywords, repo name, branch name, and scans Epic children before creating.
- **Custom field support**: Story points (`customfield_12310243`) and Epic link (`customfield_12311140`) for `redhat.atlassian.net`.
- **Retry with backoff**: 3 attempts on 429/5xx with exponential backoff and `Retry-After` support.
- **Caching**: Features and Epics lists cached with configurable TTL (default 5min).

## Configuration

| Env Variable | Flag | Default | Description |
|---|---|---|---|
| `JIRA_URL` | `--jira-url` | `https://redhat.atlassian.net` | Jira Cloud instance URL |
| `JIRA_EMAIL` | `--jira-email` | (required) | Atlassian account email |
| `JIRA_API_TOKEN` | `--jira-token` | (required) | Atlassian API token ([create here](https://id.atlassian.com/manage-profile/security/api-tokens)) |
| `JIRA_PROJECT` | `--jira-project` | `SANDBOX` | Default Jira project key |
| `GITHUB_TOKEN` | `--github-token` | (required) | GitHub PAT |
| `MCP_TRANSPORT` | `--transport` | `stdio` | Transport: `stdio` or `http` |
| `MCP_ADDRESS` | `--address` | `:8080` | HTTP listen address |
| `CACHE_TTL` | `--cache-ttl` | `5m` | Cache TTL for Jira queries |

## Authentication

Jira Cloud Basic auth (`email:token`):

1. Create an API token at https://id.atlassian.com/manage-profile/security/api-tokens
2. Set `JIRA_EMAIL` and `JIRA_API_TOKEN`

## Build & Run

```bash
make build
./bin/jira-pr-mcp
```

## Cursor Integration

Add to `.cursor/mcp.json`:

```json
{
  "mcpServers": {
    "jira-pr": {
      "command": "/path/to/jira-pr-mcp",
      "args": [],
      "env": {
        "JIRA_EMAIL": "you@redhat.com",
        "JIRA_API_TOKEN": "your-api-token",
        "GITHUB_TOKEN": "your-github-token",
        "JIRA_PROJECT": "SANDBOX"
      }
    }
  }
}
```

Or set the env vars in your shell profile and omit the `env` block.
