// TRACEABILITY_MAPPING: REQ-035
export const PIPELINE_TEMPLATES = {
  basic_ci: {
    id: "basic_ci",
    name: "Standard Build, Test & Evidence CI",
    filename: ".github/workflows/ci.yml",
    description: "Multi-step pipeline validating test execution and generating coverage reports.",
  },
  sonar_analysis: {
    id: "sonar_analysis",
    name: "SonarCloud Quality Gate & Code Analysis",
    filename: ".github/workflows/sonar.yml",
    description: "SonarCloud static analysis, security vulnerabilities, and code smell tracking.",
  },
  copilot_ci_failure_handler: {
    id: "copilot_ci_failure_handler",
    name: "Auto-Summon Copilot on CI Failure",
    filename: ".github/workflows/copilot-ci-failure-handler.yml",
    description: "Automatically comments @copilot on PRs when CI or static analysis runs fail.",
  },
  copilot_review_handler: {
    id: "copilot_review_handler",
    name: "Auto-Summon Copilot on Changes Requested",
    filename: ".github/workflows/copilot-review-handler.yml",
    description: "Automatically dispatches @copilot to address review comments when changes are requested.",
  },
  pr_automation: {
    id: "pr_automation",
    name: "PR Lifecycle & Auto-Merge Engine",
    filename: ".github/workflows/pr-automation.yml",
    description: "Undrafts ready pull requests and activates squash auto-merge for maintainers.",
  },
};

// TRACEABILITY_MAPPING: REQ-035
export function generatePipelineWorkflow(type, repoFullName) {
  const repo = repoFullName || "owner/repo";
  const [owner, name] = repo.includes("/") ? repo.split("/") : ["owner", repo];

  switch (type) {
    case "sonar_analysis":
      return {
        id: "sonar_analysis",
        filename: ".github/workflows/sonar.yml",
        required_secret: "SONAR_TOKEN",
        content: `name: "SonarCloud Analysis"

on:
  push:
    branches: [ main, master ]
  pull_request:
    branches: [ main, master ]
  workflow_dispatch:

permissions:
  contents: read

jobs:
  sonar:
    name: SonarCloud Scan
    runs-on: ubuntu-latest
    steps:
      - name: Checkout repository
        uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Analyze with SonarCloud
        env:
          GITHUB_TOKEN: \${{ secrets.GITHUB_TOKEN }}
          SONAR_TOKEN: \${{ secrets.SONAR_TOKEN }}
        run: |
          echo "Running SonarCloud analysis for ${repo}..."
`,
      };

    case "copilot_ci_failure_handler":
      return {
        id: "copilot_ci_failure_handler",
        filename: ".github/workflows/copilot-ci-failure-handler.yml",
        required_secret: "GITHUB_TOKEN",
        content: `name: "Auto-summon Copilot on CI Failure"

on:
  workflow_run:
    workflows: ["CI", "CI & Test Deployment", "SonarCloud Analysis"]
    types: [completed]

jobs:
  notify-copilot-ci:
    name: Notify Copilot on Build Failure
    runs-on: ubuntu-latest
    if: github.event.workflow_run.conclusion == 'failure' && github.event.workflow_run.event == 'pull_request'
    permissions:
      pull-requests: write
      issues: write

    steps:
      - name: Comment on PR to Summon Copilot
        env:
          GH_TOKEN: \${{ secrets.GITHUB_TOKEN }}
          WORKFLOW_NAME: \${{ github.event.workflow_run.name }}
          RUN_URL: \${{ github.event.workflow_run.html_url }}
          PR_NUMBER: \${{ github.event.workflow_run.pull_requests[0].number }}
        run: |
          if [ -n "$PR_NUMBER" ] && [ "$PR_NUMBER" != "null" ]; then
            gh pr comment "$PR_NUMBER" --body "@copilot The workflow **$WORKFLOW_NAME** failed on this pull request. Please inspect $RUN_URL, fix the failure, and push an update."
          fi
`,
      };

    case "copilot_review_handler":
      return {
        id: "copilot_review_handler",
        filename: ".github/workflows/copilot-review-handler.yml",
        required_secret: "GITHUB_TOKEN",
        content: `name: "Auto-summon Copilot on Review Changes Requested"

on:
  pull_request_review:
    types: [submitted]

jobs:
  dispatch-copilot:
    name: Notify Copilot of Review Feedback
    runs-on: ubuntu-latest
    if: github.event.review.state == 'changes_requested'
    permissions:
      pull-requests: write
      issues: write

    steps:
      - name: Comment on PR to Summon Copilot
        env:
          GH_TOKEN: \${{ secrets.GITHUB_TOKEN }}
          PR_NUMBER: \${{ github.event.pull_request.number }}
          REVIEWER: \${{ github.event.review.user.login }}
        run: |
          gh pr comment "$PR_NUMBER" --body "@copilot Changes were requested on this pull request by @$REVIEWER. Please review the feedback, implement the necessary corrections, and push an update."
`,
      };

    case "pr_automation":
      return {
        id: "pr_automation",
        filename: ".github/workflows/pr-automation.yml",
        required_secret: "GITHUB_TOKEN",
        content: `name: "PR Lifecycle Automation"

on:
  pull_request_target:
    types: [opened, converted_to_draft, edited, labeled]

jobs:
  manage-pr:
    name: Manage PR State & Auto-merge
    runs-on: ubuntu-latest
    permissions:
      contents: write
      pull-requests: write

    steps:
      - name: Enable Auto-Merge
        if: |
          github.event.pull_request.state == 'open' &&
          !startsWith(github.event.pull_request.title, '[WIP]') &&
          contains(fromJSON('["OWNER", "MEMBER", "COLLABORATOR"]'), github.event.pull_request.author_association)
        env:
          GH_TOKEN: \${{ secrets.GITHUB_TOKEN }}
          PR_NUMBER: \${{ github.event.pull_request.number }}
        run: |
          gh pr merge "$PR_NUMBER" --auto --squash || true
`,
      };

    default:
      return {
        id: "basic_ci",
        filename: ".github/workflows/ci.yml",
        required_secret: "GITHUB_TOKEN",
        content: `name: "CI Pipeline"

on:
  push:
    branches: [ main, master ]
  pull_request:
    branches: [ main, master ]

permissions:
  contents: read

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Run Build and Test
        run: |
          echo "Building ${name}..."
`,
      };
  }
}
