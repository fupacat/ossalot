package github

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TRACEABILITY_MAPPING: REQ-022
func TestSearchIssues(t *testing.T) {
	if got := extractRepoFullNameFromURL("", "invalid"); got != "" {
		t.Fatalf("expected empty fallback on malformed URLs, got %q", got)
	}
	if got := extractRepoFullNameFromURL("", "https://github.com/acme/webapp/issues/12"); got != "acme/webapp" {
		t.Fatalf("expected acme/webapp from htmlURL fallback, got %q", got)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /search/issues", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("q") == "fail" {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte(`{"items":[{"number":42,"title":"Need OpenSSF scorecard check","body":"Looking for automated standards audit","html_url":"https://github.com/acme/webapp/issues/42","repository_url":"https://api.github.com/repos/acme/webapp","user":{"login":"devuser"},"labels":[{"name":"help wanted"}]}]}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := NewClient(srv.URL, srv.URL, srv.Client())
	ctx := context.Background()

	items, err := client.SearchIssues(ctx, "ghp_tok", "scorecard")
	if err != nil || len(items) != 1 || items[0].TargetRepoFullName != "acme/webapp" || items[0].IssueNumber != 42 || len(items[0].Labels) != 1 {
		t.Fatalf("SearchIssues failed: %v, %+v", err, items)
	}

	if _, err := client.SearchIssues(ctx, "ghp_tok", "fail"); err == nil {
		t.Fatalf("expected error on 503 search response")
	}
}
