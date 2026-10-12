package github

import (
	"context"
	"errors"
	"net/http"
)

// TRACEABILITY_MAPPING: REQ-007
var ErrRepoNotOwned = errors.New("repository is not marked as owned by this workspace")

// TRACEABILITY_MAPPING: REQ-006
type RepoInspection struct {
	Owner            string          `json:"owner"`
	Name             string          `json:"name"`
	FullName         string          `json:"full_name"`
	HTMLURL          string          `json:"html_url"`
	Description      string          `json:"description"`
	PrimaryLanguage  string          `json:"primary_language"`
	DefaultBranch    string          `json:"default_branch"`
	LicenseSPDX      string          `json:"license_spdx"`
	Topics           []string        `json:"topics"`
	Stars            int             `json:"stars"`
	Forks            int             `json:"forks"`
	OpenIssues       int             `json:"open_issues"`
	FilesPresent     map[string]bool `json:"files_present"`
	ReadmeContent    string          `json:"readme_content"`
	HasCIWorkflow    bool            `json:"has_ci_workflow"`
	RecentCIPassing  bool            `json:"recent_ci_passing"`
	HasReleases      bool            `json:"has_releases"`
	HasSemverRelease bool            `json:"has_semver_release"`
	HasDependabot    bool            `json:"has_dependabot"`
	OpenSSFScore     float64         `json:"openssf_score"`
}

// TRACEABILITY_MAPPING: REQ-007
type PullRequestRequest struct {
	Owner         string `json:"owner"`
	Repo          string `json:"repo"`
	IsOwned       bool   `json:"is_owned"`
	BaseBranch    string `json:"base_branch"`
	NewBranch     string `json:"new_branch"`
	FilePath      string `json:"file_path"`
	FileContent   string `json:"file_content"`
	CommitMessage string `json:"commit_message"`
	PRTitle       string `json:"pr_title"`
	PRBody        string `json:"pr_body"`
}

// TRACEABILITY_MAPPING: REQ-006, REQ-007
type Client struct {
	APIBaseURL       string
	ScorecardBaseURL string
	HTTPClient       *http.Client
}

// TRACEABILITY_MAPPING: REQ-006
func NewClient(apiBaseURL, scorecardBaseURL string, httpClient *http.Client) *Client {
	panic("stub: REQ-006 not yet implemented")
}

// TRACEABILITY_MAPPING: REQ-005
func ParseRepoCoordinates(input string) (string, string, error) {
	panic("stub: REQ-005 not yet implemented")
}

// TRACEABILITY_MAPPING: REQ-006
func (c *Client) InspectRepository(ctx context.Context, token, owner, repo string) (RepoInspection, error) {
	panic("stub: REQ-006 not yet implemented")
}

// TRACEABILITY_MAPPING: REQ-006
func (c *Client) ListAuthenticatedRepos(ctx context.Context, token string) ([]RepoInspection, error) {
	panic("stub: REQ-006 not yet implemented")
}

// TRACEABILITY_MAPPING: REQ-007
func (c *Client) CreateRemediationPR(ctx context.Context, token string, req PullRequestRequest) (string, error) {
	panic("stub: REQ-007 not yet implemented")
}
