package api

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eric/ossalot/internal/db"
	"github.com/eric/ossalot/internal/engine/metrics"
	"github.com/eric/ossalot/internal/github"
	"github.com/eric/ossalot/internal/llm"
)

// TRACEABILITY_MAPPING: REQ-018
func TestMetricsEndpoints(t *testing.T) {
	store, err := db.OpenStore(filepath.Join(t.TempDir(), "metrics_api.db"))
	if err != nil {
		t.Fatalf("OpenStore failed: %v", err)
	}
	repo, _ := store.UpsertRepository(db.Repository{
		Owner:    "octocat",
		Name:     "hello-world",
		FullName: "octocat/hello-world",
	})

	mode := "ok"
	mux := http.NewServeMux()
	mux.HandleFunc("GET /repos/octocat/hello-world", func(w http.ResponseWriter, r *http.Request) {
		if mode == "fail_repo" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte(`{"name":"hello-world","full_name":"octocat/hello-world","stargazers_count":80,"forks_count":10,"open_issues_count":3,"default_branch":"main","owner":{"login":"octocat"}}`))
	})
	mux.HandleFunc("GET /repos/octocat/hello-world/traffic/views", func(w http.ResponseWriter, r *http.Request) {
		if mode == "fail_traffic" {
			// 403 and 404 are now an empty snapshot (REQ-047); other failures still fail the collection
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte(`{"count":200,"uniques":90}`))
	})
	mux.HandleFunc("GET /repos/octocat/hello-world/traffic/clones", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"count":40,"uniques":20}`))
	})
	mux.HandleFunc("GET /repos/octocat/hello-world/traffic/popular/referrers", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"referrer":"news.ycombinator.com","count":30,"uniques":25}]`))
	})
	mux.HandleFunc("GET /downloads/npm/hello-world", func(w http.ResponseWriter, r *http.Request) {
		if mode == "fail_registry" {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		_, _ = w.Write([]byte(`{"downloads":500}`))
	})

	upstream := httptest.NewServer(mux)
	defer upstream.Close()

	ghClient := github.NewClient(upstream.URL, upstream.URL, upstream.Client())
	regClient := metrics.NewRegistryClient(upstream.URL, upstream.Client())
	srv := NewServer(store, ghClient, llm.NewClient("", nil))
	srv.RegisterMetricsRoutes(regClient)

	doReq := func(method, path, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)
		return rec
	}

	// Missing repo checks
	if rec := doReq(http.MethodPost, "/api/v1/repos/missing/metrics/collect", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing repo collect, got %d", rec.Code)
	}
	if rec := doReq(http.MethodGet, "/api/v1/repos/missing/metrics", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing repo list, got %d", rec.Code)
	}

	// Invalid JSON body
	if rec := doReq(http.MethodPost, "/api/v1/repos/"+repo.ID+"/metrics/collect", "{bad"); rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on bad JSON body, got %d", rec.Code)
	}

	// Upstream failure branches
	mode = "fail_repo"
	if rec := doReq(http.MethodPost, "/api/v1/repos/"+repo.ID+"/metrics/collect", ""); rec.Code != http.StatusBadGateway {
		t.Fatalf("expected 502 on repo inspect error, got %d", rec.Code)
	}
	mode = "fail_traffic"
	if rec := doReq(http.MethodPost, "/api/v1/repos/"+repo.ID+"/metrics/collect", ""); rec.Code != http.StatusBadGateway {
		t.Fatalf("expected 502 on traffic error, got %d", rec.Code)
	}
	mode = "fail_registry"
	if rec := doReq(http.MethodPost, "/api/v1/repos/"+repo.ID+"/metrics/collect", `{"packages":{"npm":"hello-world"}}`); rec.Code != http.StatusBadGateway {
		t.Fatalf("expected 502 on registry error, got %d", rec.Code)
	}

	// Success with default date (empty body) and explicit date + packages
	mode = "ok"
	if rec := doReq(http.MethodPost, "/api/v1/repos/"+repo.ID+"/metrics/collect", ""); rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on collect with default date, got %d", rec.Code)
	}
	if rec := doReq(http.MethodPost, "/api/v1/repos/"+repo.ID+"/metrics/collect", `{"snapshot_date":"2026-10-12","packages":{"npm":"hello-world"}}`); rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on collect with explicit payload, got %d: %s", rec.Code, rec.Body.String())
	}

	// List snapshots
	recList := doReq(http.MethodGet, "/api/v1/repos/"+repo.ID+"/metrics", "")
	if recList.Code != http.StatusOK || !strings.Contains(recList.Body.String(), "news.ycombinator.com") {
		t.Fatalf("unexpected metrics list response: %d %s", recList.Code, recList.Body.String())
	}
}
