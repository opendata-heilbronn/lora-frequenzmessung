<!--
SYNC IMPACT REPORT
==================
Version change: [template] → 1.0.0 (initial ratification — all placeholders replaced)

Modified principles: N/A (first-time fill)

Added sections:
  - Core Principles (5 principles)
  - Technology Stack
  - Development Workflow & Quality Gates
  - Governance

Removed sections: N/A

Templates requiring updates:
  - .specify/templates/plan-template.md ✅ aligned (Constitution Check section references this document)
  - .specify/templates/spec-template.md ✅ aligned (FR/SC format consistent with principles)
  - .specify/templates/tasks-template.md ✅ aligned (test-first task ordering matches Principle II)
  - No commands/ directory found — skipped

Follow-up TODOs:
  - None — all placeholders resolved
-->

# lora-frequenzmessung Constitution

## Core Principles

### I. Microservice Separation (NON-NEGOTIABLE)

Every service MUST own a single responsibility and MUST NOT be merged with another:

- **Backend** (`backend/Backend/`, port 3001): sensor CRUD and data ingestion only. Accessible
  only via `X-Internal-Key` header — never exposed to the public internet or frontend directly.
- **ManagementAPI** (`backend/ManagementAPI/`, port 3002): the sole public-facing API. Owns JWT
  auth, TTN provisioning, firmware orchestration, and CORS. Proxies sensor CRUD to Backend.
- **Aggregator** (`backend/Aggregator/`): MQTT subscriber only. Calls Backend via internal key.
- **Frontend** (`frontend/`): MUST proxy all requests through ManagementAPI (`/api`, `/auth`
  → port 3002). Frontend MUST NOT call Backend directly.

Adding a new capability to the wrong service is a constitutional violation requiring explicit
justification in the plan's Complexity Tracking table.

### II. Test-First — Every Feature Has Tests (NON-NEGOTIABLE)

Testing is a delivery requirement, not an optional add-on:

- Every frontend feature MUST ship with at least one Playwright test covering its primary user
  journey. Ideally each discrete function has its own test.
- Go backend code MUST include unit or integration tests for new logic.
- After every feature addition, the full Playwright test suite MUST be run to catch regressions
  before the work is considered complete.
- New tests MUST be written alongside the feature code in the same commit/PR — never deferred.

Test projects in `playwright.config.ts`: `unit` (mocked, parallel) and `integration` (live,
serial, depend on a running backend). Live tests clean up created resources via the Delete button.

### III. Authentication Layers Are Fixed

Authentication boundaries MUST NOT be bypassed or renegotiated without a constitution amendment:

- Backend ↔ ManagementAPI and Backend ↔ Aggregator: `X-Internal-Key` header (env:
  `INTERNAL_API_KEY`). All three services MUST share the identical key value.
- Frontend ↔ ManagementAPI: JWT (HS256, 24 h expiry, `github.com/golang-jwt/jwt/v5`).
  Token stored in `localStorage`. All management routes require a valid token.
- CORS is configured on ManagementAPI only. Backend has no CORS.
- Credentials: single admin user via `ADMIN_USERNAME` / `ADMIN_PASSWORD` env vars.

### IV. Async for Long-Running Operations

Operations that may exceed a normal HTTP response window MUST be made asynchronous:

- Firmware builds MUST run in a goroutine. Status MUST be tracked in `sync.Map` and exposed
  via a polling endpoint (`GET /api/sensors/:uuid/build-status` every 3 s from the frontend).
- The Backend uses a single `pgx.Conn` (not a pool) — concurrent requests are NOT safe.
  Long-running handlers MUST account for this constraint.
- Status state held in `sync.Map` is ephemeral (lost on restart); this is acceptable for
  firmware build tracking.

### V. Simplicity & YAGNI

Keep implementation direct and avoid speculative abstractions:

- Sensor UUIDs are 8-char hex strings from `crypto/rand` — not full RFC UUIDs. Do not upgrade
  to full UUIDs without a clear requirement.
