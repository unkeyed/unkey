.PHONY: install setup tools fmt test-unit test-integration test-integration-long test-stress test build generate pull up clean k8s-check k8s-up k8s-down k8s-reset k8s-status start-mysql start-ctrl start-all dev

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

# One-time setup for new developers
setup:
	@[ -f go.work ] || go work init . tools

install: setup
	go mod tidy

tools: setup
	@echo "Tools are ready. Usage:"
	@echo "  go tool buf <args>"
	@echo "  go tool golangci-lint <args>"
	@echo "  go tool sqlc <args>"
	@echo "  go tool protoc-gen-go-restate <args>"

fmt: setup
	go fmt ./...
	go tool golangci-lint run

pull:
	@docker compose -f ../deployment/docker-compose.yaml pull

up: pull
	@docker compose -f ../deployment/docker-compose.yaml up -d planetscale mysql redis clickhouse s3 otel kafka restate --wait

clean:
	@docker compose -f ../deployment/docker-compose.yaml down --volumes

build: setup
	go build -o unkey .

generate: setup
	go tool buf generate --template ./buf.gen.connect.yaml --clean --path "./proto/ctrl" --path "./proto/krane" --path "./proto/vault" --path "./proto/cache" --path "./proto/gateway"
	go tool buf generate --template ./buf.gen.restate.yaml --path "./proto/hydra"
	go generate ./...
	go fmt ./...


test: test-unit

test-unit: up
	@echo "Running tests w/$(PARALLEL_PROCS) parallel test processes"
	@go test -json -failfast -timeout=15m -parallel=$(PARALLEL_PROCS) ./... | go run github.com/mfridman/tparse@ba2512e7be150bfcbd6f6220d517d3741f8f2f75 -all -smallscreen

test-stress: export INTEGRATION_TEST=true
test-stress: export SIMULATION_TEST=false
test-stress: up
	@echo "Running stress tests w/$(PARALLEL_PROCS) parallel test processes"
	@go test -tags=stress,integration,integration_long -json -failfast -timeout=15m -parallel=$(PARALLEL_PROCS) ./... | go run github.com/mfridman/tparse@ba2512e7be150bfcbd6f6220d517d3741f8f2f75 -all -smallscreen

test-integration-long: export INTEGRATION_TEST=true
test-integration-long: export SIMULATION_TEST=false
test-integration-long: up
	@echo "Running long-ish integration tests w/$(PARALLEL_PROCS) parallel test processes"
	@go test -tags=integration,integration_long -json -failfast -timeout=15m -parallel=$(PARALLEL_PROCS) ./... | go run github.com/mfridman/tparse@ba2512e7be150bfcbd6f6220d517d3741f8f2f75 -all -smallscreen

test-integration: export INTEGRATION_TEST=true
test-integration: export SIMULATION_TEST=false
test-integration: up
	@echo "Running integration tests w/$(PARALLEL_PROCS) parallel test processes"
	@go test -tags=integration -json -failfast -timeout=15m -parallel=$(PARALLEL_PROCS) ./... | go run github.com/mfridman/tparse@ba2512e7be150bfcbd6f6220d517d3741f8f2f75 -smallscreen

# ============================================================================
# Kubernetes Development Commands
# ============================================================================

k8s-check: ## Check if kubectl is available and cluster is running
	@command -v kubectl >/dev/null 2>&1 || { echo "ERROR: kubectl not found. Install from: https://kubernetes.io/docs/tasks/tools/"; exit 1; }
	@kubectl cluster-info >/dev/null 2>&1 || { echo "ERROR: Kubernetes cluster not available. Enable Kubernetes in Docker Desktop/OrbStack"; exit 1; }
	@echo "Kubernetes cluster is available"

