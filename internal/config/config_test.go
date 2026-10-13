package config

import "testing"

// TRACEABILITY_MAPPING: REQ-001
func TestLoadFromEnv(t *testing.T) {
	emptyLookup := func(string) string { return "" }
	defaults := LoadFromEnv(emptyLookup)
	if defaults.Addr != "127.0.0.1:8080" || defaults.DBPath != "ossalot.db" || defaults.WorkspaceSlug != "default" {
		t.Fatalf("unexpected defaults: %+v", defaults)
	}

	env := map[string]string{
		"OSSALOT_ADDR":      "0.0.0.0:9090",
		"OSSALOT_DB_PATH":   "/tmp/custom.db",
		"OSSALOT_WORKSPACE": "acme",
	}
	custom := LoadFromEnv(func(k string) string { return env[k] })
	if custom.Addr != "0.0.0.0:9090" || custom.DBPath != "/tmp/custom.db" || custom.WorkspaceSlug != "acme" {
		t.Fatalf("unexpected custom config: %+v", custom)
	}
}
