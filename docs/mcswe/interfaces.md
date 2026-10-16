# Interface control document

This document points to each interface's native definition; it never restates it (D6.4).

## Components

| Component | Boundary (paths it owns) |
| --- | --- |
| cli | `cmd/ossalot/` |
| config | `internal/config/` |
| db | `internal/db/` |
| github | `internal/github/` |
| llm | `internal/llm/` |
| standards | `internal/engine/standards/` |
| metrics | `internal/engine/metrics/` |
| profiler | `internal/engine/profiler/` |
| discovery | `internal/engine/discovery/` |
| community | `internal/engine/community/` |
| notifier | `internal/engine/notifier/` |
| scheduler | `internal/scheduler/` |
| api | `internal/api/` |
| web | `web/src/` |
| webassets | `web/embed.go` |

## Interfaces

Each interface has an `IF-NNN` heading and one YAML block (D6).

## IF-001 Configuration loader

```yaml
component: config
definition: internal/config/config.go#LoadFromEnv
requirements: [REQ-001]
checker: go vet ./internal/config
data_schema: internal/config/config.go#Config
```

## IF-002 Store initialization and workspace bootstrap

```yaml
component: db
definition: internal/db/store.go#OpenStore
requirements: [REQ-002]
checker: go vet ./internal/db
data_schema: internal/db/store.go#Workspace
```

## IF-003 Workspace settings persistence and secret masking

```yaml
component: db
definition: internal/db/store.go#Store.GetSettingsRedacted
requirements: [REQ-003]
checker: go vet ./internal/db
data_schema: internal/db/store.go#WorkspaceSettings
```

## IF-004 Repository and standards audit persistence

```yaml
component: db
definition: internal/db/store.go#Store.UpsertRepository
requirements: [REQ-004]
checker: go vet ./internal/db
data_schema: internal/db/store.go#Repository
```

## IF-005 GitHub repository coordinate parser

```yaml
component: github
definition: internal/github/client.go#ParseRepoCoordinates
requirements: [REQ-005]
checker: go vet ./internal/github
data_schema: null
```

## IF-006 GitHub repository and OpenSSF Scorecard inspector

```yaml
component: github
definition: internal/github/client.go#Client.InspectRepository
requirements: [REQ-006]
checker: go vet ./internal/github
data_schema: internal/github/client.go#RepoInspection
```

## IF-007 GitHub remediation pull request creator

```yaml
component: github
definition: internal/github/client.go#Client.CreateRemediationPR
requirements: [REQ-007]
checker: go vet ./internal/github
data_schema: internal/github/client.go#PullRequestRequest
```

## IF-008 Pluggable LLM file generator

```yaml
component: llm
definition: internal/llm/provider.go#Client.GenerateFile
requirements: [REQ-008]
checker: go vet ./internal/llm
data_schema: internal/llm/provider.go#GenerateRequest
```

## IF-009 OSS standards scorecard evaluator

```yaml
component: standards
definition: internal/engine/standards/auditor.go#EvaluateStandards
requirements: [REQ-009]
checker: go vet ./internal/engine/standards
data_schema: internal/engine/standards/auditor.go#AuditResult
```

## IF-010 OSS standards remediation draft builder

```yaml
component: standards
definition: internal/engine/standards/auditor.go#BuildRemediationDraft
requirements: [REQ-010]
checker: go vet ./internal/engine/standards
data_schema: internal/engine/standards/auditor.go#RemediationDraft
```

## IF-011 HTTP API router and handler server

```yaml
component: api
definition: internal/api/server.go#NewServer
requirements: [REQ-011, REQ-055]
checker: go vet ./internal/api
data_schema: null
```

## IF-012 CLI entrypoint runner

```yaml
component: cli
definition: cmd/ossalot/main.go#Run
requirements: [REQ-012]
checker: go vet ./cmd/ossalot
data_schema: null
```

## IF-013 Frontend portfolio and scorecard model

```yaml
component: web
definition: web/src/model.mjs#summarizePortfolio
requirements: [REQ-013]
checker: node --check web/src/model.mjs
data_schema: null
```

## IF-014 Metric snapshot persistence

```yaml
component: db
definition: internal/db/metrics_store.go#Store.UpsertMetricSnapshot
requirements: [REQ-014]
checker: go vet ./internal/db
data_schema: internal/db/metrics_store.go#MetricSnapshot
```

## IF-015 GitHub 14-day traffic inspector

```yaml
component: github
definition: internal/github/traffic.go#Client.FetchTrafficSnapshot
requirements: [REQ-015]
checker: go vet ./internal/github
data_schema: internal/github/traffic.go#TrafficSnapshot
```

## IF-016 Package registry download connector

