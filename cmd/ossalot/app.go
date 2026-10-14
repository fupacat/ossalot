package main

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
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

// embeddedUI returns the Vite build embedded by web/embed.go, rooted at its index.html.
// TRACEABILITY_MAPPING: REQ-052
func embeddedUI() fs.FS {
	ui, _ := fs.Sub(web.Assets, "dist") // fails only for an invalid path, and "dist" is valid
	return ui
}

// uiNotBuilt is the page for a binary built without the frontend build (ADR-029).
// TRACEABILITY_MAPPING: REQ-053
const uiNotBuilt = "The OSSAlot UI is not in this binary. Run npm ci && npm run build in web/, then rebuild ossalot (make build does both).\n"

// TRACEABILITY_MAPPING: REQ-039, REQ-052, REQ-053
func buildHandler(apiSrv http.Handler, ui fs.FS) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/api/", apiSrv)
	if _, err := fs.Stat(ui, "index.html"); err != nil {
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, uiNotBuilt, http.StatusServiceUnavailable)
		})
		return mux
	}
	mux.Handle("/", http.FileServer(http.FS(ui)))
	return mux
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
