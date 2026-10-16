package db

import (
	"os"
	"path/filepath"
	"testing"
)

// TRACEABILITY_MAPPING: REQ-002
func TestOpenStoreAndDefaultWorkspace(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "ossalot.db")

	store1, err := OpenStore(dbPath)
	if err != nil {
		t.Fatalf("OpenStore initial failed: %v", err)
	}
	ws := store1.GetDefaultWorkspace()
	if ws.Slug != "default" || ws.Mode != "personal" {
		t.Fatalf("unexpected workspace: %+v", ws)
	}

	// Re-open existing store to verify idempotent reload
	store2, err := OpenStore(dbPath)
	if err != nil {
		t.Fatalf("OpenStore reopen failed: %v", err)
	}
	ws2 := store2.GetDefaultWorkspace()
	if ws2.ID != ws.ID || ws2.Slug != "default" {
		t.Fatalf("reopened workspace mismatch: %+v", ws2)
	}

	// Corrupt file check
	corruptPath := filepath.Join(tmpDir, "corrupt.db")
	_ = os.WriteFile(corruptPath, []byte("{bad-json"), 0o600)
	if _, err := OpenStore(corruptPath); err == nil {
		t.Fatalf("expected error on corrupt db file")
	}

	// Unwritable path check (directory path instead of file)
	if _, err := OpenStore(tmpDir); err == nil {
		t.Fatalf("expected error when opening directory as db file")
	}
}

// TRACEABILITY_MAPPING: REQ-003, REQ-033
func TestWorkspaceSettingsMasking(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "settings.db")
	store, err := OpenStore(dbPath)
	if err != nil {
		t.Fatalf("OpenStore failed: %v", err)
	}

	// Initially empty secrets remain empty when redacted
	initial := store.GetSettingsRedacted()
	if initial.GitHubToken != "" || initial.LLMAPIKey != "" {
		t.Fatalf("expected empty initial secrets, got %+v", initial)
	}

	err = store.SaveSettings(WorkspaceSettings{
		GitHubToken:    "ghp_secret123",
		LLMProvider:    "openai",
		LLMAPIKey:      "sk-secret456",
		LLMFastModel:   "gpt-4o-mini",
		LLMStrongModel: "gpt-4o",
	})
	if err != nil {
		t.Fatalf("SaveSettings failed: %v", err)
	}

	redacted := store.GetSettingsRedacted()
	if redacted.GitHubToken != "***" || redacted.LLMAPIKey != "***" {
		t.Fatalf("expected masked secrets, got %+v", redacted)
	}

	raw := store.GetSettingsRaw()
	if raw.GitHubToken != "ghp_secret123" || raw.LLMAPIKey != "sk-secret456" {
		t.Fatalf("expected raw secrets, got %+v", raw)
	}

	// Saving with "***" preserves existing secrets
	err = store.SaveSettings(WorkspaceSettings{
		GitHubToken:  "***",
		LLMProvider:  "anthropic",
		LLMAPIKey:    "***",
		LLMFastModel: "claude-3-5-haiku",
		SonarToken:   "sqp_secret789",
		SonarHostURL: "https://sonarcloud.io",
		SonarOrg:     "fupacat",
		CustomKeys:   map[string]string{"codecov": "tok_123"},
	})
	if err != nil {
		t.Fatalf("SaveSettings masked update failed: %v", err)
	}
	rawAfter := store.GetSettingsRaw()
	if rawAfter.GitHubToken != "ghp_secret123" || rawAfter.LLMAPIKey != "sk-secret456" || rawAfter.LLMProvider != "anthropic" || rawAfter.SonarToken != "sqp_secret789" {
		t.Fatalf("secrets were not preserved on masked save: %+v", rawAfter)
	}
	redactedSonar := store.GetSettingsRedacted()
	if redactedSonar.SonarToken != "***" {
		t.Fatalf("expected SonarToken to be masked, got %s", redactedSonar.SonarToken)
	}

	// Saving SonarToken with "***" preserves existing sonar secret
	err = store.SaveSettings(WorkspaceSettings{
		SonarToken: "***",
	})
	if err != nil {
		t.Fatalf("SaveSettings sonar masked update failed: %v", err)
	}
	if store.GetSettingsRaw().SonarToken != "sqp_secret789" {
		t.Fatalf("sonar token was not preserved on masked save")
	}
}

