package community

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/eric/ossalot/internal/db"
)

// TRACEABILITY_MAPPING: REQ-026
func TestScanCommunities(t *testing.T) {
	defScanner := NewScanner("", nil)
	if defScanner.BaseURL != "https://hn.algolia.com" {
		t.Fatalf("unexpected default scanner base URL: %s", defScanner.BaseURL)
	}

	failPlatform := ""
	mux := http.NewServeMux()
	for _, p := range communityPlatforms {
		platform := p
		mux.HandleFunc("GET /feeds/"+platform, func(w http.ResponseWriter, r *http.Request) {
			if failPlatform == platform {
				w.WriteHeader(http.StatusBadGateway)
				return
			}
			scoreStr := "88"
			if platform == "lobsters" {
				scoreStr = "0"
			}
			_, _ = w.Write([]byte(`[{"id":"101","title":"How to audit OSS standards?","url":"https://example.com/` + platform + `/101","author":"dev1","community":"` + platform + `","score":` + scoreStr + `}]`))
		})
	}
	srv := httptest.NewServer(mux)
	defer srv.Close()

	scanner := NewScanner(srv.URL, srv.Client())
	ctx := context.Background()

	repo := db.Repository{
		ID:       "repo-1",
		Name:     "ossalot",
		FullName: "eric/ossalot",
		HTMLURL:  "https://github.com/eric/ossalot",
	}
	prof := db.CapabilityProfile{
		RepoID:            "repo-1",
		EcosystemKeywords: []string{"scorecard"},
	}

	opps, err := scanner.ScanCommunities(ctx, repo, prof)
	if err != nil || len(opps) != 5 {
		t.Fatalf("ScanCommunities failed: err=%v len=%d", err, len(opps))
	}

	// Also test with empty EcosystemKeywords fallback to repo.Name
	oppsFallback, err := scanner.ScanCommunities(ctx, repo, db.CapabilityProfile{})
	if err != nil || len(oppsFallback) != 5 {
		t.Fatalf("ScanCommunities fallback failed: err=%v len=%d", err, len(oppsFallback))
	}

	// Non-200 error branch
	failPlatform = "reddit"
	if _, err := scanner.ScanCommunities(ctx, repo, prof); err == nil {
		t.Fatalf("expected error when reddit feed returns 502")
	}

	// Network error branch
	badScanner := NewScanner("http://127.0.0.1:1", srv.Client())
	if _, err := badScanner.ScanCommunities(ctx, repo, prof); err == nil {
		t.Fatalf("expected network error on closed port")
	}
}

// TRACEABILITY_MAPPING: REQ-027
func TestGenerateMarketingPlaybooks(t *testing.T) {
	playbooks := GenerateMarketingPlaybooks(
		db.Repository{
			ID:              "repo-1",
			Owner:           "eric",
			Name:            "ossalot",
			FullName:        "eric/ossalot",
			HTMLURL:         "https://github.com/eric/ossalot",
			Description:     "Fallback description",
			PrimaryLanguage: "",
		},
		db.CapabilityProfile{},
	)
	if len(playbooks) != 4 {
		t.Fatalf("expected 4 playbooks, got %d", len(playbooks))
	}
	for _, pb := range playbooks {
		if pb.Category != "marketing_playbook" || pb.CommunityRulesNotes == "" || pb.DraftBody == "" {
			t.Fatalf("incomplete playbook: %+v", pb)
		}
	}
}
