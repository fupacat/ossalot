package github

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakeRoute struct {
	status int
	body   string
}

// fakeGitHub answers "METHOD /path?query" keys from routes, 404 otherwise, and records each request.
type fakeGitHub struct {
	routes map[string]fakeRoute
	calls  []string
	bodies map[string]string
	auth   map[string]string
}

func newFakeGitHub(t *testing.T, routes map[string]fakeRoute) (*fakeGitHub, *httptest.Server) {
	t.Helper()
	f := &fakeGitHub{routes: routes, bodies: map[string]string{}, auth: map[string]string{}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Method + " " + r.URL.Path
		if r.URL.RawQuery != "" {
			key += "?" + r.URL.RawQuery
		}
		raw, _ := io.ReadAll(r.Body)
		f.calls = append(f.calls, key)
		f.bodies[key] = string(raw)
		f.auth[key] = r.Header.Get("Authorization")
		rt, ok := f.routes[key]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(rt.status)
		_, _ = w.Write([]byte(rt.body))
	}))
	t.Cleanup(srv.Close)
	return f, srv
}

func (f *fakeGitHub) called(key string) bool {
	for _, c := range f.calls {
		if c == key {
			return true
		}
	}
	return false
}

func remediationRequest(path string) PullRequestRequest {
	return PullRequestRequest{
		Owner:         "octo",
		Repo:          "demo",
		IsOwned:       true,
		NewBranch:     "fix",
		FilePath:      path,
		FileContent:   "content",
		CommitMessage: "docs: add file",
		PRTitle:       "Add file",
		PRBody:        "Generated",
	}
}

// Routes for an owned repository whose default branch is trunk and which has no "fix" branch yet.
func baseRemediationRoutes() map[string]fakeRoute {
	return map[string]fakeRoute{
		"GET /repos/octo/demo":                     {200, `{"default_branch":"trunk"}`},
		"GET /repos/octo/demo/git/ref/heads/trunk": {200, `{"object":{"sha":"trunk-sha"}}`},
		"POST /repos/octo/demo/git/refs":           {201, `{"ref":"refs/heads/fix"}`},
		"PUT /repos/octo/demo/contents/LICENSE":    {201, `{}`},
		"POST /repos/octo/demo/pulls":              {201, `{"html_url":"https://github.com/octo/demo/pull/7"}`},
	}
}

// TRACEABILITY_MAPPING: REQ-043
func TestCreateRemediationPRUsesDefaultBranchAndCreatesHeadBranch(t *testing.T) {
	fake, srv := newFakeGitHub(t, baseRemediationRoutes())
	client := NewClient(srv.URL, srv.URL, srv.Client())

	prURL, err := client.CreateRemediationPR(context.Background(), "tok", remediationRequest("LICENSE"))
	if err != nil || prURL != "https://github.com/octo/demo/pull/7" {
		t.Fatalf("CreateRemediationPR: url=%q err=%v", prURL, err)
	}
	var ref map[string]string
	_ = json.Unmarshal([]byte(fake.bodies["POST /repos/octo/demo/git/refs"]), &ref)
	if ref["ref"] != "refs/heads/fix" || ref["sha"] != "trunk-sha" {
		t.Fatalf("head branch must be created from trunk's commit, got %v", ref)
	}
	var pr map[string]string
	_ = json.Unmarshal([]byte(fake.bodies["POST /repos/octo/demo/pulls"]), &pr)
	if pr["base"] != "trunk" || pr["head"] != "fix" {
		t.Fatalf("pull request must target the default branch, got %v", pr)
	}
	t.Logf("base=%s branch_created=%t", pr["base"], fake.called("POST /repos/octo/demo/git/refs"))
}

