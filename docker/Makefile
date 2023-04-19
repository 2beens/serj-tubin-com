.PHONY: up up-nc up-win
.PHONY: down down-nc down-win
.PHONY: stop stop-nc stop-win
.PHONY: up-prometheus up-redis down-prometheus down-redis
.PHONY: logs-fileservice logs-main
.PHONY: rebuild-fileservice rebuild-mainservice
.PHONY: redis-cli

# using regular docker on unix
up:
	@echo "Starting services ..."
	@./run.sh up

down:
	@echo "Stopping services ..."
	@./run.sh down

stop:
	@echo "Stopping services ..."
	@./run.sh stop

# using nerdctl
up-nc:
	@echo "Starting services via nerdctl ..."
	@./run.sh up --nerdctl

down-nc:
	@echo "Removing services via nerdctl ..."
	@./run.sh down --nerdctl

stop-nc:
	@echo "Stopping services via nerdctl ..."
	@./run.sh stop --nerdctl

# oh my dear windows
up-win:
	powershell.exe -Command "docker compose -p serjservice up --build -d"

down-win:
	powershell.exe -Command "docker compose -p serjservice down"

stop-win:
	powershell.exe -Command "docker compose -p serjservice stop"

up-prometheus:
	@echo "Starting Prometheus in Docker ..."
	@docker compose -p serj-tubin-com-prometheus -f docker-compose-prometheus.yml up -d

up-redis:
	@echo "Starting Redis in Docker ..."
	@docker compose -p serj-tubin-com-redis -f docker-compose-redis.yml up -d

down-prometheus:
	@echo "Stopping Prometheus in Docker ..."
	@docker compose -p serj-tubin-com-prometheus -f docker-compose-prometheus.yml down

down-redis:
	@echo "Stopping Redis in Docker ..."
	@docker compose -p serj-tubin-com-redis -f docker-compose-redis.yml down

logs-fileservice:
	@docker compose -p serjservice logs fileservice --follow

logs-main:
	@docker compose -p serjservice logs mainservice --follow

rebuild-fileservice:
	@echo "Recreating file service without building dependencies ..."
	@docker compose -p serjservice up -d --force-recreate --no-deps --build fileservice

rebuild-mainservice:
	@echo "Recreating main service without building dependencies ..."
	@docker compose -p serjservice up -d --force-recreate --no-deps --build mainservice

redis-cli:
	@echo "Accessing Redis CLI ..."
	@docker exec -it st-redis redis-cli -a $${SERJ_REDIS_PASS}
