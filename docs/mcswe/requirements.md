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

REQ-014 The database store SHALL upsert and list date-keyed `MetricSnapshot` records (`YYYY-MM-DD`) in chronological order for a repository and update the repository's `adoption_score`.

```yaml
source: ADR-009
status: active
acceptance:
  kind: categorical
  precondition: An initialized store containing a repository record.
  action: Call UpsertMetricSnapshot with two dates and a duplicate date update, then call ListMetricSnapshots.
  expected: Snapshots are deduplicated by (repo_id, snapshot_date) and returned sorted chronologically by snapshot_date ascending.
  method: test
  expected_output: "snapshots=2 latest_clones=25"
```

REQ-015 The GitHub client SHALL fetch 14-day repository traffic views, clones, and popular referrers from `/repos/{owner}/{repo}/traffic/views`, `/traffic/clones`, and `/traffic/popular/referrers`.

```yaml
source: ADR-009
status: active
acceptance:
  kind: categorical
  precondition: Mock GitHub REST endpoints serving traffic views, clones, and referrers JSON responses.
  action: Call FetchTrafficSnapshot on the GitHub client.
  expected: Returns TrafficSnapshot populated with total/unique views, total/unique clones, and referrer items.
  method: test
  expected_output: "views=120 clones=35 referrers=2"
```

REQ-016 The package registry client SHALL query download counts for configured package coordinates (`npm`, `pypi`, `crates`, `homebrew`) from the configured registry base URL and return a map of ecosystem download counts.

```yaml
source: ADR-009
status: active
acceptance:
  kind: categorical
  precondition: Mock HTTP registry server serving download count payloads for npm, pypi, crates, and homebrew.
  action: Call FetchDownloads with a map of package coordinates.
  expected: Returns download counts per ecosystem key matching the upstream responses.
  method: test
  expected_output: "npm=1500 pypi=800"
```

REQ-017 The metrics collector SHALL combine repository engagement counts, GitHub traffic data, and package registry downloads into a `MetricSnapshot` with a bounded `adoption_score` between 0 and 100 inclusive.

```yaml
source: ADR-009
status: active
acceptance:
  kind: categorical
  precondition: Repository metadata, GitHub traffic snapshot, and package registry download map.
  action: Call BuildDailySnapshot for low-adoption and high-adoption repository inputs.
  expected: Produces a MetricSnapshot with aggregated registry downloads, referrers, and an adoption_score in [0, 100].
  method: test
  expected_output: "score_bounded=true"
```

REQ-018 The HTTP API server SHALL serve `POST /api/v1/repos/{id}/metrics/collect` and `GET /api/v1/repos/{id}/metrics` to trigger snapshot collection and retrieve historical metric snapshots for a repository.

```yaml
source: ADR-009
status: active
acceptance:
  kind: categorical
  precondition: API server with registered metrics routes, a stored repository, and mock GitHub/registry servers.
  action: Send POST /api/v1/repos/{id}/metrics/collect followed by GET /api/v1/repos/{id}/metrics.
  expected: POST returns 200 with the collected MetricSnapshot and GET returns 200 with the chronological slice of snapshots.
  method: test
  expected_output: "collected=1 listed=1"
```

REQ-019 The frontend metrics model SHALL transform snapshot arrays into chronologically sorted chart series and aggregate referrer totals sorted by visit count descending.

```yaml
source: ADR-010
status: active
acceptance:
  kind: categorical
  precondition: Unordered array of metric snapshot objects containing views, clones, stars, registry downloads, and referrers.
  action: Call buildTrafficSeries and aggregateReferrers.
  expected: Returns chronologically ordered series with summed registry downloads and deduplicated referrers sorted by count descending.
  method: test
  expected_output: "series=2 topReferrer=news.ycombinator.com"
```

REQ-020 The database store SHALL upsert and retrieve `CapabilityProfile` records by `repo_id` and upsert, list, and update status for `Opportunity` records deduplicated by `external_id`.

