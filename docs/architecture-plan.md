# OSSAlot architecture plan: one server, a built frontend, automatic deploys

Status: draft for review. Nothing here is decided until it lands as ADRs in `docs/mcswe/decisions.md`.

## Goals

- Grow the UI in size and function, then do a design and UX pass, without fighting the codebase.
- Hot reload during local development.
- Deploy automatically to a stable URL on the dev domain on every merge to `main`.
- Keep the code that users actually run under MCSWE governance and tests.

## Where things stand

### The running app is an ignored script

`scripts/preview_server.go` (784 lines, `//go:build ignore`) is the server that runs the UI. It has behavior that exists nowhere else:

- It registers the metrics, discovery, community and inbox routes (`RegisterMetricsRoutes`, `RegisterDiscoveryRoutes`, `RegisterCommunityAndInboxRoutes`). `cmd/ossalot` registers none of them, so the official binary returns 404 for `/api/v1/opportunities` and others.
- It replaces `POST /api/v1/repos/{id}/remediate` with its own handler, which adds scorecard and release workflow remediation.
- It wraps the GitHub HTTP client in a `RoundTripper` that rewrites outgoing calls: it creates the branch, looks up existing file SHAs, and turns 404s on `.github/workflows/` into a message about the missing `workflow` scope. This is what makes the 1-click remediation PRs work.
- It reads the GitHub token from `GITHUB_TOKEN` or from `gh`'s `~/.config/gh/hosts.yml`.
- It starts the scheduler and runs `daily_portfolio_sync` at startup.
- It serves `web/dist` and `web/src` from disk.

`scripts/` is out of MCSWE scope and has no tests. The 100%-covered code in `internal/` isn't the full app users run, and `cmd/ossalot` can't run the app on its own.

### The UI is one hand-edited file

- `web/dist/index.html` is about 71 KB: 9 tabs, a global `state` object, 45 inline `onclick` handlers, 24 `window.*` functions, 36 `innerHTML` writes, and Tailwind loaded from `cdn.tailwindcss.com`.
- It's named and placed like a build output, and `IMPLEMENTATION_PLAN.md` describes a React + TypeScript + Vite source tree. That source was never committed. Git history shows seven commits editing `web/dist/index.html` directly.
- The only modular, tested UI code is `web/src/*_model.mjs`: pure functions with `node --test` suites, imported by the page at runtime.
- `MCSWE.md` lists `web/dist/` as out of scope, so most of the UI logic isn't governed.

### Storage is a JSON file

`internal/db` keeps state in JSON files (`ossalot.db`, `ossalot.db.metrics.json`, `ossalot.db.opportunities.json`), despite the `.db` name. That's fine for one user. It needs a persistent disk and a single running instance, so the deploy target must be a long-running server, not serverless.

### There's no deploy path

- `.github/workflows/ci.yml` runs evidence and builds a binary artifact, but deploys nothing.
- The earlier `*.trycloudflare.com` URLs were quick tunnels started by hand from an Antigravity session. They get a random hostname each time and are gone.

## Target architecture

```
                 ┌──────────────── dev domain (named Cloudflare Tunnel + Access) ───────────────┐
                 │                                                                               │
 browser ──────► │  ossalot container                                                           │
                 │    cmd/ossalot (single Go binary)                                            │
                 │      /api/v1/*  → internal/api (all routes, remediation, PR creation)       │
                 │      /*         → embedded web/dist (go:embed)                               │
                 │    volume: /data (JSON store)                                                │
                 └───────────────────────────────────────────────────────────────────────────────┘

 local dev:  browser → Vite :5173 (HMR) ──/api proxy──► go run ./cmd/ossalot :8088
```

### Backend: one binary

- `cmd/ossalot` is the only entry point. It wires every route, the scheduler and the startup sync.
- Move the PR-creation logic out of the `RoundTripper` into explicit functions in `internal/github`: create the branch if it's missing, upsert the file with its SHA, open the PR or return an existing one, and handle a missing `workflow` scope. Each gets unit tests.
- Fold the preview server's remediation handler into `internal/api`.
- Token source: `GITHUB_TOKEN` env var first, then workspace settings. Drop reading `gh`'s `hosts.yml`: it only works on a machine logged into `gh`.
- Serve the UI with `go:embed` from `web/dist`, with an SPA fallback to `index.html` for unknown non-API paths.
- Delete `scripts/preview_server.go` once nothing depends on it.

### Frontend: Vite + TypeScript + React + Tailwind

- **Vite** gives a real build, hot module replacement and a dev proxy.
- **TypeScript** catches errors in a growing UI and lets the API types be shared.
- **React** has the largest ecosystem for the design pass (shadcn/ui and Radix on top of Tailwind) and matches the original plan. Alternatives considered:
  - Svelte: lighter, smaller ecosystem.
  - Server-rendered Go with HTMX: all-Go, but weaker for chart-heavy, highly interactive screens.
- **Tailwind built at compile time**, not from the CDN. Pages then render offline and the bundle only carries the classes in use.
- **TanStack Query** for API data (caching, refetching, loading and error states) and a **router** for the tabs, so each tab has a URL.
- **API types:** start with hand-written TypeScript types next to an `api.ts` client. Later, generate them from an OpenAPI spec or from the Go structs so the two can't drift.
- The `*_model.mjs` modules stay pure logic, ported to TypeScript, with their tests moved to **Vitest**. Vitest's LCOV output feeds the existing coverage evidence.
- `web/dist/` becomes a true build artifact: gitignored, built by `npm run build` and in CI.

