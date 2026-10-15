package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"github.com/eric/ossalot/internal/db"
	"github.com/eric/ossalot/internal/scheduler"
)

// fakeUpstream stands in for GitHub, Scorecard, npm and the community APIs, and records every request.
type fakeUpstream struct {
	mu       sync.Mutex
	requests []string
	auth     map[string]string
}

func (f *fakeUpstream) RoundTrip(req *http.Request) (*http.Response, error) {
	key := req.Method + " " + req.URL.Host + req.URL.Path
	f.mu.Lock()
	f.requests = append(f.requests, key)
	f.auth[key] = req.Header.Get("Authorization")
	f.mu.Unlock()

	status, body := http.StatusOK, ""
	path := req.URL.Path
	switch {
	case req.URL.Host == "api.github.com" && strings.Count(path, "/") == 3 && strings.HasPrefix(path, "/repos/"):
		parts := strings.Split(path, "/")
		body = `{"name":"` + parts[3] + `","full_name":"` + parts[2] + "/" + parts[3] + `","default_branch":"main","owner":{"login":"` + parts[2] + `"}}`
	case strings.Contains(path, "/git/trees/"):
		body = `{"tree":[]}`
	case strings.HasSuffix(path, "/actions/runs"):
		body = `{"workflow_runs":[]}`
	case strings.HasSuffix(path, "/releases"), strings.HasSuffix(path, "/referrers"), req.URL.Host == "dev.to":
		body = `[]`
	case strings.Contains(path, "/traffic/"):
		body = `{"count":0,"uniques":0}`
	case path == "/search/issues", req.URL.Host == "api.stackexchange.com":
		body = `{"items":[]}`
	case req.URL.Host == "api.securityscorecards.dev":
		body = `{"score":5}`
	case req.URL.Host == "hn.algolia.com":
		body = `{"hits":[]}`
	case req.URL.Host == "www.reddit.com":
		body = `{"data":{"children":[]}}`
	default:
		status, body = http.StatusNotFound, `{}`
	}
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
	}, nil
}

func (f *fakeUpstream) count(prefix string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, r := range f.requests {
		if strings.HasPrefix(r, prefix) {
			n++
		}
	}
	return n
}

// startServer runs Run against dbFile with a stubbed listener and upstream, and returns the served handler.
func startServer(t *testing.T, dbFile string, env map[string]string) (http.Handler, *fakeUpstream) {
	t.Helper()
	upstream := &fakeUpstream{auth: map[string]string{}}
	origTransport, origListen := upstreamTransport, listenFunc
	t.Cleanup(func() { upstreamTransport, listenFunc = origTransport, origListen })
	upstreamTransport = upstream

	var handler http.Handler
	listenFunc = func(addr string, h http.Handler) error {
		handler = h
		return nil
	}
	if code := Run([]string{"-db", dbFile}, func(k string) string { return env[k] }); code != 0 {
		t.Fatalf("Run exited %d", code)
	}
	// Let the startup job finish so it cannot write to dbFile after the test removes its directory
	waitForJob(t, handler, enrichmentJobName)
	return handler, upstream
}

func serve(h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, path, strings.NewReader(body)))
	return rec
}

