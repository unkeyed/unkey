.PHONY: install fmt test-unit test-full build generate lint pull up down

install:
	go mod tidy

fmt: lint
	go fmt ./...
	golangci-lint run

pull:
	@docker compose -f ../deployment/docker-compose.yaml pull

up:
	@docker compose -f ../deployment/docker-compose.yaml up -d mysql redis clickhouse s3 otel kafka
	@echo "Starting ClickHouse migrations (will retry if ClickHouse isn't ready)..."
	@for i in {1..10}; do \
		echo "Migration attempt $$i..."; \
		if docker compose -f ../deployment/docker-compose.yaml run --rm clickhouse_migrator; then \
			echo "Migrations completed successfully!"; \
			break; \
		else \
			echo "Migration failed, retrying in 5 seconds..."; \
			sleep 5; \
		fi; \
	done

down:
	@docker compose -f ../deployment/docker-compose.yaml down

test-full: export INTEGRATION_TEST=true
test-full: export SIMULATION_TEST=false
test-full: up
	@echo "Running full tests... this can take more than 30min... run 'make test-unit' for faster tests"
	@go test -failfast -timeout=60m -shuffle=on -v -json ./... | tparse -all -progress -smallscreen

test-unit: up
	@go test -json -race -failfast -timeout=30m ./... | tparse -all -progress -smallscreen

build:
	go build -o unkey ./main.go

generate:
	buf generate
	go generate ./...
