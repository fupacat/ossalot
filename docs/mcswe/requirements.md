# Requirements

Each requirement is one line: its ID, then one statement containing SHALL, followed by one YAML block with its source, status and acceptance record (D5). One observable behavior per requirement. IDs are never reused; retired requirements stay, marked `retired`.

REQ-001 The configuration loader SHALL return default values (`127.0.0.1:8080`, `ossalot.db`, `default`) when environment variables are unset and override each field when `OSSALOT_ADDR`, `OSSALOT_DB_PATH`, or `OSSALOT_WORKSPACE` is set.

```yaml
source: ADR-003
status: active
acceptance:
  kind: categorical
  precondition: Environment variables OSSALOT_ADDR, OSSALOT_DB_PATH, and OSSALOT_WORKSPACE are either unset or populated with custom values.
  action: Call LoadFromEnv with an environment lookup function.
  expected: Returns Config matching defaults when unset and custom values when set.
  method: test
  expected_output: "addr=127.0.0.1:8080 db=ossalot.db workspace=default"
```

REQ-002 The database store SHALL initialize schema storage and bootstrap the `default` personal workspace idempotently on open.

```yaml
source: ADR-004
status: active
acceptance:
  kind: categorical
  precondition: A new or existing database path is provided to OpenStore.
  action: Open the store twice and query the default workspace.
  expected: The default workspace exists with slug "default" and mode "personal" without duplicate records.
  method: test
  expected_output: "slug=default mode=personal"
```

REQ-003 The database store SHALL persist workspace settings and mask non-empty `github_token` and `llm_api_key` values as `"***"` when returning redacted settings.

```yaml
source: ADR-004
status: active
acceptance:
  kind: categorical
  precondition: Workspace settings are saved with a non-empty GitHub token and LLM API key.
  action: Save settings and call GetSettingsRedacted alongside GetSettingsRaw.
  expected: GetSettingsRedacted returns "***" for both secrets while GetSettingsRaw retains the original credentials, and updating with "***" preserves existing credentials.
  method: test
  expected_output: "github_token=*** llm_api_key=***"
```

REQ-004 The database store SHALL upsert, list, retrieve, and delete repository records and their standards audits scoped to a workspace.

```yaml
source: ADR-004
status: active
acceptance:
  kind: categorical
  precondition: An initialized store with the default workspace.
  action: Upsert a repository, save a standards audit with check items, list and get the repository, then delete it.
  expected: Repository and audit records are persisted, retrieved with matching scores and check items, and removed upon deletion.
  method: test
  expected_output: "repos=1 audit_score=85 after_delete=0"
```

REQ-005 The GitHub coordinate parser SHALL extract owner and repository name from `https://github.com/<owner>/<name>` URLs or `<owner>/<name>` strings and return an error for invalid inputs.

```yaml
source: ADR-005
status: active
acceptance:
  kind: categorical
  precondition: Input strings representing full GitHub URLs, shorthand owner/repo pairs, or malformed strings.
  action: Call ParseRepoCoordinates on each input.
  expected: Valid inputs return owner and name with .git suffix stripped; invalid inputs return a non-nil error.
  method: test
  expected_output: "owner=octocat name=hello-world"
```

REQ-006 The GitHub client SHALL fetch repository metadata, file presence, README content, workflow runs, releases, and OpenSSF Scorecard data from the configured API base URLs.

```yaml
source: ADR-005
status: active
acceptance:
  kind: categorical
  precondition: Mock HTTP servers serving GitHub REST v3 and OpenSSF Scorecard responses.
  action: Call InspectRepository and ListAuthenticatedRepos on the client.
  expected: Returns populated RepoInspection and repository list matching the upstream HTTP responses.
  method: test
  expected_output: "full_name=octocat/hello-world scorecard=7.5"
```

REQ-007 The GitHub client SHALL create a branch, commit a remediation file, and open a pull request when `is_owned` is true, and return an error without network mutation when `is_owned` is false.