```yaml
source: ADR-011
status: active
acceptance:
  kind: categorical
  precondition: An initialized store containing a repository record.
  action: Save and retrieve a CapabilityProfile, upsert duplicate Opportunity records by external_id, list opportunities, and update an opportunity status.
  expected: CapabilityProfile is persisted and retrieved, opportunities are deduplicated by external_id, and status updates persist with submitted_url and outcome_notes.
  method: test
  expected_output: "profile_found=true opps=1 status=submitted"
```

REQ-021 The capability profiler SHALL synthesize a `CapabilityProfile` containing summary, problem statements, target personas, ecosystem keywords, search queries, and recommended communities from a repository and its README content.

```yaml
source: ADR-011
status: active
acceptance:
  kind: categorical
  precondition: Repository metadata with primary language, topics, description, and README content.
  action: Call BuildCapabilityProfile on repositories with and without explicit topics.
  expected: Returns CapabilityProfile with non-empty problem statements, search queries, and language-mapped recommended communities.
  method: test
  expected_output: "queries_non_empty=true"
```

REQ-022 The GitHub client SHALL query `/search/issues` with a search query string and return a slice of `IssueCandidate` items with repository full name, issue number, title, body, HTML URL, and author login.

```yaml
source: ADR-012
status: active
acceptance:
  kind: categorical
  precondition: Mock GitHub REST server serving /search/issues responses.
  action: Call SearchIssues with a query string.
  expected: Returns parsed IssueCandidate records with extracted target repository full name and issue metadata.
  method: test
  expected_output: "candidates=1 repo=acme/webapp"
```

REQ-023 The opportunity discovery engine SHALL score `IssueCandidate` items against a `CapabilityProfile`, filter out candidates from the same repository or scoring below 70, and populate `legitimacy_reason`, `pr_implementation_plan`, and `draft_body` on retained opportunities.

```yaml
source: ADR-012
status: active
acceptance:
  kind: categorical
  precondition: CapabilityProfile for a source repository and candidate issues including a self-repo issue, a low-relevance issue, and a high-relevance external issue.
  action: Call ScoreAndFilterCandidates on the candidate list.
  expected: Self-repo and low-relevance candidates are excluded; high-relevance candidate is returned with relevance_score >= 70 and non-empty pr_implementation_plan.
  method: test
  expected_output: "retained=1 score_gte_70=true"
```

REQ-024 The HTTP API server SHALL serve `POST /api/v1/repos/{id}/profile`, `GET /api/v1/repos/{id}/profile`, `POST /api/v1/repos/{id}/discover`, and `GET /api/v1/repos/{id}/opportunities` for capability profiling and issue opportunity discovery.

```yaml
source: ADR-012
status: active
acceptance:
  kind: categorical
  precondition: API server with registered discovery routes, a stored repository, and a mock GitHub search server.
  action: Call POST and GET on /profile, then POST /discover and GET /opportunities.
  expected: Profile is generated and returned with HTTP 200, and discovered opportunities scoring >= 70 are persisted and returned with HTTP 200.
  method: test
  expected_output: "profile_ok=true discovered=1"
```

REQ-025 The frontend discovery model SHALL filter opportunities by minimum score, category, and status, sort them by `relevance_score` descending, and format PR implementation plan summaries.

```yaml
source: ADR-012
status: active
acceptance:
  kind: categorical
  precondition: Array of opportunity objects with varying scores, categories, statuses, and PR plans.
  action: Call filterOpportunities and formatPRPlanSummary.
  expected: Returns matching opportunities ordered by relevance_score descending and a structured plan summary object.
  method: test
  expected_output: "filtered=1 topScore=92"
```

REQ-026 The community scanner SHALL query configured community feed endpoints (`hackernews`, `reddit`, `stackoverflow`, `devto`, `lobsters`) for a repository's capability profile keywords and return scored `Opportunity` items with platform-specific recommended actions and draft replies.

```yaml
source: ADR-013
status: active
acceptance:
  kind: categorical
  precondition: Mock community feed HTTP server returning threads for hackernews, reddit, stackoverflow, devto, and lobsters.
  action: Call ScanCommunities with a repository and CapabilityProfile.
  expected: Returns Opportunity records with matching categories, external IDs, relevance scores, and draft bodies.
  method: test
  expected_output: "scanned_categories=5"
```

