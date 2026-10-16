package api

import (
	"bytes"
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

// TRACEABILITY_MAPPING: REQ-011, REQ-055
func TestAPIServerEndpoints(t *testing.T) {
	store, err := db.OpenStore(filepath.Join(t.TempDir(), "api.db"))
	if err != nil {
		t.Fatalf("OpenStore failed: %v", err)
	}

	readmeB64 := base64.StdEncoding.EncodeToString([]byte("# Hello\n## Install\nRun\n## Usage\nUse"))
	ghMux := http.NewServeMux()
	failInspect := false
	ghMux.HandleFunc("GET /repos/octocat/hello-world", func(w http.ResponseWriter, r *http.Request) {
		if failInspect {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte(`{"name":"hello-world","full_name":"octocat/hello-world","html_url":"https://github.com/octocat/hello-world","description":"Demo","language":"Go","default_branch":"main","topics":["go"],"owner":{"login":"octocat"},"license":{"spdx_id":"MIT"}}`))
	})
	ghMux.HandleFunc("GET /repos/octocat/hello-world/git/trees/main", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"tree":[{"path":"README.md"}]}`))
	})
	ghMux.HandleFunc("GET /repos/octocat/hello-world/readme", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"content":"` + readmeB64 + `"}`))
	})
	ghMux.HandleFunc("GET /repos/octocat/hello-world/actions/runs", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"workflow_runs":[]}`))
	})
	ghMux.HandleFunc("GET /repos/octocat/hello-world/releases", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[]`))
	})
	ghMux.HandleFunc("GET /projects/github.com/octocat/hello-world", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"score":8.0}`))
	})
	failSync := false
	ghMux.HandleFunc("GET /user/repos", func(w http.ResponseWriter, r *http.Request) {
		if failSync {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(`[{"name":"synced-repo","full_name":"octocat/synced-repo","html_url":"https://github.com/octocat/synced-repo","language":"Go","owner":{"login":"octocat"}}]`))
	})
	ghMux.HandleFunc("GET /repos/octocat/hello-world/git/ref/heads/main", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"object":{"sha":"main-sha"}}`))
	})
	ghMux.HandleFunc("POST /repos/octocat/hello-world/git/refs", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"ref":"refs/heads/ossalot/remediate-contributing_guide"}`))
	})
	ghMux.HandleFunc("GET /repos/octocat/hello-world/contents/CONTRIBUTING.md", http.NotFound)
	ghMux.HandleFunc("PUT /repos/octocat/hello-world/contents/CONTRIBUTING.md", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"content":{"path":"CONTRIBUTING.md"}}`))
	})
	ghMux.HandleFunc("GET /repos/octocat/hello-world/contents/.github/workflows/sonar.yml", http.NotFound)
	ghMux.HandleFunc("PUT /repos/octocat/hello-world/contents/.github/workflows/sonar.yml", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"content":{"path":".github/workflows/sonar.yml"}}`))
	})
	ghMux.HandleFunc("POST /repos/octocat/hello-world/pulls", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"html_url":"https://github.com/octocat/hello-world/pull/10"}`))
	})

	ghSrv := httptest.NewServer(ghMux)
	defer ghSrv.Close()

	ghClient := github.NewClient(ghSrv.URL, ghSrv.URL, ghSrv.Client())
	llmClient := llm.NewClient("", nil)
	srv := NewServer(store, ghClient, llmClient)

	doReq := func(method, path, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)
		return rec
	}

	// 1. Health
	if rec := doReq(http.MethodGet, "/api/v1/health", ""); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"status":"ok"`) {
		t.Fatalf("health failed: %d %s", rec.Code, rec.Body.String())
	}

	// 2. Settings GET/PUT
	if rec := doReq(http.MethodGet, "/api/v1/settings", ""); rec.Code != http.StatusOK {
		t.Fatalf("get settings failed: %d", rec.Code)
	}
	if rec := doReq(http.MethodPut, "/api/v1/settings", "{bad"); rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on bad settings JSON, got %d", rec.Code)
	}
	if rec := doReq(http.MethodPut, "/api/v1/settings", `{"github_token":"ghp_1","llm_provider":"gemini"}`); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"***"`) {
		t.Fatalf("put settings failed: %d %s", rec.Code, rec.Body.String())
	}

	// 3. Add Repo (bad JSON, bad URL, upstream error, then success)
	if rec := doReq(http.MethodPost, "/api/v1/repos", "{bad"); rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on bad repo JSON")
	}
	if rec := doReq(http.MethodPost, "/api/v1/repos", `{"url":"invalid"}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on invalid repo URL")
	}
	failInspect = true
	if rec := doReq(http.MethodPost, "/api/v1/repos", `{"url":"octocat/hello-world","is_owned":true}`); rec.Code != http.StatusBadGateway {
		t.Fatalf("expected 502 on upstream failure, got %d", rec.Code)
	}
	failInspect = false
	if rec := doReq(http.MethodPost, "/api/v1/repos", `{"url":"octocat/hello-world","is_owned":true}`); rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 on add repo, got %d: %s", rec.Code, rec.Body.String())
	}

	// 4. List & Get Repo
	if rec := doReq(http.MethodGet, "/api/v1/repos", ""); rec.Code != http.StatusOK {
		t.Fatalf("list repos failed: %d", rec.Code)
	}
	if rec := doReq(http.MethodGet, "/api/v1/repos/repo-1", ""); rec.Code != http.StatusOK {
		t.Fatalf("get repo-1 failed: %d", rec.Code)
	}
	if rec := doReq(http.MethodGet, "/api/v1/repos/missing", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 on missing repo")
	}

	// 5. Audit Repo (missing, upstream error, success)
	if rec := doReq(http.MethodPost, "/api/v1/repos/missing/audit", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 on missing repo audit")
	}
	failInspect = true
	if rec := doReq(http.MethodPost, "/api/v1/repos/repo-1/audit", ""); rec.Code != http.StatusBadGateway {
		t.Fatalf("expected 502 on upstream audit error")
	}
	failInspect = false
	if rec := doReq(http.MethodPost, "/api/v1/repos/repo-1/audit", ""); rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on repo audit, got %d", rec.Code)
	}

	// 6. Remediate Repo (missing, bad JSON, bad check_key, draft only, create PR success, create PR forbidden on unowned)
	if rec := doReq(http.MethodPost, "/api/v1/repos/missing/remediate", `{}`); rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 on missing repo remediate")
	}
	if rec := doReq(http.MethodPost, "/api/v1/repos/repo-1/remediate", "{bad"); rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on bad remediate JSON")
	}
	if rec := doReq(http.MethodPost, "/api/v1/repos/repo-1/remediate", `{"check_key":"bad_key"}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on unsupported check_key")
	}
	if rec := doReq(http.MethodPost, "/api/v1/repos/repo-1/remediate", `{"check_key":"contributing_guide","create_pr":false}`); rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on draft generation, got %d", rec.Code)
	}
	if rec := doReq(http.MethodPost, "/api/v1/repos/repo-1/remediate", `{"check_key":"contributing_guide","create_pr":true}`); rec.Code != http.StatusOK || !bytes.Contains(rec.Body.Bytes(), []byte("pull/10")) {
		t.Fatalf("expected 200 on PR creation, got %d %s", rec.Code, rec.Body.String())
	}

	// Custom file remediation (REQ-055) with defaults and custom fields
	if rec := doReq(http.MethodPost, "/api/v1/repos/repo-1/remediate", `{"check_key":"sonar_analysis","file_path":".github/workflows/sonar.yml","file_content":"name: Sonar","create_pr":false}`); rec.Code != http.StatusOK || !bytes.Contains(rec.Body.Bytes(), []byte(".github/workflows/sonar.yml")) {
		t.Fatalf("expected 200 on custom file draft, got %d %s", rec.Code, rec.Body.String())
	}
	if rec := doReq(http.MethodPost, "/api/v1/repos/repo-1/remediate", `{"check_key":"sonar_analysis","file_path":".github/workflows/sonar.yml","file_content":"name: Sonar","create_pr":true}`); rec.Code != http.StatusOK || !bytes.Contains(rec.Body.Bytes(), []byte("pull/10")) {
		t.Fatalf("expected 200 on custom file PR creation with defaults, got %d %s", rec.Code, rec.Body.String())
	}
	if rec := doReq(http.MethodPost, "/api/v1/repos/repo-1/remediate", `{"check_key":"sonar_analysis","file_path":".github/workflows/sonar.yml","file_content":"name: Sonar","branch_name":"custom-br","commit_message":"ci: custom","pr_title":"chore: custom","pr_body":"custom body","create_pr":true}`); rec.Code != http.StatusOK || !bytes.Contains(rec.Body.Bytes(), []byte("pull/10")) {
		t.Fatalf("expected 200 on custom file PR creation with custom fields, got %d %s", rec.Code, rec.Body.String())
	}

	// Mark repo-1 unowned to test forbidden PR creation branch
	_, _ = store.UpsertRepository(db.Repository{
		ID:       "repo-1",
		Owner:    "octocat",
		Name:     "hello-world",
		FullName: "octocat/hello-world",
		IsOwned:  false,
	})
	if rec := doReq(http.MethodPost, "/api/v1/repos/repo-1/remediate", `{"check_key":"contributing_guide","create_pr":true}`); rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 on unowned PR creation, got %d", rec.Code)
	}

	// 7. Sync Repos (error then success)
	failSync = true
	if rec := doReq(http.MethodPost, "/api/v1/repos/sync", ""); rec.Code != http.StatusBadGateway {
		t.Fatalf("expected 502 on sync failure")
	}
	failSync = false
	if rec := doReq(http.MethodPost, "/api/v1/repos/sync", ""); rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on sync success, got %d", rec.Code)
	}

	// 8. Delete Repo
	if rec := doReq(http.MethodDelete, "/api/v1/repos/repo-1", ""); rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on delete repo-1")
	}
	if rec := doReq(http.MethodDelete, "/api/v1/repos/repo-1", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 on second delete")
	}
}
