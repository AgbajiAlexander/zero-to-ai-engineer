# ZERO → AI ENGINEER — Developer Makefile
#
# Common targets for local development.
# All commands assume you are running from the repository root.
#
# Prerequisites: Go, Docker, docker compose

.DEFAULT_GOAL := help

API_DIR   := apps/api
BINARY    := bin/server

# ── Help ──────────────────────────────────────────────────────────────────────
.PHONY: help
help: ## Show this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
	  awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

# ── Go API ────────────────────────────────────────────────────────────────────
.PHONY: build
build: ## Build the Go API binary
	@mkdir -p bin
	cd $(API_DIR) && go build -o ../../$(BINARY) ./cmd/server
	@echo "Built: $(BINARY)"

.PHONY: run
run: ## Run the Go API locally (reads .env)
	@cd $(API_DIR) && \
	  export $$(grep -v '^#' ../../.env | xargs) 2>/dev/null; \
	  go run ./cmd/server

.PHONY: test
test: ## Run all Go tests (short mode — skips integration tests)
	cd $(API_DIR) && go test -short -count=1 ./...

.PHONY: test-all
test-all: ## Run all Go tests including integration tests (requires DATABASE_URL)
	cd $(API_DIR) && go test -count=1 ./...

.PHONY: test-race
test-race: ## Run all Go tests with the race detector
	cd $(API_DIR) && go test -race -short -count=1 ./...

.PHONY: fmt
fmt: ## Format all Go source files
	cd $(API_DIR) && gofmt -w .

.PHONY: vet
vet: ## Run go vet on all packages
	cd $(API_DIR) && go vet ./...

.PHONY: check
check: fmt vet test ## Format, vet, and test (all short)

# ── Docker Compose ────────────────────────────────────────────────────────────
.PHONY: up
up: ## Start the full development environment (postgres + migrate + api)
	docker compose up --build

.PHONY: up-db
up-db: ## Start only PostgreSQL
	docker compose up postgres

.PHONY: down
down: ## Stop all containers (keep volumes)
	docker compose down

.PHONY: down-clean
down-clean: ## Stop all containers and remove volumes (destroys local data)
	docker compose down -v

.PHONY: logs
logs: ## Follow logs from all running containers
	docker compose logs -f

.PHONY: migrate
migrate: ## Run database migrations
	docker compose run --rm migrate

# ── Utilities ─────────────────────────────────────────────────────────────────
.PHONY: tidy
tidy: ## Tidy Go module dependencies
	cd $(API_DIR) && go mod tidy
