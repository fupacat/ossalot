package main

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/eric/ossalot/internal/api"
	"github.com/eric/ossalot/internal/config"
	"github.com/eric/ossalot/internal/db"
	"github.com/eric/ossalot/internal/engine/community"
	"github.com/eric/ossalot/internal/engine/metrics"
	"github.com/eric/ossalot/internal/engine/notifier"
	"github.com/eric/ossalot/internal/github"
	"github.com/eric/ossalot/internal/llm"
	"github.com/eric/ossalot/internal/scheduler"
	"github.com/eric/ossalot/web"
)

// upstreamTransport carries every request to GitHub, the registries and the community APIs.
// TRACEABILITY_MAPPING: REQ-039
var upstreamTransport http.RoundTripper = http.DefaultTransport

// TRACEABILITY_MAPPING: REQ-042
const enrichmentJobName = "daily_portfolio_sync"

// TRACEABILITY_MAPPING: REQ-042
var enrichmentSteps = []string{
	"/api/v1/repos/%s/audit",
	"/api/v1/repos/%s/metrics/collect",
	"/api/v1/repos/%s/profile",
	"/api/v1/repos/%s/discover",
	"/api/v1/repos/%s/community/scan",
	"/api/v1/repos/%s/playbooks",
}

// TRACEABILITY_MAPPING: REQ-039, REQ-041, REQ-042
func newServer(store *db.Store, envLookup config.EnvLookupFunc) (*api.Server, *scheduler.Runner) {
	ghClient := github.NewClient("https://api.github.com", "https://api.securityscorecards.dev", &http.Client{
		Timeout:   20 * time.Second,
		Transport: upstreamTransport,
	})
	ghClient.TokenFunc = func(requestToken string) string {
		return resolveGitHubToken(requestToken, envLookup)
	}
	// The registry and feed clients speak internal contracts that the live transports translate,
	// so their base URLs are never dialed.
	regClient := metrics.NewRegistryClient("http://registry.ossalot.internal", &http.Client{
		Timeout:   10 * time.Second,
		Transport: &metrics.LiveRegistryTransport{Base: upstreamTransport, NPMBaseURL: "https://api.npmjs.org"},
	})
	scanner := community.NewScanner("http://feeds.ossalot.internal", &http.Client{
		Timeout: 15 * time.Second,
		Transport: &community.LiveFeedTransport{
			Base:             upstreamTransport,
			AlgoliaURL:       "https://hn.algolia.com",
			StackExchangeURL: "https://api.stackexchange.com",
			RedditURL:        "https://www.reddit.com",
			DevToURL:         "https://dev.to",
		},
	})
	runner := scheduler.NewRunner()

	srv := api.NewServer(store, ghClient, llm.NewClient("", nil))
	srv.RegisterMetricsRoutes(regClient)
	srv.RegisterDiscoveryRoutes()
	srv.RegisterCommunityAndInboxRoutes(scanner, notifier.NewNotifier(nil), runner)
	registerEnrichmentJob(runner, store, srv)
	return srv, runner
}

// TRACEABILITY_MAPPING: REQ-039, REQ-040
func buildHandler(apiSrv http.Handler) http.Handler {
	dist, _ := fs.Sub(web.Assets, "dist")
	src, _ := fs.Sub(web.Assets, "src")
	mux := http.NewServeMux()
	mux.Handle("/api/", apiSrv)
	mux.Handle("/src/", http.StripPrefix("/src/", withModuleType(http.FileServer(http.FS(src)))))
	mux.Handle("/", http.FileServer(http.FS(dist)))
	return mux
}

// withModuleType sets the JavaScript type for .mjs files, which the OS MIME tables may not know.
// TRACEABILITY_MAPPING: REQ-040
func withModuleType(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".mjs") {
			w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		}
		next.ServeHTTP(w, r)
	})
}

// TRACEABILITY_MAPPING: REQ-041
func resolveGitHubToken(settingsToken string, envLookup config.EnvLookupFunc) string {
	if settingsToken != "" {
		return settingsToken
	}
	return envLookup("GITHUB_TOKEN")
}

// TRACEABILITY_MAPPING: REQ-042
func registerEnrichmentJob(runner *scheduler.Runner, store *db.Store, apiSrv http.Handler) {
	runner.RegisterJob(enrichmentJobName, func(ctx context.Context) error {
		failed := 0
		for _, repo := range store.ListRepositories() {
			for _, step := range enrichmentSteps {
				// No request body: metrics collection gets no package coordinates (ADR-024)
				req, _ := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf(step, repo.ID), http.NoBody)
				rec := &statusRecorder{header: http.Header{}, status: http.StatusOK}
				apiSrv.ServeHTTP(rec, req)
				if rec.status >= http.StatusBadRequest {
					failed++
				}
			}
		}
		if failed > 0 {
			return fmt.Errorf("%d enrichment steps failed", failed)
		}
		return nil
	})
}

// statusRecorder keeps only the status of an internal API call.
// TRACEABILITY_MAPPING: REQ-042
type statusRecorder struct {
	header http.Header
	status int
}

// TRACEABILITY_MAPPING: REQ-042
func (s *statusRecorder) Header() http.Header { return s.header }

// TRACEABILITY_MAPPING: REQ-042
func (s *statusRecorder) Write(b []byte) (int, error) { return len(b), nil }

// TRACEABILITY_MAPPING: REQ-042
func (s *statusRecorder) WriteHeader(status int) { s.status = status }
