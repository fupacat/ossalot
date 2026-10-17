# Decisions

Each decision has an `ADR-NNN` heading and one YAML block (D4). Only resolved decisions may be cited as a requirement's source.

## Bounds

What this project will not do (D4.4):

- Automatically submit posts, comments, or pull requests to external unowned repositories or third-party communities (Reddit, Hacker News, Stack Overflow, Dev.to, Lobsters) without explicit human review and action.
- Require CGO or an external database server for default single-user local execution.
- Expose workspace credentials (GitHub tokens, LLM API keys, webhook URLs) in plaintext API responses or logs.

## ADR-001 Adopt MCSWE 1.0

```yaml
status: resolved
context: The project needs every behavior traced from decision to code to test, with evidence, across both the Go backend (cmd/, internal/) and frontend (web/src/).
decision: Follow the MCSWE 1.0 standard for everything under the manifest's scope.
consequences: Definitions merge before code; every in-scope line is anchored; the audit runs before every merge.
superseded_by: null
```

## ADR-002 Safety-critical components

```yaml
status: resolved
context: O2.6 requires MC/DC coverage for safety-critical components, and O2.7 requires this choice to be recorded either way.
decision: No components in OSSAlot are safety-critical; OSSAlot is an open-source repository analytics, standards auditing, and community marketing platform whose failure cannot cause physical harm or safety hazards.
consequences: The standard 100% line and branch coverage bar from requirements-based tests (O2.5) applies to all in-scope components; MC/DC coverage (O2.6) is not required.
superseded_by: null
```

## ADR-003 Go standard-library HTTP server and single-binary architecture

```yaml
status: resolved
context: OSSAlot requires a single-binary backend with zero external runtime service dependencies, deterministic routing, and environment-driven configuration.
decision: Build the backend in Go 1.22+ using net/http pattern routing, structured JSON APIs under /api/v1, and environment variable configuration (OSSALOT_ADDR, OSSALOT_DB_PATH, OSSALOT_WORKSPACE).
consequences: Compiles to a self-contained binary with zero third-party router dependencies and predictable request handling.
superseded_by: null
```

## ADR-004 Tenant-aware persistence store with masked credentials

```yaml
status: resolved
context: OSSAlot must support zero-login personal mode via a default workspace while keeping a multi-tenant data model for repositories, standards audits, and workspace credentials.
decision: Persist workspaces, workspace settings, repositories, standards audits, check items, and scan jobs in a tenant-scoped store that automatically bootstraps the "default" workspace and masks non-empty tokens/keys as "***" on external reads.
consequences: Local users get immediate persistence without manual setup, and sensitive tokens are never leaked via settings GET responses.
superseded_by: null
```

## ADR-005 Rate-limited GitHub REST integration and owned-repo guardrails

```yaml
status: resolved
context: OSSAlot inspects public and owned GitHub repositories, checks OpenSSF Scorecard results, and opens remediation pull requests only on repositories owned by the user.
decision: Use a rate-limited GitHub REST client that parses repository URLs or owner/name pairs, fetches repository metadata/files/workflows/releases/scorecards, and enforces is_owned == true before creating any remediation branch or pull request.
consequences: Prevents accidental pull request creation on unowned repositories while supporting full standards inspection on any public repository.
superseded_by: null
```

## ADR-006 Four-pillar OSS Standards Scorecard and Remediation Engine

```yaml
status: resolved
context: Maintainers need an objective 0-100 standards score across community governance, documentation quality, technical hygiene, and supply-chain checks, paired with actionable file generation for missing artifacts.
decision: Evaluate repositories across four 25-point pillars (community_files, documentation_quality, technical_hygiene, security_openssf) producing pass/warn/fail check items and generating tailored markdown/workflow files for any non-passing check.
consequences: Every repository receives a reproducible 0-100 standards score and ready-to-commit remediation files.
superseded_by: null
```

## ADR-007 Pluggable LLM provider with deterministic template fallback

```yaml
status: resolved
context: Users may configure Gemini, OpenAI, Anthropic, or Ollama for AI-authored remediation files, or run OSSAlot without an LLM API key configured.
decision: Define a unified LLM provider interface supporting gemini, openai, anthropic, and ollama, falling back to deterministic repository-aware markdown templates when no API key is configured.
consequences: Remediation works out-of-the-box without mandatory paid API keys while supporting LLM synthesis when configured.
superseded_by: null
```