```yaml
component: metrics
definition: internal/engine/metrics/collector.go#RegistryClient.FetchDownloads
requirements: [REQ-016]
checker: go vet ./internal/engine/metrics
data_schema: null
```

## IF-017 Daily metric snapshot builder and adoption scorer

```yaml
component: metrics
definition: internal/engine/metrics/collector.go#BuildDailySnapshot
requirements: [REQ-017]
checker: go vet ./internal/engine/metrics
data_schema: internal/db/metrics_store.go#MetricSnapshot
```

## IF-018 HTTP API metrics collection and time-series endpoints

```yaml
component: api
definition: internal/api/metrics_handlers.go#Server.RegisterMetricsRoutes
requirements: [REQ-018]
checker: go vet ./internal/api
data_schema: internal/db/metrics_store.go#MetricSnapshot
```

## IF-019 Frontend time-series and referrer attribution model

```yaml
component: web
definition: web/src/metrics_model.mjs#buildTrafficSeries
requirements: [REQ-019]
checker: node --check web/src/metrics_model.mjs
data_schema: null
```

## IF-020 Capability profile and opportunity persistence

```yaml
component: db
definition: internal/db/opportunities_store.go#Store.SaveCapabilityProfile
requirements: [REQ-020]
checker: go vet ./internal/db
data_schema: internal/db/opportunities_store.go#Opportunity
```

## IF-021 Repository capability profiler

```yaml
component: profiler
definition: internal/engine/profiler/profiler.go#BuildCapabilityProfile
requirements: [REQ-021]
checker: go vet ./internal/engine/profiler
data_schema: internal/db/opportunities_store.go#CapabilityProfile
```

## IF-022 GitHub issue candidate search client

```yaml
component: github
definition: internal/github/search.go#Client.SearchIssues
requirements: [REQ-022]
checker: go vet ./internal/github
data_schema: internal/github/search.go#IssueCandidate
```

## IF-023 Opportunity relevance scorer and PR plan generator

```yaml
component: discovery
definition: internal/engine/discovery/finder.go#ScoreAndFilterCandidates
requirements: [REQ-023]
checker: go vet ./internal/engine/discovery
data_schema: internal/db/opportunities_store.go#Opportunity
```

## IF-024 HTTP API capability profile and opportunity discovery endpoints

```yaml
component: api
definition: internal/api/discovery_handlers.go#Server.RegisterDiscoveryRoutes
requirements: [REQ-024]
checker: go vet ./internal/api
data_schema: internal/db/opportunities_store.go#Opportunity
```

## IF-025 Frontend opportunity filter and PR plan model

```yaml
component: web
definition: web/src/discovery_model.mjs#filterOpportunities
requirements: [REQ-025]
checker: node --check web/src/discovery_model.mjs
data_schema: null
```

## IF-026 Multi-platform community scanner

```yaml
component: community
definition: internal/engine/community/scanner.go#Scanner.ScanCommunities
requirements: [REQ-026]
checker: go vet ./internal/engine/community
data_schema: internal/db/opportunities_store.go#Opportunity
```

## IF-027 Outbound marketing playbook generator

```yaml
component: community
definition: internal/engine/community/scanner.go#GenerateMarketingPlaybooks
requirements: [REQ-027]
checker: go vet ./internal/engine/community
data_schema: internal/db/opportunities_store.go#Opportunity
```

## IF-028 High-value opportunity webhook notifier

```yaml
component: notifier
definition: internal/engine/notifier/webhook.go#Notifier.NotifyHighValueOpportunity
requirements: [REQ-028]
checker: go vet ./internal/engine/notifier
data_schema: internal/db/opportunities_store.go#Opportunity
```

## IF-029 Background and on-demand scan job runner

```yaml
component: scheduler
definition: internal/scheduler/runner.go#Runner.TriggerJob
requirements: [REQ-029]
checker: go vet ./internal/scheduler
data_schema: internal/scheduler/runner.go#ScanJobRecord
```

## IF-030 HTTP API community, playbook, inbox, and scheduler endpoints

```yaml
component: api
definition: internal/api/community_handlers.go#Server.RegisterCommunityAndInboxRoutes
requirements: [REQ-030]
checker: go vet ./internal/api
data_schema: internal/db/opportunities_store.go#Opportunity
```

## IF-031 Frontend action inbox and campaign attribution model

```yaml
component: web
definition: web/src/inbox_model.mjs#groupInboxByStatus
requirements: [REQ-031]
checker: node --check web/src/inbox_model.mjs
data_schema: null
```

## IF-032 Continuous integration and test deployment workflow

```yaml
component: cli
definition: .github/workflows/ci.yml
requirements: [REQ-032, REQ-054]
checker: python3 scripts/run_evidence.py test
data_schema: null
```

