package discovery

import (
	"testing"

	"github.com/eric/ossalot/internal/db"
	"github.com/eric/ossalot/internal/github"
)

// TRACEABILITY_MAPPING: REQ-023
func TestScoreIgnoresUnrelatedLabels(t *testing.T) {
	repo := db.Repository{ID: "repo-1", Name: "ossalot", FullName: "eric/ossalot"}
	profile := db.CapabilityProfile{RepoID: "repo-1", EcosystemKeywords: []string{"scorecard", "standards"}}
	candidates := []github.IssueCandidate{
		{
			TargetRepoFullName: "octocat/other",
			IssueNumber:        7,
			Title:              "scorecard standards gap",
			HTMLURL:            "https://github.com/octocat/other/issues/7",
			Labels:             []string{"bug"},
		},
	}

	got := ScoreAndFilterCandidates(repo, profile, candidates)
	// 45 base + 20 per matched keyword; the "bug" label adds nothing
	if len(got) != 1 || got[0].RelevanceScore != 85 {
		t.Fatalf("expected one opportunity scored 85 with no label bonus, got %+v", got)
	}
}
