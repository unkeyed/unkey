.PHONY: install fmt test-unit test-full build generate lint pull up down

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
	@docker compose -f ../deployment/docker-compose.yaml up --force-recreate -d mysql planetscale redis clickhouse s3 otel
	@docker compose -f ../deployment/docker-compose.yaml up -d mysql planetscale redis clickhouse s3 otel
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

test-full: export INTEGRATION_TEST=true
test-full: export SIMULATION_TEST=false
test-full: up
	@echo "Running full tests... this can take more than 30min... run 'make test-unit' for faster tests"
	@echo "Using $(PARALLEL_PROCS) parallel test processes"
	@go test -json -failfast -timeout=15m -parallel=$(PARALLEL_PROCS) ./... | tparse -all -progress -smallscreen

test-unit: up
	@echo "Using $(PARALLEL_PROCS) parallel test processes"
	@go test -json -failfast -timeout=15m -parallel=$(PARALLEL_PROCS) -race ./... | tparse -all -progress -smallscreen

build:
	go build -o unkey ./main.go

generate:
	buf generate
	go generate ./...
