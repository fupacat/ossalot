package discovery

import (
	"testing"

	"github.com/eric/ossalot/internal/db"
	"github.com/eric/ossalot/internal/github"
)

// TRACEABILITY_MAPPING: REQ-023
func TestScoreAndFilterCandidates(t *testing.T) {
	repo := db.Repository{
		ID:       "repo-1",
		Name:     "ossalot",
		FullName: "eric/ossalot",
	}
	profile := db.CapabilityProfile{
		RepoID:            "repo-1",
		EcosystemKeywords: []string{"scorecard", "standards", "remediation"},
	}

	candidates := []github.IssueCandidate{
		{
			// Self-repo issue: must be excluded
			TargetRepoFullName: "eric/ossalot",
			IssueNumber:        1,
			Title:              "scorecard standards remediation",
		},
		{
			// Low-relevance issue (< 70): must be excluded
			TargetRepoFullName: "acme/unrelated",
			IssueNumber:        2,
			Title:              "Fix CSS button alignment",
		},
		{
			// High-relevance external issue (exceeds 100 cap -> clamped to 100)
			TargetRepoFullName: "acme/platform",
			IssueNumber:        42,
			Title:              "Automate OpenSSF scorecard and standards remediation",
			Body:               "We need scorecard and standards checks.",
			HTMLURL:            "https://github.com/acme/platform/issues/42",
			Author:             "alice",
			Labels:             []string{"help wanted", "enhancement"},
		},
	}

	retained := ScoreAndFilterCandidates(repo, profile, candidates)
	if len(retained) != 1 {
		t.Fatalf("expected 1 retained opportunity, got %d", len(retained))
	}
	if retained[0].RelevanceScore != 100 || retained[0].ExternalID != "gh:acme/platform#42" || retained[0].PRImplementationPlan == "" {
		t.Fatalf("unexpected retained opportunity: %+v", retained[0])
	}
}
