package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/eric/ossalot/internal/db"
	"github.com/eric/ossalot/internal/engine/standards"
	"github.com/eric/ossalot/internal/github"
	"github.com/eric/ossalot/internal/llm"
)

// TRACEABILITY_MAPPING: REQ-011
type Server struct {
	store    *db.Store
	ghClient *github.Client
	llm      *llm.Client
	mux      *http.ServeMux
}

// TRACEABILITY_MAPPING: REQ-011
func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

// TRACEABILITY_MAPPING: REQ-011, REQ-055, REQ-056
func NewServer(store *db.Store, ghClient *github.Client, llmClient *llm.Client) *Server {
	s := &Server{
		store:    store,
		ghClient: ghClient,
		llm:      llmClient,
		mux:      http.NewServeMux(),
	}
	s.mux.HandleFunc("GET /api/v1/health", s.handleHealth)
	s.mux.HandleFunc("GET /api/v1/settings", s.handleGetSettings)
	s.mux.HandleFunc("PUT /api/v1/settings", s.handlePutSettings)
	s.mux.HandleFunc("GET /api/v1/repos", s.handleListRepos)
	s.mux.HandleFunc("POST /api/v1/repos", s.handleAddRepo)
	s.mux.HandleFunc("POST /api/v1/repos/sync", s.handleSyncRepos)
	s.mux.HandleFunc("GET /api/v1/repos/{id}", s.handleGetRepo)
	s.mux.HandleFunc("PATCH /api/v1/repos/{id}", s.handlePatchRepo)
	s.mux.HandleFunc("DELETE /api/v1/repos/{id}", s.handleDeleteRepo)
	s.mux.HandleFunc("POST /api/v1/repos/{id}/audit", s.handleAuditRepo)
	s.mux.HandleFunc("POST /api/v1/repos/{id}/remediate", s.handleRemediateRepo)
	s.mux.HandleFunc("POST /api/v1/repos/{id}/issues", s.handleCreateIssue)
	return s
}

// TRACEABILITY_MAPPING: REQ-011
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// TRACEABILITY_MAPPING: REQ-011
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	ws := s.store.GetDefaultWorkspace()
	writeJSON(w, http.StatusOK, map[string]any{
		"status":    "ok",
		"workspace": ws,
	})
}

// TRACEABILITY_MAPPING: REQ-011
func (s *Server) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.store.GetSettingsRedacted())
}

// TRACEABILITY_MAPPING: REQ-011
func (s *Server) handlePutSettings(w http.ResponseWriter, r *http.Request) {
	var body db.WorkspaceSettings
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid settings JSON"})
		return
	}
	_ = s.store.SaveSettings(body)
	writeJSON(w, http.StatusOK, s.store.GetSettingsRedacted())
}

// TRACEABILITY_MAPPING: REQ-011
func (s *Server) handleListRepos(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.store.ListRepositories())
}

