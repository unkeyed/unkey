.PHONY: install fmt test-unit test-full build generate lint pull build-docker

install:
	go mod tidy

fmt: lint
	@go fmt ./...

pull:
	docker pull mysql:latest
	docker pull redis:latest
	docker pull grafana/otel-lgtm:latest

build-docker:
	docker build -t apiv2:latest .

test-full: pull build-docker
	@export INTEGRATION_TEST=true && \
	export SIMULATION_TEST=false && \
	echo "Running full tests... this can take more than 30min... run 'make test-unit' for faster tests" && \
	go test -failfast -timeout=60m -shuffle=on -v -json ./... | tparse -all -progress -smallscreen

test-unit:
	go test -json -race -failfast -timeout=30m ./... | tparse -all -progress -smallscreen

build:
	go build -o unkey ./main.go

generate:
	go generate ./...
	# buf generate

lint:
	@golangci-lint run
