package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TRACEABILITY_MAPPING: REQ-054
func TestCIBuildsFrontendBeforeEvidence(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatalf("read CI workflow: %v", err)
	}
	wf := string(raw)
	at := func(marker string) int {
		i := strings.Index(wf, marker)
		if i < 0 {
			t.Fatalf("CI workflow has no %q", marker)
		}
		return i
	}
	node := at("actions/setup-node")
	build := at("npm run build")
	npmCI := at("npm ci")
	if !strings.Contains(wf[node:build], "working-directory: web") {
		t.Fatalf("the frontend build step must run in web/")
	}
	if !(node < npmCI && npmCI < build && build < at("run_evidence.py test") && build < at("run_evidence.py coverage") && build < at("go build")) {
		t.Fatalf("the frontend build must follow the Node.js setup and precede the evidence steps and go build")
	}
	t.Logf("frontend_build_step=present order=node<build<evidence<go_build")
}