// TRACEABILITY_MAPPING: REQ-043
func TestCreateRemediationPRBranchHandling(t *testing.T) {
	ctx := context.Background()

	// An existing head branch is reused, and an explicit base branch skips the repository lookup
	routes := baseRemediationRoutes()
	routes["GET /repos/octo/demo/git/ref/heads/fix"] = fakeRoute{200, `{"object":{"sha":"fix-sha"}}`}
	fake, srv := newFakeGitHub(t, routes)
	req := remediationRequest("LICENSE")
	req.BaseBranch = "release"
	if _, err := NewClient(srv.URL, srv.URL, srv.Client()).CreateRemediationPR(ctx, "tok", req); err != nil {
		t.Fatalf("existing branch: %v", err)
	}
	if fake.called("POST /repos/octo/demo/git/refs") || fake.called("GET /repos/octo/demo") {
		t.Fatalf("existing branch with explicit base must not create a branch or look up the repository: %v", fake.calls)
	}

	failures := map[string]func(map[string]fakeRoute){
		"repository lookup fails": func(r map[string]fakeRoute) { r["GET /repos/octo/demo"] = fakeRoute{500, `{}`} },
		"head branch lookup fails": func(r map[string]fakeRoute) {
			r["GET /repos/octo/demo/git/ref/heads/fix"] = fakeRoute{500, `{}`}
		},
		"base branch lookup fails": func(r map[string]fakeRoute) { delete(r, "GET /repos/octo/demo/git/ref/heads/trunk") },
		"branch creation fails":    func(r map[string]fakeRoute) { r["POST /repos/octo/demo/git/refs"] = fakeRoute{422, `{}`} },
	}
	for name, breakRoute := range failures {
		routes := baseRemediationRoutes()
		breakRoute(routes)
		_, srv := newFakeGitHub(t, routes)
		if _, err := NewClient(srv.URL, srv.URL, srv.Client()).CreateRemediationPR(ctx, "tok", remediationRequest("LICENSE")); err == nil {
			t.Fatalf("%s: expected an error", name)
		}
	}
}

// TRACEABILITY_MAPPING: REQ-044
func TestCreateRemediationPRUpdatesExistingFile(t *testing.T) {
	ctx := context.Background()
	routes := baseRemediationRoutes()
	routes["GET /repos/octo/demo/contents/LICENSE?ref=fix"] = fakeRoute{200, `{"sha":"existing"}`}
	fake, srv := newFakeGitHub(t, routes)
	if _, err := NewClient(srv.URL, srv.URL, srv.Client()).CreateRemediationPR(ctx, "tok", remediationRequest("LICENSE")); err != nil {
		t.Fatalf("CreateRemediationPR: %v", err)
	}
	var put map[string]string
	_ = json.Unmarshal([]byte(fake.bodies["PUT /repos/octo/demo/contents/LICENSE"]), &put)
	if put["sha"] != "existing" {
		t.Fatalf("the PUT must carry the existing file's SHA, got %v", put)
	}
	t.Logf("put_sha=%s", put["sha"])

	// A new file is committed without a SHA
	fake, srv = newFakeGitHub(t, baseRemediationRoutes())
	if _, err := NewClient(srv.URL, srv.URL, srv.Client()).CreateRemediationPR(ctx, "tok", remediationRequest("LICENSE")); err != nil {
		t.Fatalf("new file: %v", err)
	}
	put = nil
	_ = json.Unmarshal([]byte(fake.bodies["PUT /repos/octo/demo/contents/LICENSE"]), &put)
	if _, ok := put["sha"]; ok {
		t.Fatalf("a new file must not carry a SHA, got %v", put)
	}

	// A failing file lookup stops the flow before the commit
	routes = baseRemediationRoutes()
	routes["GET /repos/octo/demo/contents/LICENSE?ref=fix"] = fakeRoute{500, `{}`}
	fake, srv = newFakeGitHub(t, routes)
	if _, err := NewClient(srv.URL, srv.URL, srv.Client()).CreateRemediationPR(ctx, "tok", remediationRequest("LICENSE")); err == nil {
		t.Fatalf("expected an error when the file lookup fails")
	}
	if fake.called("PUT /repos/octo/demo/contents/LICENSE") {
		t.Fatalf("no commit may follow a failed file lookup")
	}
}

