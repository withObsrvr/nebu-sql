#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

go build -o ./nebu-sql ./cmd/nebu-sql

processors=(
  token-transfer
  contract-events
  contract-invocation
  transaction-stats
  ledger-change-stats
  account-effects
)

start_ledger="${1:-60200000}"
stop_ledger="${2:-60200000}"

for p in "${processors[@]}"; do
  echo "=== $p ==="
  if ! command -v "$p" >/dev/null 2>&1; then
    echo "missing on PATH; installing via nebu..."
    nebu install "$p"
  fi

  if ! "$p" --describe-json >/dev/null 2>&1; then
    echo "skip: $p does not support --describe-json"
    echo
    continue
  fi

  ./nebu-sql -c "select count(*) as n from nebu('$p', start = $start_ledger, stop = $stop_ledger)"
  ./nebu-sql -c "select * from nebu('$p', start = $start_ledger, stop = $stop_ledger) limit 1"
  echo
 done