REQ-027 The marketing playbook generator SHALL produce outbound `Opportunity` playbooks (`Show HN`, `Reddit Deep-Dive`, `Dev.to Tutorial`, `StackOverflow Answer Guide`) with populated `community_rules_notes`, `draft_title`, and `draft_body` for a repository.

```yaml
source: ADR-013
status: active
acceptance:
  kind: categorical
  precondition: Repository record and CapabilityProfile.
  action: Call GenerateMarketingPlaybooks for the repository.
  expected: Returns 4 playbooks with category "marketing_playbook", non-empty community_rules_notes, and repository-specific draft bodies.
  method: test
  expected_output: "playbooks=4"
```

REQ-028 The webhook notifier SHALL send HTTP POST JSON notifications to configured Slack and Discord webhook URLs when an opportunity has `relevance_score >= 85` and perform no HTTP request when `relevance_score < 85` or no webhook URLs are configured.

```yaml
source: ADR-014
status: active
acceptance:
  kind: categorical
  precondition: Mock webhook HTTP server and WorkspaceSettings with Slack and Discord webhook URLs.
  action: Call NotifyHighValueOpportunity with a low-score opportunity (75) and a high-score opportunity (90).
  expected: Low-score opportunity sends 0 webhook requests; high-score opportunity sends requests to configured webhooks and returns the count of notified endpoints.
  method: test
  expected_output: "low_sent=0 high_sent=2"
```

REQ-029 The job scheduler SHALL register named scan jobs, execute a job on demand while recording `ScanJobRecord` status (`completed` or `failed`), and return the list of executed job records.

```yaml
source: ADR-014
status: active
acceptance:
  kind: categorical
  precondition: A Runner with registered succeeding and failing job functions.
  action: Call TriggerJob for a succeeding job, a failing job, and an unregistered job name, then call ListJobs.
  expected: Succeeding job records status "completed", failing job records status "failed", unregistered job returns an error, and ListJobs returns all recorded executions.
  method: test
  expected_output: "jobs=2 first=completed second=failed"
```

REQ-030 The HTTP API server SHALL serve `POST /api/v1/repos/{id}/community/scan`, `POST /api/v1/repos/{id}/playbooks`, `GET /api/v1/opportunities`, `PUT /api/v1/opportunities/{id}/status`, `GET /api/v1/jobs`, and `POST /api/v1/jobs/{name}/run`.

```yaml
source: ADR-014
status: active
acceptance:
  kind: categorical
  precondition: API server with registered community, inbox, notifier, and scheduler routes.
  action: Invoke community scan, playbook generation, global opportunities list, opportunity status update, and job trigger/list endpoints.
  expected: Endpoints return expected HTTP status codes (200, 400, 404, 502) and persist opportunities and job execution records.
  method: test
  expected_output: "inbox_updated=true job_completed=true"
```

REQ-031 The frontend inbox model SHALL group opportunities by status columns (`new`, `saved`, `drafted`, `submitted`, `dismissed`) and calculate campaign attribution traffic and star deltas from before and after metric snapshots.

```yaml
source: ADR-014
status: active
acceptance:
  kind: categorical
  precondition: Array of opportunities across statuses and baseline/post-submission metric snapshots.
  action: Call groupInboxByStatus and computeCampaignAttribution.
  expected: Returns opportunities bucketed into the five status arrays and exact deltas for stars, unique views, and unique clones.
  method: test
  expected_output: "starsDelta=25 viewsDelta=140"
```

REQ-032 The continuous integration workflow SHALL execute on pushes and pull requests to validate MCSWE test and coverage evidence and compile the single-binary test distribution executable.

```yaml
source: ADR-015
status: active
acceptance:
  kind: categorical
  precondition: A GitHub Actions runner environment with Go 1.22+ and Node.js 20+.
  action: Execute python3 scripts/run_evidence.py test, python3 scripts/run_evidence.py coverage, and go build ./cmd/ossalot.
  expected: All tests pass, statement coverage report generates at 100%, and the ossalot test binary builds without compilation errors.
  method: test
  expected_output: "coverage: 100.0% of statements"
```

