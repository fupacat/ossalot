package main

import (
	"context"
	"flag"
	"io"
	"net/http"
	"os"

	"github.com/eric/ossalot/internal/config"
	"github.com/eric/ossalot/internal/db"
)

// TRACEABILITY_MAPPING: REQ-012
var (
	exitFunc   = os.Exit
	listenFunc = http.ListenAndServe
)

// TRACEABILITY_MAPPING: REQ-012, REQ-039, REQ-042, REQ-052
func Run(args []string, envLookup config.EnvLookupFunc) int {
	cfg := config.LoadFromEnv(envLookup)
	fs := flag.NewFlagSet("ossalot", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	addr := fs.String("addr", cfg.Addr, "HTTP listen address")
	dbPath := fs.String("db", cfg.DBPath, "Database file path")
	checkOnly := fs.Bool("check", false, "Initialize store and exit 0")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	store, err := db.OpenStore(*dbPath)
	if err != nil {
		return 2
	}
	srv, runner := newServer(store, envLookup)
	if *checkOnly {
		return 0
	}
	go func() {
		_, _ = runner.TriggerJob(context.Background(), enrichmentJobName)
	}()
	if err := listenFunc(*addr, buildHandler(srv, embeddedUI())); err != nil {
		return 2
	}
	return 0
}

// TRACEABILITY_MAPPING: REQ-012
func main() {
	exitFunc(Run(os.Args[1:], os.Getenv))
}
