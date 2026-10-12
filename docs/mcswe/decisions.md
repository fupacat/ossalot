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
context: The project needs every behavior traced from decision to code to test, with evidence, across both the Go backend (cmd/, internal/) and React/TypeScript frontend (web/src/).
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
