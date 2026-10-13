package db

import (
	"path/filepath"
	"testing"
)

// TRACEABILITY_MAPPING: REQ-020
func TestCapabilityProfileAndOpportunityStore(t *testing.T) {
	store, err := OpenStore(filepath.Join(t.TempDir(), "opps.db"))
	if err != nil {
		t.Fatalf("OpenStore failed: %v", err)
	}

	// Missing repo_id rejected
	if err := store.SaveCapabilityProfile(CapabilityProfile{}); err == nil {
		t.Fatalf("expected error on empty repo_id")
	}
	if _, ok := store.GetCapabilityProfile("repo-1"); ok {
		t.Fatalf("expected false when profile does not exist")
	}

	// Insert and update profile
	prof := CapabilityProfile{
		RepoID:            "repo-1",
		Summary:           "Initial summary",
		ProblemStatements: []string{"rate limiting"},
		SearchQueries:     []string{"label:help-wanted rate limit"},
	}
	if err := store.SaveCapabilityProfile(prof); err != nil {
		t.Fatalf("SaveCapabilityProfile failed: %v", err)
	}
	prof.Summary = "Updated summary"
	if err := store.SaveCapabilityProfile(prof); err != nil {
		t.Fatalf("SaveCapabilityProfile update failed: %v", err)
	}
	gotProf, ok := store.GetCapabilityProfile("repo-1")
	if !ok || gotProf.Summary != "Updated summary" {
		t.Fatalf("unexpected profile: ok=%v %+v", ok, gotProf)
	}

	// UpsertOpportunity validation and deduplication by external_id
	if _, err := store.UpsertOpportunity(Opportunity{}); err == nil {
		t.Fatalf("expected error when external_id is empty")
	}

	opp1, err := store.UpsertOpportunity(Opportunity{
		RepoID:         "repo-1",
		Category:       "github_issue",
		ExternalID:     "gh:acme/webapp#42",
		Title:          "Need token bucket limiter",
		RelevanceScore: 88,
	})
	if err != nil || opp1.ID != "opp-1" || opp1.Status != "new" {
		t.Fatalf("unexpected opp1: %+v, err=%v", opp1, err)
	}

	// Duplicate external_id updates in place
	opp2, err := store.UpsertOpportunity(Opportunity{
		RepoID:         "repo-1",
		Category:       "github_issue",
		ExternalID:     "gh:acme/webapp#42",
		Title:          "Need token bucket limiter (updated)",
		RelevanceScore: 92,
		Status:         "saved",
	})
	if err != nil || opp2.ID != opp1.ID || opp2.RelevanceScore != 92 {
		t.Fatalf("expected deduplicated update, got %+v err=%v", opp2, err)
	}

	list := store.ListOpportunities("repo-1")
	if len(list) != 1 || list[0].Title != "Need token bucket limiter (updated)" {
		t.Fatalf("unexpected opportunities list: %+v", list)
	}
	allList := store.ListOpportunities("")
	if len(allList) != 1 {
		t.Fatalf("expected 1 opportunity in global list, got %d", len(allList))
	}

	updated, ok := store.UpdateOpportunityStatus(opp1.ID, "submitted", "https://github.com/acme/webapp/pull/99", "Merged by maintainer")
	if !ok || updated.Status != "submitted" || updated.SubmittedURL != "https://github.com/acme/webapp/pull/99" {
		t.Fatalf("UpdateOpportunityStatus failed: ok=%v %+v", ok, updated)
	}
	if _, ok := store.UpdateOpportunityStatus("missing-id", "dismissed", "", ""); ok {
		t.Fatalf("expected false updating missing opportunity")
	}
}
