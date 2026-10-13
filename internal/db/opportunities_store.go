package db

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

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
type opportunitiesState struct {
	Profiles      []CapabilityProfile `json:"profiles"`
	Opportunities []Opportunity       `json:"opportunities"`
	NextOppSeq    int                 `json:"next_opp_seq"`
}

// TRACEABILITY_MAPPING: REQ-020
func (s *Store) opportunitiesFilePath() string {
	return s.path + ".opportunities.json"
}

// TRACEABILITY_MAPPING: REQ-020
func (s *Store) loadOpportunitiesStateLocked() opportunitiesState {
	data, err := os.ReadFile(s.opportunitiesFilePath())
	if err != nil || len(data) == 0 {
		return opportunitiesState{
			Profiles:      []CapabilityProfile{},
			Opportunities: []Opportunity{},
			NextOppSeq:    1,
		}
	}
	var st opportunitiesState
	_ = json.Unmarshal(data, &st)
	return st
}

// TRACEABILITY_MAPPING: REQ-020
func (s *Store) saveOpportunitiesStateLocked(st opportunitiesState) error {
	raw, _ := json.MarshalIndent(st, "", "  ")
	return os.WriteFile(s.opportunitiesFilePath(), raw, 0o600)
}

// TRACEABILITY_MAPPING: REQ-020
func (s *Store) SaveCapabilityProfile(profile CapabilityProfile) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if profile.RepoID == "" {
		return errors.New("repo_id is required")
	}
	st := s.loadOpportunitiesStateLocked()
	updated := false
	for i, existing := range st.Profiles {
		if existing.RepoID == profile.RepoID {
			st.Profiles[i] = profile
			updated = true
			break
		}
	}
	if !updated {
		st.Profiles = append(st.Profiles, profile)
	}
	return s.saveOpportunitiesStateLocked(st)
}

// TRACEABILITY_MAPPING: REQ-020
func (s *Store) GetCapabilityProfile(repoID string) (CapabilityProfile, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	st := s.loadOpportunitiesStateLocked()
	for _, p := range st.Profiles {
		if p.RepoID == repoID {
			return p, true
		}
	}
	return CapabilityProfile{}, false
}

// TRACEABILITY_MAPPING: REQ-020
func (s *Store) UpsertOpportunity(opp Opportunity) (Opportunity, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if opp.ExternalID == "" {
		return Opportunity{}, errors.New("external_id is required")
	}
	st := s.loadOpportunitiesStateLocked()
	opp.WorkspaceID = s.state.Workspace.ID
	if opp.Status == "" {
		opp.Status = "new"
	}
	for i, existing := range st.Opportunities {
		if existing.ExternalID == opp.ExternalID {
			opp.ID = existing.ID
			st.Opportunities[i] = opp
			return opp, s.saveOpportunitiesStateLocked(st)
		}
	}
	opp.ID = fmt.Sprintf("opp-%d", st.NextOppSeq)
	st.NextOppSeq++
	st.Opportunities = append(st.Opportunities, opp)
	return opp, s.saveOpportunitiesStateLocked(st)
}

// TRACEABILITY_MAPPING: REQ-020
func (s *Store) ListOpportunities(repoID string) []Opportunity {
	s.mu.RLock()
	defer s.mu.RUnlock()
	st := s.loadOpportunitiesStateLocked()
	var out []Opportunity
	for _, item := range st.Opportunities {
		if repoID == "" || item.RepoID == repoID {
			out = append(out, item)
		}
	}
	return out
}

// TRACEABILITY_MAPPING: REQ-020
func (s *Store) UpdateOpportunityStatus(id, status, submittedURL, outcomeNotes string) (Opportunity, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := s.loadOpportunitiesStateLocked()
	for i, item := range st.Opportunities {
		if item.ID == id {
			st.Opportunities[i].Status = status
			st.Opportunities[i].SubmittedURL = submittedURL
			st.Opportunities[i].OutcomeNotes = outcomeNotes
			_ = s.saveOpportunitiesStateLocked(st)
			return st.Opportunities[i], true
		}
	}
	return Opportunity{}, false
}
