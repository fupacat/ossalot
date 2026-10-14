package github

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TRACEABILITY_MAPPING: REQ-005
func TestParseRepoCoordinates(t *testing.T) {
	owner, name, err := ParseRepoCoordinates("https://github.com/octocat/hello-world.git")
	if err != nil || owner != "octocat" || name != "hello-world" {
		t.Fatalf("unexpected parse result: %s/%s err=%v", owner, name, err)
	}

	owner2, name2, err := ParseRepoCoordinates("http://github.com/eric/ossalot/")
	if err != nil || owner2 != "eric" || name2 != "ossalot" {
		t.Fatalf("unexpected parse result for http URL: %s/%s err=%v", owner2, name2, err)
	}

	if _, _, err := ParseRepoCoordinates("invalid-single-segment"); err == nil {
		t.Fatalf("expected error for invalid input")
	}
	if _, _, err := ParseRepoCoordinates("/missing-owner"); err == nil {
		t.Fatalf("expected error for empty owner")
	}
}

// TRACEABILITY_MAPPING: REQ-006
func TestInspectAndListRepositories(t *testing.T) {
	// Verify default NewClient initialization
	defaultClient := NewClient("", "", nil)
	if defaultClient.APIBaseURL != "https://api.github.com" || defaultClient.ScorecardBaseURL != "https://api.securityscorecards.dev" {
		t.Fatalf("unexpected defaults: %+v", defaultClient)
	}

	readmeB64 := base64.StdEncoding.EncodeToString([]byte("# Hello World\n## Installation\n`go install`\n## Usage\nExample"))
	mux := http.NewServeMux()
	mux.HandleFunc("GET /repos/octocat/hello-world", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"name":"hello-world","full_name":"octocat/hello-world","html_url":"https://github.com/octocat/hello-world","description":"Demo repo","language":"Go","default_branch":"","stargazers_count":42,"forks_count":5,"open_issues_count":2,"topics":["go","oss"],"owner":{"login":"octocat"},"license":{"spdx_id":"MIT"}}`))
	})
	mux.HandleFunc("GET /repos/octocat/hello-world/git/trees/main", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"tree":[{"path":"README.md"},{"path":"LICENSE"},{"path":".github/workflows/ci.yml"},{"path":".github/dependabot.yml"}]}`))
	})
	mux.HandleFunc("GET /repos/octocat/hello-world/readme", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"content":"` + readmeB64 + `"}`))
	})
	mux.HandleFunc("GET /repos/octocat/hello-world/actions/runs", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"workflow_runs":[{"conclusion":"success"}]}`))
	})
	mux.HandleFunc("GET /repos/octocat/hello-world/releases", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"tag_name":"v1.0.0"}]`))
	})
	mux.HandleFunc("GET /projects/github.com/octocat/hello-world", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"score":7.5}`))
	})
	mux.HandleFunc("GET /user/repos", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer ghp_tok" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(`[{"name":"hello-world","full_name":"octocat/hello-world","html_url":"https://github.com/octocat/hello-world","language":"Go","default_branch":"main","owner":{"login":"octocat"}}]`))
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := NewClient(srv.URL, srv.URL, srv.Client())
	ctx := context.Background()

	insp, err := client.InspectRepository(ctx, "ghp_tok", "octocat", "hello-world")
	if err != nil {
		t.Fatalf("InspectRepository failed: %v", err)
	}
	if insp.FullName != "octocat/hello-world" || !insp.HasCIWorkflow || !insp.RecentCIPassing || !insp.HasSemverRelease || !insp.HasDependabot || insp.OpenSSFScore != 7.5 {
		t.Fatalf("unexpected inspection: %+v", insp)
	}

	repos, err := client.ListAuthenticatedRepos(ctx, "ghp_tok")
	if err != nil || len(repos) != 1 || repos[0].FullName != "octocat/hello-world" {
		t.Fatalf("ListAuthenticatedRepos failed: %v, %+v", err, repos)
	}

	if _, err := client.ListAuthenticatedRepos(ctx, "wrong"); err == nil {
		t.Fatalf("expected error on unauthorized ListAuthenticatedRepos")
	}
	if _, err := client.InspectRepository(ctx, "", "missing", "repo"); err == nil {
		t.Fatalf("expected error on 404 InspectRepository")
	}

	// Network failure branch
	badClient := NewClient("http://127.0.0.1:1", "http://127.0.0.1:1", srv.Client())
	if _, err := badClient.InspectRepository(ctx, "", "octocat", "hello-world"); err == nil {
		t.Fatalf("expected network error on closed port")
	}
}

