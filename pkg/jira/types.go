package jira

import (
	"encoding/json"
	"strings"
	"time"
)

type Issue struct {
	Key    string      `json:"key"`
	Self   string      `json:"self"`
	Fields IssueFields `json:"fields"`
}

type IssueFields struct {
	Summary     string          `json:"summary"`
	RawDesc     json.RawMessage `json:"description"`
	Description string          `json:"-"`
	Status      Status          `json:"status"`
	IssueType   IssueType       `json:"issuetype"`
	Assignee    *User           `json:"assignee"`
	Reporter    *User           `json:"reporter"`
	Labels      []string        `json:"labels"`
	Created     string          `json:"created"`
	Updated     string          `json:"updated"`
	Comment     *Comments       `json:"comment,omitempty"`
	IssueLinks  []Link          `json:"issuelinks,omitempty"`
	Parent      *ParentRef      `json:"parent,omitempty"`

	StoryPoints float64          `json:"-"`
	EpicLink    string           `json:"-"`
	RawFields   json.RawMessage  `json:"-"`
}

// ParseDescription extracts plain text from either a string (v2) or ADF object (v3).
func (f *IssueFields) ParseDescription() {
	if len(f.RawDesc) == 0 || string(f.RawDesc) == "null" {
		f.Description = ""
		return
	}
	// Try plain string first (v2 compat)
	var s string
	if err := json.Unmarshal(f.RawDesc, &s); err == nil {
		f.Description = s
		return
	}
	// ADF object (v3) — extract text from content nodes
	var doc adfDoc
	if err := json.Unmarshal(f.RawDesc, &doc); err == nil {
		f.Description = doc.PlainText()
		return
	}
	f.Description = string(f.RawDesc)
}

type adfDoc struct {
	Content []adfNode `json:"content"`
}

type adfNode struct {
	Type    string    `json:"type"`
	Text    string    `json:"text,omitempty"`
	Content []adfNode `json:"content,omitempty"`
}

func (d *adfDoc) PlainText() string {
	var b strings.Builder
	for _, node := range d.Content {
		node.collectText(&b)
		b.WriteString("\n")
	}
	return strings.TrimSpace(b.String())
}

func (n *adfNode) collectText(b *strings.Builder) {
	if n.Text != "" {
		b.WriteString(n.Text)
	}
	for _, child := range n.Content {
		child.collectText(b)
	}
}

type Status struct {
	Name string `json:"name"`
}

type IssueType struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

type User struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
}

type Comments struct {
	Total    int       `json:"total"`
	Comments []Comment `json:"comments"`
}

type Comment struct {
	ID      string `json:"id"`
	Body    string `json:"body"`
	Author  *User  `json:"author"`
	Created string `json:"created"`
}

type Link struct {
	Type       LinkType `json:"type"`
	InwardIss  *Issue   `json:"inwardIssue,omitempty"`
	OutwardIss *Issue   `json:"outwardIssue,omitempty"`
}

type LinkType struct {
	Name    string `json:"name"`
	Inward  string `json:"inward"`
	Outward string `json:"outward"`
}

type ParentRef struct {
	Key    string      `json:"key"`
	Fields ParentField `json:"fields"`
}

type ParentField struct {
	Summary   string    `json:"summary"`
	Status    Status    `json:"status"`
	IssueType IssueType `json:"issuetype"`
}

type SearchResult struct {
	StartAt    int     `json:"startAt"`
	MaxResults int     `json:"maxResults"`
	Total      int     `json:"total"`
	Issues     []Issue `json:"issues"`
}

type Sprint struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	State string `json:"state"`
}

type SprintList struct {
	Values []Sprint `json:"values"`
}

type SprintIssues struct {
	Issues []Issue `json:"issues"`
	Total  int     `json:"total"`
}

type RemoteLink struct {
	ID     int              `json:"id"`
	Self   string           `json:"self"`
	Object RemoteLinkObject `json:"object"`
}

type RemoteLinkRequest struct {
	GlobalID string           `json:"globalId"`
	Object   RemoteLinkObject `json:"object"`
}

type RemoteLinkObject struct {
	URL   string `json:"url"`
	Title string `json:"title"`
}

type RemoteLinkResponse struct {
	ID   int    `json:"id"`
	Self string `json:"self"`
}

type CommentRequest struct {
	Body *ADFDocument `json:"body"`
}

type CommentResponse struct {
	ID      string `json:"id"`
	Created string `json:"created"`
}

type CreateIssueRequest struct {
	Fields CreateIssueFields `json:"fields"`
}

type CreateIssueFields struct {
	Project     ProjectRef     `json:"project"`
	IssueType   IssueTypeRef   `json:"issuetype"`
	Summary     string         `json:"summary"`
	Description *ADFDocument   `json:"description,omitempty"`
	Labels      []string       `json:"labels,omitempty"`
	Parent      *ParentKeyRef  `json:"parent,omitempty"`
	Security    *SecurityLevel `json:"security,omitempty"`
	StoryPoints *float64       `json:"customfield_12310243,omitempty"`
}

// ADFDocument is an Atlassian Document Format wrapper for Jira Cloud v3 API.
type ADFDocument struct {
	Type    string        `json:"type"`
	Version int           `json:"version"`
	Content []ADFContent  `json:"content"`
}

type ADFContent struct {
	Type    string       `json:"type"`
	Content []ADFText    `json:"content,omitempty"`
}

type ADFText struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// NewADFDescription creates an ADF document from a plain text string.
func NewADFDescription(text string) *ADFDocument {
	if text == "" {
		return nil
	}
	return &ADFDocument{
		Type:    "doc",
		Version: 1,
		Content: []ADFContent{
			{
				Type: "paragraph",
				Content: []ADFText{
					{Type: "text", Text: text},
				},
			},
		},
	}
}

type ProjectRef struct {
	ID  string `json:"id,omitempty"`
	Key string `json:"key,omitempty"`
}

type IssueTypeRef struct {
	Name string `json:"name"`
}

type ParentKeyRef struct {
	Key string `json:"key"`
}

type SecurityLevel struct {
	ID string `json:"id"`
}

type CreateIssueResponse struct {
	ID   string `json:"id"`
	Key  string `json:"key"`
	Self string `json:"self"`
}

type MySelfResponse struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
}

type UserSearchResult struct {
	AccountID   string `json:"accountId"`
	DisplayName string `json:"displayName"`
	EmailAddr   string `json:"emailAddress,omitempty"`
	Active      bool   `json:"active"`
}

type Transition struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	To   Status `json:"to"`
}

type TransitionsResponse struct {
	Transitions []Transition `json:"transitions"`
}

// CacheEntry holds a cached value with expiration.
type CacheEntry struct {
	Value     interface{}
	ExpiresAt time.Time
}
