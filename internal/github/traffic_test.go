package github

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TRACEABILITY_MAPPING: REQ-015
func TestFetchTrafficSnapshot(t *testing.T) {
	step := "all_ok"
	mux := http.NewServeMux()
	mux.HandleFunc("GET /repos/octocat/hello-world/traffic/views", func(w http.ResponseWriter, r *http.Request) {
		if step == "fail_views" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte(`{"count":120,"uniques":45}`))
	})
	mux.HandleFunc("GET /repos/octocat/hello-world/traffic/clones", func(w http.ResponseWriter, r *http.Request) {
		if step == "fail_clones" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte(`{"count":35,"uniques":18}`))
	})
	mux.HandleFunc("GET /repos/octocat/hello-world/traffic/popular/referrers", func(w http.ResponseWriter, r *http.Request) {
		if step == "fail_referrers" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte(`[{"referrer":"news.ycombinator.com","count":50,"uniques":40},{"referrer":"reddit.com","count":25,"uniques":20}]`))
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := NewClient(srv.URL, srv.URL, srv.Client())
	ctx := context.Background()

	snap, err := client.FetchTrafficSnapshot(ctx, "ghp_tok", "octocat", "hello-world")
	if err != nil {
		t.Fatalf("FetchTrafficSnapshot failed: %v", err)
	}
	if snap.ViewsTotal != 120 || snap.ViewsUnique != 45 || snap.ClonesTotal != 35 || snap.ClonesUnique != 18 || len(snap.Referrers) != 2 {
		t.Fatalf("unexpected traffic snapshot: %+v", snap)
	}

	step = "fail_views"
	if _, err := client.FetchTrafficSnapshot(ctx, "ghp_tok", "octocat", "hello-world"); err == nil {
		t.Fatalf("expected error on views failure")
	}
	step = "fail_clones"
	if _, err := client.FetchTrafficSnapshot(ctx, "ghp_tok", "octocat", "hello-world"); err == nil {
		t.Fatalf("expected error on clones failure")
	}
	step = "fail_referrers"
	if _, err := client.FetchTrafficSnapshot(ctx, "ghp_tok", "octocat", "hello-world"); err == nil {
		t.Fatalf("expected error on referrers failure")
	}
}
