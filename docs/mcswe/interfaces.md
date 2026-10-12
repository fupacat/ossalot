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
| api | `internal/api/` |
| web | `web/src/` |

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
requirements: [REQ-011]
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
