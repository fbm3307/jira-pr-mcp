package github

import (
	"testing"
)

func TestParsePRURL(t *testing.T) {
	tests := []struct {
		name       string
		url        string
		wantOwner  string
		wantRepo   string
		wantNumber int
		wantErr    bool
	}{
		{
			name:       "valid PR URL",
			url:        "https://github.com/codeready-toolchain/host-operator/pull/123",
			wantOwner:  "codeready-toolchain",
			wantRepo:   "host-operator",
			wantNumber: 123,
		},
		{
			name:       "valid PR URL with trailing slash",
			url:        "https://github.com/owner/repo/pull/456/",
			wantOwner:  "owner",
			wantRepo:   "repo",
			wantNumber: 456,
		},
		{
			name:       "valid PR URL with extra path segments",
			url:        "https://github.com/org/project/pull/789/files",
			wantOwner:  "org",
			wantRepo:   "project",
			wantNumber: 789,
		},
		{
			name:    "not a github URL",
			url:     "https://gitlab.com/owner/repo/pull/123",
			wantErr: true,
		},
		{
			name:    "missing pull segment",
			url:     "https://github.com/owner/repo/issues/123",
			wantErr: true,
		},
		{
			name:    "invalid PR number",
			url:     "https://github.com/owner/repo/pull/abc",
			wantErr: true,
		},
		{
			name:    "too few path segments",
			url:     "https://github.com/owner",
			wantErr: true,
		},
		{
			name:    "empty URL",
			url:     "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, number, err := ParsePRURL(tt.url)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParsePRURL(%q) expected error, got nil", tt.url)
				}
				return
			}
			if err != nil {
				t.Errorf("ParsePRURL(%q) unexpected error: %v", tt.url, err)
				return
			}
			if owner != tt.wantOwner {
				t.Errorf("owner = %q, want %q", owner, tt.wantOwner)
			}
			if repo != tt.wantRepo {
				t.Errorf("repo = %q, want %q", repo, tt.wantRepo)
			}
			if number != tt.wantNumber {
				t.Errorf("number = %d, want %d", number, tt.wantNumber)
			}
		})
	}
}
