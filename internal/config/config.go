package config

// TRACEABILITY_MAPPING: REQ-001
type Config struct {
	Addr          string `json:"addr"`
	DBPath        string `json:"db_path"`
	WorkspaceSlug string `json:"workspace_slug"`
}

// TRACEABILITY_MAPPING: REQ-001
type EnvLookupFunc func(key string) string

// TRACEABILITY_MAPPING: REQ-001
func LoadFromEnv(lookup EnvLookupFunc) Config {
	cfg := Config{
		Addr:          "127.0.0.1:8080",
		DBPath:        "ossalot.db",
		WorkspaceSlug: "default",
	}
	if v := lookup("OSSALOT_ADDR"); v != "" {
		cfg.Addr = v
	}
	if v := lookup("OSSALOT_DB_PATH"); v != "" {
		cfg.DBPath = v
	}
	if v := lookup("OSSALOT_WORKSPACE"); v != "" {
		cfg.WorkspaceSlug = v
	}
	return cfg
}
