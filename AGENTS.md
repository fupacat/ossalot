# AGENTS.md

## MCSWE

This repository follows the MCSWE 1.0 standard; the manifest is `MCSWE.md`.

- Merge definitions before code: decisions, requirements and interfaces in `docs/mcswe/` are reviewed and merged before any implementation of them.
- Put a `TRACEABILITY_MAPPING: REQ-NNN` comment above every function, class, block or job you add or change in scope, and above every test.
- Start any behavior change by amending `docs/mcswe/requirements.md`. Never reuse a requirement ID.
- Before asking for merge, run the audit: the manifest's test and coverage commands, then regenerate `docs/mcswe/trace-matrix.md` and write an audit record. Never edit those two by hand.
- Approval is the owner merging the pull request on GitHub (ADR-033). Don't write approval trailers, and don't merge a pull request unless the owner asks for that specific merge. Before the merge, the audit lists the branch's commits as pending approval.
