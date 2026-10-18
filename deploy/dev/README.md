# Dev environment: Docker Desktop + Cloudflare Tunnel (ADR-036)

```
push to main → CI test-and-build (GitHub-hosted)
            → publish-image: ghcr.io/fupacat/ossalot:main and :<sha>   (GitHub-hosted)
            → deploy-dev: docker compose pull && up -d, then checks    (self-hosted runner on this host)
browser → Cloudflare Access → Cloudflare Tunnel (cloudflared container) → ossalot:8080
```

CI and the image build run in GitHub. Only the last step runs on your machine, through a GitHub Actions self-hosted runner that connects out to GitHub, so the host needs no open port.

## One-time setup

1. **Cloudflare tunnel.** Zero Trust → Networks → Tunnels → Create a tunnel (Cloudflared) named `ossalot-dev`. Copy the token from the install command. Under Public Hostname, add `ossalot.fupacat.dev` → `HTTP` → `ossalot:8080`.
2. **Cloudflare Access.** Zero Trust → Access → Applications → Add → Self-hosted for `ossalot.fupacat.dev`, policy Allow, include Emails = your address. Do this before anything else is reachable: the app holds a GitHub token that can open issues and pull requests.
3. **GitHub environment.** Repo → Settings → Environments → New `dev`. Add secrets `OSSALOT_GITHUB_TOKEN` (the token the app uses) and `TUNNEL_TOKEN` (from step 1). Set Deployment branches to `main` only.
4. **Self-hosted runner.** Repo → Settings → Actions → Runners → New self-hosted runner, follow the steps for your OS on the machine running Docker Desktop, and add the label `ossalot-dev`. Run it as a service so it survives reboots. On Windows, install the **Linux** runner inside the WSL distro that has Docker Desktop's WSL integration turned on, because the job's steps run in bash.
   The deploy job logs in to GHCR with its own `GITHUB_TOKEN`, using a Docker config that is deleted after the job. A private `ghcr.io/fupacat/ossalot` package therefore needs no extra setup: the image's source label links the package to this repository.
5. **Public repo guard.** The repo is public. Settings → Actions → General → set fork pull request workflows to "Require approval for all external contributors" (the default only covers first-time contributors). The deploy job only runs on pushes to `main` in the `dev` environment, so pull requests never reach the runner.

Push to `main` (or re-run the CI workflow) and the stack comes up. `deploy-dev` passes only when `GET /api/v1/health` answers and cloudflared's `/ready` reports a live tunnel connection. Both are probed over the stack's network, so a bad `TUNNEL_TOKEN` fails the deploy.

## Day to day

- Deployed version: `docker compose -p ossalot-dev ps`, or the `dev` environment's page on GitHub.
- Logs: `docker compose -p ossalot-dev logs -f ossalot`
- Roll back: re-run the `deploy-dev` job of an older CI run, or run `OSSALOT_TAG=<sha> docker compose up -d` here with the tokens in `.env`.
- Data lives in the `ossalot-dev_ossalot-data` volume and survives deploys.
