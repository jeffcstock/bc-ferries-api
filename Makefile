.PHONY: rebuild up down restart logs build clean

# Rebuild everything from scratch (stops, removes volumes, rebuilds, starts)
rebuild:
	docker-compose down -v
	docker-compose build
	docker-compose up -d

# Start containers
up:
	docker-compose up -d

# Stop containers
down:
	docker-compose down

# Stop containers and remove volumes
clean:
	docker-compose down -v

# Restart containers (without rebuilding)
restart:
	docker-compose restart

# View logs
logs:
	docker-compose logs -f

# Build containers without starting
build:
	docker-compose build

# Show container status
status:
	docker-compose ps