## IF-033 Extended workspace credentials and secret masking

```yaml
component: db
definition: internal/db/store.go#Store.SaveSettings
requirements: [REQ-033]
checker: go vet ./internal/db
data_schema: internal/db/store.go#WorkspaceSettings
```

## IF-034 Setup and permission-gating evaluator

```yaml
component: web
definition: web/src/setup_model.mjs#evaluateSetupCapabilities
requirements: [REQ-034]
checker: node --check web/src/setup_model.mjs
data_schema: null
```

## IF-035 CI and automated pipeline template generator

```yaml
component: web
definition: web/src/pipeline_model.mjs#generatePipelineWorkflow
requirements: [REQ-035]
checker: node --check web/src/pipeline_model.mjs
data_schema: null
```

## IF-036 Code hygiene and agent-readiness config generator

```yaml
component: web
definition: web/src/hygiene_model.mjs#generateHygieneConfigs
requirements: [REQ-036]
checker: node --check web/src/hygiene_model.mjs
data_schema: null
```

## IF-037 SonarCloud onboarding and local Community Edition guide

```yaml
component: web
definition: web/src/sonar_model.mjs#evaluateSonarOnboarding
requirements: [REQ-037]
checker: node --check web/src/sonar_model.mjs
data_schema: null
```

## IF-038 SonarCloud token and GitHub secret deep-links

```yaml
component: web
definition: web/src/sonar_model.mjs#evaluateSonarOnboarding
requirements: [REQ-038]
checker: node --check web/src/sonar_model.mjs
data_schema: null
```

## IF-039 Single server assembly

```yaml
component: cli
definition: cmd/ossalot/app.go#buildHandler
requirements: [REQ-039, REQ-052, REQ-053]
checker: go vet ./cmd/ossalot
data_schema: null
```

## IF-040 Embedded UI assets

```yaml
component: webassets
definition: web/embed.go#Assets
requirements: [REQ-052, REQ-053]
checker: go vet ./web
data_schema: null
```

## IF-041 GitHub token resolution

```yaml
component: cli
definition: cmd/ossalot/app.go#resolveGitHubToken
requirements: [REQ-041]
checker: go vet ./cmd/ossalot
data_schema: internal/db/store.go#WorkspaceSettings
```

## IF-042 Startup portfolio enrichment job

```yaml
component: cli
definition: cmd/ossalot/app.go#registerEnrichmentJob
requirements: [REQ-042]
checker: go vet ./cmd/ossalot
data_schema: internal/scheduler/runner.go#ScanJobRecord
```

## IF-043 Remediation pull request flow

```yaml
component: github
definition: internal/github/client.go#Client.CreateRemediationPR
requirements: [REQ-043, REQ-044, REQ-045, REQ-046]
checker: go vet ./internal/github
data_schema: internal/github/client.go#PullRequestRequest
```

## IF-044 Traffic fallback for restricted repositories

```yaml
component: github
definition: internal/github/traffic.go#Client.FetchTrafficSnapshot
requirements: [REQ-047]
checker: go vet ./internal/github
data_schema: internal/github/traffic.go#TrafficSnapshot
```

## IF-045 Issue search retry

```yaml
component: github
definition: internal/github/search.go#Client.SearchIssues
requirements: [REQ-048]
checker: go vet ./internal/github
data_schema: internal/github/search.go#IssueCandidate
```

## IF-046 Live package registry adapter

```yaml
component: metrics
definition: internal/engine/metrics/live_registry.go#LiveRegistryTransport
requirements: [REQ-049]
checker: go vet ./internal/engine/metrics
data_schema: null
```

## IF-047 Live community feed adapter

```yaml
component: community
definition: internal/engine/community/live_feeds.go#LiveFeedTransport
requirements: [REQ-050]
checker: go vet ./internal/engine/community
data_schema: internal/engine/community/scanner.go#communityFeedItem
```

## IF-048 Workflow remediation templates

```yaml
component: standards
definition: internal/engine/standards/auditor.go#BuildRemediationDraft
requirements: [REQ-051]
checker: go vet ./internal/engine/standards
data_schema: internal/engine/standards/auditor.go#RemediationDraft
```

## IF-049 GitHub issue creator

```yaml
component: github
definition: internal/github/client.go#Client.CreateIssue
requirements: [REQ-055]
checker: go vet ./internal/github
data_schema: internal/github/client.go#IssueRequest
```

## IF-050 CI failure Copilot summon workflow

```yaml
component: cli
definition: .github/workflows/copilot-ci-failure-handler.yml
requirements: [REQ-056]
checker: python3 scripts/run_evidence.py test
data_schema: null
```
