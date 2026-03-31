package github

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// ParsePRURL extracts owner, repo, and PR number from a GitHub PR URL.
// Supports: https://github.com/{owner}/{repo}/pull/{number}
func ParsePRURL(prURL string) (owner, repo string, number int, err error) {
	u, err := url.Parse(prURL)
	if err != nil {
		return "", "", 0, fmt.Errorf("invalid URL: %w", err)
	}

	if u.Host != "github.com" {
		return "", "", 0, fmt.Errorf("not a GitHub URL (host: %s)", u.Host)
	}

	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 4 || parts[2] != "pull" {
		return "", "", 0, fmt.Errorf("invalid GitHub PR URL format, expected: https://github.com/{owner}/{repo}/pull/{number}")
	}

	owner = parts[0]
	repo = parts[1]
	number, err = strconv.Atoi(parts[3])
	if err != nil {
		return "", "", 0, fmt.Errorf("invalid PR number %q: %w", parts[3], err)
	}

	return owner, repo, number, nil
}
