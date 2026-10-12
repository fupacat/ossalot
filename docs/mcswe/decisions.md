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
