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
	panic("stub: REQ-001 not yet implemented")
}
