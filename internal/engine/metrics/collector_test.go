package metrics

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/eric/ossalot/internal/github"
)

// TRACEABILITY_MAPPING: REQ-016
func TestFetchDownloads(t *testing.T) {
	defClient := NewRegistryClient("", nil)
	if defClient.BaseURL != "https://api.npmjs.org" {
		t.Fatalf("unexpected default base URL: %s", defClient.BaseURL)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /downloads/npm/ossalot-js", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"downloads":1500}`))
	})
	mux.HandleFunc("GET /downloads/pypi/ossalot-py", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"downloads":800}`))
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	rc := NewRegistryClient(srv.URL, srv.Client())
	ctx := context.Background()

	counts, err := rc.FetchDownloads(ctx, map[string]string{
		"npm":  "ossalot-js",
		"pypi": "ossalot-py",
	})
	if err != nil || counts["npm"] != 1500 || counts["pypi"] != 800 {
		t.Fatalf("FetchDownloads failed: %v, %+v", err, counts)
	}

	// Non-200 error branch
	if _, err := rc.FetchDownloads(ctx, map[string]string{"crates": "missing"}); err == nil {
		t.Fatalf("expected error on 404 registry lookup")
	}

	// Network failure branch
	badClient := NewRegistryClient("http://127.0.0.1:1", srv.Client())
	if _, err := badClient.FetchDownloads(ctx, map[string]string{"npm": "ossalot-js"}); err == nil {
		t.Fatalf("expected network error on closed port")
	}
}

// TRACEABILITY_MAPPING: REQ-017
func TestBuildDailySnapshot(t *testing.T) {
	highSnap := BuildDailySnapshot(
		"repo-1",
		"2026-10-12",
		github.RepoInspection{Stars: 500, Forks: 50, OpenIssues: 4},
		github.TrafficSnapshot{
			ViewsTotal:   300,
			ViewsUnique:  100,
			ClonesTotal:  40,
			ClonesUnique: 20,
			Referrers: []github.ReferrerItem{
				{Referrer: "news.ycombinator.com", Count: 50, Uniques: 40},
			},
		},
		map[string]int{"npm": 2000},
	)
	if highSnap.AdoptionScore != 100 || len(highSnap.Referrers) != 1 || highSnap.RegistryDownloads["npm"] != 2000 {
		t.Fatalf("unexpected high snapshot: %+v", highSnap)
	}

	negSnap := BuildDailySnapshot(
		"repo-1",
		"2026-10-12",
		github.RepoInspection{Stars: -50},
		github.TrafficSnapshot{},
		nil,
	)
	if negSnap.AdoptionScore != 0 {
		t.Fatalf("expected 0 floor on adoption score, got %d", negSnap.AdoptionScore)
	}
}
