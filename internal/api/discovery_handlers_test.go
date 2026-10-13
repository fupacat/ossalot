package api

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eric/ossalot/internal/db"
	"github.com/eric/ossalot/internal/github"
	"github.com/eric/ossalot/internal/llm"
)

// TRACEABILITY_MAPPING: REQ-024
func TestDiscoveryRoutes(t *testing.T) {
	store, err := db.OpenStore(filepath.Join(t.TempDir(), "discovery_api.db"))
	if err != nil {
		t.Fatalf("OpenStore failed: %v", err)
	}
	repo, _ := store.UpsertRepository(db.Repository{
		Owner:           "eric",
		Name:            "ossalot",
		FullName:        "eric/ossalot",
		PrimaryLanguage: "Go",
		Topics:          []string{"scorecard", "standards"},
	})

	mode := "ok"
	readmeB64 := base64.StdEncoding.EncodeToString([]byte("# OSSAlot\n## Features\nScorecard checks"))
	mux := http.NewServeMux()
	mux.HandleFunc("GET /repos/eric/ossalot", func(w http.ResponseWriter, r *http.Request) {
		if mode == "fail_inspect" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte(`{"name":"ossalot","full_name":"eric/ossalot","default_branch":"main","owner":{"login":"eric"}}`))
	})
	mux.HandleFunc("GET /repos/eric/ossalot/readme", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"content":"` + readmeB64 + `"}`))
	})
	mux.HandleFunc("GET /search/issues", func(w http.ResponseWriter, r *http.Request) {
		if mode == "fail_search" {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte(`{"items":[{"number":15,"title":"Need scorecard and standards automation","body":"Looking for scorecard tool","html_url":"https://github.com/acme/service/issues/15","repository_url":"https://api.github.com/repos/acme/service","user":{"login":"bob"},"labels":[{"name":"help wanted"}]}]}`))
	})

	upstream := httptest.NewServer(mux)
	defer upstream.Close()

	srv := NewServer(store, github.NewClient(upstream.URL, upstream.URL, upstream.Client()), llm.NewClient("", nil))
	srv.RegisterDiscoveryRoutes()

	doReq := func(method, path string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, nil)
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)
		return rec
	}

	// 1. Missing profile and missing repo checks
	if rec := doReq(http.MethodGet, "/api/v1/repos/"+repo.ID+"/profile"); rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 before profile exists, got %d", rec.Code)
	}
	if rec := doReq(http.MethodPost, "/api/v1/repos/missing/profile"); rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 on missing repo profile POST")
	}
	if rec := doReq(http.MethodPost, "/api/v1/repos/missing/discover"); rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 on missing repo discover POST")
	}
	if rec := doReq(http.MethodGet, "/api/v1/repos/missing/opportunities"); rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 on missing repo opportunities GET")
	}

	// 2. Discover auto-builds profile when profile does not exist yet
	mode = "fail_search"
	if rec := doReq(http.MethodPost, "/api/v1/repos/"+repo.ID+"/discover"); rec.Code != http.StatusBadGateway {
		t.Fatalf("expected 502 on search failure, got %d", rec.Code)
	}

	// 3. Explicit POST /profile error and success
	mode = "fail_inspect"
	if rec := doReq(http.MethodPost, "/api/v1/repos/"+repo.ID+"/profile"); rec.Code != http.StatusBadGateway {
		t.Fatalf("expected 502 on inspect failure, got %d", rec.Code)
	}
	mode = "ok"
	if rec := doReq(http.MethodPost, "/api/v1/repos/"+repo.ID+"/profile"); rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on POST /profile, got %d", rec.Code)
	}
	if rec := doReq(http.MethodGet, "/api/v1/repos/"+repo.ID+"/profile"); rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on GET /profile, got %d", rec.Code)
	}

	// 4. POST /discover and GET /opportunities
	if rec := doReq(http.MethodPost, "/api/v1/repos/"+repo.ID+"/discover"); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "gh:acme/service#15") {
		t.Fatalf("unexpected discover response: %d %s", rec.Code, rec.Body.String())
	}
	if rec := doReq(http.MethodGet, "/api/v1/repos/"+repo.ID+"/opportunities"); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "gh:acme/service#15") {
		t.Fatalf("unexpected opportunities list response: %d %s", rec.Code, rec.Body.String())
	}
}
