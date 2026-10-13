package community

import (
	"context"
	"net/http"

	"github.com/eric/ossalot/internal/db"
)

// TRACEABILITY_MAPPING: REQ-026
type Scanner struct {
	BaseURL    string
	HTTPClient *http.Client
}

// TRACEABILITY_MAPPING: REQ-026
func NewScanner(baseURL string, httpClient *http.Client) *Scanner {
	panic("stub: REQ-026 not yet implemented")
}

// TRACEABILITY_MAPPING: REQ-026
func (s *Scanner) ScanCommunities(ctx context.Context, repo db.Repository, profile db.CapabilityProfile) ([]db.Opportunity, error) {
	panic("stub: REQ-026 not yet implemented")
}

// TRACEABILITY_MAPPING: REQ-027
func GenerateMarketingPlaybooks(repo db.Repository, profile db.CapabilityProfile) []db.Opportunity {
	panic("stub: REQ-027 not yet implemented")
}
