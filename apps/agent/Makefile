.PHONY: install fmt test build race lint generate

install:
	go mod tidy

fmt: lint
	go fmt ./...

test:
	go test -cover -json -failfast ./... | tparse -all -progress

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