// TRACEABILITY_MAPPING: REQ-045
func TestCreateRemediationPRReturnsExistingPullRequest(t *testing.T) {
	ctx := context.Background()
	exists := fakeRoute{422, `{"message":"Validation Failed","errors":[{"message":"A pull request already exists for octo:fix."}]}`}
	listKey := "GET /repos/octo/demo/pulls?state=open&head=octo%3Afix"

	routes := baseRemediationRoutes()
	routes["POST /repos/octo/demo/pulls"] = exists
	routes[listKey] = fakeRoute{200, `[{"html_url":"https://github.com/octo/demo/pull/3"}]`}
	_, srv := newFakeGitHub(t, routes)
	prURL, err := NewClient(srv.URL, srv.URL, srv.Client()).CreateRemediationPR(ctx, "tok", remediationRequest("LICENSE"))
	if err != nil || prURL != "https://github.com/octo/demo/pull/3" {
		t.Fatalf("expected the existing pull request: url=%q err=%v", prURL, err)
	}
	t.Logf("pr_url=existing err=%v", err)

	failures := map[string]func(map[string]fakeRoute){
		"list fails": func(r map[string]fakeRoute) { r[listKey] = fakeRoute{500, `{}`} },
		"none open":  func(r map[string]fakeRoute) { r[listKey] = fakeRoute{200, `[]`} },
		"other 422": func(r map[string]fakeRoute) {
			r["POST /repos/octo/demo/pulls"] = fakeRoute{422, `{"message":"No commits between trunk and fix"}`}
		},
		"server error": func(r map[string]fakeRoute) {
			r["POST /repos/octo/demo/pulls"] = fakeRoute{500, `A pull request already exists`}
		},
		"unreadable response": func(r map[string]fakeRoute) { r["POST /repos/octo/demo/pulls"] = fakeRoute{201, `not json`} },
	}
	for name, change := range failures {
		routes := baseRemediationRoutes()
		routes["POST /repos/octo/demo/pulls"] = exists
		change(routes)
		_, srv := newFakeGitHub(t, routes)
		if _, err := NewClient(srv.URL, srv.URL, srv.Client()).CreateRemediationPR(ctx, "tok", remediationRequest("LICENSE")); err == nil {
			t.Fatalf("%s: expected an error", name)
		}
	}
}

// TRACEABILITY_MAPPING: REQ-046
func TestCreateRemediationPRWorkflowScope(t *testing.T) {
	ctx := context.Background()
	routes := baseRemediationRoutes()
	fake, srv := newFakeGitHub(t, routes)
	_, err := NewClient(srv.URL, srv.URL, srv.Client()).CreateRemediationPR(ctx, "tok", remediationRequest(".github/workflows/scorecard.yml"))
	if !errors.Is(err, ErrWorkflowScopeRequired) || !strings.Contains(err.Error(), "'workflow' OAuth scope") {
		t.Fatalf("expected ErrWorkflowScopeRequired naming the workflow scope, got %v", err)
	}
	if fake.called("POST /repos/octo/demo/pulls") {
		t.Fatalf("no pull request may be requested after a refused workflow commit")
	}
	t.Logf("err=ErrWorkflowScopeRequired pr_requested=%t", fake.called("POST /repos/octo/demo/pulls"))

	// Other failures, and 404s outside .github/workflows/, keep the GitHub error
	routes = baseRemediationRoutes()
	routes["PUT /repos/octo/demo/contents/.github/workflows/scorecard.yml"] = fakeRoute{500, `{}`}
	_, srv = newFakeGitHub(t, routes)
	if _, err := NewClient(srv.URL, srv.URL, srv.Client()).CreateRemediationPR(ctx, "tok", remediationRequest(".github/workflows/scorecard.yml")); err == nil || errors.Is(err, ErrWorkflowScopeRequired) {
		t.Fatalf("a 500 on a workflow file must surface as a GitHub error, got %v", err)
	} else if !strings.Contains(err.Error(), "unexpected HTTP status 500") {
		t.Fatalf("the GitHub error must name its status, got %q", err.Error())
	}
	routes = baseRemediationRoutes()
	delete(routes, "PUT /repos/octo/demo/contents/LICENSE")
	_, srv = newFakeGitHub(t, routes)
	if _, err := NewClient(srv.URL, srv.URL, srv.Client()).CreateRemediationPR(ctx, "tok", remediationRequest("LICENSE")); err == nil || errors.Is(err, ErrWorkflowScopeRequired) {
		t.Fatalf("a 404 outside .github/workflows/ must surface as a GitHub error, got %v", err)
	}
}

// TRACEABILITY_MAPPING: REQ-047
func TestFetchTrafficSnapshotRestrictedRepository(t *testing.T) {
	ctx := context.Background()
	for _, status := range []int{http.StatusForbidden, http.StatusNotFound} {
		routes := map[string]fakeRoute{
			"GET /repos/octo/demo/traffic/views":             {status, `{"message":"Must have push access"}`},
			"GET /repos/octo/demo/traffic/clones":            {status, `{}`},
			"GET /repos/octo/demo/traffic/popular/referrers": {status, `{}`},
		}
		_, srv := newFakeGitHub(t, routes)
		snap, err := NewClient(srv.URL, srv.URL, srv.Client()).FetchTrafficSnapshot(ctx, "tok", "octo", "demo")
		if err != nil || snap.ViewsTotal != 0 || snap.ClonesTotal != 0 || len(snap.Referrers) != 0 {
			t.Fatalf("status %d: expected an empty snapshot without error, got %+v err=%v", status, snap, err)
		}
		t.Logf("views=%d clones=%d referrers=%d err=%v", snap.ViewsTotal, snap.ClonesTotal, len(snap.Referrers), err)
	}
}

