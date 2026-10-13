package metrics

import (
	"context"
	"net/http"

	"github.com/eric/ossalot/internal/db"
	"github.com/eric/ossalot/internal/github"
)

// TRACEABILITY_MAPPING: REQ-016
type RegistryClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

// TRACEABILITY_MAPPING: REQ-016
func NewRegistryClient(baseURL string, httpClient *http.Client) *RegistryClient {
	panic("stub: REQ-016 not yet implemented")
}

// TRACEABILITY_MAPPING: REQ-016
func (rc *RegistryClient) FetchDownloads(ctx context.Context, packages map[string]string) (map[string]int, error) {
	panic("stub: REQ-016 not yet implemented")
}

// TRACEABILITY_MAPPING: REQ-017
func BuildDailySnapshot(repoID, snapshotDate string, insp github.RepoInspection, traffic github.TrafficSnapshot, downloads map[string]int) db.MetricSnapshot {
	panic("stub: REQ-017 not yet implemented")
}