// TRACEABILITY_MAPPING: REQ-011
func (s *Server) handleAddRepo(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL     string `json:"url"`
		IsOwned bool   `json:"is_owned"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	owner, name, err := github.ParseRepoCoordinates(req.URL)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	settings := s.store.GetSettingsRaw()
	insp, err := s.ghClient.InspectRepository(r.Context(), settings.GitHubToken, owner, name)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	eval := standards.EvaluateStandards(insp)
	now := time.Now().UTC().Format(time.RFC3339)
	saved, _ := s.store.UpsertRepository(db.Repository{
		Owner:           owner,
		Name:            name,
		FullName:        insp.FullName,
		HTMLURL:         insp.HTMLURL,
		Description:     insp.Description,
		PrimaryLanguage: insp.PrimaryLanguage,
		Topics:          insp.Topics,
		IsOwned:         req.IsOwned,
		StandardsScore:  eval.OverallScore,
		LastSyncedAt:    now,
	})
	_ = s.store.SaveAudit(saved.ID, db.AuditRecord{
		ID:             fmt.Sprintf("audit-%s", saved.ID),
		RepoID:         saved.ID,
		OverallScore:   eval.OverallScore,
		CategoryScores: eval.CategoryScores,
		Checks:         eval.Checks,
		CreatedAt:      now,
	})
	updated, _ := s.store.GetRepository(saved.ID)
	writeJSON(w, http.StatusCreated, updated)
}

// TRACEABILITY_MAPPING: REQ-011
func (s *Server) handleSyncRepos(w http.ResponseWriter, r *http.Request) {
	settings := s.store.GetSettingsRaw()
	discovered, err := s.ghClient.ListAuthenticatedRepos(r.Context(), settings.GitHubToken)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	now := time.Now().UTC().Format(time.RFC3339)
	for _, item := range discovered {
		_, _ = s.store.UpsertRepository(db.Repository{
			Owner:           item.Owner,
			Name:            item.Name,
			FullName:        item.FullName,
			HTMLURL:         item.HTMLURL,
			Description:     item.Description,
			PrimaryLanguage: item.PrimaryLanguage,
			Topics:          item.Topics,
			IsOwned:         true,
			LastSyncedAt:    now,
		})
	}
	writeJSON(w, http.StatusOK, s.store.ListRepositories())
}

// TRACEABILITY_MAPPING: REQ-011
func (s *Server) handleGetRepo(w http.ResponseWriter, r *http.Request) {
	repo, ok := s.store.GetRepository(r.PathValue("id"))
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "repository not found"})
		return
	}
	writeJSON(w, http.StatusOK, repo)
}

// TRACEABILITY_MAPPING: REQ-011
func (s *Server) handleDeleteRepo(w http.ResponseWriter, r *http.Request) {
	if !s.store.DeleteRepository(r.PathValue("id")) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "repository not found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"deleted": true})
}

// TRACEABILITY_MAPPING: REQ-011
func (s *Server) handleAuditRepo(w http.ResponseWriter, r *http.Request) {
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
	eval := standards.EvaluateStandards(insp)
	now := time.Now().UTC().Format(time.RFC3339)
	_ = s.store.SaveAudit(repo.ID, db.AuditRecord{
		ID:             fmt.Sprintf("audit-%s", repo.ID),
		RepoID:         repo.ID,
		OverallScore:   eval.OverallScore,
		CategoryScores: eval.CategoryScores,
		Checks:         eval.Checks,
		CreatedAt:      now,
	})
	updated, _ := s.store.GetRepository(repo.ID)
	writeJSON(w, http.StatusOK, updated)
}

// TRACEABILITY_MAPPING: REQ-011, REQ-043
func (s *Server) handleRemediateRepo(w http.ResponseWriter, r *http.Request) {
	repo, ok := s.store.GetRepository(r.PathValue("id"))
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "repository not found"})
		return
	}
	var req struct {
		CheckKey string `json:"check_key"`
		CreatePR bool   `json:"create_pr"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request JSON"})
		return
	}
	settings := s.store.GetSettingsRaw()
	draft, err := standards.BuildRemediationDraft(r.Context(), s.llm, settings, repo, req.CheckKey)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	prURL := ""
	if req.CreatePR {
		prURL, err = s.ghClient.CreateRemediationPR(r.Context(), settings.GitHubToken, github.PullRequestRequest{
			Owner:         repo.Owner,
			Repo:          repo.Name,
			IsOwned:       repo.IsOwned,
			BaseBranch:    "", // the client targets the repository's default branch
			NewBranch:     draft.BranchName,
			FilePath:      draft.FilePath,
			FileContent:   draft.Content,
			CommitMessage: draft.CommitMessage,
			PRTitle:       draft.PRTitle,
			PRBody:        draft.PRBody,
		})
		if err != nil {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": err.Error()})
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"draft":  draft,
		"pr_url": prURL,
	})
}

// TRACEABILITY_MAPPING: REQ-055
func (s *Server) handleCreateIssue(w http.ResponseWriter, r *http.Request) {
	repo, ok := s.store.GetRepository(r.PathValue("id"))
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "repository not found"})
		return
	}
	var req struct {
		Title  string   `json:"title"`
		Body   string   `json:"body"`
		Labels []string `json:"labels,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request JSON"})
		return
	}
	if req.Title == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "title is required"})
		return
	}
	settings := s.store.GetSettingsRaw()
	res, err := s.ghClient.CreateIssue(r.Context(), settings.GitHubToken, github.IssueRequest{
		Owner:   repo.Owner,
		Repo:    repo.Name,
		IsOwned: repo.IsOwned,
		Title:   req.Title,
		Body:    req.Body,
		Labels:  req.Labels,
	})
	if err != nil {
		if errors.Is(err, github.ErrRepoNotOwned) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// TRACEABILITY_MAPPING: REQ-056
func (s *Server) handlePatchRepo(w http.ResponseWriter, r *http.Request) {
	var patch db.RepoPatch
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request JSON"})
		return
	}
	updated, ok, err := s.store.PatchRepository(r.PathValue("id"), patch)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "repository not found"})
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