// TRACEABILITY_MAPPING: REQ-004
func TestRepositoryAndAuditLifecycle(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "repos.db")
	store, err := OpenStore(dbPath)
	if err != nil {
		t.Fatalf("OpenStore failed: %v", err)
	}

	saved, err := store.UpsertRepository(Repository{
		Owner:           "octocat",
		Name:            "hello-world",
		FullName:        "octocat/hello-world",
		HTMLURL:         "https://github.com/octocat/hello-world",
		PrimaryLanguage: "Go",
		IsOwned:         true,
	})
	if err != nil || saved.ID == "" {
		t.Fatalf("UpsertRepository insert failed: %v, %+v", err, saved)
	}

	audit := AuditRecord{
		ID:             "audit-1",
		OverallScore:   85,
		CategoryScores: map[string]int{"community_files": 25, "documentation_quality": 20, "technical_hygiene": 20, "security_openssf": 20},
		Checks: []CheckItemRecord{
			{CheckKey: "license_valid", Category: "community_files", Status: "pass", Score: 5, Weight: 5},
		},
	}
	if err := store.SaveAudit(saved.ID, audit); err != nil {
		t.Fatalf("SaveAudit failed: %v", err)
	}
	if err := store.SaveAudit("nonexistent-id", audit); err == nil {
		t.Fatalf("expected error saving audit for nonexistent repo")
	}

	// Upsert existing repo by FullName preserves LatestAudit when nil
	updated, err := store.UpsertRepository(Repository{
		Owner:           "octocat",
		Name:            "hello-world",
		FullName:        "octocat/hello-world",
		Description:     "Updated description",
		PrimaryLanguage: "Go",
		IsOwned:         true,
		StandardsScore:  85,
	})
	if err != nil || updated.ID != saved.ID || updated.LatestAudit == nil || updated.LatestAudit.OverallScore != 85 {
		t.Fatalf("UpsertRepository update failed to preserve audit: %+v, err=%v", updated, err)
	}

	list := store.ListRepositories()
	if len(list) != 1 || list[0].Description != "Updated description" {
		t.Fatalf("unexpected repository list: %+v", list)
	}

	got, ok := store.GetRepository(saved.ID)
	if !ok || got.FullName != "octocat/hello-world" {
		t.Fatalf("GetRepository failed: ok=%v, got=%+v", ok, got)
	}
	if _, ok := store.GetRepository("missing"); ok {
		t.Fatalf("expected false for missing repo")
	}

	if !store.DeleteRepository(saved.ID) {
		t.Fatalf("expected true deleting existing repo")
	}
	if store.DeleteRepository(saved.ID) {
		t.Fatalf("expected false deleting already-removed repo")
	}
	if len(store.ListRepositories()) != 0 {
		t.Fatalf("expected 0 repos after delete")
	}
}

// TRACEABILITY_MAPPING: REQ-056
func TestPatchRepository(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "patch.db")
	store, err := OpenStore(dbPath)
	if err != nil {
		t.Fatalf("OpenStore failed: %v", err)
	}

	repo, err := store.UpsertRepository(Repository{
		Owner:    "fupacat",
		Name:     "ossalot",
		FullName: "fupacat/ossalot",
		IsOwned:  false,
	})
	if err != nil {
		t.Fatalf("UpsertRepository failed: %v", err)
	}

	// Patching non-existent repo
	if _, ok, err := store.PatchRepository("missing-id", RepoPatch{}); err != nil || ok {
		t.Fatalf("expected ok=false for missing repo, got ok=%v err=%v", ok, err)
	}

	isOwned := true
	defBranch := "main"
	baseBranch := "main"
	prefix := "copilot/"
	handle := "@copilot"
	prompt := "Run 'go test ./...' before opening PR"
	labels := "copilot, agent"
	desc := "OSS maintenance cockpit"

	patched, ok, err := store.PatchRepository(repo.ID, RepoPatch{
		IsOwned:       &isOwned,
		DefaultBranch: &defBranch,
		BaseBranch:    &baseBranch,
		BranchPrefix:  &prefix,
		CopilotHandle: &handle,
		CopilotPrompt: &prompt,
		IssueLabels:   &labels,
		Description:   &desc,
	})
	if err != nil || !ok {
		t.Fatalf("PatchRepository failed: ok=%v err=%v", ok, err)
	}

	if !patched.IsOwned || patched.DefaultBranch != "main" || patched.BaseBranch != "main" ||
		patched.BranchPrefix != "copilot/" || patched.CopilotHandle != "@copilot" ||
		patched.CopilotPrompt != prompt || patched.IssueLabels != labels || patched.Description != desc {
		t.Fatalf("unexpected patched repo: %+v", patched)
	}

	// Re-upserting preserves configured fields
	reupserted, err := store.UpsertRepository(Repository{
		ID:       repo.ID,
		FullName: "fupacat/ossalot",
		Owner:    "fupacat",
		Name:     "ossalot",
	})
	if err != nil {
		t.Fatalf("re-upsert failed: %v", err)
	}
	if reupserted.DefaultBranch != "main" || reupserted.BaseBranch != "main" || reupserted.CopilotHandle != "@copilot" {
		t.Fatalf("upsert failed to preserve config: %+v", reupserted)
	}

	// Flush error path
	store.path = filepath.Join(t.TempDir(), "missing-dir", "db.json")
	if _, _, err := store.PatchRepository(repo.ID, RepoPatch{IsOwned: &isOwned}); err == nil {
		t.Fatalf("expected flush error for invalid path")
	}
}

