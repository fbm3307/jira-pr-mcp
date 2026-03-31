package github

type PRDetails struct {
	Owner       string        `json:"owner"`
	Repo        string        `json:"repo"`
	Number      int           `json:"number"`
	Title       string        `json:"title"`
	Body        string        `json:"body"`
	State       string        `json:"state"`
	HeadBranch  string        `json:"head_branch"`
	BaseBranch  string        `json:"base_branch"`
	Labels      []string      `json:"labels"`
	URL         string        `json:"url"`
	Commits     []CommitInfo  `json:"commits"`
	ChangedFiles []FileChange `json:"changed_files"`
}

type CommitInfo struct {
	SHA     string `json:"sha"`
	Message string `json:"message"`
}

type FileChange struct {
	Filename  string `json:"filename"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
	Status    string `json:"status"`
}
