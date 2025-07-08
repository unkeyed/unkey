.PHONY: install fmt test-unit test-full build generate lint build-docker pull

install:
	go mod tidy

fmt: lint
	@go fmt ./...

pull:
	@docker compose -f ../deployment/docker-compose.yaml pull 

build-docker:
	@docker compose -f ../deployment/docker-compose.yaml build mysql 

test-full: pull build-docker
	@export INTEGRATION_TEST=true && \
	export SIMULATION_TEST=false && \
	echo "Running full tests... this can take more than 30min... run 'make test-unit' for faster tests" && \
	go test -failfast -timeout=60m -shuffle=on -v -json ./... | tparse -all -progress -smallscreen

test-unit: pull build-docker
	go test -json -race -failfast -timeout=30m ./... | tparse -all -progress -smallscreen

build:
	go build -o unkey ./main.go

generate:
	go generate ./...
	# buf generate

lint:
	@golangci-lint run
