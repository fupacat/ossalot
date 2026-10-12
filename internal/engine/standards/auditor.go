package standards

import (
	"context"

	"github.com/eric/ossalot/internal/db"
	"github.com/eric/ossalot/internal/github"
	"github.com/eric/ossalot/internal/llm"
)

// TRACEABILITY_MAPPING: REQ-009
type AuditResult struct {
	OverallScore   int                  `json:"overall_score"`
	CategoryScores map[string]int       `json:"category_scores"`
	Checks         []db.CheckItemRecord `json:"checks"`
}

// TRACEABILITY_MAPPING: REQ-010
type RemediationDraft struct {
	CheckKey      string `json:"check_key"`
	FilePath      string `json:"file_path"`
	BranchName    string `json:"branch_name"`
	CommitMessage string `json:"commit_message"`
	PRTitle       string `json:"pr_title"`
	PRBody        string `json:"pr_body"`
	Content       string `json:"content"`
}

// TRACEABILITY_MAPPING: REQ-009
func EvaluateStandards(insp github.RepoInspection) AuditResult {
	panic("stub: REQ-009 not yet implemented")
}

// TRACEABILITY_MAPPING: REQ-010
func BuildRemediationDraft(ctx context.Context, llmClient *llm.Client, settings db.WorkspaceSettings, repo db.Repository, checkKey string) (RemediationDraft, error) {
	panic("stub: REQ-010 not yet implemented")
}