// TRACEABILITY_MAPPING: REQ-007
func TestCreateRemediationPR(t *testing.T) {
	mux := http.NewServeMux()
	branchExists := func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"object":{"sha":"abc"}}`))
	}
	mux.HandleFunc("GET /repos/octocat/hello-world/git/ref/heads/ossalot/security-policy", branchExists)
	mux.HandleFunc("GET /repos/octocat/fail-pr/git/ref/heads/ossalot/security-policy", branchExists)
	mux.HandleFunc("GET /repos/octocat/hello-world/contents/SECURITY.md", http.NotFound)
	mux.HandleFunc("GET /repos/octocat/fail-pr/contents/SECURITY.md", http.NotFound)
	mux.HandleFunc("PUT /repos/octocat/hello-world/contents/SECURITY.md", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"content":{"path":"SECURITY.md"}}`))
	})
	mux.HandleFunc("POST /repos/octocat/hello-world/pulls", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"html_url":"https://github.com/octocat/hello-world/pull/1"}`))
	})
	mux.HandleFunc("PUT /repos/octocat/fail-pr/contents/SECURITY.md", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"content":{"path":"SECURITY.md"}}`))
	})
	mux.HandleFunc("POST /repos/octocat/fail-pr/pulls", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := NewClient(srv.URL, srv.URL, srv.Client())
	ctx := context.Background()

	// Unowned repo must be rejected with ErrRepoNotOwned
	_, err := client.CreateRemediationPR(ctx, "ghp_tok", PullRequestRequest{
		Owner:   "octocat",
		Repo:    "hello-world",
		IsOwned: false,
	})
	if !errors.Is(err, ErrRepoNotOwned) {
		t.Fatalf("expected ErrRepoNotOwned, got %v", err)
	}

	// Owned repo succeeds
	prURL, err := client.CreateRemediationPR(ctx, "ghp_tok", PullRequestRequest{
		Owner:         "octocat",
		Repo:          "hello-world",
		IsOwned:       true,
		BaseBranch:    "main",
		NewBranch:     "ossalot/security-policy",
		FilePath:      "SECURITY.md",
		FileContent:   "# Security Policy",
		CommitMessage: "docs: add SECURITY.md",
		PRTitle:       "Add SECURITY.md",
		PRBody:        "Generated by OSSAlot",
	})
	if err != nil || prURL != "https://github.com/octocat/hello-world/pull/1" {
		t.Fatalf("CreateRemediationPR failed: url=%s err=%v", prURL, err)
	}

	// PUT content failure
	if _, err := client.CreateRemediationPR(ctx, "ghp_tok", PullRequestRequest{
		Owner:    "octocat",
		Repo:     "missing",
		IsOwned:  true,
		FilePath: "SECURITY.md",
	}); err == nil {
		t.Fatalf("expected error on PUT failure")
	}

	// POST PR failure
	if _, err := client.CreateRemediationPR(ctx, "ghp_tok", PullRequestRequest{
		Owner:      "octocat",
		Repo:       "fail-pr",
		IsOwned:    true,
		BaseBranch: "main",
		NewBranch:  "ossalot/security-policy",
		FilePath:   "SECURITY.md",
	}); err == nil {
		t.Fatalf("expected error on POST PR failure")
	}
}
