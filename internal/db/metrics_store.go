package db

// TRACEABILITY_MAPPING: REQ-014
type ReferrerStat struct {
	Referrer string `json:"referrer"`
	Count    int    `json:"count"`
	Uniques  int    `json:"uniques"`
}

// TRACEABILITY_MAPPING: REQ-014
type MetricSnapshot struct {
	RepoID            string          `json:"repo_id"`
	SnapshotDate      string          `json:"snapshot_date"`
	Stars             int             `json:"stars"`
	Forks             int             `json:"forks"`
	OpenIssues        int             `json:"open_issues"`
	ViewsTotal        int             `json:"views_total"`
	ViewsUnique       int             `json:"views_unique"`
	ClonesTotal       int             `json:"clones_total"`
	ClonesUnique      int             `json:"clones_unique"`
	Referrers         []ReferrerStat  `json:"referrers"`
	RegistryDownloads map[string]int  `json:"registry_downloads"`
	AdoptionScore     int             `json:"adoption_score"`
}

// TRACEABILITY_MAPPING: REQ-014
func (s *Store) UpsertMetricSnapshot(snap MetricSnapshot) error {
	panic("stub: REQ-014 not yet implemented")
}

// TRACEABILITY_MAPPING: REQ-014
func (s *Store) ListMetricSnapshots(repoID string) []MetricSnapshot {
	panic("stub: REQ-014 not yet implemented")
}
