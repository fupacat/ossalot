package community

import (
	"strings"
	"testing"

	"github.com/eric/ossalot/internal/db"
)

// TRACEABILITY_MAPPING: REQ-027
func TestMarketingPlaybooksUseRepositoryLanguageAndSummary(t *testing.T) {
	playbooks := GenerateMarketingPlaybooks(
		db.Repository{
			ID:              "repo-1",
			Owner:           "eric",
			Name:            "ossalot",
			FullName:        "eric/ossalot",
			HTMLURL:         "https://github.com/eric/ossalot",
			Description:     "Repository description",
			PrimaryLanguage: "Go",
		},
		db.CapabilityProfile{Summary: "profile summary"},
	)

	var showHN *db.Opportunity
	for i := range playbooks {
		if strings.Contains(playbooks[i].DraftTitle, "Show HN") {
			showHN = &playbooks[i]
		}
	}
	if showHN == nil {
		t.Fatalf("expected a Show HN playbook, got %+v", playbooks)
	}
	// The profile summary is used when set, and the repository language is named in the draft
	if !strings.Contains(showHN.DraftTitle, "profile summary") {
		t.Fatalf("Show HN title should use the profile summary: %q", showHN.DraftTitle)
	}
	if !strings.Contains(showHN.DraftBody, " in Go ") {
		t.Fatalf("Show HN body should name the repository language: %q", showHN.DraftBody)
	}
}
