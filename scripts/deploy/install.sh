#!/usr/bin/env bash
# One-time host setup for the Cloudflare dev environment (ADR-035).
# Usage: sudo ./install.sh <hostname>   e.g. sudo ./install.sh ossalot-dev.example.com
# Needs: gh (logged in for root, or GH_TOKEN in /etc/ossalot/env) and cloudflared.
set -euo pipefail

HOSTNAME_ARG="${1:?usage: install.sh <public hostname>}"
INSTALL_DIR=/opt/ossalot
HERE="$(cd "$(dirname "$0")" && pwd)"

for bin in gh cloudflared; do
  command -v "$bin" >/dev/null || { echo "missing $bin"; exit 1; }
done

id ossalot >/dev/null 2>&1 || useradd --system --home "$INSTALL_DIR" --shell /usr/sbin/nologin ossalot
install -d -o ossalot -g ossalot "$INSTALL_DIR" "$INSTALL_DIR/data"
install -d -m 0750 /etc/ossalot
if [ ! -f /etc/ossalot/env ]; then
  install -m 0600 "$HERE/env.example" /etc/ossalot/env
  echo "edit /etc/ossalot/env (GITHUB_TOKEN, GH_TOKEN) before continuing"; exit 1
fi

install -m 0755 "$HERE/update.sh" "$INSTALL_DIR/update.sh"
install -m 0644 "$HERE"/systemd/*.service "$HERE"/systemd/*.timer /etc/systemd/system/
systemctl daemon-reload

# Tunnel: cloudflared tunnel login must have been run once (writes ~/.cloudflared/cert.pem)
cloudflared tunnel info ossalot-dev >/dev/null 2>&1 || cloudflared tunnel create ossalot-dev
cloudflared tunnel route dns ossalot-dev "$HOSTNAME_ARG" || true
install -d /etc/cloudflared
tunnel_id=$(cloudflared tunnel info ossalot-dev -o json | python3 -c 'import json,sys; print(json.load(sys.stdin)["id"])')
cp ~/.cloudflared/"$tunnel_id".json /etc/cloudflared/
cat > /etc/cloudflared/config.yml <<CFG
tunnel: $tunnel_id
credentials-file: /etc/cloudflared/$tunnel_id.json
ingress:
  - hostname: $HOSTNAME_ARG
    service: http://127.0.0.1:8080
  - service: http_status:404
CFG
cloudflared service install 2>/dev/null || systemctl restart cloudflared

set -a; . /etc/ossalot/env; set +a
"$INSTALL_DIR/update.sh"
systemctl enable --now ossalot.service ossalot-update.timer
echo "done: https://$HOSTNAME_ARG  (now add a Cloudflare Access application for it)"
