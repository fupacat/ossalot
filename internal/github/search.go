package github

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

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
func extractRepoFullNameFromURL(repoAPIURL, htmlURL string) string {
	if idx := strings.Index(repoAPIURL, "/repos/"); idx >= 0 {
		return strings.Trim(repoAPIURL[idx+len("/repos/"):], "/")
	}
	cleaned := strings.TrimPrefix(htmlURL, "https://github.com/")
	parts := strings.Split(cleaned, "/")
	if len(parts) >= 2 {
		return parts[0] + "/" + parts[1]
	}
	return ""
}

// TRACEABILITY_MAPPING: REQ-022
func (c *Client) SearchIssues(ctx context.Context, token, query string) ([]IssueCandidate, error) {
	var resp struct {
		Items []struct {
			Number        int    `json:"number"`
			Title         string `json:"title"`
			Body          string `json:"body"`
			HTMLURL       string `json:"html_url"`
			RepositoryURL string `json:"repository_url"`
			User          struct {
				Login string `json:"login"`
			} `json:"user"`
			Labels []struct {
				Name string `json:"name"`
			} `json:"labels"`
		} `json:"items"`
	}
	searchURL := fmt.Sprintf("%s/search/issues?q=%s&per_page=20", c.APIBaseURL, url.QueryEscape(query))
	if err := c.doJSON(ctx, http.MethodGet, searchURL, token, nil, &resp); err != nil {
		return nil, err
	}
	out := make([]IssueCandidate, 0, len(resp.Items))
	for _, item := range resp.Items {
		labels := make([]string, 0, len(item.Labels))
		for _, l := range item.Labels {
			labels = append(labels, l.Name)
		}
		out = append(out, IssueCandidate{
			TargetRepoFullName: extractRepoFullNameFromURL(item.RepositoryURL, item.HTMLURL),
			IssueNumber:        item.Number,
			Title:              item.Title,
			Body:               item.Body,
			HTMLURL:            item.HTMLURL,
			Author:             item.User.Login,
			Labels:             labels,
		})
	}
	return out, nil
}