## ADR-008 Frontend portfolio and scorecard state model

```yaml
status: resolved
context: The web UI needs deterministic state transformations for portfolio health summaries, check status badge mapping, and remediation payload construction.
decision: Implement a modular frontend domain model in web/src/model.mjs that computes portfolio aggregates, normalizes check statuses, and builds remediation payloads consumed by the SPA views.
consequences: Frontend logic is testable in isolation under Node's test runner with 100% line and branch coverage.
superseded_by: null
```

## ADR-009 Daily metric snapshots and 14-day GitHub traffic persistence

```yaml
status: resolved
context: GitHub's traffic API only retains views, clones, and top referrers for 14 days, so historical adoption trends are lost unless archived into daily snapshots alongside package registry download counts.
decision: Persist date-keyed MetricSnapshot records per repository (YYYY-MM-DD) storing stars, forks, watchers, open issues, views, unique views, clones, unique clones, top referrers, registry downloads across ecosystems (npm, pypi, crates, homebrew, docker, golang), and a composite 0-100 adoption score.
consequences: Repeat collections on the same UTC date upsert the daily snapshot idempotently while preserving multi-month traffic and download history beyond GitHub's 14-day window.
superseded_by: null
```

## ADR-010 Frontend time-series chart and referrer attribution model

```yaml
status: resolved
context: The Repo Detail view requires chronologically ordered time-series points for Recharts and aggregated referrer traffic totals across snapshot windows.
decision: Provide pure functions in web/src/metrics_model.mjs to sort and project daily metric snapshots into chart series and to sum referrer counts and unique visitors across snapshots.
consequences: Time-series transformation and referrer aggregation are deterministic and verifiable under Node's test runner.
superseded_by: null
```

## ADR-011 Repository capability profiling and opportunity store

```yaml
status: resolved
context: Keyword-only search produces noisy, low-value matches; OSSAlot needs a structured CapabilityProfile per repository and a deduplicated Opportunity store.
decision: Extract a structured CapabilityProfile (summary, problem statements, target personas, ecosystem keywords, search queries, recommended communities) from repository metadata and README content, and persist profiles and deduplicated Opportunity records keyed by external_id.
consequences: Enables targeted GitHub issue discovery and community listening without duplicate inbox entries.
superseded_by: null
```

## ADR-012 Relevance-gated GitHub opportunity discovery and PR plan synthesis

```yaml
status: resolved
context: Suggesting a library on tangential GitHub issues is spammy; maintainers need high-signal matches (>= 70 relevance) with a concrete PR implementation plan.
decision: Query GitHub /search/issues using capability profile queries, exclude issues belonging to the source repository itself, score relevance on a 0-100 scale, retain only matches scoring >= 70, and synthesize a PRImplementationPlan and technical draft comment.
consequences: Only genuine external matches with actionable integration plans enter the opportunity inbox.
superseded_by: null
```

## ADR-013 Dual-mode community listening and marketing playbook engine

```yaml
status: resolved
context: Open-source maintainers need to discover active discussions asking for solutions their project provides across Hacker News, Reddit, Stack Overflow, Dev.to, and Lobsters, and draft non-spammy community launch posts that respect each platform's self-promotion norms.
decision: Implement a multi-platform community feed scanner and a deterministic marketing playbook generator in internal/engine/community/ producing Opportunity records with community_rules_notes and platform-native drafts.
consequences: Both inbound community threads and outbound launch playbooks flow into the unified Opportunity inbox.
superseded_by: null
```

## ADR-014 Webhook notifications, job scheduler, and campaign attribution inbox

```yaml
status: resolved
context: High-relevance opportunities (>= 85) should optionally trigger Slack/Discord webhook alerts, background scans need job status tracking, and submitted posts should be correlated against traffic and star deltas.
decision: Implement a threshold-gated webhook notifier (internal/engine/notifier/), an in-memory/persisted job runner (internal/scheduler/), unified inbox and scan API endpoints (internal/api/community_handlers.go), and a frontend inbox and attribution calculator (web/src/inbox_model.mjs).
consequences: Completes the end-to-end discovery, notification, action workflow, and attribution loop.
superseded_by: null
```

## ADR-015 Automated CI and test deployment pipeline

