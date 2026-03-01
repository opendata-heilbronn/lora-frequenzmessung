.PHONY: help backend management aggregator frontend infra migrate test build

ENV_FILE := $(CURDIR)/backend/.env
LOAD_ENV := set -a && source $(ENV_FILE) && set +a

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'

# ── Infrastructure ────────────────────────────────────────────────────────────

infra: ## Start TimescaleDB, Mosquitto and Grafana via Docker Compose
	docker compose up -d timescaledb mosquitto grafana

infra-down: ## Stop infrastructure containers
	docker compose down

migrate: ## Run DB migrations against local TimescaleDB (reads DB_DSN from backend/.env)
	$(LOAD_ENV) && migrate -database "$$DB_DSN" -path backend/Backend/migrations up

# ── Backend services ──────────────────────────────────────────────────────────

backend: ## Run the internal Backend API (port 3001)
	cd backend && $(LOAD_ENV) && go run ./Backend

management: ## Run the ManagementAPI with JWT auth (port 3002)
	cd backend && $(LOAD_ENV) && go run ./ManagementAPI

aggregator: ## Run the MQTT Aggregator
	cd backend && $(LOAD_ENV) && go run ./Aggregator

# ── Frontend ──────────────────────────────────────────────────────────────────

frontend: ## Run the Vite dev server (port 5173)
	cd frontend && npm run dev

# ── Testing ───────────────────────────────────────────────────────────────────

test: ## Run all Go tests
	cd backend && go test ./...

test-frontend: ## Run Playwright unit tests (mocked, no backend needed)
	cd frontend && npx playwright test --project=unit

test-frontend-live: ## Run Playwright integration tests (needs running backend)
	cd frontend && npx playwright test --project=integration

# ── Build ─────────────────────────────────────────────────────────────────────

build: ## Build all Go binaries to /tmp
	cd backend && go build -o /tmp/backend-bin ./Backend
	cd backend && go build -o /tmp/management-bin ./ManagementAPI
	cd backend && go build -o /tmp/aggregator-bin ./Aggregator
	cd frontend && npm run build

# ── Dev shortcuts ─────────────────────────────────────────────────────────────

dev: ## Start infra + all backend services + frontend in parallel
	$(MAKE) infra
	$(MAKE) -j4 backend management aggregator frontend