REQ-033 The database store SHALL persist SonarQube credentials (`sonar_token`, `sonar_host_url`, `sonar_org`) and custom API keys in `WorkspaceSettings` and mask non-empty values as `"***"` when returning redacted settings.

```yaml
source: ADR-016
status: active
acceptance:
  kind: categorical
  precondition: Workspace settings saved with non-empty sonar_token, sonar_host_url, sonar_org, and custom_keys.
  action: Call SaveSettings and query GetSettingsRedacted alongside GetSettingsRaw.
  expected: GetSettingsRedacted returns "***" for sonar_token while preserving the original value in GetSettingsRaw.
  method: test
  expected_output: "sonar_token=***"
```

REQ-034 The frontend setup model SHALL evaluate workspace settings and report permission gating status (`enabled`, `gated`, `missing_secret`) across platform capabilities including repository inspection, standard remediation, workflow mutation, SonarQube quality analysis, and LLM synthesis.

```yaml
source: ADR-016
status: active
acceptance:
  kind: categorical
  precondition: WorkspaceSettings object with varying levels of credentials configured.
  action: Call evaluateSetupCapabilities with unconfigured, basic GitHub, and full token settings.
  expected: Returns capability map indicating which features are unlocked, which require additional scopes (such as workflow scope or Sonar token), and required next steps.
  method: test
  expected_output: "inspect=unlocked workflows=gated"
```

REQ-035 The frontend pipeline model SHALL generate GitHub Actions workflow definitions for standard CI, SonarCloud analysis, auto-Copilot failure/review summoning, and PR lifecycle automation for a repository coordinate.

```yaml
source: ADR-017
status: active
acceptance:
  kind: categorical
  precondition: Repository coordinates and desired pipeline type (basic_ci, sonar, copilot_failure_handler, copilot_review_handler, pr_automation).
  action: Call generatePipelineWorkflow for each type.
  expected: Returns workflow filename, YAML content containing required permissions and actions, and installation instructions.
  method: test
  expected_output: "filename=.github/workflows/sonar.yml"
```

REQ-036 The frontend hygiene model SHALL generate code hygiene and agent-readiness configurations for `.editorconfig`, `.gitattributes`, `.pre-commit-config.yaml`, and ecosystem-specific linter/formatter configurations.

```yaml
source: ADR-018
status: active
acceptance:
  kind: categorical
  precondition: Target repository language and ecosystem specifications.
  action: Call generateHygieneConfigs for Go, Java, TypeScript, and Python.
  expected: Returns configuration files with standard line endings, indentation rules, git attributes, and pre-commit hooks.
  method: test
  expected_output: "files_count=3 has_editorconfig=true"
```

REQ-037 The frontend sonar onboarding model SHALL evaluate SonarCloud organization and project existence, produce GitHub registration/import deep-links for public repositories, and generate local Community Edition setup instructions.

```yaml
source: ADR-019
status: active
acceptance:
  kind: categorical
  precondition: Repository coordinates, organization search response, and component search response.
  action: Call evaluateSonarOnboarding with registered/unregistered orgs and imported/unimported projects.
  expected: Returns correct onboarding status (registered_and_imported, registered_not_imported, not_registered), registration deep-links, and local CE docker instructions.
  method: test
  expected_output: "status=registered_and_imported has_ce_guide=true"
```

REQ-038 The frontend sonar onboarding model SHALL provide a SonarCloud token generation deep-link and a GitHub Actions secret configuration deep-link for the `SONAR_TOKEN` secret of a repository.

```yaml
source: ADR-020
status: active
acceptance:
  kind: categorical
  precondition: Repository coordinates with an owner and a name.
  action: Call evaluateSonarOnboarding for a repository owned by an organization that is registered on SonarCloud.
  expected: Returns a token generation URL on SonarCloud and a GitHub Actions secret URL for the repository, with the secret named SONAR_TOKEN.
  method: test
  expected_output: "tokenURL=https://sonarcloud.io/account/security secretURL=https://github.com/owner/repo/settings/secrets/actions/new secretName=SONAR_TOKEN"
```

