#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT_DIR"

GOOS_VALUE="${GOOS:-linux}"
GOARCH_VALUE="${GOARCH:-amd64}"
OUTPUT_DIR="temp/${GOOS_VALUE}_${GOARCH_VALUE}"

go mod tidy
go test ./...

mkdir -p "$OUTPUT_DIR"
CGO_ENABLED=0 GOOS="$GOOS_VALUE" GOARCH="$GOARCH_VALUE" go build -o "${OUTPUT_DIR}/main" main.go

echo "built ${OUTPUT_DIR}/main"
