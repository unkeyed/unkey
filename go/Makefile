.PHONY: install tools fmt test-unit test-integration test-integration-long test-stress test build generate pull up clean

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

tools:
	go install github.com/bufbuild/buf/cmd/buf@v1.57.0
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.4.0
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.28.0

fmt:
	go fmt ./...
	golangci-lint run

pull:
	@docker compose -f ../deployment/docker-compose.yaml pull

up: pull
	@docker compose -f ../deployment/docker-compose.yaml up -d planetscale mysql redis clickhouse s3 otel


clean:
	@docker compose -f ../deployment/docker-compose.yaml down --volumes

build:
	go build -o unkey ./main.go

generate:
	buf generate
	go generate ./...

test: test-unit

test-unit: up
	@echo "Running tests w/$(PARALLEL_PROCS) parallel test processes"
	@go test -json -failfast -timeout=15m -parallel=$(PARALLEL_PROCS) -race ./... | tparse -all -progress -smallscreen

test-stress: export INTEGRATION_TEST=true
test-stress: export SIMULATION_TEST=false
test-stress: up
	@echo "Running stress tests w/$(PARALLEL_PROCS) parallel test processes"
	@go test -tags=stress,integration,integration_long -json -failfast -timeout=15m -parallel=$(PARALLEL_PROCS) ./... | tparse -all -progress -smallscreen

test-integration-long: export INTEGRATION_TEST=true
test-integration-long: export SIMULATION_TEST=false
test-integration-long: up
	@echo "Running long-ish integration tests w/$(PARALLEL_PROCS) parallel test processes"
	@go test -tags=integration,integration_long -json -failfast -timeout=15m -parallel=$(PARALLEL_PROCS) ./... | tparse -all -progress -smallscreen

test-integration: export INTEGRATION_TEST=true
test-integration: export SIMULATION_TEST=false
test-integration: up
	@echo "Running integration tests w/$(PARALLEL_PROCS) parallel test processes"
	@go test -tags=integration -json -failfast -timeout=15m -parallel=$(PARALLEL_PROCS) ./... | tparse -all -progress -smallscreen
