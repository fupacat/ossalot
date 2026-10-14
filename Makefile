# Local development and builds (ADR-031). Tooling, outside the MCSWE manifest's scope.
#
#   make dev    Go API on 127.0.0.1:8080 plus Vite with hot reload on http://127.0.0.1:5173
#   make build  frontend build, then the ossalot binary with the UI embedded

GO ?= go
# Use GITHUB_TOKEN when set, else the token from a gh login
TOKEN = $${GITHUB_TOKEN:-$$(gh auth token 2>/dev/null)}

.PHONY: dev build web

web/node_modules: web/package-lock.json
	cd web && npm ci
	@touch web/node_modules

web: web/node_modules
	cd web && npm run build

dev: web/node_modules
	@echo "UI with hot reload: http://127.0.0.1:5173   (API on 127.0.0.1:8080)"
	@trap 'kill 0' INT TERM EXIT; \
	GITHUB_TOKEN="$(TOKEN)" $(GO) run ./cmd/ossalot & \
	(cd web && npm run dev) & \
	wait

build: web
	$(GO) build -o ossalot ./cmd/ossalot
