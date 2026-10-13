package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// TRACEABILITY_MAPPING: REQ-008
type GenerateRequest struct {
	Provider        string `json:"provider"`
	APIKey          string `json:"api_key"`
	Model           string `json:"model"`
	RepoFullName    string `json:"repo_full_name"`
	RepoDescription string `json:"repo_description"`
	PrimaryLanguage string `json:"primary_language"`
	CheckKey        string `json:"check_key"`
}

// TRACEABILITY_MAPPING: REQ-008
type Client struct {
	EndpointURL string
	HTTPClient  *http.Client
}

// TRACEABILITY_MAPPING: REQ-008
func NewClient(endpointURL string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{
		EndpointURL: endpointURL,
		HTTPClient:  httpClient,
	}
}

// TRACEABILITY_MAPPING: REQ-008
func renderDeterministicTemplate(req GenerateRequest) string {
	lang := req.PrimaryLanguage
	if lang == "" {
		lang = "software"
	}
	switch req.CheckKey {
	case "contributing_guide":
		return fmt.Sprintf("# Contributing to %s\n\nThank you for helping improve %s (%s).\n\n## Development Workflow\n1. Fork the repository and create a feature branch.\n2. Add unit tests for all behavior changes.\n3. Open a Pull Request with a clear summary.\n", req.RepoFullName, req.RepoFullName, lang)
	case "code_of_conduct":
		return fmt.Sprintf("# Contributor Covenant Code of Conduct\n\n## Our Pledge\nWe as contributors and maintainers of %s pledge to make participation a harassment-free experience for everyone.\n", req.RepoFullName)
	case "security_policy":
		return fmt.Sprintf("# Security Policy for %s\n\n## Reporting a Vulnerability\nPlease report suspected security vulnerabilities privately via GitHub Security Advisories on %s rather than public issues.\n", req.RepoFullName, req.RepoFullName)
	case "license_valid":
		return fmt.Sprintf("MIT License\n\nCopyright (c) 2026 %s Contributors\n\nPermission is hereby granted, free of charge, to any person obtaining a copy\nof this software and associated documentation files (the \"Software\"), to deal\nin the Software without restriction, including without limitation the rights\nto use, copy, modify, merge, publish, distribute, sublicense, and/or sell\ncopies of the Software, and to permit persons to whom the Software is\nfurnished to do so, subject to the following conditions:\n\nThe above copyright notice and this permission notice shall be included in all\ncopies or substantial portions of the Software.\n\nTHE SOFTWARE IS PROVIDED \"AS IS\", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR\nIMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,\nFITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE\nAUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER\nLIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,\nOUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE\nSOFTWARE.\n", req.RepoFullName)
	case "issue_templates":
		return fmt.Sprintf("---\nname: Bug report\nabout: Create a report to help improve %s\n---\n\n## Steps to Reproduce\n1.\n\n## Expected Behavior\n", req.RepoFullName)
	case "pr_template":
		return fmt.Sprintf("## Summary\nDescribe the changes to %s.\n\n## Verification\n- [ ] Tests added and passing\n", req.RepoFullName)
	case "ci_workflows":
		return fmt.Sprintf("name: CI\non:\n  push:\n    branches: [main]\n  pull_request:\njobs:\n  build:\n    runs-on: ubuntu-latest\n    steps:\n      - uses: actions/checkout@v4\n      - name: Verify %s (%s)\n        run: echo \"Running CI checks\"\n", req.RepoFullName, lang)
	default:
		return fmt.Sprintf("# %s\n\n%s\n\n## Installation\nSee project instructions for %s.\n\n## Quickstart\nRun the default command to get started.\n", req.RepoFullName, req.RepoDescription, lang)
	}
}

// TRACEABILITY_MAPPING: REQ-008
func (c *Client) GenerateFile(ctx context.Context, req GenerateRequest) (string, error) {
	if req.APIKey == "" || c.EndpointURL == "" {
		return renderDeterministicTemplate(req), nil
	}
	payload, _ := json.Marshal(req)
	httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, c.EndpointURL, bytes.NewReader(payload))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+req.APIKey)

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("llm provider returned status %d", resp.StatusCode)
	}
	var out struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil || out.Content == "" {
		return "", fmt.Errorf("invalid llm response payload")
	}
	return out.Content, nil
}
