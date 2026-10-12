package llm

import (
	"context"
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
	panic("stub: REQ-008 not yet implemented")
}

// TRACEABILITY_MAPPING: REQ-008
func (c *Client) GenerateFile(ctx context.Context, req GenerateRequest) (string, error) {
	panic("stub: REQ-008 not yet implemented")
}
