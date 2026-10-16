package github

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// TRACEABILITY_MAPPING: REQ-007
var ErrRepoNotOwned = errors.New("repository is not marked as owned by this workspace")

// TRACEABILITY_MAPPING: REQ-046
var ErrWorkflowScopeRequired = errors.New("GitHub requires the 'workflow' OAuth scope to create or edit files in .github/workflows/; grant it with 'gh auth refresh -s workflow', or copy the generated file and commit it yourself")

// TRACEABILITY_MAPPING: REQ-043, REQ-045, REQ-046, REQ-047
type StatusError struct {
	StatusCode int
	URL        string
	Body       string
}

// TRACEABILITY_MAPPING: REQ-043, REQ-045, REQ-046, REQ-047
func (e *StatusError) Error() string {
	return fmt.Sprintf("unexpected HTTP status %d for %s", e.StatusCode, e.URL)
}

// TRACEABILITY_MAPPING: REQ-043, REQ-044, REQ-046, REQ-047
func hasStatus(err error, codes ...int) bool {
	var se *StatusError
	if !errors.As(err, &se) {
		return false
	}
	for _, code := range codes {
		if se.StatusCode == code {
			return true
		}
	}
	return false
}

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

// TRACEABILITY_MAPPING: REQ-006, REQ-007, REQ-041
type Client struct {
	APIBaseURL       string
	ScorecardBaseURL string
	HTTPClient       *http.Client
	// TokenFunc, when set, picks the token for each request to APIBaseURL from the caller's token.
	TokenFunc func(requestToken string) string
}

