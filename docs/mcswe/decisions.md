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

