# Cloudflare dev environment (ADR-035)

The dev instance runs the binary from the latest green CI run on `main` on a Linux x86-64 host you own, exposed through a Cloudflare Tunnel and protected by Cloudflare Access. The host pulls new builds; nothing in CI needs Cloudflare credentials or inbound access to the host.

```
push to main → CI (ci.yml) uploads ossalot-test-binary
host: ossalot-update.timer (5 min) → update.sh → gh run download → -check → restart ossalot.service
browser → Cloudflare Access → Tunnel (cloudflared) → 127.0.0.1:8080
```

## One-time setup

1. On the host, install [`gh`](https://cli.github.com/) and [`cloudflared`](https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/downloads/).
2. `cloudflared tunnel login` and pick the zone (domain) on your Cloudflare account.
3. Create a fine-grained GitHub token with read access to `fupacat/ossalot` (Contents, Actions) for `GH_TOKEN`; `GITHUB_TOKEN` is the token the app itself uses.
4. `sudo ./scripts/deploy/install.sh ossalot-dev.<your-domain>`. The first run copies `env.example` to `/etc/ossalot/env` and stops; fill it in and run it again.
5. Cloudflare dashboard → Zero Trust → Access → Applications → Add → Self-hosted: domain `ossalot-dev.<your-domain>`, policy Allow, include Emails = your address. Do this before sharing the URL: the app holds a GitHub token that can open issues and pull requests.

## Day to day

- Deploy now: `sudo systemctl start ossalot-update.service`
- Logs: `journalctl -u ossalot -u ossalot-update -f`
- Deployed commit: `cat /opt/ossalot/deployed-sha`
- Roll back: stop the timer, download an older run with `gh run download <run-id> -R fupacat/ossalot -n ossalot-test-binary`, install it to `/opt/ossalot/ossalot` and restart.

Data lives in `/opt/ossalot/data` and survives deploys. CI keeps artifacts for 7 days, so a host offline longer than that waits for the next push to `main`.