```yaml
status: resolved
context: To minimize manual interventions and maintain rapid iteration, every push and pull request must automatically run the full MCSWE test and coverage evidence suite and build deployable distribution artifacts for the test environment.
decision: Define a GitHub Actions workflow in .github/workflows/ci.yml that checks out the repository, installs Go and Node.js dependencies, verifies MCSWE evidence (test and 100% coverage), builds the single-binary OSSAlot executable, and publishes the test release artifact.
consequences: Regressions are detected automatically on PRs and pushes; test environment binaries are built deterministically in CI without manual local intervention.
superseded_by: null
```

## ADR-016 Granular setup, permission-gated capability model, and extensible credentials

```yaml
status: resolved
context: Maintainers need transparent setup guidance that distinguishes base read-only repository inspection from mutation actions (opening PRs, updating workflow files, SonarQube integration, package registries, and LLM reasoning) with capability gating that clearly indicates which secrets unlock which features.
decision: Extend WorkspaceSettings with SonarQube credentials (sonar_token, sonar_host_url, sonar_org), custom API keys, and implement a pure setup model in web/src/setup_model.mjs that computes capability gates (inspect, remediate_standard, remediate_workflows, sonarqube, webhooks, llm_synthesis) and reports prerequisite completion status.
consequences: Users can clearly see base access vs gated requirements; sensitive credentials are stored securely and masked on read.
superseded_by: null
```

## ADR-017 CI and automated pipeline templates

```yaml
status: resolved
context: Projects need repeatable, automated CI/CD and bot automation templates (such as the multi-job matrix, SonarCloud analysis, auto-Copilot CI failure responder, PR lifecycle auto-merge, and release automation seen on tsql-scriptdom-java) that can be inspected and generated on demand.
decision: Implement a pipeline template catalog and model in web/src/pipeline_model.mjs that formats ready-to-commit GitHub Actions workflow configurations for standard CI, SonarCloud, Copilot summoning, PR automation, and release flows.
consequences: Maintainers can choose between baseline validation and full autonomous pipeline automation with zero boilerplate guesswork.
superseded_by: null
```

## ADR-018 Code hygiene and agent-readiness verification configs

```yaml
status: resolved
context: To match the verification infrastructure and agent-readiness standards analyzed by girdle, projects need structured configurations for linters, formatters, editorconfig, gitattributes, git hooks, and structural indexing.
decision: Implement a code hygiene generator in web/src/hygiene_model.mjs that generates tailored configurations (.editorconfig, .gitattributes, pre-commit config, and linter/formatter configs) across common language ecosystems.
consequences: Maintains consistent code formatting, drift prevention, and agent-readiness verification across tracked repositories.
superseded_by: null
```

## ADR-019 Automated SonarCloud onboarding detection, registration deep-links, and local Community Edition guide

```yaml
status: resolved
context: Setting up SonarQube/SonarCloud can be daunting for maintainers. OSSAlot should detect if the repository owner already has a SonarCloud organization, whether the project has been imported, offer direct GitHub OAuth registration/import links for public projects, and generate turnkey Docker instructions for local Community Edition instances.
decision: Implement SonarCloud onboarding inspection functions in web/src/sonar_model.mjs that parse organization and component status payloads, compute registration and project creation deep-links, and output Docker instructions for local SonarQube Community Edition.
consequences: Streamlines static analysis setup for both free cloud users and self-hosted on-premises users.
superseded_by: null
```

## ADR-020 Token and GitHub secret deep-links for SonarCloud onboarding

```yaml
status: resolved
context: After a SonarCloud project is imported, the analysis job still needs a SONAR_TOKEN secret in the GitHub repository. Maintainers currently have to find the token page and the secrets page themselves. This was first proposed as an amendment to REQ-037, which is superseded here so that the REQ-037 identifier keeps its committed meaning.
decision: Extend evaluateSonarOnboarding in web/src/sonar_model.mjs to return the SonarCloud token generation URL and the GitHub Actions new-secret URL for the repository, with the secret name SONAR_TOKEN.
consequences: Maintainers can finish SonarCloud setup from the dashboard without searching for settings pages. The links are fixed URL patterns, so they are not checked against live SonarCloud or GitHub pages.
superseded_by: null
```

## ADR-021 Enforcement start for the definitions-first history check