// TRACEABILITY_MAPPING: REQ-048
func TestSearchIssuesRetriesWithKeyword(t *testing.T) {
	ctx := context.Background()
	item := `{"items":[{"number":5,"title":"Parser crash","body":"original body","html_url":"https://github.com/a/b/issues/5","repository_url":"https://api.github.com/repos/a/b","user":{"login":"dev"},"labels":[{"name":"bug"}]}]}`
	broadKey := "GET /search/issues?q=is%3Aissue+is%3Aopen+parser&per_page=20"
	routes := map[string]fakeRoute{
		"GET /search/issues?q=is%3Aissue+language%3AGo+parser&per_page=20": {200, `{"items":[]}`},
		broadKey: {200, item},
	}
	fake, srv := newFakeGitHub(t, routes)
	got, err := NewClient(srv.URL, srv.URL, srv.Client()).SearchIssues(ctx, "tok", "is:issue language:Go parser")
	if err != nil || len(got) != 1 {
		t.Fatalf("expected one item from the broader search, got %v err=%v", got, err)
	}
	if got[0].Body != "original body" || len(got[0].Labels) != 1 || got[0].Labels[0] != "bug" {
		t.Fatalf("issue fields must be returned unmodified, got %+v", got[0])
	}
	retries := 0
	for _, c := range fake.calls {
		if c == broadKey {
			retries++
		}
	}
	if retries != 1 {
		t.Fatalf("expected exactly one retry, got %d: %v", retries, fake.calls)
	}
	t.Logf("retries=%d items=%d body_unchanged=true labels_unchanged=true", retries, len(got))

	// Results on the first query, and queries with only qualifiers, are not retried
	for _, query := range []string{"is:issue is:open parser", "is:issue label:bug"} {
		fake, srv := newFakeGitHub(t, map[string]fakeRoute{
			broadKey: {200, item},
			"GET /search/issues?q=is%3Aissue+label%3Abug&per_page=20": {200, `{"items":[]}`},
		})
		if _, err := NewClient(srv.URL, srv.URL, srv.Client()).SearchIssues(ctx, "tok", query); err != nil {
			t.Fatalf("%q: %v", query, err)
		}
		if len(fake.calls) != 1 {
			t.Fatalf("%q: expected one search, got %v", query, fake.calls)
		}
	}

	// A failing retry is an error
	delete(routes, broadKey)
	_, srv = newFakeGitHub(t, routes)
	if _, err := NewClient(srv.URL, srv.URL, srv.Client()).SearchIssues(ctx, "tok", "is:issue language:Go parser"); err == nil {
		t.Fatalf("expected an error when the retry fails")
	}

	if kw := primarySearchKeyword(`is:issue "parser"`); kw != "parser" {
		t.Fatalf("quotes must be trimmed from the keyword, got %q", kw)
	}
}

// TRACEABILITY_MAPPING: REQ-041
func TestTokenFuncAppliesOnlyToGitHubAPI(t *testing.T) {
	api, apiSrv := newFakeGitHub(t, map[string]fakeRoute{
		"GET /repos/octo/demo": {200, `{"name":"demo","default_branch":"main"}`},
	})
	scorecard, scSrv := newFakeGitHub(t, map[string]fakeRoute{
		"GET /projects/github.com/octo/demo": {200, `{"score":5}`},
	})
	client := NewClient(apiSrv.URL, scSrv.URL, http.DefaultClient)
	client.TokenFunc = func(requestToken string) string {
		if requestToken != "" {
			return requestToken
		}
		return "env-token"
	}
	if _, err := client.InspectRepository(context.Background(), "", "octo", "demo"); err != nil {
		t.Fatalf("InspectRepository: %v", err)
	}
	if got := api.auth["GET /repos/octo/demo"]; got != "Bearer env-token" {
		t.Fatalf("GitHub API requests must carry the resolved token, got %q", got)
	}
	if got := scorecard.auth["GET /projects/github.com/octo/demo"]; got != "" {
		t.Fatalf("the Scorecard API must never receive the GitHub token, got %q", got)
	}
}
