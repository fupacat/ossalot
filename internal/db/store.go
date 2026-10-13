package db

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
)

// TRACEABILITY_MAPPING: REQ-002
type Workspace struct {
	ID   string `json:"id"`
	Slug string `json:"slug"`
	Name string `json:"name"`
	Mode string `json:"mode"`
}

// TRACEABILITY_MAPPING: REQ-003
type WorkspaceSettings struct {
	WorkspaceID       string `json:"workspace_id"`
	GitHubToken       string `json:"github_token"`
	LLMProvider       string `json:"llm_provider"`
	LLMAPIKey         string `json:"llm_api_key"`
	LLMFastModel      string `json:"llm_fast_model"`
	LLMStrongModel    string `json:"llm_strong_model"`
	SlackWebhookURL   string `json:"slack_webhook_url"`
	DiscordWebhookURL string `json:"discord_webhook_url"`
}

// TRACEABILITY_MAPPING: REQ-004
type CheckItemRecord struct {
	CheckKey        string `json:"check_key"`
	Category        string `json:"category"`
	Status          string `json:"status"`
	Score           int    `json:"score"`
	Weight          int    `json:"weight"`
	Details         string `json:"details"`
	RemediationType string `json:"remediation_type"`
}

// TRACEABILITY_MAPPING: REQ-004
type AuditRecord struct {
	ID             string            `json:"id"`
	RepoID         string            `json:"repo_id"`
	OverallScore   int               `json:"overall_score"`
	CategoryScores map[string]int    `json:"category_scores"`
	Checks         []CheckItemRecord `json:"checks"`
	CreatedAt      string            `json:"created_at"`
}

// TRACEABILITY_MAPPING: REQ-004
type Repository struct {
	ID              string       `json:"id"`
	WorkspaceID     string       `json:"workspace_id"`
	Owner           string       `json:"owner"`
	Name            string       `json:"name"`
	FullName        string       `json:"full_name"`
	HTMLURL         string       `json:"html_url"`
	Description     string       `json:"description"`
	PrimaryLanguage string       `json:"primary_language"`
	Topics          []string     `json:"topics"`
	IsOwned         bool         `json:"is_owned"`
	StandardsScore  int          `json:"standards_score"`
	LastSyncedAt    string       `json:"last_synced_at"`
	LatestAudit     *AuditRecord `json:"latest_audit,omitempty"`
}

// TRACEABILITY_MAPPING: REQ-002, REQ-003, REQ-004
type persistedState struct {
	Workspace Workspace         `json:"workspace"`
	Settings  WorkspaceSettings `json:"settings"`
	Repos     []Repository      `json:"repos"`
	NextSeq   int               `json:"next_seq"`
}

// TRACEABILITY_MAPPING: REQ-002, REQ-003, REQ-004
type Store struct {
	mu    sync.RWMutex
	path  string
	state persistedState
}

// TRACEABILITY_MAPPING: REQ-002, REQ-003, REQ-004
func (s *Store) flushLocked() error {
	raw, _ := json.MarshalIndent(s.state, "", "  ")
	return os.WriteFile(s.path, raw, 0o600)
}

// TRACEABILITY_MAPPING: REQ-002
func OpenStore(path string) (*Store, error) {
	s := &Store{
		path: path,
		state: persistedState{
			Workspace: Workspace{
				ID:   "ws-default",
				Slug: "default",
				Name: "Personal Workspace",
				Mode: "personal",
			},
			Settings: WorkspaceSettings{
				WorkspaceID:    "ws-default",
				LLMProvider:    "gemini",
				LLMFastModel:   "gemini-2.5-flash",
				LLMStrongModel: "gemini-2.5-pro",
			},
			Repos:   []Repository{},
			NextSeq: 1,
		},
	}
	if data, err := os.ReadFile(path); err == nil && len(data) > 0 {
		if err := json.Unmarshal(data, &s.state); err != nil {
			return nil, err
		}
	}
	if err := s.flushLocked(); err != nil {
		return nil, err
	}
	return s, nil
}

// TRACEABILITY_MAPPING: REQ-002
func (s *Store) GetDefaultWorkspace() Workspace {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state.Workspace
}

// TRACEABILITY_MAPPING: REQ-003
func (s *Store) SaveSettings(settings WorkspaceSettings) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	settings.WorkspaceID = s.state.Workspace.ID
	if settings.GitHubToken == "***" {
		settings.GitHubToken = s.state.Settings.GitHubToken
	}
	if settings.LLMAPIKey == "***" {
		settings.LLMAPIKey = s.state.Settings.LLMAPIKey
	}
	s.state.Settings = settings
	return s.flushLocked()
}

// TRACEABILITY_MAPPING: REQ-003
func (s *Store) GetSettingsRaw() WorkspaceSettings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state.Settings
}

// TRACEABILITY_MAPPING: REQ-003
func (s *Store) GetSettingsRedacted() WorkspaceSettings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := s.state.Settings
	if out.GitHubToken != "" {
		out.GitHubToken = "***"
	}
	if out.LLMAPIKey != "" {
		out.LLMAPIKey = "***"
	}
	return out
}

// TRACEABILITY_MAPPING: REQ-004
func (s *Store) UpsertRepository(repo Repository) (Repository, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	repo.WorkspaceID = s.state.Workspace.ID
	for i, existing := range s.state.Repos {
		if (repo.ID != "" && existing.ID == repo.ID) || (repo.FullName != "" && existing.FullName == repo.FullName) {
			repo.ID = existing.ID
			if repo.LatestAudit == nil {
				repo.LatestAudit = existing.LatestAudit
			}
			s.state.Repos[i] = repo
			return repo, s.flushLocked()
		}
	}
	if repo.ID == "" {
		repo.ID = fmt.Sprintf("repo-%d", s.state.NextSeq)
		s.state.NextSeq++
	}
	s.state.Repos = append(s.state.Repos, repo)
	return repo, s.flushLocked()
}

// TRACEABILITY_MAPPING: REQ-004
func (s *Store) SaveAudit(repoID string, audit AuditRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, existing := range s.state.Repos {
		if existing.ID == repoID {
			audit.RepoID = repoID
			copyAudit := audit
			s.state.Repos[i].StandardsScore = audit.OverallScore
			s.state.Repos[i].LatestAudit = &copyAudit
			return s.flushLocked()
		}
	}
	return errors.New("repository not found")
}

// TRACEABILITY_MAPPING: REQ-004
func (s *Store) ListRepositories() []Repository {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Repository, len(s.state.Repos))
	copy(out, s.state.Repos)
	return out
}

// TRACEABILITY_MAPPING: REQ-004
func (s *Store) GetRepository(id string) (Repository, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, r := range s.state.Repos {
		if r.ID == id {
			return r, true
		}
	}
	return Repository{}, false
}

// TRACEABILITY_MAPPING: REQ-004
func (s *Store) DeleteRepository(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, r := range s.state.Repos {
		if r.ID == id {
			s.state.Repos = append(s.state.Repos[:i], s.state.Repos[i+1:]...)
			_ = s.flushLocked()
			return true
		}
	}
	return false
}
