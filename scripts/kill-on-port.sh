#!/usr/bin/env bash
set -euo pipefail

# kill-on-port.sh - Kill all processes listening on a given TCP port (default: 8080)
# Usage: ./kill-on-port.sh [PORT]

PORT="${1:-8080}"

# Try lsof first
PIDS=""
if command -v lsof >/dev/null 2>&1; then
  PIDS=$(lsof -t -i :"$PORT" 2>/dev/null || true)
fi

# Fallback to ss if lsof didn't return anything
if [ -z "$PIDS" ] && command -v ss >/dev/null 2>&1; then
  PIDS=$(ss -ltnp 2>/dev/null | awk -v port=":$PORT" '$0 ~ port { match($0, /pid=[0-9]+/, m); if (RSTART) { pid=substr($0, RSTART+4, RLENGTH-4); print pid }}' | sort -u)
fi

if [ -z "$PIDS" ]; then
  echo "No processes found listening on port $PORT"
  exit 0
fi

echo "Found PIDs listening on port $PORT: $PIDS"

# Send SIGTERM first
echo "$PIDS" | tr ' ' '\n' | xargs -r -n1 sh -c 'echo "TERM -> $0"; kill "$0"' || true

# Wait briefly for processes to exit
for i in 1 2 3 4 5; do
  sleep 1
  REMAINING=""
  for p in $PIDS; do
    if kill -0 "$p" 2>/dev/null; then
      REMAINING="$REMAINING $p"
    fi
  done
  REMAINING=$(echo "$REMAINING" | xargs || true)
  if [ -z "$REMAINING" ]; then
    echo "Processes terminated gracefully"
    exit 0
  fi
  PIDS="$REMAINING"
done

# Force kill remaining
echo "Forcing kill -9 on: $PIDS"
echo "$PIDS" | tr ' ' '\n' | xargs -r -n1 sh -c 'echo "KILL -> $0"; kill -9 "$0"' || true

echo "Done"
