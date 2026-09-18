#!/usr/bin/env bash
# Repeatable device benchmark for adb-claw realtime work.
# Usage: scripts/bench-device.sh [serial] [rounds]
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SERIAL="${1:-}"
ROUNDS="${2:-5}"
BIN="${ROOT}/bin/adb-claw"

if [[ ! -x "$BIN" ]]; then
  (cd "$ROOT/src" && make build)
fi

args=()
if [[ -n "$SERIAL" ]]; then
  args+=(-s "$SERIAL")
fi

for mode in stream pull; do
  "$BIN" "${args[@]}" bench --rounds "$ROUNDS" --capture "$mode"
done