REQ-039 The CLI runner SHALL serve the base API routes (REQ-011), the metrics routes (REQ-018), the profiling and discovery routes (REQ-024) and the community, inbox and job routes (REQ-030) from one listener.

```yaml
source: ADR-022
status: active
acceptance:
  kind: categorical
  precondition: Run is called with a stubbed listener that captures the handler and a temporary database path.
  action: Send GET /api/v1/health, GET /api/v1/opportunities and GET /api/v1/jobs to the captured handler.
  expected: Each request returns status 200.
  method: test
  expected_output: "health=200 opportunities=200 jobs=200"
```

REQ-040 The CLI runner SHALL serve the embedded `web/dist` files at `/` and the embedded `web/src` modules at `/src/`, with `.mjs` files served with content type `text/javascript`.

```yaml
source: ADR-023
status: retired  # replaced by REQ-052 (ADR-029): Vite bundles the modules, so /src/ is no longer served
acceptance:
  kind: categorical
  precondition: Run is called with a stubbed listener that captures the handler.
  action: Send GET / and GET /src/model.mjs to the captured handler.
  expected: / returns 200 with the embedded index.html; /src/model.mjs returns 200 with content type text/javascript.
  method: test
  expected_output: "index=200 module=200 module_type=text/javascript"
```

REQ-041 The CLI runner SHALL use the workspace settings GitHub token for GitHub requests when it is set, use the `GITHUB_TOKEN` environment variable when the settings token is empty, and never write the environment token to the store.

```yaml
source: ADR-025
status: active
acceptance:
  kind: categorical
  precondition: A store with and without a saved GitHub token, and an environment lookup that returns a GITHUB_TOKEN value.
  action: Resolve the GitHub token for both stores, then read the raw settings back from the store.
  expected: The saved token wins when present; the environment token is used when no token is saved; the store's saved token is unchanged in both cases.
  method: test
  expected_output: "settings_wins=true env_fallback=true env_persisted=false"
```

REQ-042 The CLI runner SHALL register a `daily_portfolio_sync` job that runs the standards audit, metrics collection without package coordinates, capability profiling, issue discovery, community scan and playbook generation for each tracked repository, and SHALL trigger that job once when the server starts.

```yaml
source: ADR-026
status: active
acceptance:
  kind: categorical
  precondition: A store with two tracked repositories and stubbed GitHub, registry and community endpoints that record requests.
  action: Start the server through Run with a stubbed listener and wait for the startup job record.
  expected: The job history contains one completed daily_portfolio_sync record, and each repository received the six enrichment steps with no package coordinates in the metrics request.
  method: test
  expected_output: "job=daily_portfolio_sync status=completed repos=2 steps_per_repo=6 packages=0"
```

REQ-043 The GitHub client SHALL open a remediation pull request against the repository's default branch when the request names no base branch, and SHALL create the head branch from the default branch's latest commit when the head branch does not exist.

```yaml
source: ADR-024
status: active
acceptance:
  kind: categorical
  precondition: A mock GitHub API whose repository default branch is "trunk" and which has no branch named by the request.
  action: Call CreateRemediationPR for an owned repository with an empty BaseBranch.
  expected: The client creates refs/heads/<head> at trunk's commit SHA, commits the file to the head branch, and opens the pull request with base "trunk".
  method: test
  expected_output: "base=trunk branch_created=true"
```

REQ-044 The GitHub client SHALL include the existing file's SHA when committing a remediation file that already exists on the head branch.

```yaml
source: ADR-024
status: active
acceptance:
  kind: categorical
  precondition: A mock GitHub API where the head branch exists and already contains the target file with a known SHA.
  action: Call CreateRemediationPR for an owned repository.
  expected: The contents PUT request carries the existing file's SHA.
  method: test
  expected_output: "put_sha=existing"
```

