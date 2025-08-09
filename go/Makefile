.PHONY: install fmt test-unit test-integration test-integration-long test-stress build generate lint pull up down

# Detect OS and set GOMAXPROCS accordingly
UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Linux)
    DETECTED_PROCS := $(shell nproc)
else ifeq ($(UNAME_S),Darwin)
    DETECTED_PROCS := $(shell sysctl -n hw.ncpu)
else
    DETECTED_PROCS := 4
endif

GOMAXPROCS_VAL := $(or $(GOMAXPROCS),$(DETECTED_PROCS))
PARALLEL_PROCS := $(shell if [ $(GOMAXPROCS_VAL) -gt 1 ]; then expr $(GOMAXPROCS_VAL) / 2; else echo 1; fi)

install:
	go mod tidy

fmt: lint
	go fmt ./...
	golangci-lint run

pull:
	@docker compose -f ../deployment/docker-compose.yaml pull

up:
	@docker compose -f ../deployment/docker-compose.yaml up -d planetscale mysql redis clickhouse s3 otel
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
	@docker compose -f ../deployment/docker-compose.yaml down --volumes

test-stress: export INTEGRATION_TEST=true
test-stress: export SIMULATION_TEST=false
test-stress: up
	@echo "Running stress tests w/$(PARALLEL_PROCS) parallel test processes"
	@go test -tags=stress,integration,integration_long -json -failfast -timeout=15m -parallel=$(PARALLEL_PROCS) ./... | tparse -all -progress -smallscreen

test-integration-long: export INTEGRATION_TEST=true
test-integration-long: export SIMULATION_TEST=false
test-integration-long: up
	@echo "Running integrations tests w/$(PARALLEL_PROCS) parallel test processes"
	@go test -tags=integration,integration-long -json -failfast -timeout=15m -parallel=$(PARALLEL_PROCS) ./... | tparse -all -progress -smallscreen

test-integration: export INTEGRATION_TEST=true
test-integration: export SIMULATION_TEST=false
test-integration: up
	@echo "Running tests w/$(PARALLEL_PROCS) parallel test processes"
	@go test -tags=integration -json -failfast -timeout=15m -parallel=$(PARALLEL_PROCS) ./... | tparse -all -progress -smallscreen

test-unit: up
	@echo "Running tests w/$(PARALLEL_PROCS) parallel test processes"
	@go test -json -failfast -timeout=15m -parallel=$(PARALLEL_PROCS) -race ./... | tparse -all -progress -smallscreen

build:
	go build -o unkey ./main.go

generate:
	buf generate
	go generate ./...
