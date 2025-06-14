.PHONY: all build install proto clean test lint dev

# Build variables
BINARY_NAME=assetmanagerd
VERSION?=dev
LDFLAGS=-ldflags "-X main.version=${VERSION}"

all: proto build

proto:
	@echo "Generating protobuf code..."
	buf generate

build: proto
	@echo "Building assetmanagerd..."
	go build ${LDFLAGS} -o bin/${BINARY_NAME} ./cmd/assetmanagerd

install: build
	@echo "Installing assetmanagerd..."
	sudo install -m 755 bin/${BINARY_NAME} /usr/local/bin/
	@echo "Creating directories..."
	sudo mkdir -p /opt/assetmanagerd /opt/vm-assets
	sudo chown assetmanagerd:assetmanagerd /opt/assetmanagerd
	@echo "Installing systemd service..."
	sudo cp contrib/systemd/assetmanagerd.service /etc/systemd/system/
	sudo systemctl daemon-reload
	@echo "Installation complete. Run 'sudo systemctl start assetmanagerd' to start the service."

clean:
	@echo "Cleaning..."
	rm -rf bin/ gen/

test:
	@echo "Running tests..."
	go test -v ./...

lint:
	@echo "Running linters..."
	golangci-lint run

dev:
	@echo "Running in development mode..."
	go run ./cmd/assetmanagerd

# Create user and group for the service
create-user:
	@echo "Creating assetmanagerd user and group..."
	sudo useradd -r -s /bin/false -d /opt/assetmanagerd -m assetmanagerd || true

# Development helpers
run-local:
	UNKEY_ASSETMANAGERD_STORAGE_BACKEND=local \
	UNKEY_ASSETMANAGERD_LOCAL_STORAGE_PATH=/tmp/vm-assets \
	UNKEY_ASSETMANAGERD_DATABASE_PATH=/tmp/assets.db \
	UNKEY_ASSETMANAGERD_CACHE_DIR=/tmp/asset-cache \
	UNKEY_ASSETMANAGERD_OTEL_ENABLED=false \
	go run ./cmd/assetmanagerd