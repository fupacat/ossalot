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

// TRACEABILITY_MAPPING: REQ-056
func TestCopilotReviewHandlerDefinition(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "copilot-review-handler.yml"))
	if err != nil {
		t.Fatalf("read Copilot review workflow: %v", err)
	}
	wf := string(raw)
	for _, marker := range []string{
		"name: \"Auto-summon Copilot on Review Changes Requested\"",
		"pull_request_review:",
		"types: [submitted]",
		"github.event.review.state == 'changes_requested'",
		"GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}",
		"@copilot Changes were requested on this pull request",
	} {
		if !strings.Contains(wf, marker) {
			t.Fatalf("review workflow missing %q", marker)
		}
	}
	t.Logf("workflow=present trigger=pull_request_review secret=GITHUB_TOKEN")
}
