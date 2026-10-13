package db

import (
	"encoding/json"
	"errors"
	"os"
	"sort"
)

// TRACEABILITY_MAPPING: REQ-014
type ReferrerStat struct {
	Referrer string `json:"referrer"`
	Count    int    `json:"count"`
	Uniques  int    `json:"uniques"`
}

// TRACEABILITY_MAPPING: REQ-014
type MetricSnapshot struct {
	RepoID            string         `json:"repo_id"`
	SnapshotDate      string         `json:"snapshot_date"`
	Stars             int            `json:"stars"`
	Forks             int            `json:"forks"`
	OpenIssues        int            `json:"open_issues"`
	ViewsTotal        int            `json:"views_total"`
	ViewsUnique       int            `json:"views_unique"`
	ClonesTotal       int            `json:"clones_total"`
	ClonesUnique      int            `json:"clones_unique"`
	Referrers         []ReferrerStat `json:"referrers"`
	RegistryDownloads map[string]int `json:"registry_downloads"`
	AdoptionScore     int            `json:"adoption_score"`
}

// TRACEABILITY_MAPPING: REQ-014
func (s *Store) metricsFilePath() string {
	return s.path + ".metrics.json"
}

// TRACEABILITY_MAPPING: REQ-014
func (s *Store) loadAllSnapshotsLocked() []MetricSnapshot {
	data, err := os.ReadFile(s.metricsFilePath())
	if err != nil || len(data) == 0 {
		return []MetricSnapshot{}
	}
	var list []MetricSnapshot
	_ = json.Unmarshal(data, &list)
	return list
}

// TRACEABILITY_MAPPING: REQ-014
func (s *Store) saveAllSnapshotsLocked(list []MetricSnapshot) error {
	raw, _ := json.MarshalIndent(list, "", "  ")
	return os.WriteFile(s.metricsFilePath(), raw, 0o600)
}

// TRACEABILITY_MAPPING: REQ-014
func (s *Store) UpsertMetricSnapshot(snap MetricSnapshot) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	foundRepo := false
	for _, r := range s.state.Repos {
		if r.ID == snap.RepoID {
			foundRepo = true
			break
		}
	}
	if !foundRepo {
		return errors.New("repository not found")
	}

	all := s.loadAllSnapshotsLocked()
	updated := false
	for i, existing := range all {
		if existing.RepoID == snap.RepoID && existing.SnapshotDate == snap.SnapshotDate {
			all[i] = snap
			updated = true
			break
		}
	}
	if !updated {
		all = append(all, snap)
	}
	return s.saveAllSnapshotsLocked(all)
}

// TRACEABILITY_MAPPING: REQ-014
func (s *Store) ListMetricSnapshots(repoID string) []MetricSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	all := s.loadAllSnapshotsLocked()
	var out []MetricSnapshot
	for _, item := range all {
		if item.RepoID == repoID {
			out = append(out, item)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].SnapshotDate < out[j].SnapshotDate
	})
	return out
}
