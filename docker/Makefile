.PHONY: up-prometheus up-redis down-prometheus down-redis logs-fileservice logs-main rebuild-fileservice rebuild-mainservice redis-cli up-nc down-nc

up-nc:
	@echo "Starting services via nerdctl ..."
	@./run.sh up --nerdctl

down-nc:
	@echo "Stopping services via nerdctl ..."
	@./run.sh down --nerdctl

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
