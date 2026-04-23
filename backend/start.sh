#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT_DIR"

if command -v gf >/dev/null 2>&1; then
  exec gf run main.go
fi

exec go run main.go
