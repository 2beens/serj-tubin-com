.PHONY: up down stop restart
.PHONY: up-monitoring down-monitoring stop-monitoring
.PHONY: logs-fileservice logs-main logs-all
.PHONY: rebuild-fileservice rebuild-mainservice
.PHONY: redis-cli clean

# Core application services
up:
	@echo "Starting core services..."
	@docker compose up --build -d

down:
	@echo "Stopping core services..."
	@docker compose down

stop:
	@echo "Stopping core services..."
	@docker compose stop

restart:
	@echo "Restarting core services..."
	@docker compose restart

# Monitoring stack (optional)
up-monitoring:
	@echo "Starting monitoring stack..."
	@docker compose -f docker-compose.monitoring.yml up -d

down-monitoring:
	@echo "Stopping monitoring stack..."
	@docker compose -f docker-compose.monitoring.yml down

stop-monitoring:
	@echo "Stopping monitoring stack..."
	@docker compose -f docker-compose.monitoring.yml stop

# Combined operations
up-all:
	@echo "Starting all services (core + monitoring)..."
	@docker compose up --build -d
	@docker compose -f docker-compose.monitoring.yml up -d

down-all:
	@echo "Stopping all services..."
	@docker compose down
	@docker compose -f docker-compose.monitoring.yml down

# Individual service operations
up-mainservice:
	@echo "Starting main service and dependencies..."
	@docker compose up mainservice --build -d

up-fileservice:
	@echo "Starting file service and dependencies..."
	@docker compose up fileservice --build -d

# Logs
logs-fileservice:
	@docker compose logs fileservice --follow

logs-main:
	@docker compose logs mainservice --follow

logs-all:
	@docker compose logs --follow

# Rebuild services
rebuild-fileservice:
	@echo "Rebuilding file service..."
	@docker compose up -d --force-recreate --no-deps --build fileservice

rebuild-mainservice:
	@echo "Rebuilding main service..."
	@docker compose up -d --force-recreate --no-deps --build mainservice

# Utilities
redis-cli:
	@echo "Accessing Redis CLI..."
	@docker exec -it st-redis redis-cli -a $${SERJ_REDIS_PASS}

clean:
	@echo "Cleaning up unused Docker resources..."
	@docker system prune -f
	@docker volume prune -f

# Help
help:
	@echo "Available commands:"
	@echo "  up              - Start core services"
	@echo "  down            - Stop and remove core services"
	@echo "  stop            - Stop core services"
	@echo "  restart         - Restart core services"
	@echo "  up-monitoring   - Start monitoring stack"
	@echo "  down-monitoring - Stop monitoring stack"
	@echo "  up-all          - Start all services"
	@echo "  down-all        - Stop all services"
	@echo "  logs-*          - View logs for specific services"
	@echo "  rebuild-*       - Rebuild specific services"
	@echo "  redis-cli       - Access Redis CLI"
	@echo "  clean           - Clean up unused Docker resources"