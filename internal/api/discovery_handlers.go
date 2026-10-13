package api

import (
	"net/http"

	"github.com/eric/ossalot/internal/db"
	"github.com/eric/ossalot/internal/engine/discovery"
	"github.com/eric/ossalot/internal/engine/profiler"
)

// TRACEABILITY_MAPPING: REQ-024
func (s *Server) RegisterDiscoveryRoutes() {
	s.mux.HandleFunc("POST /api/v1/repos/{id}/profile", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := s.store.GetRepository(r.PathValue("id"))
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "repository not found"})
			return
		}
		settings := s.store.GetSettingsRaw()
		insp, err := s.ghClient.InspectRepository(r.Context(), settings.GitHubToken, repo.Owner, repo.Name)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
			return
		}
		prof := profiler.BuildCapabilityProfile(repo, insp.ReadmeContent)
		_ = s.store.SaveCapabilityProfile(prof)
		writeJSON(w, http.StatusOK, prof)
	})

	s.mux.HandleFunc("GET /api/v1/repos/{id}/profile", func(w http.ResponseWriter, r *http.Request) {
		prof, ok := s.store.GetCapabilityProfile(r.PathValue("id"))
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "capability profile not found"})
			return
		}
		writeJSON(w, http.StatusOK, prof)
	})

	s.mux.HandleFunc("POST /api/v1/repos/{id}/discover", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := s.store.GetRepository(r.PathValue("id"))
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "repository not found"})
			return
		}
		prof, ok := s.store.GetCapabilityProfile(repo.ID)
		if !ok {
			prof = profiler.BuildCapabilityProfile(repo, "")
			_ = s.store.SaveCapabilityProfile(prof)
		}
		settings := s.store.GetSettingsRaw()
		candidates, err := s.ghClient.SearchIssues(r.Context(), settings.GitHubToken, prof.SearchQueries[0])
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
			return
		}
		scored := discovery.ScoreAndFilterCandidates(repo, prof, candidates)
		saved := make([]db.Opportunity, 0, len(scored))
		for _, opp := range scored {
			persisted, _ := s.store.UpsertOpportunity(opp)
			saved = append(saved, persisted)
		}
		writeJSON(w, http.StatusOK, saved)
	})

	s.mux.HandleFunc("GET /api/v1/repos/{id}/opportunities", func(w http.ResponseWriter, r *http.Request) {
		if _, ok := s.store.GetRepository(r.PathValue("id")); !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "repository not found"})
			return
		}
		writeJSON(w, http.StatusOK, s.store.ListOpportunities(r.PathValue("id")))
	})
}
