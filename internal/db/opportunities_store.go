package db

// TRACEABILITY_MAPPING: REQ-020
type CapabilityProfile struct {
	RepoID                 string   `json:"repo_id"`
	Summary                string   `json:"summary"`
	ProblemStatements      []string `json:"problem_statements"`
	TargetPersonas         []string `json:"target_personas"`
	EcosystemKeywords      []string `json:"ecosystem_keywords"`
	SearchQueries          []string `json:"search_queries"`
	RecommendedCommunities []string `json:"recommended_communities"`
	UpdatedAt              string   `json:"updated_at"`
}

// TRACEABILITY_MAPPING: REQ-020
type Opportunity struct {
	ID                   string `json:"id"`
	WorkspaceID          string `json:"workspace_id"`
	RepoID               string `json:"repo_id"`
	Category             string `json:"category"`
	ExternalID           string `json:"external_id"`
	Title                string `json:"title"`
	URL                  string `json:"url"`
	SourceAuthor         string `json:"source_author"`
	SourceCommunity      string `json:"source_community"`
	RelevanceScore       int    `json:"relevance_score"`
	LegitimacyReason     string `json:"legitimacy_reason"`
	RecommendedAction    string `json:"recommended_action"`
	DraftTitle           string `json:"draft_title"`
	DraftBody            string `json:"draft_body"`
	PRImplementationPlan string `json:"pr_implementation_plan"`
	CommunityRulesNotes  string `json:"community_rules_notes"`
	Status               string `json:"status"`
	SubmittedURL         string `json:"submitted_url"`
	OutcomeNotes         string `json:"outcome_notes"`
	DiscoveredAt         string `json:"discovered_at"`
}

// TRACEABILITY_MAPPING: REQ-020
func (s *Store) SaveCapabilityProfile(profile CapabilityProfile) error {
	panic("stub: REQ-020 not yet implemented")
}

// TRACEABILITY_MAPPING: REQ-020
func (s *Store) GetCapabilityProfile(repoID string) (CapabilityProfile, bool) {
	panic("stub: REQ-020 not yet implemented")
}

// TRACEABILITY_MAPPING: REQ-020
func (s *Store) UpsertOpportunity(opp Opportunity) (Opportunity, error) {
	panic("stub: REQ-020 not yet implemented")
}

// TRACEABILITY_MAPPING: REQ-020
func (s *Store) ListOpportunities(repoID string) []Opportunity {
	panic("stub: REQ-020 not yet implemented")
}

// TRACEABILITY_MAPPING: REQ-020
func (s *Store) UpdateOpportunityStatus(id, status, submittedURL, outcomeNotes string) (Opportunity, bool) {
	panic("stub: REQ-020 not yet implemented")
}
