package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eric/ossalot/internal/db"
	"github.com/eric/ossalot/internal/engine/community"
	"github.com/eric/ossalot/internal/engine/notifier"
	"github.com/eric/ossalot/internal/github"
	"github.com/eric/ossalot/internal/llm"
	"github.com/eric/ossalot/internal/scheduler"
)

// TRACEABILITY_MAPPING: REQ-030
func TestCommunityAndInboxRoutes(t *testing.T) {
	store, err := db.OpenStore(filepath.Join(t.TempDir(), "comm_api.db"))
	if err != nil {
		t.Fatalf("OpenStore failed: %v", err)
	}
	repo, _ := store.UpsertRepository(db.Repository{
		Owner:           "eric",
		Name:            "ossalot",
		FullName:        "eric/ossalot",
		HTMLURL:         "https://github.com/eric/ossalot",
		PrimaryLanguage: "Go",
	})

	failScan := false
	mux := http.NewServeMux()
	for _, p := range []string{"hackernews", "reddit", "stackoverflow", "devto", "lobsters"} {
		platform := p
		mux.HandleFunc("GET /feeds/"+platform, func(w http.ResponseWriter, r *http.Request) {
			if failScan {
				w.WriteHeader(http.StatusBadGateway)
				return
			}
			_, _ = w.Write([]byte(`[{"id":"9","title":"Best OSS standards tool?","url":"https://example.com/` + platform + `/9","author":"alice","community":"` + platform + `","score":90}]`))
		})
	}
	upstream := httptest.NewServer(mux)
	defer upstream.Close()

	scanner := community.NewScanner(upstream.URL, upstream.Client())
	notif := notifier.NewNotifier(upstream.Client())
	runner := scheduler.NewRunner()
	runner.RegisterJob("daily_sync", func(ctx context.Context) error { return nil })

	srv := NewServer(store, github.NewClient(upstream.URL, upstream.URL, upstream.Client()), llm.NewClient("", nil))
	srv.RegisterCommunityAndInboxRoutes(scanner, notif, runner)

	doReq := func(method, path, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)
		return rec
	}

	// 1. Missing repo checks
	if rec := doReq(http.MethodPost, "/api/v1/repos/missing/community/scan", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 on missing repo community scan")
	}
	if rec := doReq(http.MethodPost, "/api/v1/repos/missing/playbooks", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 on missing repo playbooks")
	}

	// 2. Community scan upstream failure then success
	failScan = true
	if rec := doReq(http.MethodPost, "/api/v1/repos/"+repo.ID+"/community/scan", ""); rec.Code != http.StatusBadGateway {
		t.Fatalf("expected 502 on community scan error, got %d", rec.Code)
	}
	failScan = false
	if rec := doReq(http.MethodPost, "/api/v1/repos/"+repo.ID+"/community/scan", ""); rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on community scan, got %d", rec.Code)
	}

	// 3. Playbooks generation
	if rec := doReq(http.MethodPost, "/api/v1/repos/"+repo.ID+"/playbooks", ""); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "marketing_playbook") {
		t.Fatalf("unexpected playbooks response: %d %s", rec.Code, rec.Body.String())
	}

	// 4. Global opportunities list & status update
	if rec := doReq(http.MethodGet, "/api/v1/opportunities", ""); rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on GET /opportunities")
	}
	if rec := doReq(http.MethodPut, "/api/v1/opportunities/opp-1/status", `{}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on empty status")
	}
	if rec := doReq(http.MethodPut, "/api/v1/opportunities/missing/status", `{"status":"saved"}`); rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 on missing opportunity")
	}
	if rec := doReq(http.MethodPut, "/api/v1/opportunities/opp-1/status", `{"status":"submitted","submitted_url":"https://news.ycombinator.com/item?id=1","outcome_notes":"Front page"}`); rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on status update, got %d", rec.Code)
	}

	// 5. Scheduler jobs trigger and list
	if rec := doReq(http.MethodPost, "/api/v1/jobs/unknown/run", ""); rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on unknown job trigger")
	}
	if rec := doReq(http.MethodPost, "/api/v1/jobs/daily_sync/run", ""); rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on daily_sync trigger, got %d", rec.Code)
	}
	if rec := doReq(http.MethodGet, "/api/v1/jobs", ""); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "daily_sync") {
		t.Fatalf("unexpected jobs list response: %d %s", rec.Code, rec.Body.String())
	}
}
