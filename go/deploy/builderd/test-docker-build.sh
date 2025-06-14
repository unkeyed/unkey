#!/bin/bash

# Test script to verify Docker build functionality
set -e

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
