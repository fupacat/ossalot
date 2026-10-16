package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TRACEABILITY_MAPPING: REQ-056
func TestCopilotCIFailureWorkflowDefinition(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "copilot-ci-failure-handler.yml"))
	if err != nil {
		t.Fatalf("read copilot ci failure workflow: %v", err)
	}
	wf := string(raw)
	for _, marker := range []string{
		"workflow_run:",
		"types: [completed]",
		"github.event.workflow_run.conclusion == 'failure'",
		"github.event.workflow_run.event == 'pull_request'",
		"GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}",
		"gh pr comment",
		"@copilot",
	} {
		if !strings.Contains(wf, marker) {
			t.Fatalf("workflow missing %q", marker)
		}
	}
	t.Log("trigger=workflow_run condition=failure_and_pull_request commenter=gh_pr_comment")
}
