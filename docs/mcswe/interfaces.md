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

## Interfaces

Each interface has an `IF-NNN` heading and one YAML block (D6).

No interfaces defined yet. Add interface entries in a definitions change before implementing code.