// TRACEABILITY_MAPPING: REQ-006
func NewClient(apiBaseURL, scorecardBaseURL string, httpClient *http.Client) *Client {
	if apiBaseURL == "" {
		apiBaseURL = "https://api.github.com"
	}
	if scorecardBaseURL == "" {
		scorecardBaseURL = "https://api.securityscorecards.dev"
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{
		APIBaseURL:       strings.TrimRight(apiBaseURL, "/"),
		ScorecardBaseURL: strings.TrimRight(scorecardBaseURL, "/"),
		HTTPClient:       httpClient,
	}
}

// TRACEABILITY_MAPPING: REQ-005
func ParseRepoCoordinates(input string) (string, string, error) {
	cleaned := strings.TrimSpace(input)
	cleaned = strings.TrimPrefix(cleaned, "https://github.com/")
	cleaned = strings.TrimPrefix(cleaned, "http://github.com/")
	cleaned = strings.Trim(cleaned, "/")
	cleaned = strings.TrimSuffix(cleaned, ".git")
	parts := strings.Split(cleaned, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", errors.New("invalid repository coordinates: expected owner/repo or https://github.com/owner/repo")
	}
	return parts[0], parts[1], nil
}

// TRACEABILITY_MAPPING: REQ-006, REQ-007, REQ-041
func (c *Client) doJSON(ctx context.Context, method, url, token string, body any, out any) error {
	var bodyReader *bytes.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(raw)
	} else {
		bodyReader = bytes.NewReader(nil)
	}
	req, _ := http.NewRequestWithContext(ctx, method, url, bodyReader)
	req.Header.Set("Accept", "application/vnd.github+json")
	// Only GitHub API requests get the resolved token; other hosts (Scorecard) never see it
	if c.TokenFunc != nil && strings.HasPrefix(url, c.APIBaseURL) {
		token = c.TokenFunc(token)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
		return &StatusError{StatusCode: resp.StatusCode, URL: url, Body: string(raw)}
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// TRACEABILITY_MAPPING: REQ-006
type ghRepoPayload struct {
	Name            string   `json:"name"`
	FullName        string   `json:"full_name"`
	HTMLURL         string   `json:"html_url"`
	Description     string   `json:"description"`
	Language        string   `json:"language"`
	DefaultBranch   string   `json:"default_branch"`
	StargazersCount int      `json:"stargazers_count"`
	ForksCount      int      `json:"forks_count"`
	OpenIssuesCount int      `json:"open_issues_count"`
	Topics          []string `json:"topics"`
	Owner           struct {
		Login string `json:"login"`
	} `json:"owner"`
	License struct {
		SPDXID string `json:"spdx_id"`
	} `json:"license"`
}

// TRACEABILITY_MAPPING: REQ-006
func (c *Client) InspectRepository(ctx context.Context, token, owner, repo string) (RepoInspection, error) {
	var meta ghRepoPayload
	repoURL := fmt.Sprintf("%s/repos/%s/%s", c.APIBaseURL, owner, repo)
	if err := c.doJSON(ctx, http.MethodGet, repoURL, token, nil, &meta); err != nil {
		return RepoInspection{}, err
	}
	branch := meta.DefaultBranch
	if branch == "" {
		branch = "main"
	}
	insp := RepoInspection{
		Owner:           owner,
		Name:            repo,
		FullName:        meta.FullName,
		HTMLURL:         meta.HTMLURL,
		Description:     meta.Description,
		PrimaryLanguage: meta.Language,
		DefaultBranch:   branch,
		LicenseSPDX:     meta.License.SPDXID,
		Topics:          meta.Topics,
		Stars:           meta.StargazersCount,
		Forks:           meta.ForksCount,
		OpenIssues:      meta.OpenIssuesCount,
		FilesPresent:    map[string]bool{},
	}

	var treeResp struct {
		Tree []struct {
			Path string `json:"path"`
		} `json:"tree"`
	}
	treeURL := fmt.Sprintf("%s/repos/%s/%s/git/trees/%s?recursive=1", c.APIBaseURL, owner, repo, branch)
	if err := c.doJSON(ctx, http.MethodGet, treeURL, token, nil, &treeResp); err == nil {
		for _, item := range treeResp.Tree {
			upper := strings.ToUpper(item.Path)
			insp.FilesPresent[upper] = true
			if strings.HasPrefix(item.Path, ".github/workflows/") {
				insp.HasCIWorkflow = true
			}
			if item.Path == ".github/dependabot.yml" || item.Path == "renovate.json" {
				insp.HasDependabot = true
			}
		}
	}

	var readmeResp struct {
		Content string `json:"content"`
	}
	readmeURL := fmt.Sprintf("%s/repos/%s/%s/readme", c.APIBaseURL, owner, repo)
	if err := c.doJSON(ctx, http.MethodGet, readmeURL, token, nil, &readmeResp); err == nil {
		decoded, _ := base64.StdEncoding.DecodeString(strings.ReplaceAll(readmeResp.Content, "\n", ""))
		insp.ReadmeContent = string(decoded)
	}

	var runsResp struct {
		WorkflowRuns []struct {
			Conclusion string `json:"conclusion"`
		} `json:"workflow_runs"`
	}
	runsURL := fmt.Sprintf("%s/repos/%s/%s/actions/runs?per_page=5", c.APIBaseURL, owner, repo)
	if err := c.doJSON(ctx, http.MethodGet, runsURL, token, nil, &runsResp); err == nil && len(runsResp.WorkflowRuns) > 0 {
		insp.RecentCIPassing = runsResp.WorkflowRuns[0].Conclusion == "success"
	}

	var releases []struct {
		TagName string `json:"tag_name"`
	}
	relURL := fmt.Sprintf("%s/repos/%s/%s/releases?per_page=5", c.APIBaseURL, owner, repo)
	if err := c.doJSON(ctx, http.MethodGet, relURL, token, nil, &releases); err == nil && len(releases) > 0 {
		insp.HasReleases = true
		if strings.HasPrefix(releases[0].TagName, "v") {
			insp.HasSemverRelease = true
		}
	}

	var scorecard struct {
		Score float64 `json:"score"`
	}
	scURL := fmt.Sprintf("%s/projects/github.com/%s/%s", c.ScorecardBaseURL, owner, repo)
	if err := c.doJSON(ctx, http.MethodGet, scURL, "", nil, &scorecard); err == nil {
		insp.OpenSSFScore = scorecard.Score
	}

	return insp, nil
}

// TRACEABILITY_MAPPING: REQ-006
func (c *Client) ListAuthenticatedRepos(ctx context.Context, token string) ([]RepoInspection, error) {
	var list []ghRepoPayload
	url := fmt.Sprintf("%s/user/repos?per_page=50&sort=updated", c.APIBaseURL)
	if err := c.doJSON(ctx, http.MethodGet, url, token, nil, &list); err != nil {
		return nil, err
	}
	out := make([]RepoInspection, 0, len(list))
	for _, item := range list {
		out = append(out, RepoInspection{
			Owner:           item.Owner.Login,
			Name:            item.Name,
			FullName:        item.FullName,
			HTMLURL:         item.HTMLURL,
			Description:     item.Description,
			PrimaryLanguage: item.Language,
			DefaultBranch:   item.DefaultBranch,
			LicenseSPDX:     item.License.SPDXID,
			Topics:          item.Topics,
			Stars:           item.StargazersCount,
			Forks:           item.ForksCount,
			OpenIssues:      item.OpenIssuesCount,
			FilesPresent:    map[string]bool{},
		})
	}
	return out, nil
}

// TRACEABILITY_MAPPING: REQ-007, REQ-043, REQ-044, REQ-045, REQ-046
func (c *Client) CreateRemediationPR(ctx context.Context, token string, req PullRequestRequest) (string, error) {
	if !req.IsOwned {
		return "", ErrRepoNotOwned
	}
	repoURL := fmt.Sprintf("%s/repos/%s/%s", c.APIBaseURL, req.Owner, req.Repo)
	base := req.BaseBranch
	if base == "" {
		var meta ghRepoPayload
		if err := c.doJSON(ctx, http.MethodGet, repoURL, token, nil, &meta); err != nil {
			return "", err
		}
		base = meta.DefaultBranch
	}
	if err := c.ensureBranch(ctx, token, repoURL, base, req.NewBranch); err != nil {
		return "", err
	}

	putPayload := map[string]string{
		"message": req.CommitMessage,
		"content": base64.StdEncoding.EncodeToString([]byte(req.FileContent)),
		"branch":  req.NewBranch,
	}
	sha, err := c.existingFileSHA(ctx, token, repoURL, req.FilePath, req.NewBranch)
	if err != nil {
		return "", err
	}
	if sha != "" {
		putPayload["sha"] = sha
	}
	var putResp map[string]any
	if err := c.doJSON(ctx, http.MethodPut, repoURL+"/contents/"+req.FilePath, token, putPayload, &putResp); err != nil {
		// GitHub answers 404, not 403, when the token lacks the workflow scope
		if strings.HasPrefix(req.FilePath, ".github/workflows/") && hasStatus(err, http.StatusNotFound) {
			return "", ErrWorkflowScopeRequired
		}
		return "", err
	}

	prPayload := map[string]string{
		"title": req.PRTitle,
		"head":  req.NewBranch,
		"base":  base,
		"body":  req.PRBody,
	}
	var prResp struct {
		HTMLURL string `json:"html_url"`
	}
	if err := c.doJSON(ctx, http.MethodPost, repoURL+"/pulls", token, prPayload, &prResp); err != nil {
		var se *StatusError
		if errors.As(err, &se) && se.StatusCode == http.StatusUnprocessableEntity && strings.Contains(se.Body, "A pull request already exists") {
			return c.openPullRequestURL(ctx, token, repoURL, req.Owner, req.NewBranch)
		}
		return "", err
	}
	return prResp.HTMLURL, nil
}

// TRACEABILITY_MAPPING: REQ-043
func (c *Client) ensureBranch(ctx context.Context, token, repoURL, base, branch string) error {
	var existing map[string]any
	err := c.doJSON(ctx, http.MethodGet, repoURL+"/git/ref/heads/"+branch, token, nil, &existing)
	if !hasStatus(err, http.StatusNotFound) {
		return err
	}
	var baseRef struct {
		Object struct {
			SHA string `json:"sha"`
		} `json:"object"`
	}
	if err := c.doJSON(ctx, http.MethodGet, repoURL+"/git/ref/heads/"+base, token, nil, &baseRef); err != nil {
		return err
	}
	var created map[string]any
	return c.doJSON(ctx, http.MethodPost, repoURL+"/git/refs", token, map[string]string{
		"ref": "refs/heads/" + branch,
		"sha": baseRef.Object.SHA,
	}, &created)
}

// TRACEABILITY_MAPPING: REQ-044
func (c *Client) existingFileSHA(ctx context.Context, token, repoURL, path, branch string) (string, error) {
	var file struct {
		SHA string `json:"sha"`
	}
	err := c.doJSON(ctx, http.MethodGet, fmt.Sprintf("%s/contents/%s?ref=%s", repoURL, path, url.QueryEscape(branch)), token, nil, &file)
	if hasStatus(err, http.StatusNotFound) {
		return "", nil
	}
	return file.SHA, err
}

// TRACEABILITY_MAPPING: REQ-045
func (c *Client) openPullRequestURL(ctx context.Context, token, repoURL, owner, branch string) (string, error) {
	var prs []struct {
		HTMLURL string `json:"html_url"`
	}
	listURL := fmt.Sprintf("%s/pulls?state=open&head=%s", repoURL, url.QueryEscape(owner+":"+branch))
	if err := c.doJSON(ctx, http.MethodGet, listURL, token, nil, &prs); err != nil {
		return "", err
	}
	if len(prs) == 0 {
		return "", fmt.Errorf("GitHub reported an existing pull request for %s, but none is open", branch)
	}
	return prs[0].HTMLURL, nil
}

// TRACEABILITY_MAPPING: REQ-055
type IssueRequest struct {
	Owner   string   `json:"owner"`
	Repo    string   `json:"repo"`
	IsOwned bool     `json:"is_owned"`
	Title   string   `json:"title"`
	Body    string   `json:"body"`
	Labels  []string `json:"labels,omitempty"`
}

// TRACEABILITY_MAPPING: REQ-055
type IssueResult struct {
	Number  int    `json:"number"`
	URL     string `json:"url"`
	HTMLURL string `json:"html_url"`
}

// TRACEABILITY_MAPPING: REQ-055
func (c *Client) CreateIssue(ctx context.Context, token string, req IssueRequest) (IssueResult, error) {
	if !req.IsOwned {
		return IssueResult{}, ErrRepoNotOwned
	}
	repoURL := fmt.Sprintf("%s/repos/%s/%s/issues", c.APIBaseURL, req.Owner, req.Repo)
	payload := map[string]any{
		"title": req.Title,
		"body":  req.Body,
	}
	if len(req.Labels) > 0 {
		payload["labels"] = req.Labels
	}
	var ghResp struct {
		Number  int    `json:"number"`
		URL     string `json:"url"`
		HTMLURL string `json:"html_url"`
	}
	if err := c.doJSON(ctx, http.MethodPost, repoURL, token, payload, &ghResp); err != nil {
		return IssueResult{}, err
	}
	return IssueResult{
		Number:  ghResp.Number,
		URL:     ghResp.URL,
		HTMLURL: ghResp.HTMLURL,
	}, nil
}
