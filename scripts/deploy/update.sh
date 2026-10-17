#!/usr/bin/env bash
# Pulls the binary from the latest green CI run on main and restarts ossalot
# when it changed (ADR-035). Run by ossalot-update.timer; safe to run by hand.
set -euo pipefail

REPO="${OSSALOT_REPO:-fupacat/ossalot}"
INSTALL_DIR="${OSSALOT_HOME:-/opt/ossalot}"
STATE="$INSTALL_DIR/deployed-sha"

run=$(gh run list -R "$REPO" --workflow ci.yml --branch main --event push \
  --status success --limit 1 --json databaseId,headSha \
  --jq '.[0] | "\(.databaseId) \(.headSha)"')
[ -n "$run" ] || { echo "no green run on main yet"; exit 0; }
read -r run_id sha <<<"$run"

if [ -f "$STATE" ] && [ "$(cat "$STATE")" = "$sha" ]; then
  exit 0
fi

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
gh run download "$run_id" -R "$REPO" -n ossalot-test-binary -D "$tmp"
chmod +x "$tmp/ossalot"
OSSALOT_DB_PATH="$tmp/check.db" "$tmp/ossalot" -check

install -m 0755 "$tmp/ossalot" "$INSTALL_DIR/ossalot.new"
mv -f "$INSTALL_DIR/ossalot.new" "$INSTALL_DIR/ossalot"
echo "$sha" > "$STATE"
systemctl restart ossalot.service
echo "deployed $sha (run $run_id)"
