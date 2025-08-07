.PHONY: install fmt test build race lint generate

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
PARALLEL_PROCS := $(shell echo $$(($(GOMAXPROCS_VAL) > 1 ? $(GOMAXPROCS_VAL)/2 : 1)))

install:
	@go mod tidy

fmt: lint
	@go fmt ./...

test:
	TESTCONTAINERS_RYUK_DISABLED=true go test -json -count=1 -parallel=$(PARALLEL_PROCS) -failfast ./pkg/... ./services/... | tparse -all -progress -smallscreen

build:
	go build -o unkey ./cmd/main.go

race:
	go install github.com/amit-davidson/Chronos/cmd/chronos
	~/go/bin/chronos --file=./cmd/main.go --mod=$$(pwd)

lint:
	golangci-lint run

generate:
	go get github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen
	mkdir -p ./pkg/openapi
	go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config=./pkg/openapi/config.yaml ./pkg/openapi/openapi.json
	buf generate