```yaml
status: resolved
context: O6.1 requires each requirement's definition to be approved before its code enters scope. The repository adopted MCSWE in 743ad99, but its Phase 1 to Phase 3 history places definitions and code in the same commits, and the standard's history check was added in 4cff376. Checking history from before that point would fail on work done before the rule was checked.
decision: The O6.1 history check runs only for code that entered scope after commit 4cff376 (set as definitions_first_from in MCSWE.md). Earlier commits are not tested by the check. Their findings remain visible in the audit history, and the standard's exception rules do not apply to O6.1 (X1.3).
consequences: REQ-001 to REQ-031 are not flagged by the history check. Work from 4cff376 onward is checked. This baseline should be reviewed by the project owner before the next release audit.
superseded_by: null
```

## ADR-022 Single server entry point

```yaml
status: resolved
context: The app users run is scripts/preview_server.go, which is outside the manifest's scope, untested, and built with the ignore tag. It registers the metrics, discovery, community, inbox and job routes, replaces the remediation handler, wraps the GitHub, registry and community HTTP clients, loads the GitHub token, starts an enrichment job and serves the UI. cmd/ossalot registers only the base routes and serves no UI, so the governed binary cannot run the app.
decision: cmd/ossalot becomes the only server. It wires every API route, the scheduler, the startup enrichment job and the UI from one listener on the configured address (REQ-001 default 127.0.0.1:8080). Behavior the preview server adds moves into cmd/ and internal/ under the requirements that cite this decision. scripts/preview_server.go is deleted once nothing depends on it.
consequences: The code users run is the code the audit governs. Running the app no longer depends on go run of an ignored script. Behavior not carried forward by ADR-024 disappears with the preview server.
superseded_by: null
```

## ADR-023 UI assets embedded in the binary

```yaml
status: resolved
context: web/dist/index.html imports its helper modules from /src/*.mjs at runtime, so the UI needs both web/dist and web/src served. The preview server read both from disk relative to its working directory, which breaks when the binary runs anywhere else, such as a container.
decision: A Go package at web/embed.go embeds web/dist and the web/src modules with go:embed. The server serves web/dist at / and web/src at /src/, with .mjs files served as text/javascript. Test files under web/src are not embedded. web/embed.go joins the manifest's scope as the webassets component.
consequences: One binary carries the UI and runs from any directory. UI changes need a rebuild of the binary. Phase 2 of the architecture plan replaces the embedded files with a Vite build output but keeps this serving contract.
superseded_by: null
```

## ADR-024 Live provider behavior carried forward from the preview server

```yaml
status: resolved
context: The preview server's HTTP wrappers do three kinds of work. Some fix real GitHub API behavior (default branches, missing head branches, existing files and pull requests, the workflow OAuth scope, traffic endpoints on repositories the token cannot push to). Some translate the internal /feeds and /downloads contracts to real public APIs. Some change data to pass filters, which the requirements do not allow - appending the search keyword to issue bodies and adding an enhancement label so issues pass the discovery threshold of 70 (REQ-023), rewriting language:GAP to "tsql OR antlr" for one repository, replacing the keyword "ossalot" with "open source maintainer", and guessing that a repository's npm package and Go module share its name during enrichment.
decision: Carry forward, as explicit and tested behavior in internal/github, internal/engine/metrics and internal/engine/community, the GitHub API fixes, the traffic fallback, the zero-result issue search retry, and the registry and community feed translations. Do not carry forward the injected keywords, the synthetic labels, the repository-specific query rewrites or the package-name guesses. The community feed translation keeps its fixed per-platform scores and labels the lobsters platform, which has no search API and is answered from Ask HN, as Ask HN.
consequences: Discovery shows only issues that meet REQ-023 on their real content, so fewer opportunities may appear than with the preview server. Package download counts appear only for packages the caller names. The fixed community scores remain heuristics, not measured relevance; a later decision can replace them.
superseded_by: null
```

## ADR-025 GitHub token source

```yaml
status: resolved
context: The preview server took the GitHub token from the GITHUB_TOKEN environment variable or from gh's ~/.config/gh/hosts.yml, and saved it into the workspace settings. Reading hosts.yml only works on a machine logged in to gh, and saving an environment token into the store copies a credential the operator did not ask to store. A deployed container gets its credentials from its environment.
decision: GitHub requests use the workspace settings token when one is set, and otherwise the GITHUB_TOKEN environment variable. The environment token is never written to the store. gh's hosts.yml is not read.
consequences: Deployments pass GITHUB_TOKEN in the environment. Local users who relied on gh login export GITHUB_TOKEN, for example from gh auth token. A token saved in settings still takes precedence.
superseded_by: null
```

## ADR-026 Startup portfolio enrichment