```yaml
source: ADR-005
status: active
acceptance:
  kind: categorical
  precondition: A target repository with is_owned set to true or false and a mock GitHub REST server.
  action: Call CreateRemediationPR for an owned repository and for an unowned repository.
  expected: Owned repository returns the created pull request URL; unowned repository returns ErrRepoNotOwned.
  method: test
  expected_output: "pr_url=https://github.com/octocat/hello-world/pull/1"
```

REQ-008 The LLM provider SHALL generate remediation content via the configured provider HTTP endpoint when an API key is present and return a deterministic repository-specific markdown template when no API key is configured.

```yaml
source: ADR-007
status: active
acceptance:
  kind: categorical
  precondition: Provider configured either with an empty API key or with a mock LLM HTTP endpoint.
  action: Call GenerateFile for a check key under both configurations.
  expected: Returns deterministic template content when unauthenticated and LLM response content when authenticated.
  method: test
  expected_output: "mode=template_and_llm_verified"
```

REQ-009 The OSS standards auditor SHALL evaluate a `RepoInspection` across the four 25-point pillars (`community_files`, `documentation_quality`, `technical_hygiene`, `security_openssf`) and return an `AuditResult` with an overall score between 0 and 100 and per-check statuses (`pass`, `warn`, `fail`).

```yaml
source: ADR-006
status: active
acceptance:
  kind: categorical
  precondition: RepoInspection snapshots representing complete, partial, and minimal repositories.
  action: Call EvaluateStandards on each snapshot.
  expected: Complete repository scores 100 with all checks passing; minimal repository scores < 40 with failing checks recorded.
  method: test
  expected_output: "complete=100 minimal_has_failures=true"
```

REQ-010 The OSS standards remediator SHALL generate a `RemediationDraft` containing target file path, branch name, commit message, pull request title, and file content for any supported check key.

```yaml
source: ADR-006
status: active
acceptance:
  kind: categorical
  precondition: Repository metadata and a supported check key such as contributing_guide or security_policy.
  action: Call BuildRemediationDraft with the repository and check key.
  expected: Returns a populated RemediationDraft with non-empty file path, branch name, and generated markdown content.
  method: test
  expected_output: "path=CONTRIBUTING.md branch=ossalot/remediate-contributing_guide"
```

REQ-011 The HTTP API server SHALL serve JSON endpoints under `/api/v1` for health check, workspace settings read/update, repository add/sync/list/get/delete, standards audit execution, and remediation draft/PR creation.

```yaml
source: ADR-003
status: active
acceptance:
  kind: categorical
  precondition: An initialized API server backed by a store, mock GitHub server, and LLM provider.
  action: Send HTTP requests to /api/v1/health, /api/v1/settings, /api/v1/repos, /api/v1/repos/{id}/audit, and /api/v1/repos/{id}/remediate.
  expected: Endpoints return expected HTTP status codes (200, 201, 400, 404) and valid JSON payloads.
  method: test
  expected_output: "status=ok"
```

REQ-012 The CLI runner SHALL support `-addr`, `-db`, and `-check` flags, returning exit code 0 when `-check` is set and returning exit code 2 when invalid flags or an unwritable database path are supplied.

```yaml
source: ADR-003
status: active
acceptance:
  kind: categorical
  precondition: Command-line argument slices with -check, invalid flags, or an invalid database path.
  action: Call Run with each argument slice.
  expected: Returns 0 for -check and 2 for invalid flags or database initialization failure.
  method: test
  expected_output: "check_exit=0 error_exit=2"
```

REQ-013 The frontend portfolio model SHALL compute portfolio summary statistics, map check statuses to badge metadata, and build remediation request payloads from UI inputs.

```yaml
source: ADR-008
status: active
acceptance:
  kind: categorical
  precondition: Array of repository objects with standards scores, ownership flags, and check items.
  action: Call summarizePortfolio, getStatusBadge, and buildRemediationRequest.
  expected: Returns exact total count, rounded average score, owned count, failing check count, badge metadata, and request payload.
  method: test
  expected_output: "totalRepos=2 avgStandardsScore=80 ownedRepos=1"
```
