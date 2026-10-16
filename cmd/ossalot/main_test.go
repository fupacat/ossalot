package main

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

// TRACEABILITY_MAPPING: REQ-012
func TestRunAndMain(t *testing.T) {
	tmpDir := t.TempDir()
	dbFile := filepath.Join(tmpDir, "cli.db")
	emptyEnv := func(string) string { return "" }

	// 1. -check mode returns 0
	if code := Run([]string{"-db", dbFile, "-check"}, emptyEnv); code != 0 {
		t.Fatalf("expected exit 0 on -check, got %d", code)
	}

	// 2. Invalid flag returns 2
	if code := Run([]string{"-unknown-flag"}, emptyEnv); code != 2 {
		t.Fatalf("expected exit 2 on invalid flag, got %d", code)
	}

	// 3. Unwritable db path returns 2
	if code := Run([]string{"-db", tmpDir, "-check"}, emptyEnv); code != 2 {
		t.Fatalf("expected exit 2 on directory db path, got %d", code)
	}

	// 4. ListenAndServe success and error paths
	origListen := listenFunc
	defer func() { listenFunc = origListen }()

	listenFunc = func(addr string, handler http.Handler) error {
		return nil
	}
	if code := Run([]string{"-db", dbFile, "-addr", "127.0.0.1:0"}, emptyEnv); code != 0 {
		t.Fatalf("expected exit 0 when listenFunc succeeds, got %d", code)
	}

	listenFunc = func(addr string, handler http.Handler) error {
		return errors.New("bind failed")
	}
	if code := Run([]string{"-db", dbFile, "-addr", "127.0.0.1:0"}, emptyEnv); code != 2 {
		t.Fatalf("expected exit 2 when listenFunc errors, got %d", code)
	}

	// 5. Exercise main() with overridden exitFunc and os.Args
	origExit := exitFunc
	origArgs := os.Args
	defer func() {
		exitFunc = origExit
		os.Args = origArgs
	}()
	capturedCode := -1
	exitFunc = func(code int) {
		capturedCode = code
	}
	os.Args = []string{"ossalot", "-db", dbFile, "-check"}
	main()
	if capturedCode != 0 {
		t.Fatalf("expected main() to exit 0, got %d", capturedCode)
	}
}

// TRACEABILITY_MAPPING: REQ-032
func TestCIWorkflowDefinition(t *testing.T) {
	wfPath := filepath.Join("..", "..", ".github", "workflows", "ci.yml")
	content, err := os.ReadFile(wfPath)
	if err != nil {
		t.Fatalf("failed to read CI workflow file: %v", err)
	}
	s := string(content)
	if !containsStr(s, "python3 scripts/run_evidence.py test") {
		t.Errorf("missing test evidence command in workflow")
	}
	if !containsStr(s, "python3 scripts/run_evidence.py coverage") {
		t.Errorf("missing coverage evidence command in workflow")
	}
	if !containsStr(s, "go build") {
		t.Errorf("missing go build command in workflow")
	}
}

// TRACEABILITY_MAPPING: REQ-035
func TestPRAutomationWorkflowDefinition(t *testing.T) {
	wfPath := filepath.Join("..", "..", ".github", "workflows", "pr-automation.yml")
	content, err := os.ReadFile(wfPath)
	if err != nil {
		t.Fatalf("failed to read PR automation workflow file: %v", err)
	}
	s := string(content)
	if !containsStr(s, "pull_request_target") {
		t.Errorf("missing pull_request_target trigger in PR automation workflow")
	}
	if !containsStr(s, "secrets.GITHUB_TOKEN") {
		t.Errorf("missing GITHUB_TOKEN secret in PR automation workflow")
	}
	if !containsStr(s, "gh pr merge \"$PR_NUMBER\" --auto --squash") {
		t.Errorf("missing auto-merge command in PR automation workflow")
	}
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || filepath.Base(s) != "" && stringContains(s, substr))
}

func stringContains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

