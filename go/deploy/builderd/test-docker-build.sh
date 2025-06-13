#!/bin/bash

# Test script to verify Docker build functionality
set -e

echo "Starting builderd service for testing..."

# Start builderd in background with test configuration
UNKEY_BUILDERD_OTEL_ENABLED=false \
UNKEY_BUILDERD_SCRATCH_DIR=/tmp/builderd-test/scratch \
UNKEY_BUILDERD_ROOTFS_OUTPUT_DIR=/tmp/builderd-test/rootfs \
UNKEY_BUILDERD_WORKSPACE_DIR=/tmp/builderd-test/workspace \
UNKEY_BUILDERD_DATABASE_DATA_DIR=/tmp/builderd-test/data \
./build/builderd &

BUILDERD_PID=$!

# Wait for service to start
sleep 3

# Function to cleanup on exit
cleanup() {
    echo "Cleaning up..."
    kill $BUILDERD_PID 2>/dev/null || true
    rm -rf /tmp/builderd-test
}
trap cleanup EXIT

# Test health endpoint
echo "Testing health endpoint..."
curl -f http://localhost:8082/health

echo -e "\n\nTesting metrics endpoint (if enabled)..."
curl -f http://localhost:8082/metrics || echo "Metrics endpoint not available (OpenTelemetry disabled)"

echo -e "\n\nTesting Docker build with ghcr.io/unkeyed/unkey:f4cfee5..."

# Create test build request
curl -X POST http://localhost:8082/builder.v1.BuilderService/CreateBuild \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: test-tenant" \
  -H "X-Customer-ID: test-customer" \
  -d '{
    "config": {
      "source": {
        "docker_image": {
          "image_uri": "ghcr.io/unkeyed/unkey:f4cfee5"
        }
      },
      "target": {
        "microvm_rootfs": {
          "format": "ext4",
          "init_strategy": "INIT_STRATEGY_DIRECT_EXEC"
        }
      },
      "strategy": {
        "docker_strategy": {
          "optimize_for_size": true,
          "remove_dev_packages": true
        }
      },
      "tenant": {
        "tenant_id": "test-tenant",
        "customer_id": "test-customer",
        "tier": "TENANT_TIER_FREE"
      }
    }
  }'

echo -e "\n\nBuild test completed successfully!"
echo "Check /tmp/builderd-test/rootfs/ for extracted rootfs"