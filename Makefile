.PHONY: help rebuild up down restart logs build clean status

# Default target - show help
help:
	@echo "Available targets:"
	@echo "  make rebuild  - Rebuild everything from scratch (stops, removes volumes, rebuilds, starts)"
	@echo "  make up       - Start containers"
	@echo "  make down     - Stop containers"
	@echo "  make clean    - Stop containers and remove volumes"
	@echo "  make restart  - Restart containers (without rebuilding)"
	@echo "  make logs     - View logs (follows)"
	@echo "  make build    - Build containers without starting"
	@echo "  make status   - Show container status"
	@echo "  make help     - Show this help message"

# Rebuild everything from scratch (stops, removes volumes, rebuilds, starts)
rebuild:
	docker compose down -v
	docker compose build
	docker compose up -d

# Start containers
up:
	docker compose up -d

# Stop containers
down:
	docker compose down

# Stop containers and remove volumes
clean:
	docker compose down -v

# Restart containers (without rebuilding)
restart:
	docker compose restart

# View logs
logs:
	docker compose logs -f

# Build containers without starting
build:
	docker compose build

# Show container status
status:
	docker compose ps
