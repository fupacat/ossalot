package db

import "sync"

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
type Store struct {
	mu   sync.RWMutex
	path string
}

// TRACEABILITY_MAPPING: REQ-002
func OpenStore(path string) (*Store, error) {
	panic("stub: REQ-002 not yet implemented")
}

// TRACEABILITY_MAPPING: REQ-002
func (s *Store) GetDefaultWorkspace() Workspace {
	panic("stub: REQ-002 not yet implemented")
}

// TRACEABILITY_MAPPING: REQ-003
func (s *Store) SaveSettings(settings WorkspaceSettings) error {
	panic("stub: REQ-003 not yet implemented")
}

// TRACEABILITY_MAPPING: REQ-003
func (s *Store) GetSettingsRaw() WorkspaceSettings {
	panic("stub: REQ-003 not yet implemented")
}

// TRACEABILITY_MAPPING: REQ-003
func (s *Store) GetSettingsRedacted() WorkspaceSettings {
	panic("stub: REQ-003 not yet implemented")
}

// TRACEABILITY_MAPPING: REQ-004
func (s *Store) UpsertRepository(repo Repository) (Repository, error) {
	panic("stub: REQ-004 not yet implemented")
}

// TRACEABILITY_MAPPING: REQ-004
func (s *Store) SaveAudit(repoID string, audit AuditRecord) error {
	panic("stub: REQ-004 not yet implemented")
}

// TRACEABILITY_MAPPING: REQ-004
func (s *Store) ListRepositories() []Repository {
	panic("stub: REQ-004 not yet implemented")
}

// TRACEABILITY_MAPPING: REQ-004
func (s *Store) GetRepository(id string) (Repository, bool) {
	panic("stub: REQ-004 not yet implemented")
}

// TRACEABILITY_MAPPING: REQ-004
func (s *Store) DeleteRepository(id string) bool {
	panic("stub: REQ-004 not yet implemented")
}