REQ-045 The GitHub client SHALL return the URL of the open pull request for the head branch when GitHub reports that a pull request for that branch already exists.

```yaml
source: ADR-024
status: active
acceptance:
  kind: categorical
  precondition: A mock GitHub API whose pull request creation returns 422 "A pull request already exists" and whose pull request list returns one open pull request for the head branch.
  action: Call CreateRemediationPR for an owned repository.
  expected: Returns the existing pull request's html_url and no error.
  method: test
  expected_output: "pr_url=existing err=nil"
```

REQ-046 The GitHub client SHALL return `ErrWorkflowScopeRequired`, with guidance to grant the `workflow` OAuth scope, when committing a file under `.github/workflows/` returns 404.

```yaml
source: ADR-024
status: active
acceptance:
  kind: categorical
  precondition: A mock GitHub API whose contents PUT returns 404 for .github/workflows/scorecard.yml.
  action: Call CreateRemediationPR for that file on an owned repository.
  expected: Returns an error that matches ErrWorkflowScopeRequired and whose message names the workflow scope; no pull request is requested.
  method: test
  expected_output: "err=ErrWorkflowScopeRequired pr_requested=false"
```

REQ-047 The GitHub client SHALL return zero view and clone counts and an empty referrer list, without error, when a traffic endpoint responds with 403 or 404.

```yaml
source: ADR-024
status: active
acceptance:
  kind: categorical
  precondition: A mock GitHub API whose traffic endpoints return 403 for one repository and 404 for another.
  action: Call FetchTrafficSnapshot for both repositories.
  expected: Both snapshots have zero views, zero clones and no referrers, and no error is returned.
  method: test
  expected_output: "views=0 clones=0 referrers=0 err=nil"
```

REQ-048 The GitHub client SHALL repeat an issue search once with the query `is:issue is:open <keyword>` when the original query returns no items and contains a keyword, where the keyword is the first term that is not an `is:`, `language:` or `label:` qualifier, and SHALL return the issue fields unmodified.

```yaml
source: ADR-024
status: active
acceptance:
  kind: categorical
  precondition: A mock search API that returns no items for "is:issue language:Go parser" and one item for "is:issue is:open parser".
  action: Call SearchIssues with "is:issue language:Go parser".
  expected: The client sends the broader query once and returns the single item with its body and labels exactly as the API returned them.
  method: test
  expected_output: "retries=1 items=1 body_unchanged=true labels_unchanged=true"
```

REQ-049 The package registry adapter SHALL answer `/downloads/npm/{package}` for unscoped npm packages with the npm registry's last-day download count, and SHALL answer every other ecosystem, scoped npm packages and failed lookups with 0 downloads.

```yaml
source: ADR-024
status: active
acceptance:
  kind: categorical
  precondition: A mock npm downloads API that returns 1234 for "left-pad" and 404 for "missing-pkg".
  action: Send requests for npm/left-pad, npm/missing-pkg, npm/@scope/pkg and pypi/requests through the adapter.
  expected: left-pad returns 1234; the other three return 0; every response is status 200.
  method: test
  expected_output: "left-pad=1234 missing=0 scoped=0 pypi=0"
```

REQ-050 The community feed adapter SHALL answer `/feeds/{platform}?q=<keyword>` for `hackernews`, `stackoverflow`, `reddit`, `devto` and `lobsters` from Hacker News Algolia story search, Stack Exchange search, Reddit search, Dev.to articles by tag and Hacker News Algolia Ask HN search respectively, with fixed scores of 82 (91 above 50 points) for Hacker News and Ask HN items, 88 for Stack Overflow, 86 for Reddit and 84 for Dev.to, using `open source` when the keyword is empty.

```yaml
source: ADR-024
status: active
acceptance:
  kind: categorical
  precondition: Mock Algolia, Stack Exchange, Reddit and Dev.to APIs that each return known items.
  action: Send /feeds/{platform}?q=parser for each of the five platforms and /feeds/hackernews with no q.
  expected: Each response lists the mock items with the platform's community label (lobsters items labeled Ask HN) and fixed score; the request without q queries "open source".
  method: test
  expected_output: "platforms=5 lobsters_label=Ask_HN empty_q=open_source"
```

