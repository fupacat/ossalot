package metrics

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

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
	if baseURL == "" {
		baseURL = "https://api.npmjs.org"
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &RegistryClient{
		BaseURL:    strings.TrimRight(baseURL, "/"),
		HTTPClient: httpClient,
	}
}

// TRACEABILITY_MAPPING: REQ-016
func (rc *RegistryClient) FetchDownloads(ctx context.Context, packages map[string]string) (map[string]int, error) {
	out := map[string]int{}
	for eco, pkg := range packages {
		url := fmt.Sprintf("%s/downloads/%s/%s", rc.BaseURL, eco, pkg)
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		resp, err := rc.HTTPClient.Do(req)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			resp.Body.Close()
			return nil, fmt.Errorf("registry %s returned status %d", eco, resp.StatusCode)
		}
		var payload struct {
			Downloads int `json:"downloads"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&payload)
		resp.Body.Close()
		out[eco] = payload.Downloads
	}
	return out, nil
}

// TRACEABILITY_MAPPING: REQ-017
func BuildDailySnapshot(repoID, snapshotDate string, insp github.RepoInspection, traffic github.TrafficSnapshot, downloads map[string]int) db.MetricSnapshot {
	refs := make([]db.ReferrerStat, 0, len(traffic.Referrers))
	for _, r := range traffic.Referrers {
		refs = append(refs, db.ReferrerStat{
			Referrer: r.Referrer,
			Count:    r.Count,
			Uniques:  r.Uniques,
		})
	}
	dlCopy := map[string]int{}
	totalDownloads := 0
	for k, v := range downloads {
		dlCopy[k] = v
		totalDownloads += v
	}

	score := (insp.Stars / 5) + (insp.Forks * 2) + (traffic.ViewsUnique / 5) + (traffic.ClonesUnique * 2) + (totalDownloads / 50)
	if score > 100 {
		score = 100
	}
	if score < 0 {
		score = 0
	}

	return db.MetricSnapshot{
		RepoID:            repoID,
		SnapshotDate:      snapshotDate,
		Stars:             insp.Stars,
		Forks:             insp.Forks,
		OpenIssues:        insp.OpenIssues,
		ViewsTotal:        traffic.ViewsTotal,
		ViewsUnique:       traffic.ViewsUnique,
		ClonesTotal:       traffic.ClonesTotal,
		ClonesUnique:      traffic.ClonesUnique,
		Referrers:         refs,
		RegistryDownloads: dlCopy,
		AdoptionScore:     score,
	}
}
