# LoRa Crowd Density Measurement

> Measure crowd density in public spaces using BLE scanning on ESP32 sensors, transmitted via LoRaWAN (TTN), stored in TimescaleDB, and visualised in a web UI and Grafana.

[![MIT License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Codeberg](https://img.shields.io/badge/source-Codeberg-2185D0.svg)](https://codeberg.org/cfhn/lora-frequenzmessung)
[![CI](https://ci.codeberg.org/api/badges/cfhn/lora-frequenzmessung/status.svg)](https://codeberg.org/cfhn/lora-frequenzmessung/actions)

---

> **Canonical repository:** https://codeberg.org/cfhn/lora-frequenzmessung
>
> Mirrors exist on GitHub and GitLab for visibility, but they are read-only.
> **Contributions, issues, and pull requests belong on Codeberg.**

---

## Introduction

This project provides an end-to-end system for measuring crowd density in public spaces. Small ESP32-based sensors scan for nearby Bluetooth devices and transmit a count over LoRaWAN via The Things Network. A backend stack ingests the data and makes it available for real-time dashboards.

**Data flow:**

```
ESP32 (BLE scan)
    └─ LoRaWAN (TTN) ─→ MQTT Broker
                              └─ Aggregator ─→ Backend API ─→ TimescaleDB
                                                                    └─→ Grafana
                                                                    └─→ Web UI (via ManagementAPI)
```

**Maintained by** [Code for Heilbronn (CFHN)](https://codeforheilbronn.de/).

---

## Architecture

### Component Overview

| Service        | Language   | Port  | Purpose                                              |
|----------------|------------|-------|------------------------------------------------------|
| Backend        | Go/Fiber   | 3001  | Internal sensor CRUD + data ingestion (X-Internal-Key protected) |
| ManagementAPI  | Go/Fiber   | 3002  | Public API: JWT auth, TTN registration, firmware builds |
| Aggregator     | Go         | —     | MQTT subscriber → forwards data to Backend           |
| Frontend       | Vue 3/Vite | 5173  | Management web UI                                    |
| TimescaleDB    | PostgreSQL | 5432  | Time-series sensor data storage                      |
| Mosquitto      | MQTT       | 1883  | Message broker (TTN bridge)                          |
| Grafana        | —          | 3000  | Dashboard / visualisation                            |

### Request Paths

```
Browser  ──→  Frontend (Vite, :5173)
                  └─ /api/*  ──→  ManagementAPI (:3002)  ──→  Backend (:3001)
                  └─ /auth/* ──→  ManagementAPI (:3002)

TTN      ──→  MQTT (:1883)
                  └─  Aggregator  ──→  Backend (:3001)  ──→  TimescaleDB
```

<!-- Screenshots / diagrams can be added here once available -->

---

## Hardware Requirements

- **Heltec WiFi LoRa 32 V3** (ESP32-S3) — one per measurement location
- A LoRaWAN gateway in range
- A [The Things Network](https://www.thethingsnetwork.org/) application
- A server to run the backend stack (Docker + Docker Compose)

---

## Getting Started

### Prerequisites

| Tool | Version |
|------|---------|
| Go | 1.24+ |
| Node.js | 18+ |
| Docker & Docker Compose | latest |
| PlatformIO CLI | latest (for firmware builds) |
| `golang-migrate` | latest |

Install `golang-migrate` (macOS):

```bash
brew install golang-migrate
```

### Quick Start (Development)

```bash
# 1. Clone
git clone https://codeberg.org/cfhn/lora-frequenzmessung.git
cd lora-frequenzmessung

# 2. Configure environment
cp .env.local.dist backend/.env
# Edit backend/.env and fill in all required values (see Environment Variables below)

# 3. Start infrastructure (TimescaleDB, Mosquitto, Grafana)
make infra

# 4. Run database migrations
migrate -database "$DB_DSN" -path db/migrations up

# 5. Start all services in parallel
make dev
```

`make dev` starts the Backend (`:3001`), ManagementAPI (`:3002`), Aggregator, and Frontend (`:5173`) in parallel.

### Environment Variables

Copy `.env.local.dist` to `backend/.env` and fill in the required values.

| Variable | Required | Service(s) | Description |
|---|---|---|---|
| `DB_DSN` | Yes | Backend | PostgreSQL connection string |
| `INTERNAL_API_KEY` | Yes | Backend, ManagementAPI, Aggregator | Shared secret for internal service auth |
| `BACKEND_INTERNAL_URL` | Yes | ManagementAPI, Aggregator | URL of the internal Backend API (e.g. `http://localhost:3001`) |
| `JWT_SECRET` | Yes | ManagementAPI | HS256 signing key — min 32 characters |
| `ADMIN_USERNAME` | Yes | ManagementAPI | Admin login username |
| `ADMIN_PASSWORD` | Yes | ManagementAPI | Admin login password |
| `BROKER_HOST` | Yes | Aggregator | MQTT broker hostname |
| `BROKER_PORT` | Yes | Aggregator | MQTT broker port (usually `1883`) |
| `MQTT_USERNAME` | Yes | Aggregator | MQTT username (TTN app ID + `@ttn`) |
| `MQTT_PASSWORD` | Yes | Aggregator | MQTT API key from TTN |
| `TOPIC` | Yes | Aggregator | TTN MQTT topic to subscribe to |
| `TTN_APP_ID` | No | ManagementAPI | TTN application ID (for device provisioning) |
| `TTN_API_KEY` | No | ManagementAPI | TTN API key (for device provisioning) |
| `SENSOR_PAX_PATH` | No | ManagementAPI | Path to firmware source directory (default: `./sensor-pax`) |
| `CORS_ORIGINS` | No | ManagementAPI | Comma-separated allowed origins (e.g. `http://localhost:5173`) |

---

## Development Commands

All commands are run from the repository root via `make`. Run `make help` to see the full list.

```bash
make help              # Show all available targets

# Infrastructure
make infra             # Start TimescaleDB, Mosquitto, Grafana
make infra-down        # Stop infrastructure containers

# Backend services (each in its own terminal, or use make dev)
make backend           # Run internal Backend API (port 3001)
make management        # Run ManagementAPI (port 3002)
make aggregator        # Run MQTT Aggregator

# Frontend
make frontend          # Run Vite dev server (port 5173)

# All services at once
make dev               # Start infra + all services in parallel

# Testing
make test              # Run all Go tests
make test-frontend     # Run Playwright unit tests (mocked, no backend needed)
make test-frontend-live # Run Playwright integration tests (needs running backend)

# Build
make build             # Build all Go binaries + frontend bundle
```

---

## Testing

### Go Tests

```bash
cd backend
go test ./...
```

### Frontend — Playwright

There are two test projects configured in `frontend/playwright.config.ts`:

| Project | Command | Requires backend? |
|---|---|---|
| `unit` | `make test-frontend` | No (uses mocked API) |
| `integration` | `make test-frontend-live` | Yes (hits real backend) |

Run a single spec:

```bash
cd frontend
npx playwright test e2e/add-sensor.spec.ts
```

---

## Production Deployment

Docker images are published to `codeberg.org/cfhn/lora-frequenzmessung/<service>`.

Use `docker-compose-deploy.yml` to deploy the full stack:

```bash
docker compose -f docker-compose-deploy.yml up -d
```

All environment variables listed in the table above must be set in the production environment. For `POSTGRES_PASSWORD` and `GF_ADMIN_PASSWORD`, **change the defaults** — the development defaults (`password` / `admin`) are insecure.

---

## Project Structure

```
lora-frequenzmessung/
├── backend/               # All Go source code (single go.mod)
│   ├── Backend/           # Internal HTTP API (port 3001)
│   ├── ManagementAPI/     # Public JWT API (port 3002)
│   ├── Aggregator/        # MQTT subscriber
│   ├── structs/           # Shared Go types
│   └── internal/          # Shared internal packages
├── frontend/              # Vue 3 + Vite management UI
│   └── e2e/               # Playwright tests
├── sensor-pax/            # ESP32 firmware (PlatformIO / libpax)
├── db/
│   └── migrations/        # SQL migrations (golang-migrate)
├── docker/                # Dockerfiles + service configs
├── docs/                  # Additional documentation
├── hardware/              # Hardware schematics / notes
├── docker-compose.yml     # Development infrastructure
├── docker-compose-deploy.yml # Production deployment
└── Makefile               # Developer shortcuts
```

---

## Contributing

Contributions are welcome! This project is maintained by volunteers at [Code for Heilbronn (CFHN)](https://codeforheilbronn.de/).

**All contributions go through Codeberg:**

1. Fork the repository on [Codeberg](https://codeberg.org/cfhn/lora-frequenzmessung)
2. Create a feature branch (`git checkout -b feature/my-feature`)
3. Commit your changes following the existing code style
4. Open a pull request on Codeberg

Please open an issue first for larger changes so we can discuss the approach.

---

## License

MIT © 2026 [Open Data Heilbronn](https://codeforheilbronn.de/)

See [LICENSE](LICENSE) for the full text.
