---
mcswe: "1.0"
scope:
  - cmd/
  - internal/
  - web/src/
  - .github/workflows/
out_of_scope:
  - .gitignore
  - IMPLEMENTATION_PLAN.md
  - go.mod
  - reports/
  - scripts/
  - web/dist/
documents:
  decisions: docs/mcswe/decisions.md
  requirements: docs/mcswe/requirements.md
  interfaces: docs/mcswe/interfaces.md
  exceptions: docs/mcswe/exceptions.md
  trace_matrix: docs/mcswe/trace-matrix.md
  audits: docs/mcswe/audits/
evidence:
  test_command: "python3 scripts/run_evidence.py test"
  test_report: "reports/junit.xml"
  coverage_command: "python3 scripts/run_evidence.py coverage"
  coverage_report: "reports/lcov.info"
  per_requirement_command: "python3 scripts/run_evidence.py per-req {tests}"
safety_critical: []         # Resolved in ADR-002: no safety-critical components
generators: []
audit_schedule: before each release
definitions_first_from: "4cff376"   # ADR-021: O6.1 is checked only for code added after this commit
banned_terms_extra: []
---

# MCSWE manifest

This project follows the MCSWE 1.0 standard. The block above is the machine-readable manifest (D3). Everything under `scope` must meet every objective; paths under `out_of_scope` are not yet governed.

## How work happens here

1. Definitions first: decisions, requirements and interfaces are reviewed and merged before code.
2. Implementation second: code and tests carry `TRACEABILITY_MAPPING:` anchors, and the audit runs before merge.
3. Behavior changes start as amendments to the requirements.

See `docs/mcswe/` for the required documents.
