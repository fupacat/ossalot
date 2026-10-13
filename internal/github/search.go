package github

import "context"

// TRACEABILITY_MAPPING: REQ-022
type IssueCandidate struct {
	TargetRepoFullName string   `json:"target_repo_full_name"`
	IssueNumber        int      `json:"issue_number"`
	Title              string   `json:"title"`
	Body               string   `json:"body"`
	HTMLURL            string   `json:"html_url"`
	Author             string   `json:"author"`
	Labels             []string `json:"labels"`
}

// TRACEABILITY_MAPPING: REQ-022
func (c *Client) SearchIssues(ctx context.Context, token, query string) ([]IssueCandidate, error) {
	panic("stub: REQ-022 not yet implemented")
}
