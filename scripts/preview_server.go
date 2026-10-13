//go:build ignore

package main

import (
	"bufio"
	"context"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/eric/ossalot/internal/api"
	"github.com/eric/ossalot/internal/db"
	"github.com/eric/ossalot/internal/engine/community"
	"github.com/eric/ossalot/internal/engine/metrics"
	"github.com/eric/ossalot/internal/engine/notifier"
	"github.com/eric/ossalot/internal/github"
	"github.com/eric/ossalot/internal/llm"
	"github.com/eric/ossalot/internal/scheduler"
)

func loadGHCLIToken() string {
	if envTok := os.Getenv("GITHUB_TOKEN"); envTok != "" {
		return envTok
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	f, err := os.Open(filepath.Join(home, ".config", "gh", "hosts.yml"))
	if err != nil {
		return ""
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "oauth_token:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "oauth_token:"))
		}
	}
	return ""
}

func main() {
	addr := ":8088"
	if v := os.Getenv("OSSALOT_PREVIEW_ADDR"); v != "" {
		addr = v
	}
	dbPath := "ossalot.db"
	if v := os.Getenv("OSSALOT_DB_PATH"); v != "" {
		dbPath = v
	}

	store, err := db.OpenStore(dbPath)
	if err != nil {
		log.Fatalf("OpenStore failed: %v", err)
	}

	// If no GitHub token is stored yet, bootstrap from ~/.config/gh/hosts.yml or GITHUB_TOKEN
	rawSettings := store.GetSettingsRaw()
	if rawSettings.GitHubToken == "" {
		if tok := loadGHCLIToken(); tok != "" {
			rawSettings.GitHubToken = tok
			_ = store.SaveSettings(rawSettings)
			log.Printf("Loaded GitHub authentication token from local gh config")
		}
	}

	// Connect to live GitHub API and OpenSSF Scorecard API
	ghClient := github.NewClient("https://api.github.com", "https://api.securityscorecards.dev", nil)
	regClient := metrics.NewRegistryClient("https://api.npmjs.org", nil)
	scanner := community.NewScanner("https://hn.algolia.com", nil)
	notif := notifier.NewNotifier(nil)
	runner := scheduler.NewRunner()
	runner.RegisterJob("daily_portfolio_sync", func(ctx context.Context) error { return nil })

	apiSrv := api.NewServer(store, ghClient, llm.NewClient("", nil))
	apiSrv.RegisterMetricsRoutes(regClient)
	apiSrv.RegisterDiscoveryRoutes()
	apiSrv.RegisterCommunityAndInboxRoutes(scanner, notif, runner)

	rootMux := http.NewServeMux()
	rootMux.Handle("/api/", apiSrv)
	rootMux.Handle("/src/", http.StripPrefix("/src/", http.FileServer(http.Dir("web/src"))))
	rootMux.Handle("/", http.FileServer(http.Dir("web/dist")))

	log.Printf("OSSAlot live server listening on %s (db=%s)", addr, dbPath)
	log.Fatal(http.ListenAndServe(addr, rootMux))
}
