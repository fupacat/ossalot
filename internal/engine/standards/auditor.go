package standards

import (
	"context"
	"errors"
	"fmt"
	"strings"

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
func boolCheck(key, category string, ok bool, weight int, passMsg, failMsg string) db.CheckItemRecord {
	if ok {
		return db.CheckItemRecord{
			CheckKey:        key,
			Category:        category,
			Status:          "pass",
			Score:           weight,
			Weight:          weight,
			Details:         passMsg,
			RemediationType: "generate_file",
		}
	}
	return db.CheckItemRecord{
		CheckKey:        key,
		Category:        category,
		Status:          "fail",
		Score:           0,
		Weight:          weight,
		Details:         failMsg,
		RemediationType: "generate_file",
	}
}

// TRACEABILITY_MAPPING: REQ-009
func threeStateCheck(key, category string, full, partial bool, weight, partialScore int, passMsg, warnMsg, failMsg string) db.CheckItemRecord {
	if full {
		return db.CheckItemRecord{
			CheckKey:        key,
			Category:        category,
			Status:          "pass",
			Score:           weight,
			Weight:          weight,
			Details:         passMsg,
			RemediationType: "generate_file",
		}
	}
	if partial {
		return db.CheckItemRecord{
			CheckKey:        key,
			Category:        category,
			Status:          "warn",
			Score:           partialScore,
			Weight:          weight,
			Details:         warnMsg,
			RemediationType: "generate_file",
		}
	}
	return db.CheckItemRecord{
		CheckKey:        key,
		Category:        category,
		Status:          "fail",
		Score:           0,
		Weight:          weight,
		Details:         failMsg,
		RemediationType: "generate_file",
	}
}

// TRACEABILITY_MAPPING: REQ-009
func EvaluateStandards(insp github.RepoInspection) AuditResult {
	fp := insp.FilesPresent
	hasLicense := insp.LicenseSPDX != "" || fp["LICENSE"]
	hasContrib := fp["CONTRIBUTING.MD"] || fp[".GITHUB/CONTRIBUTING.MD"]
	hasCoC := fp["CODE_OF_CONDUCT.MD"] || fp[".GITHUB/CODE_OF_CONDUCT.MD"]
	hasSec := fp["SECURITY.MD"] || fp[".GITHUB/SECURITY.MD"]
	hasIssueTpl := fp[".GITHUB/ISSUE_TEMPLATE/BUG_REPORT.MD"] || fp[".GITHUB/ISSUE_TEMPLATE.MD"]
	hasPRTpl := fp[".GITHUB/PULL_REQUEST_TEMPLATE.MD"]

	readmeUpper := strings.ToUpper(insp.ReadmeContent)
	readmeFull := len(insp.ReadmeContent) >= 80 && strings.Contains(readmeUpper, "INSTALL") && strings.Contains(readmeUpper, "USAGE")
	readmePartial := len(insp.ReadmeContent) > 0

	metaFull := insp.Description != "" && len(insp.Topics) > 0
	metaPartial := insp.Description != "" || len(insp.Topics) > 0

	checks := []db.CheckItemRecord{
		boolCheck("license_valid", "community_files", hasLicense, 5, "Valid open-source license detected", "Missing LICENSE file"),
		boolCheck("contributing_guide", "community_files", hasContrib, 5, "CONTRIBUTING.md present", "Missing CONTRIBUTING.md guide"),
		boolCheck("code_of_conduct", "community_files", hasCoC, 5, "CODE_OF_CONDUCT.md present", "Missing CODE_OF_CONDUCT.md"),
		boolCheck("security_policy", "community_files", hasSec, 5, "SECURITY.md policy present", "Missing SECURITY.md vulnerability policy"),
		threeStateCheck("issue_templates", "community_files", hasIssueTpl && hasPRTpl, hasIssueTpl || hasPRTpl, 5, 3, "Issue and PR templates present", "Partial issue/PR templates present", "Missing issue and PR templates"),
		threeStateCheck("readme_comprehensive", "documentation_quality", readmeFull, readmePartial, 15, 8, "README includes installation and usage sections", "README exists but lacks installation or usage sections", "Missing README.md"),
		threeStateCheck("repo_metadata_topics", "documentation_quality", metaFull, metaPartial, 10, 5, "Repository description and topics configured", "Repository missing either description or topics", "Repository missing description and topics"),
		threeStateCheck("ci_workflows", "technical_hygiene", insp.HasCIWorkflow && insp.RecentCIPassing, insp.HasCIWorkflow, 15, 8, "CI workflow present and passing", "CI workflow present but latest run not passing", "No GitHub Actions CI workflows detected"),
		threeStateCheck("release_automation", "technical_hygiene", insp.HasSemverRelease, insp.HasReleases, 10, 5, "Semantic Versioning releases published", "Releases present without vX.Y.Z semver tag", "No GitHub releases published"),
		threeStateCheck("openssf_scorecard", "security_openssf", insp.OpenSSFScore >= 7.0, insp.OpenSSFScore >= 4.0, 15, 8, "OpenSSF Scorecard >= 7.0", "OpenSSF Scorecard between 4.0 and 6.9", "OpenSSF Scorecard < 4.0 or unavailable"),
		boolCheck("dependency_automation", "security_openssf", insp.HasDependabot, 10, "Automated dependency updates configured", "Missing Dependabot or Renovate configuration"),
	}

	catScores := map[string]int{
		"community_files":       0,
		"documentation_quality": 0,
		"technical_hygiene":     0,
		"security_openssf":      0,
	}
	overall := 0
	for _, c := range checks {
		catScores[c.Category] += c.Score
		overall += c.Score
	}
	return AuditResult{
		OverallScore:   overall,
		CategoryScores: catScores,
		Checks:         checks,
	}
}

// TRACEABILITY_MAPPING: REQ-010
var checkFilePaths = map[string]string{
	"license_valid":         "LICENSE",
	"contributing_guide":    "CONTRIBUTING.md",
	"code_of_conduct":       "CODE_OF_CONDUCT.md",
	"security_policy":       "SECURITY.md",
	"issue_templates":       ".github/ISSUE_TEMPLATE/bug_report.md",
	"pr_template":           ".github/PULL_REQUEST_TEMPLATE.md",
	"ci_workflows":          ".github/workflows/ci.yml",
	"readme_comprehensive":  "README.md",
	"dependency_automation": ".github/dependabot.yml",
}

// TRACEABILITY_MAPPING: REQ-051
type workflowTemplate struct {
	filePath string
	content  string
}

// TRACEABILITY_MAPPING: REQ-051
var workflowTemplates = map[string]workflowTemplate{
	"openssf_scorecard": {
		filePath: ".github/workflows/scorecard.yml",
		content: `# OpenSSF Scorecard Workflow
name: Scorecard supply-chain security
on:
  branch_protection_rule:
  schedule:
    - cron: '30 1 * * 6'
  push:
    branches: [ "main", "master" ]
permissions: read-all
jobs:
  analysis:
    name: Scorecard analysis
    runs-on: ubuntu-latest
    permissions:
      security-events: write
      id-token: write
    steps:
      - name: "Checkout code"
        uses: actions/checkout@v4
      - name: "Run analysis"
        uses: ossf/scorecard-action@v2.4.0
        with:
          results_file: results.sarif
          results_format: sarif
          publish_results: true
`,
	},
	"release_automation": {
		filePath: ".github/workflows/release.yml",
		content: `# Automated Release Workflow
name: Release
on:
  push:
    tags:
      - 'v*'
permissions:
  contents: write
jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - name: Create Release
        uses: softprops/action-gh-release@v2
        with:
          generate_release_notes: true
`,
	},
}

// TRACEABILITY_MAPPING: REQ-010, REQ-051
func BuildRemediationDraft(ctx context.Context, llmClient *llm.Client, settings db.WorkspaceSettings, repo db.Repository, checkKey string) (RemediationDraft, error) {
	// Workflow checks get fixed templates; the LLM provider is never asked for CI configuration
	if tpl, ok := workflowTemplates[checkKey]; ok {
		return RemediationDraft{
			CheckKey:      checkKey,
			FilePath:      tpl.filePath,
			BranchName:    fmt.Sprintf("ossalot/remediate-%s", checkKey),
			CommitMessage: fmt.Sprintf("ci: add %s via OSSAlot remediation", tpl.filePath),
			PRTitle:       fmt.Sprintf("chore(oss): add %s to improve OSS standards compliance", tpl.filePath),
			PRBody:        fmt.Sprintf("Automated OSS standards remediation for check `%s` on `%s`.", checkKey, repo.FullName),
			Content:       tpl.content,
		}, nil
	}
	filePath, ok := checkFilePaths[checkKey]
	if !ok {
		return RemediationDraft{}, errors.New("unsupported check key for file remediation")
	}
	content, err := llmClient.GenerateFile(ctx, llm.GenerateRequest{
		Provider:        settings.LLMProvider,
		APIKey:          settings.LLMAPIKey,
		Model:           settings.LLMStrongModel,
		RepoFullName:    repo.FullName,
		RepoDescription: repo.Description,
		PrimaryLanguage: repo.PrimaryLanguage,
		CheckKey:        checkKey,
	})
	if err != nil {
		return RemediationDraft{}, err
	}
	return RemediationDraft{
		CheckKey:      checkKey,
		FilePath:      filePath,
		BranchName:    fmt.Sprintf("ossalot/remediate-%s", checkKey),
		CommitMessage: fmt.Sprintf("docs: add %s via OSSAlot standards remediation", filePath),
		PRTitle:       fmt.Sprintf("chore(oss): add %s to improve OSS standards compliance", filePath),
		PRBody:        fmt.Sprintf("Automated OSS standards remediation for check `%s` on `%s`.", checkKey, repo.FullName),
		Content:       content,
	}, nil
}
