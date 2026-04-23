# AIGateway Console Backend

## Current State

- `backend/` is now the active Go + GoFrame backend.
- `backend-java-legacy/` keeps the original Java/Spring Boot implementation as the migration reference baseline.
- The current codebase now covers:
  - `P1` foundation: scaffold, middleware, config, client abstractions, landing/static layout
  - `P2` platform core: `session`, `user`, `system`, `dashboard`, `healthz`, `landing`
  - `P3` first-cut gateway resource domain: `route`, `domain`, `tls`, `service`, `service-source`, `proxy`, `wasm`, `ai`, `mcp`
- The current gateway resource implementation is intentionally an in-memory/fake-client migration step for review and contract stabilization. Full K8s/CRD parity remains follow-up work.

## Directory Guide

- `api/`: HTTP request/response definitions.
- `internal/controller/`: GoFrame controllers grouped by domain.
- `internal/service/`: business services, no separate `logic/` layer.
- `internal/dao`, `internal/model/do`, `internal/model/entity`: Portal DB access layer for later phases.
- `internal/middleware/`: trace, access log, auth, and response middleware.
- `internal/job/`: cron and reconcile jobs.
- `utility/clients/`: adapters for K8s, Portal DB, and Grafana.
- `test/contracts/`: contract fixtures to compare Go behavior with `backend-java-legacy/`.

## Quick Start

```bash
cd backend
go test ./...
go run main.go
```

Default server address is `:8080`.

## Build

```bash
cd backend
./build.sh
```

## Next Migration Phases

- `P4`: Portal DB domain (`consumer`, `org`, `invite`, `model assets`, `stats`)
- `P5`: jobs and reconcile
- `P6`: frontend cutover and explicit `aigateway` rename cleanup
