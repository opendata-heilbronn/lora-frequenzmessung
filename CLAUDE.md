# CLAUDE.md
# key rules: 
* for each feature we build we builed in the frontent we build at least one playwrite test, at best for each function one, to ensure we dont break features in the future.
* after adding a new feature run playwrite tests to ensure we did not break somthing
* for each code we write we add tests 
* 
This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Test Commands

### Backend (Go, from `backend/` directory)
```bash
go build -o /tmp/backend-bin ./Backend      # build API server
go build -o /tmp/aggregator-bin ./Aggregator # build MQTT aggregator
go test ./...                                # all Go tests
go test ./Backend/ -v -run TestName          # single test
```

### Frontend (Vue 3/Vite, from `frontend/` directory)
```bash
npm run dev           # dev server on :5173
npm run build         # production build (includes vue-tsc type check)
npm run test:e2e      # all Playwright tests (unit + integration)
npx playwright test e2e/add-sensor.spec.ts              # single spec file
npx playwright test e2e/add-sensor-live.spec.ts --workers=1  # live tests (need running backend)
```

### Infrastructure
```bash
docker-compose up -d   # TimescaleDB(:5432), Mosquitto(:1883), Grafana(:3000)
migrate -database "$DB_DSN" -path db/migrations up  # run DB migrations
```

## Architecture

```
ESP32 Sensor → TTN (LoRaWAN) → MQTT → Aggregator → Backend API → TimescaleDB
                                                         ↑
                                              Frontend (Vue 3) ──→ Browser USB Flash
```

**Backend** (`backend/Backend/`, Fiber v3, port 3001): HTTP API for sensor CRUD, TTN registration, firmware building/serving. Uses a single `pgx.Conn` (not a pool — not safe for concurrent requests).

**Aggregator** (`backend/Aggregator/`): MQTT subscriber on TTN broker. Fetches sensor list via `GET /api/sensors` with 30s in-memory cache. Parses base64 TTN payloads (format: `uuid,type_id,value,...` where type 0=density, 1=battery).

**Frontend** (`frontend/`, Vue 3 + Vite): Management UI. Vite proxies `/api/*` to backend. Uses `esp-web-tools` web component for USB flashing (Chrome/Edge only).

**Sensor Firmware** (`sensor-pax/`, PlatformIO): Targets Heltec WiFi LoRa 32 V3 (ESP32-S3). Uses libpax for BLE/WiFi scanning. LoRa is optional — controlled by `ENABLE_LORA` in generated `customs.h`.

**Single Go module** at `backend/go.mod` covers both Backend and Aggregator packages.

## Key Implementation Details

- **Sensor UUIDs**: 8-char hex from `crypto/rand` (not full UUIDs)
- **Firmware builds are async**: `POST /api/sensors/:uuid/build-firmware` starts a goroutine, status tracked in `sync.Map` (lost on restart). Frontend polls `GET /api/sensors/:uuid/build-status` every 3s
- **Firmware customization**: Copies `sensor-pax/` to temp dir, generates `src/customs.h` with sensor UUID + optional LoRa keys, runs `platformio run`
- **`SENSOR_PAX_PATH`**: Defaults to `./sensor-pax`, resolves relative to CWD with fallback to `../sensor-pax` (for running from `backend/` dir)
- **TTN registration**: Two-step — POST to Identity Server (create device), PUT to Join Server (assign keys). Env vars: `TTN_APP_ID`, `TTN_API_KEY`
- **Structs** (`backend/structs/Clients.go`): Have both `yaml:` and `json:` tags for backward compatibility

## Playwright Test Setup

Two projects in `playwright.config.ts`:
- **`unit`**: Mocked API tests (`page.route()` intercepts), run in parallel
- **`integration`**: Live tests (`*-live.spec.ts`), hit real backend, run serially after unit tests via `dependencies`

Live integration tests create real sensors/TTN devices and clean up via the Delete button. They need the backend running.

## Environment Variables

Template: `.env.local.dist`. Key vars:
- `DB_DSN` — Postgres connection string
- `BROKER_HOST`, `BROKER_PORT`, `MQTT_USERNAME`, `MQTT_PASSWORD`, `TOPIC` — MQTT/TTN
- `BACKEND_URL` — used by Aggregator to reach Backend
- `TTN_APP_ID`, `TTN_API_KEY` — TTN device provisioning
- `SENSOR_PAX_PATH` — path to firmware source (default `./sensor-pax`)

## Database

TimescaleDB (PostgreSQL 17). Migrations in `db/migrations/`:
- `000001`: `sensor_data` hypertable (time-series sensor readings)
- `000002`: `sensors` table (device registry with TTN credentials)

<!-- SPECKIT START -->
Current active plan: specs/002-sensor-location-change/plan.md
For additional context: specs/002-sensor-location-change/research.md, specs/002-sensor-location-change/data-model.md, specs/002-sensor-location-change/contracts/api.md
<!-- SPECKIT END -->