```yaml
status: resolved
context: The preview server registered a daily_portfolio_sync job that audits, collects metrics for, profiles, discovers issues for, scans communities for and generates playbooks for every tracked repository, and ran it once at startup. Nothing schedules it daily despite its name. The jobs API (REQ-030) lists it and can run it on demand.
decision: cmd/ossalot registers the daily_portfolio_sync job with those six steps for each tracked repository and triggers it once in the background when the server starts. Metrics collection in this job names no packages (ADR-024). The name is kept because the jobs API exposes it. Periodic scheduling is out of scope for this decision.
consequences: Data is fresh after each start without a manual run. Startup makes GitHub and community API calls for every tracked repository.
superseded_by: null
```

## ADR-027 Workflow remediation templates

```yaml
status: resolved
context: The standards auditor reports openssf_scorecard and release_automation checks (REQ-009), but the remediator has no file for them, so remediation returns "unsupported check key". The preview server answered these two keys itself with fixed GitHub Actions workflows.
decision: The remediator returns fixed workflow drafts for openssf_scorecard (.github/workflows/scorecard.yml, running ossf/scorecard-action) and release_automation (.github/workflows/release.yml, creating a GitHub release on v* tags), without calling the LLM provider, with ci-prefixed commit messages.
consequences: Every check the auditor can fail on these two keys has a remediation. Creating these files on GitHub needs a token with the workflow scope (REQ-046).
superseded_by: null
```

## ADR-028 Vite build for the frontend

```yaml
status: resolved
context: The UI exists only as hand-edited build output, web/dist/index.html, with no source. Tailwind comes from the Play CDN and compiles styles in the browser, so pages need the CDN at runtime. There is no dev server or hot reload. This is phase 2 of docs/architecture-plan.md.
decision: web/ becomes a Vite project. The markup moves to web/index.html and the page's inline module script moves unchanged to web/app/main.js; the web/src models it imports are bundled. npm run build writes web/dist/, which is generated and ignored by git except for a placeholder file. Tailwind 4 compiles at build time through its Vite plugin, scanning web/index.html, web/app/ and web/src/, with the page's customizations (class dark mode, slate-850 and slate-950) carried into the CSS entry. The page's current look is an iteration mock with no design intent, so differences from the Play CDN's Tailwind 3 output, such as the version 4 palette, are accepted. TypeScript is configured for new code, but no file is converted in this phase. The model tests stay on node --test until the TypeScript port in phase 4 moves them to Vitest.
consequences: UI changes edit source, and web/dist is regenerated. Pages render without the Tailwind CDN. Colors and spacing may shift slightly with Tailwind 4; the design phase sets the real look. A Go build embeds the real UI only after npm ci and npm run build. Moving the model tests to Vitest is deferred from phase 2, where the architecture plan placed it.
superseded_by: null
```

## ADR-029 Serving the built UI

```yaml
status: resolved
context: ADR-023 embeds web/dist and the web/src modules and serves /src/ because the page imported the modules at runtime. Vite bundles those modules into web/dist/assets/, so /src/ is no longer requested. web/dist is now generated, and go:embed refuses to compile a pattern that matches no files.
decision: web/embed.go embeds all of web/dist, including a committed placeholder file, so Go compiles on a fresh clone without a frontend build. The server serves web/dist at / and stops serving /src/; this replaces the /src/ part of ADR-023, whose decision to embed the UI in the binary stands. When the embedded build has no index.html, / answers 503 with a message naming the frontend build command.
consequences: go build and go test work without Node. A binary built without the frontend build serves the 503 message until it is rebuilt with the UI. Release builds, in CI and later in the container, run the frontend build first.
superseded_by: null
```

## ADR-030 Frontend build in CI

```yaml
status: resolved
context: CI (REQ-032) builds the test binary with go build. With web/dist generated, a binary built without the frontend build serves only the 503 message (ADR-029).
decision: The CI workflow runs npm ci and npm run build in web/ after setting up Node and before the evidence steps and the Go build, so the test artifact embeds the real UI.
consequences: CI depends on web/package-lock.json being committed. A broken frontend build fails CI before the Go steps run.
superseded_by: null
```

## ADR-031 Local development loop

