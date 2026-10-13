package api

import (
	"encoding/json"
	"net/http"

	"github.com/eric/ossalot/internal/db"
	"github.com/eric/ossalot/internal/engine/community"
	"github.com/eric/ossalot/internal/engine/notifier"
	"github.com/eric/ossalot/internal/engine/profiler"
	"github.com/eric/ossalot/internal/scheduler"
)

// TRACEABILITY_MAPPING: REQ-030
func (s *Server) RegisterCommunityAndInboxRoutes(scanner *community.Scanner, notif *notifier.Notifier, runner *scheduler.Runner) {
	s.mux.HandleFunc("POST /api/v1/repos/{id}/community/scan", func(w http.ResponseWriter, r *http.Request) {
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
		discovered, err := scanner.ScanCommunities(r.Context(), repo, prof)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
			return
		}
		settings := s.store.GetSettingsRaw()
		saved := make([]db.Opportunity, 0, len(discovered))
		for _, opp := range discovered {
			persisted, _ := s.store.UpsertOpportunity(opp)
			_, _ = notif.NotifyHighValueOpportunity(r.Context(), settings, persisted)
			saved = append(saved, persisted)
		}
		writeJSON(w, http.StatusOK, saved)
	})

	s.mux.HandleFunc("POST /api/v1/repos/{id}/playbooks", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := s.store.GetRepository(r.PathValue("id"))
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "repository not found"})
			return
		}
		prof, _ := s.store.GetCapabilityProfile(repo.ID)
		playbooks := community.GenerateMarketingPlaybooks(repo, prof)
		saved := make([]db.Opportunity, 0, len(playbooks))
		for _, pb := range playbooks {
			persisted, _ := s.store.UpsertOpportunity(pb)
			saved = append(saved, persisted)
		}
		writeJSON(w, http.StatusOK, saved)
	})

	s.mux.HandleFunc("GET /api/v1/opportunities", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, s.store.ListOpportunities(""))
	})

	s.mux.HandleFunc("PUT /api/v1/opportunities/{id}/status", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Status       string `json:"status"`
			SubmittedURL string `json:"submitted_url"`
			OutcomeNotes string `json:"outcome_notes"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Status == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid status payload"})
			return
		}
		updated, ok := s.store.UpdateOpportunityStatus(r.PathValue("id"), req.Status, req.SubmittedURL, req.OutcomeNotes)
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "opportunity not found"})
			return
		}
		writeJSON(w, http.StatusOK, updated)
	})

	s.mux.HandleFunc("GET /api/v1/jobs", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, runner.ListJobs())
	})

	s.mux.HandleFunc("POST /api/v1/jobs/{name}/run", func(w http.ResponseWriter, r *http.Request) {
		rec, err := runner.TriggerJob(r.Context(), r.PathValue("name"))
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, rec)
	})
}