- `SENSOR_PAX_PATH` resolves relative to CWD with a `../sensor-pax` fallback; do not introduce
  a configuration service for a two-path lookup.
- Structs carry both `yaml:` and `json:` tags for backward compatibility — preserve both tags
  on any struct touched in `backend/structs/`.
- No half-finished implementations. No feature flags unless explicitly required by a spec.

## Technology Stack

The following stack is fixed for this project. Deviations require explicit justification.

| Layer | Technology | Version / Notes |
|---|---|---|
| Backend API | Go + Fiber v3 | Port 3001, single `pgx.Conn` |
| Management API | Go + Fiber v3 | Port 3002, JWT, CORS |
| Aggregator | Go | MQTT subscriber, 30 s sensor cache |
| Frontend | Vue 3 + Vite | Port 5173 (dev) / 8080 (prod) |
| Database | TimescaleDB (PostgreSQL 17) | `sensor_data` hypertable + `sensors` table |
| Message Broker | Mosquitto | Port 1883 |
| Dashboards | Grafana | Port 3000 |
| Firmware | PlatformIO (ESP32-S3) | `sensor-pax/`, Heltec WiFi LoRa 32 V3 |
| Go module | `backend/go.mod` | Single module for Backend, ManagementAPI, Aggregator |
| E2E Testing | Playwright | `unit` + `integration` projects |
| TTN Integration | TTN API | 4-step provisioning (IS → JS → NS → AS) |

Infrastructure is managed via `docker-compose up -d`. DB schema changes MUST use migrate
migrations in `db/migrations/` — direct DDL edits to the database are prohibited.

## Development Workflow & Quality Gates

### Build & Test

```bash
# Backend (from backend/)
go build -o /tmp/backend-bin ./Backend
go build -o /tmp/management-bin ./ManagementAPI
go build -o /tmp/aggregator-bin ./Aggregator
go test ./...

# Frontend (from frontend/)
npm run dev            # dev server :5173
npm run build          # production build (includes vue-tsc type check)
npm run test:e2e       # all Playwright tests
```

### Quality Gates (enforce before merge)

1. `go test ./...` MUST pass with zero failures.
2. `npm run build` MUST succeed (type errors block merge).
3. `npm run test:e2e` MUST pass — both `unit` and `integration` projects.
4. No new public route on Backend (all Backend routes MUST be `/internal/*`).
5. No hardcoded secrets — all sensitive values MUST be in env vars.

### Feature Delivery Checklist

- [ ] Backend route added under `/internal/` if applicable.
- [ ] ManagementAPI proxy/handler added if user-facing.
- [ ] Playwright test written for every new frontend function.
- [ ] Go test written for every new backend handler.
- [ ] `npm run test:e2e` run and passing before marking complete.
- [ ] Environment variable documented if added.
- [ ] DB migration created if schema changed.

## Governance

This constitution supersedes all other documented practices. When a CLAUDE.md instruction
conflicts with this constitution, the constitution takes precedence; update CLAUDE.md to align.

**Amendment procedure**:
1. Propose the amendment in a PR description with rationale.
2. Bump `CONSTITUTION_VERSION` following semantic versioning:
   - MAJOR: principle removed or fundamentally redefined.
   - MINOR: new principle or major section added.
   - PATCH: clarification, wording, or typo fix.
3. Update `LAST_AMENDED_DATE` to today's date (ISO 8601).
4. Run the consistency propagation checklist (plan, spec, tasks templates) and note any
   template updates in the Sync Impact Report comment at the top of this file.

**Compliance review**: Every PR description MUST include a "Constitution Check" section
confirming no principles are violated, or explicitly justifying any deviation in the plan's
Complexity Tracking table.

**Runtime guidance**: See `CLAUDE.md` for build commands, test invocations, and
environment variable details.

**Version**: 1.0.0 | **Ratified**: 2026-05-01 | **Last Amended**: 2026-05-01