```yaml
status: resolved
context: Iterating on the UI needs hot reload. Today every UI change means editing the built file and reloading against a running Go server.
decision: make dev runs the Go server (cmd/ossalot on 127.0.0.1:8080) and the Vite dev server (127.0.0.1:5173) together. Vite proxies /api to :8080, updates styles in place and reloads the page when the script changes. make build runs the frontend build, then go build. The Makefile and the web build configuration (package.json, package-lock.json, and the Vite and TypeScript configuration) are tooling outside the manifest's scope, like scripts/.
consequences: During development the UI is opened on :5173; :8080 serves whatever UI the running binary embeds. Tooling changes need no requirement amendments.
superseded_by: null
```

## ADR-032 The page script stays outside scope until it is ported

```yaml
status: resolved
context: Moving the inline script (about 780 lines) and the markup into web/ turns them into source. The manifest's scope covers web/src/, so placing them there would require requirements, anchors and full line and branch coverage for nine tabs whose code phase 4 rewrites in React, tab by tab. Today the same code is outside scope as web/dist.
decision: The markup lives in web/index.html and the script in web/app/; both are listed as out of scope. Each tab enters scope when phase 4 ports it, with its requirements defined first. Logic that needs governance before then moves into the web/src models.
consequences: This phase does not change how much of the page is governed; the existing gap becomes explicit in the manifest instead of hidden in a build output.
superseded_by: null
```

## ADR-033 Approval is the owner's pull request merge

```yaml
status: resolved
context: Approvals were recorded as Approved-by and Approves trailers on extra commits. For a solo project that meant an empty approval commit for every change, written by the agent at the owner's request, and a merge that raced ahead of its approval left main unapproved (#7, #8). Merging a reviewed pull request on GitHub is already the owner's explicit acceptance of the change.
decision: A pull request merged on GitHub approves every commit it brings in. The audit reads approvals from merge commits on the mainline's first-parent history that GitHub committed with a "Merge pull request #N" subject; the merge's author is the approver, and an approver who wrote the change is recorded as a self-approval (O4.4). Commits that have not reached the mainline are reported as pending approval, not as findings. Commits that reach the mainline without a pull request merge remain unapproved findings. Approved-by trailers stay valid. Agents do not merge pull requests unless the owner asks for that specific merge, and do not write approval trailers.
consequences: No approval commits. The audit before a merge reports the branch's own commits as pending. Every change to main goes through a pull request merged on GitHub; squash and rebase merges are not recognized and would show as unapproved. Earlier merged pull requests now count as approvals, which clears the outstanding O4.1 on 91c4f12.
superseded_by: null
```

## ADR-034 Issue dispatching to repository agents for context-aware remediation and automation

```yaml
status: resolved
context: Blindly creating pull requests with static templates overwrites existing repository files, destroying existing working CI pipelines (such as Node builds and test suites) or generating mismatched formats (such as README markdown in dependabot.yml). Modern repositories have resident coding agents (such as GitHub Copilot Workspace, @copilot, or CI bot automations) that have full codebase context, understand package managers, and can inspect existing configurations.
decision: Implement an Issue Dispatcher in the GitHub client (CreateIssue) and API server (POST /api/v1/repos/{id}/issues). The action files a structured GitHub issue that tags @copilot with technical objectives, repository context, non-destructive guardrails (do not overwrite existing working CI/build steps), and reference templates. Owned-repo enforcement is preserved.
consequences: Maintainers can trigger standards remediation and pipeline automation safely without risk of destructive file overwrites. The resident coding agent performs context-aware analysis and opens a verified, non-destructive pull request.
superseded_by: null
```

## ADR-035 Dev environment on a Cloudflare Tunnel, pulled from CI

```yaml
status: resolved
context: The owner wants a hosted dev instance on Cloudflare that follows the latest build. OSSAlot is a Go server keeping its data in files on disk, so it cannot run on Workers or Pages, and Cloudflare Containers have ephemeral disk. The app holds a GitHub token that can create issues and pull requests.
decision: The dev instance runs on a host the owner controls, exposed through a Cloudflare Tunnel and gated by a Cloudflare Access policy for the owner's email. A systemd timer on the host downloads the ossalot-test-binary artifact of the latest successful push run of ci.yml on main, runs it with -check, and restarts the service when the commit changed. The tooling lives in scripts/deploy/, outside the manifest's scope; ci.yml is unchanged.
consequences: No Cloudflare or host credentials in GitHub, and no inbound port on the host. Deploys lag main by up to five minutes. Data persists on the host across deploys. A host offline longer than the 7-day artifact retention waits for the next push to main.
superseded_by: null
```
