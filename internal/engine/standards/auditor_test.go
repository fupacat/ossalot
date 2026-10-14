package standards

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/eric/ossalot/internal/db"
	"github.com/eric/ossalot/internal/github"
	"github.com/eric/ossalot/internal/llm"
)

// TRACEABILITY_MAPPING: REQ-009
func TestEvaluateStandardsCompletePartialAndMinimal(t *testing.T) {
	// 1. Complete repository (100 points)
	complete := github.RepoInspection{
		FullName:    "octocat/hello-world",
		Description: "Complete open-source library",
		LicenseSPDX: "MIT",
		Topics:      []string{"go", "oss"},
		FilesPresent: map[string]bool{
			"CONTRIBUTING.MD":                      true,
			"CODE_OF_CONDUCT.MD":                   true,
			"SECURITY.MD":                          true,
			".GITHUB/ISSUE_TEMPLATE/BUG_REPORT.MD": true,
			".GITHUB/PULL_REQUEST_TEMPLATE.MD":     true,
		},
		ReadmeContent:    "# Hello World\nThis is a comprehensive README with more than eighty characters.\n## Installation\n`go get`\n## Usage\nRun it.",
		HasCIWorkflow:    true,
		RecentCIPassing:  true,
		HasReleases:      true,
		HasSemverRelease: true,
		HasDependabot:    true,
		OpenSSFScore:     8.5,
	}
	resComplete := EvaluateStandards(complete)
	if resComplete.OverallScore != 100 {
		t.Fatalf("expected 100 score for complete repo, got %d (%+v)", resComplete.OverallScore, resComplete.CategoryScores)
	}

	// 2. Partial repository (exercises every warn path)
	partial := github.RepoInspection{
		FullName:    "octocat/partial",
		Description: "Only description, no topics",
		FilesPresent: map[string]bool{
			".GITHUB/ISSUE_TEMPLATE.MD": true,
		},
		ReadmeContent: "Short readme without sections",
		HasCIWorkflow: true,
		HasReleases:   true,
		OpenSSFScore:  5.0,
	}
	resPartial := EvaluateStandards(partial)
	if resPartial.OverallScore <= 0 || resPartial.OverallScore >= 100 {
		t.Fatalf("expected intermediate score for partial repo, got %d", resPartial.OverallScore)
	}

	// 3. Minimal repository (0 points, all fail)
	minimal := github.RepoInspection{
		FullName:     "octocat/minimal",
		FilesPresent: map[string]bool{},
	}
	resMinimal := EvaluateStandards(minimal)
	if resMinimal.OverallScore != 0 {
		t.Fatalf("expected 0 score for minimal repo, got %d", resMinimal.OverallScore)
	}
}

// TRACEABILITY_MAPPING: REQ-010
func TestBuildRemediationDraft(t *testing.T) {
	ctx := context.Background()
	llmClient := llm.NewClient("", nil)
	repo := db.Repository{
		FullName:        "octocat/hello-world",
		Description:     "Sample repo",
		PrimaryLanguage: "Go",
	}

	draft, err := BuildRemediationDraft(ctx, llmClient, db.WorkspaceSettings{}, repo, "contributing_guide")
	if err != nil {
		t.Fatalf("BuildRemediationDraft failed: %v", err)
	}
	if draft.FilePath != "CONTRIBUTING.md" || draft.BranchName != "ossalot/remediate-contributing_guide" || !strings.Contains(draft.Content, "octocat/hello-world") {
		t.Fatalf("unexpected draft: %+v", draft)
	}

	// Unsupported check key error branch
	if _, err := BuildRemediationDraft(ctx, llmClient, db.WorkspaceSettings{}, repo, "unknown_key"); err == nil {
		t.Fatalf("expected error on unsupported check key")
	}

	// LLM error branch
	errSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer errSrv.Close()
	failingLLM := llm.NewClient(errSrv.URL, errSrv.Client())
	if _, err := BuildRemediationDraft(ctx, failingLLM, db.WorkspaceSettings{LLMAPIKey: "sk-1"}, repo, "security_policy"); err == nil {
		t.Fatalf("expected error when LLM provider fails")
	}
}

// TRACEABILITY_MAPPING: REQ-051
func TestBuildRemediationDraftWorkflowTemplates(t *testing.T) {
	llmCalls := 0
	llmSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		llmCalls++
		_, _ = w.Write([]byte(`{"content":"from the LLM"}`))
	}))
	defer llmSrv.Close()
	llmClient := llm.NewClient(llmSrv.URL, llmSrv.Client())
	settings := db.WorkspaceSettings{LLMAPIKey: "sk-1"}
	repo := db.Repository{FullName: "octo/demo"}

	paths := map[string]string{}
	for _, key := range []string{"openssf_scorecard", "release_automation"} {
		draft, err := BuildRemediationDraft(context.Background(), llmClient, settings, repo, key)
		if err != nil {
			t.Fatalf("%s: %v", key, err)
		}
		if !strings.HasPrefix(draft.CommitMessage, "ci:") || !strings.Contains(draft.Content, "runs-on: ubuntu-latest") || draft.BranchName != "ossalot/remediate-"+key {
			t.Fatalf("%s: unexpected draft %+v", key, draft)
		}
		paths[key] = draft.FilePath
	}
	if paths["openssf_scorecard"] != ".github/workflows/scorecard.yml" || paths["release_automation"] != ".github/workflows/release.yml" || llmCalls != 0 {
		t.Fatalf("unexpected paths %v or LLM calls %d", paths, llmCalls)
	}
	t.Logf("scorecard=%s release=%s llm_calls=%d", paths["openssf_scorecard"], paths["release_automation"], llmCalls)
}
