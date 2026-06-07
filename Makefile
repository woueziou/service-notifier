.PHONY: all dev dev-docker server client run migrate-up migrate-down docker-up docker-down clean

GO=go
NPM=npm

all: dev

# --- Server (runs natively on host) ---
# Assumes postgres and redis are already running (e.g. in Docker) on localhost:5432 and localhost:6379.

server:
	cd server && $(GO) run ./cmd/notifier

server-build:
	cd server && $(GO) build -o bin/notifier ./cmd/notifier

server-test:
	cd server && $(GO) test ./...

server-lint:
	cd server && $(GO) vet ./...

swag:
	cd server && swag init -g cmd/notifier/main.go

# --- Client ---

client:
	cd client && $(NPM) run dev

client-build:
	cd client && $(NPM) run build

client-install:
	cd client && $(NPM) install

# --- Docker (runs everything in containers) ---
# The compose file relies on external "postgres" and "redis" containers
# on the same Docker network.

docker-up:
	docker compose up -d

docker-down:
	docker compose down

docker-build:
	docker compose build

docker-logs:
	docker compose logs -f

# --- Development (native server + docker deps) ---

dev:
	@echo "Starting mailpit (SMTP test server)..."
	docker compose up -d mailpit
	@echo ""
	@echo "Make sure postgres and redis are already running."
	@echo "  docker ps | grep -E 'postgres|redis'"
	@echo ""
	@sleep 2
	$(MAKE) server

dev-docker:
	@echo "Starting all containers (requires external postgres+redis network)..."
	docker compose up -d --build

# --- Database Migrations ---

migrate-up:
	cd server && migrate -path migrations -database "$(DATABASE_URL)" up

migrate-down:
	cd server && migrate -path migrations -database "$(DATABASE_URL)" down 1

# --- Clean ---

clean:
	cd server && rm -rf bin/