REQ-051 The OSS standards remediator SHALL return fixed GitHub Actions workflow drafts for `openssf_scorecard` at `.github/workflows/scorecard.yml` and `release_automation` at `.github/workflows/release.yml`, with `ci:` commit messages, without calling the LLM provider.

```yaml
source: ADR-027
status: active
acceptance:
  kind: categorical
  precondition: An LLM client configured with a mock endpoint that records calls.
  action: Call BuildRemediationDraft for openssf_scorecard and release_automation.
  expected: Returns drafts with the two workflow paths, workflow YAML content, commit messages starting with "ci:", and no request to the LLM endpoint.
  method: test
  expected_output: "scorecard=.github/workflows/scorecard.yml release=.github/workflows/release.yml llm_calls=0"
```

REQ-052 The CLI runner SHALL serve the files of the embedded `web/dist` build at their paths under `/`, including the bundled files under `/assets/`, and SHALL answer `/src/` paths with 404.

```yaml
source: ADR-029
status: active
acceptance:
  kind: categorical
  precondition: The server handler is built over an embedded file system holding index.html and assets/app.js.
  action: Send GET /, GET /assets/app.js and GET /src/model.mjs to the handler.
  expected: / returns 200 with index.html, /assets/app.js returns 200 with a JavaScript content type, and /src/model.mjs returns 404.
  method: test
  expected_output: "index=200 asset=200 src=404"
```

REQ-053 The CLI runner SHALL answer `GET /` with status 503 and a message naming `npm run build` when the embedded `web/dist` build has no `index.html`.

```yaml
source: ADR-029
status: active
acceptance:
  kind: categorical
  precondition: The server handler is built over an embedded file system that holds only the placeholder file.
  action: Send GET / to the handler.
  expected: Returns 503 with a body that names npm run build; API routes still answer.
  method: test
  expected_output: "index=503 names_build=true api=200"
```

REQ-054 The continuous integration workflow SHALL run `npm ci` and `npm run build` in `web/` after setting up Node.js and before the MCSWE evidence steps and the Go build.

```yaml
source: ADR-030
status: active
acceptance:
  kind: categorical
  precondition: The repository's .github/workflows/ci.yml.
  action: Read the workflow's steps in order.
  expected: A step runs npm ci and npm run build with web as its working directory, after the Node.js setup step and before the test evidence, coverage evidence and go build steps.
  method: test
  expected_output: "frontend_build_step=present order=node<build<evidence<go_build"
```

REQ-055 The GitHub client and API server SHALL support creating GitHub issues on owned repositories with structured maintainer directives and @copilot agent prompts, returning the created issue URL and number, and SHALL reject issue creation on unowned repositories with HTTP 403.

```yaml
source: ADR-034
status: active
acceptance:
  kind: categorical
  precondition: An initialized API server backed by a store holding owned and unowned repositories, and a mock GitHub REST server.
  action: Send POST /api/v1/repos/{id}/issues with title, body, and labels for an owned repository and for an unowned repository.
  expected: Returns status 200 with issue HTML URL and number for the owned repository; returns status 403 for the unowned repository.
  method: test
  expected_output: "issue_url=https://github.com/octocat/hello-world/issues/1 unowned=403"
```

REQ-056 The repository SHALL include a dedicated GitHub Actions workflow at `.github/workflows/copilot-ci-failure-handler.yml` that listens for completed `workflow_run` events and comments `@copilot` on pull requests when CI workflows fail.

```yaml
source: ADR-035
status: active
acceptance:
  kind: categorical
  precondition: The repository workflow file `.github/workflows/copilot-ci-failure-handler.yml`.
  action: Read the workflow trigger, job condition, and run command.
  expected: The workflow is triggered by completed workflow_run events, gates execution to pull_request failures, and uses `gh pr comment` to post an `@copilot` remediation request.
  method: test
  expected_output: "trigger=workflow_run condition=failure_and_pull_request commenter=gh_pr_comment"
```
