.PHONY: pull build down migrate-clickhouse migrate-clickhouse-reset integration generate-sql nuke-docker

pull:
	docker compose -f ./deployment/docker-compose.yaml pull

build: pull
	docker compose -f ./deployment/docker-compose.yaml build

down:
	docker compose -f ./deployment/docker-compose.yaml down

up: down build
	docker compose -f ./deployment/docker-compose.yaml up -d

integration: up
	@cd apps/api && \
	$(MAKE) seed && \
	pnpm test:integration

generate-sql:
	@cd internal/db && \
	pnpm drizzle-kit generate --dialect=mysql

nuke-docker:
	docker stop $$(docker ps -aq)
	docker system prune -af
	docker volume prune --all -f
