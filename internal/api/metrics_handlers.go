package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/eric/ossalot/internal/engine/metrics"
)

// TRACEABILITY_MAPPING: REQ-018
func (s *Server) RegisterMetricsRoutes(regClient *metrics.RegistryClient) {
	s.mux.HandleFunc("POST /api/v1/repos/{id}/metrics/collect", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := s.store.GetRepository(r.PathValue("id"))
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "repository not found"})
			return
		}
		var req struct {
			SnapshotDate string            `json:"snapshot_date"`
			Packages     map[string]string `json:"packages"`
		}
		if r.ContentLength > 0 {
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request JSON"})
				return
			}
		}
		if req.SnapshotDate == "" {
			req.SnapshotDate = time.Now().UTC().Format("2006-01-02")
		}
		settings := s.store.GetSettingsRaw()
		insp, err := s.ghClient.InspectRepository(r.Context(), settings.GitHubToken, repo.Owner, repo.Name)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
			return
		}
		traffic, err := s.ghClient.FetchTrafficSnapshot(r.Context(), settings.GitHubToken, repo.Owner, repo.Name)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
			return
		}
		downloads, err := regClient.FetchDownloads(r.Context(), req.Packages)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
			return
		}
		snap := metrics.BuildDailySnapshot(repo.ID, req.SnapshotDate, insp, traffic, downloads)
		_ = s.store.UpsertMetricSnapshot(snap)
		writeJSON(w, http.StatusOK, snap)
	})

	s.mux.HandleFunc("GET /api/v1/repos/{id}/metrics", func(w http.ResponseWriter, r *http.Request) {
		if _, ok := s.store.GetRepository(r.PathValue("id")); !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "repository not found"})
			return
		}
		writeJSON(w, http.StatusOK, s.store.ListMetricSnapshots(r.PathValue("id")))
	})
}