k8s-up: k8s-check ## Deploy all services to current Kubernetes cluster
	@echo "Building Docker images..."
	@docker build -t unkey/mysql:latest -f ../deployment/Dockerfile.mysql ../
	@docker build -t unkey/clickhouse:latest -f ../deployment/Dockerfile.clickhouse ../
	@docker build -t unkey:latest .
	@echo "Creating namespace..."
	@kubectl apply -f k8s/manifests/namespace.yaml
	@echo "Applying Kubernetes manifests..."
	@kubectl apply -f k8s/manifests/
	@echo "Waiting for services to be ready..."
	@kubectl wait --for=condition=ready pod -l app=mysql -n unkey --timeout=180s
	@kubectl wait --for=condition=ready pod -l app=clickhouse -n unkey --timeout=180s
	@kubectl wait --for=condition=ready pod -l app=s3 -n unkey --timeout=180s
	@kubectl wait --for=condition=ready pod -l app=restate -n unkey --timeout=180s
	@kubectl wait --for=condition=ready pod -l app=api -n unkey --timeout=180s
	@kubectl wait --for=condition=ready pod -l app=gw -n unkey --timeout=180s
	@kubectl wait --for=condition=ready pod -l app=ctrl -n unkey --timeout=180s
	@kubectl wait --for=condition=ready pod -l app=krane -n unkey --timeout=180s
	@kubectl wait --for=condition=ready pod -l app=dashboard -n unkey --timeout=180s
	@echo "Kubernetes environment is ready!"
	@echo ""
	@echo "Services accessible via NodePort on localhost - check actual ports with:"
	@echo ""
	@echo "Check NodePort assignments: make k8s-ports"
	@make k8s-status

k8s-stop: ## Stop all pods (scale deployments to 0)
	@echo "Stopping all pods..."
	@kubectl scale deployment --all --replicas=0 -n unkey
	@echo "All pods stopped"

k8s-down: ## Delete all services from current Kubernetes cluster
	@echo "Deleting all services..."
	@kubectl delete namespace unkey --ignore-not-found=true
	@echo "Services deleted"

k8s-reset: k8s-down k8s-up ## Reset the entire Kubernetes environment

# Helper function to deploy a service
define deploy-service
	@echo "Starting $(2)..."
	$(if $(3),@docker build -t $(3) $(4))
	@kubectl apply -f k8s/manifests/namespace.yaml
	@kubectl apply -f k8s/manifests/$(1).yaml
	@kubectl wait --for=condition=ready pod -l app=$(1) -n unkey --timeout=180s
	$(if $(5),@kubectl wait --for=condition=ready pod -l app=$(5) -n unkey --timeout=180s)
	@echo "$(2) is ready!"
endef

start-mysql: k8s-check ## Deploy only MySQL
	$(call deploy-service,mysql,MySQL,unkey/mysql:latest,-f ../deployment/Dockerfile.mysql ../)

start-clickhouse: k8s-check ## Deploy only ClickHouse
	@docker build -t unkey/clickhouse:latest -f ../deployment/Dockerfile.clickhouse ../
	$(call deploy-service,clickhouse,ClickHouse)

start-redis: k8s-check ## Deploy only Redis
	$(call deploy-service,redis,Redis)

start-s3: k8s-check ## Deploy only S3 (MinIO)
	$(call deploy-service,s3,S3 (MinIO))

start-planetscale: k8s-check ## Deploy only PlanetScale HTTP driver
	$(call deploy-service,planetscale,PlanetScale HTTP driver)

start-observability: k8s-check ## Deploy only Observability stack
	$(call deploy-service,observability,Observability stack,,,otel-collector)

start-restate: k8s-check ## Deploy only Restate
	$(call deploy-service,restate,Restate)

start-api: k8s-check ## Deploy only API service (3 replicas)
	$(call deploy-service,api,API,unkey:latest,.)


start-ctrl: k8s-check ## Deploy only ctrl service
	$(call deploy-service,ctrl,Control Plane,unkey:latest,.)

start-dashboard: k8s-check ## Deploy only dashboard service
	$(call deploy-service,dashboard,Dashboard,unkey/dashboard:latest,-f ../apps/dashboard/Dockerfile ../)

start-unkey-services: start-api start-ctrl start-dashboard ## Deploy all Unkey services

start-dependencies: start-mysql start-clickhouse start-redis start-s3 start-planetscale start-observability start-restate ## Deploy all dependency services

start-all: start-dependencies start-unkey-services ## Deploy all services individually


image:
	@docker build -t unkey:dev .

dev: image## Start with Tilt (if available) or fallback to k8s-up
	@if command -v tilt >/dev/null 2>&1; then \
		echo "Starting with Tilt..."; \
		tilt up; \
	else \
		echo "Tilt not found, using k8s-up instead"; \
		echo "Install Tilt from: https://docs.tilt.dev/install.html"; \
		make k8s-up; \
	fi