// waitForJob polls the jobs API until the named job has a record.
func waitForJob(t *testing.T, h http.Handler, name string) scheduler.ScanJobRecord {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		var records []scheduler.ScanJobRecord
		_ = json.Unmarshal(serve(h, http.MethodGet, "/api/v1/jobs", "").Body.Bytes(), &records)
		for _, r := range records {
			if r.JobName == name {
				return r
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("job %s never ran", name)
	return scheduler.ScanJobRecord{}
}

// TRACEABILITY_MAPPING: REQ-039
func TestServerServesAllAPIRoutes(t *testing.T) {
	h, _ := startServer(t, filepath.Join(t.TempDir(), "routes.db"), nil)
	codes := map[string]int{}
	for _, path := range []string{"/api/v1/health", "/api/v1/opportunities", "/api/v1/jobs"} {
		codes[path] = serve(h, http.MethodGet, path, "").Code
	}
	if codes["/api/v1/health"] != 200 || codes["/api/v1/opportunities"] != 200 || codes["/api/v1/jobs"] != 200 {
		t.Fatalf("every API route must be served from the one listener, got %v", codes)
	}
	t.Logf("health=%d opportunities=%d jobs=%d", codes["/api/v1/health"], codes["/api/v1/opportunities"], codes["/api/v1/jobs"])
}

// TRACEABILITY_MAPPING: REQ-052
func TestServerServesBuiltUI(t *testing.T) {
	ui := fstest.MapFS{
		"index.html":    {Data: []byte("<!DOCTYPE html><title>OSSAlot</title>")},
		"assets/app.js": {Data: []byte("console.log('ui')")},
	}
	api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	h := buildHandler(api, ui)

	index := serve(h, http.MethodGet, "/", "")
	if index.Code != http.StatusOK || !strings.Contains(index.Body.String(), "<title>OSSAlot") {
		t.Fatalf("expected index.html at /, got %d %q", index.Code, index.Body.String())
	}
	asset := serve(h, http.MethodGet, "/assets/app.js", "")
	if asset.Code != http.StatusOK || !strings.Contains(asset.Header().Get("Content-Type"), "javascript") {
		t.Fatalf("expected the bundled script with a JavaScript type, got %d %q", asset.Code, asset.Header().Get("Content-Type"))
	}
	src := serve(h, http.MethodGet, "/src/model.mjs", "")
	if src.Code != http.StatusNotFound {
		t.Fatalf("/src/ is no longer served, got %d", src.Code)
	}

	// The binary embeds web/dist, whose committed placeholder exists with or without a build
	if _, err := fs.Stat(embeddedUI(), ".gitkeep"); err != nil {
		t.Fatalf("embeddedUI must be rooted at web/dist: %v", err)
	}
	t.Logf("index=%d asset=%d src=%d", index.Code, asset.Code, src.Code)
}

// TRACEABILITY_MAPPING: REQ-053
func TestServerWithoutBuiltUI(t *testing.T) {
	ui := fstest.MapFS{".gitkeep": {Data: []byte{}}}
	api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	h := buildHandler(api, ui)

	index := serve(h, http.MethodGet, "/", "")
	namesBuild := strings.Contains(index.Body.String(), "npm run build")
	if index.Code != http.StatusServiceUnavailable || !namesBuild {
		t.Fatalf("expected 503 naming npm run build, got %d %q", index.Code, index.Body.String())
	}
	apiCode := serve(h, http.MethodGet, "/api/v1/health", "").Code
	if apiCode != http.StatusOK {
		t.Fatalf("API routes must still answer without a UI, got %d", apiCode)
	}
	t.Logf("index=%d names_build=%t api=%d", index.Code, namesBuild, apiCode)
}

// TRACEABILITY_MAPPING: REQ-041
func TestGitHubTokenResolution(t *testing.T) {
	env := func(k string) string { return map[string]string{"GITHUB_TOKEN": "env-token"}[k] }
	if got := resolveGitHubToken("saved-token", env); got != "saved-token" {
		t.Fatalf("the settings token must win, got %q", got)
	}
	if got := resolveGitHubToken("", env); got != "env-token" {
		t.Fatalf("GITHUB_TOKEN must be used when no token is saved, got %q", got)
	}

	// Through the server: the environment token is sent to GitHub but never stored
	dbFile := filepath.Join(t.TempDir(), "token.db")
	h, upstream := startServer(t, dbFile, map[string]string{"GITHUB_TOKEN": "env-token"})
	if rec := serve(h, http.MethodPost, "/api/v1/repos", `{"url":"octo/demo"}`); rec.Code != http.StatusCreated && rec.Code != http.StatusOK {
		t.Fatalf("add repo: %d %s", rec.Code, rec.Body.String())
	}
	if got := upstream.auth["GET api.github.com/repos/octo/demo"]; got != "Bearer env-token" {
		t.Fatalf("GitHub must receive the environment token, got %q", got)
	}
	reopened, err := db.OpenStore(dbFile)
	if err != nil {
		t.Fatalf("reopen store: %v", err)
	}
	envPersisted := reopened.GetSettingsRaw().GitHubToken != ""
	if envPersisted {
		t.Fatalf("the environment token must never be written to the store")
	}

	// A saved token takes precedence over the environment
	saved := filepath.Join(t.TempDir(), "saved.db")
	store, err := db.OpenStore(saved)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	settings := store.GetSettingsRaw()
	settings.GitHubToken = "saved-token"
	if err := store.SaveSettings(settings); err != nil {
		t.Fatalf("save settings: %v", err)
	}
	h, upstream = startServer(t, saved, map[string]string{"GITHUB_TOKEN": "env-token"})
	serve(h, http.MethodPost, "/api/v1/repos", `{"url":"octo/demo"}`)
	if got := upstream.auth["GET api.github.com/repos/octo/demo"]; got != "Bearer saved-token" {
		t.Fatalf("the saved token must win over GITHUB_TOKEN, got %q", got)
	}
	t.Logf("settings_wins=true env_fallback=true env_persisted=%t", envPersisted)
}

// TRACEABILITY_MAPPING: REQ-042
func TestEnrichmentJobSteps(t *testing.T) {
	store, err := db.OpenStore(filepath.Join(t.TempDir(), "steps.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	for _, name := range []string{"alpha", "beta"} {
		if _, err := store.UpsertRepository(db.Repository{ID: "repo-" + name, Owner: "octo", Name: name, FullName: "octo/" + name}); err != nil {
			t.Fatalf("add repo: %v", err)
		}
	}

	var calls []string
	var bodies int64
	failPath := ""
	recorder := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		raw, _ := io.ReadAll(r.Body)
		bodies += int64(len(raw))
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == failPath {
			w.WriteHeader(http.StatusBadGateway)
		}
		_, _ = w.Write([]byte(`{}`))
	})
	runner := scheduler.NewRunner()
	registerEnrichmentJob(runner, store, recorder)

	record, err := runner.TriggerJob(context.Background(), enrichmentJobName)
	if err != nil || record.Status != "completed" {
		t.Fatalf("expected a completed job, got %+v err=%v", record, err)
	}
	want := []string{}
	for _, id := range []string{"repo-alpha", "repo-beta"} {
		for _, step := range []string{"audit", "metrics/collect", "profile", "discover", "community/scan", "playbooks"} {
			want = append(want, "POST /api/v1/repos/"+id+"/"+step)
		}
	}
	got := strings.Join(calls, "\n")
	if got != strings.Join(want, "\n") || bodies != 0 {
		t.Fatalf("each repository must get the six steps in order with no request bodies:\n%s\nbodies=%d", got, bodies)
	}

	// A failing step is reported, and the remaining steps still run
	calls, failPath = nil, "/api/v1/repos/repo-alpha/discover"
	record, err = runner.TriggerJob(context.Background(), enrichmentJobName)
	if err == nil || record.Status != "failed" || len(calls) != 12 {
		t.Fatalf("expected a failed job after all 12 steps, got %+v err=%v calls=%d", record, err, len(calls))
	}
	t.Logf("job=%s status=completed repos=2 steps_per_repo=6 packages=%d", enrichmentJobName, bodies)
}

// TRACEABILITY_MAPPING: REQ-042
func TestStartupRunsEnrichmentJob(t *testing.T) {
	dbFile := filepath.Join(t.TempDir(), "startup.db")
	store, err := db.OpenStore(dbFile)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	for _, name := range []string{"alpha", "beta"} {
		if _, err := store.UpsertRepository(db.Repository{ID: "repo-" + name, Owner: "octo", Name: name, FullName: "octo/" + name}); err != nil {
			t.Fatalf("add repo: %v", err)
		}
	}

	h, upstream := startServer(t, dbFile, nil)
	record := waitForJob(t, h, enrichmentJobName)
	if record.Status != "completed" {
		t.Fatalf("expected the startup job to complete, got %+v", record)
	}
	if n := upstream.count("GET api.npmjs.org"); n != 0 {
		t.Fatalf("enrichment must not look up guessed packages, got %d npm requests", n)
	}
	for _, repo := range []string{"alpha", "beta"} {
		var snaps []json.RawMessage
		_ = json.Unmarshal(serve(h, http.MethodGet, "/api/v1/repos/repo-"+repo+"/metrics", "").Body.Bytes(), &snaps)
		if len(snaps) != 1 || serve(h, http.MethodGet, "/api/v1/repos/repo-"+repo+"/profile", "").Code != http.StatusOK {
			t.Fatalf("%s: expected one metric snapshot and a profile after startup enrichment", repo)
		}
	}
	if !bytes.Contains(serve(h, http.MethodGet, "/api/v1/jobs", "").Body.Bytes(), []byte(enrichmentJobName)) {
		t.Fatalf("the jobs API must list the startup job")
	}
}
