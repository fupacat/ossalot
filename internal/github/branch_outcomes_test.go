package github

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TRACEABILITY_MAPPING: REQ-006
func TestInspectRepositoryFallbacks(t *testing.T) {
	failing := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /repos/octocat/hello-world", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"name":"hello-world","full_name":"octocat/hello-world","default_branch":"trunk"}`))
	})
	mux.HandleFunc("GET /repos/octocat/hello-world/git/trees/trunk", failing)
	mux.HandleFunc("GET /repos/octocat/hello-world/readme", failing)
	mux.HandleFunc("GET /repos/octocat/hello-world/actions/runs", failing)
	mux.HandleFunc("GET /repos/octocat/hello-world/releases", failing)
	mux.HandleFunc("GET /projects/github.com/octocat/hello-world", failing)

	srv := httptest.NewServer(mux)
	defer srv.Close()

	insp, err := NewClient(srv.URL, srv.URL, srv.Client()).InspectRepository(context.Background(), "", "octocat", "hello-world")
	if err != nil {
		t.Fatalf("a failing optional endpoint must not fail the inspection: %v", err)
	}
	if insp.DefaultBranch != "trunk" {
		t.Fatalf("expected the reported default branch trunk, got %q", insp.DefaultBranch)
	}
	if len(insp.FilesPresent) != 0 || insp.HasCIWorkflow || insp.ReadmeContent != "" || insp.RecentCIPassing || insp.HasReleases || insp.OpenSSFScore != 0 {
		t.Fatalf("failed endpoints should leave their fields empty: %+v", insp)
	}
}

// TRACEABILITY_MAPPING: REQ-006
func TestInspectRepositoryNonSemverRelease(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /repos/octocat/hello-world", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"name":"hello-world","full_name":"octocat/hello-world","default_branch":"main"}`))
	})
	mux.HandleFunc("GET /repos/octocat/hello-world/releases", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"tag_name":"1.0.0"}]`))
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	insp, err := NewClient(srv.URL, srv.URL, srv.Client()).InspectRepository(context.Background(), "", "octocat", "hello-world")
	if err != nil {
		t.Fatalf("InspectRepository failed: %v", err)
	}
	if !insp.HasReleases || insp.HasSemverRelease {
		t.Fatalf("a release tag without a v prefix is a release but not semver: %+v", insp)
	}
}