Layout:

```
web/
  index.html            # Vite entry (small shell)
  package.json
  vite.config.ts        # dev proxy /api → :8088
  tsconfig.json
  src/
    main.tsx
    api/                # typed client + query hooks
    models/             # ported *_model logic + Vitest tests
    tabs/               # one folder per tab
    components/         # shared UI (later: design system)
  dist/                 # generated, gitignored, embedded by Go
```

### Local development

- `make dev` starts both processes:
  - `vite` on `:5173` with HMR, proxying `/api` to `:8088`;
  - the Go server on `:8088`, optionally under `air` so it rebuilds on save.
- Open `http://localhost:5173`. UI edits show up immediately; Go edits restart the server.
- `make build` runs `npm ci && npm run build`, then `go build ./cmd/ossalot` with the embedded UI.

### Deploy

1. **Container.** Multi-stage `Dockerfile`: a Node stage builds `web/dist`, a Go stage builds `cmd/ossalot` with the UI embedded, and a small runtime image (distroless or alpine) runs it. The data directory is a mounted volume.
2. **CI.** On every merge to `main`, after the evidence jobs pass, build the image and push it to GHCR, tagged with the commit SHA and `dev`.
3. **Host.** A machine you control (home server, WSL box or a small VPS) runs the container plus `cloudflared` with a **named tunnel**. The tunnel maps a fixed hostname on the dev domain, e.g. `ossalot.dev.<domain>`, to the container.
4. **Rollout.** Watchtower on the host pulls the new `dev` image and restarts the container. If you want CI to report deploy success, use a deploy job that connects over the tunnel or SSH instead.
5. **Access control.** Put **Cloudflare Access** in front of the hostname, limited to your account. The app holds a GitHub token that can push branches and open PRs, so it mustn't be public.
6. **Secrets.** `GITHUB_TOKEN` and LLM keys come from the host's environment file, never from the image. The token needs the `workflow` scope for pipeline PRs.

Alternative host: Fly.io or Render with a persistent volume avoids running a machine, but you lose the Cloudflare-only setup and pay for the volume. The container is the same either way.

## MCSWE implications

- New ADRs: single server entry point, frontend stack, build and deploy pipeline, access control.
- Requirement amendments before any code, per `AGENTS.md`. This covers the consolidated routes, the PR-creation behavior, token loading, static serving, and the UI behaviors being ported.
- Scope changes in `MCSWE.md`: `web/` source stays in scope (`web/src/`); `web/dist/` stays out as a generated artifact. Add `Dockerfile` and the deploy workflow to scope.
- `scripts/run_evidence.py` has to run Vitest and merge its LCOV with Go's. TypeScript branch coverage must stay at the current bar.
- Traceability anchors move with the code: the `// TRACEABILITY_MAPPING:` comments carry over into `.ts`/`.tsx` files.

## Phases

### Phase 1: one server
- ADRs and requirement amendments for consolidating the servers.
- Move routes, remediation, PR creation, token loading, scheduler start-up and static serving into `cmd/ossalot` and `internal/`, with tests.
- Delete `scripts/preview_server.go`.
- **Done when:** `go run ./cmd/ossalot` serves the current UI with every feature the preview server had, and the audit has no new findings.

### Phase 2: a real frontend build, same UI
- Add Vite, TypeScript and built Tailwind. Move the inline script and markup from `web/dist/index.html` into `web/src/`, unchanged in behavior.
- Port the `*_model.mjs` tests to Vitest and wire their coverage into the evidence scripts.
- `make dev` with hot reload.
- **Done when:** the UI looks and works the same, `web/dist` is generated, and hot reload works locally.

Phase 2 can run in parallel with phase 1.

### Phase 3: automatic deploy
- Dockerfile, GHCR push in CI, named tunnel, Watchtower, Cloudflare Access.
- **Done when:** a merge to `main` shows up at the dev hostname within minutes, behind Access.

Depends on phase 1, since the container needs a single real server.

### Phase 4: React, one tab at a time
- Mount React components into the existing tab containers, one tab per change, starting with the smallest.
- Wire pending UI gaps as tabs are ported, such as the "1-Click PR to Repo" pipeline button, which today only shows the YAML.
- **Done when:** no tab uses `innerHTML` or inline handlers, and the old global `state` is gone.

### Phase 5: design and UX
- Adopt a component library (shadcn/ui on Radix) and design tokens. Then do the visual and UX pass on a codebase built for it.

## Risks

- **Porting the PR logic** out of the `RoundTripper` could change edge-case behavior: existing branches, existing files, the missing-scope message. Write tests from the current behavior before moving it.
- **The JSON store** can't support more than one running instance or concurrent writers across processes. That's fine for a single-user dev deployment. Revisit it (SQLite is the natural next step) before any multi-user or hosted mode.
- **A self-hosted machine** is a single point of failure for the dev URL. That's acceptable for a dev environment.
- **The MCSWE overhead** of every phase is real. Phases 1 and 2 are mostly moves, which keeps the requirement amendments small.

## Open questions

1. Which machine hosts the container: this WSL box, another home machine, or a VPS?
2. What is the dev hostname?
3. Is React confirmed, or do you want to weigh Svelte first?
4. Should CI deploy every merge to `main`, or only tagged builds?
5. Is a separate production deployment planned, or only the dev domain for now?
