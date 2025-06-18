#!/bin/bash

# Script to register VM assets with assetmanagerd
# This script registers the assets downloaded by setup-vm-assets.sh

set -euo pipefail

ASSETS_DIR="/opt/vm-assets"
ASSETMANAGER_ENDPOINT="${UNKEY_METALD_ASSETMANAGER_ENDPOINT:-http://localhost:8083}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Generate asset IDs (using simple names for testing)
KERNEL_ID="kernel-vmlinux"
ROOTFS_ID="rootfs-bionic"

# Check if assets exist
if [ ! -f "$ASSETS_DIR/vmlinux" ]; then
    log_error "Kernel file not found at $ASSETS_DIR/vmlinux"
    log_info "Run setup-vm-assets.sh first to download assets"
    exit 1
fi

if [ ! -f "$ASSETS_DIR/rootfs.ext4" ]; then
    log_error "Rootfs file not found at $ASSETS_DIR/rootfs.ext4"
    log_info "Run setup-vm-assets.sh first to download assets"
    exit 1
fi

# Get file sizes
KERNEL_SIZE=$(stat -c%s "$ASSETS_DIR/vmlinux")
ROOTFS_SIZE=$(stat -c%s "$ASSETS_DIR/rootfs.ext4")

# Create sharded directories for assetmanagerd
log_info "Creating sharded directories for assets..."
sudo mkdir -p "/opt/vm-assets/ke"
sudo mkdir -p "/opt/vm-assets/ro"

# Copy assets to sharded locations (first 2 chars of ID as directory)
log_info "Copying assets to sharded locations..."
sudo cp "$ASSETS_DIR/vmlinux" "/opt/vm-assets/ke/$KERNEL_ID"
sudo cp "$ASSETS_DIR/rootfs.ext4" "/opt/vm-assets/ro/$ROOTFS_ID"

# Set proper ownership
sudo chown -R assetmanagerd:assetmanagerd /opt/vm-assets/

# Register kernel with assetmanagerd
log_info "Registering kernel asset with assetmanagerd..."
curl -X POST "$ASSETMANAGER_ENDPOINT/asset.v1.AssetManagerService/RegisterAsset" \
  -H "Content-Type: application/json" \
  -d "{
    \"asset\": {
      \"id\": \"$KERNEL_ID\",
      \"name\": \"Firecracker Kernel\",
      \"type\": \"ASSET_TYPE_KERNEL\",
      \"backend\": \"STORAGE_BACKEND_LOCAL\",
      \"location\": \"ke/$KERNEL_ID\",
      \"size_bytes\": $KERNEL_SIZE,
      \"created_by\": \"setup-script\",
      \"labels\": {
        \"os\": \"linux\",
        \"version\": \"5.10\",
        \"arch\": \"x86_64\"
      }
    }
  }" && log_success "Kernel registered" || log_error "Failed to register kernel"

# Register rootfs with assetmanagerd
log_info "Registering rootfs asset with assetmanagerd..."
curl -X POST "$ASSETMANAGER_ENDPOINT/asset.v1.AssetManagerService/RegisterAsset" \
  -H "Content-Type: application/json" \
  -d "{
    \"asset\": {
      \"id\": \"$ROOTFS_ID\",
      \"name\": \"Ubuntu Bionic Rootfs\",
      \"type\": \"ASSET_TYPE_ROOTFS\",
      \"backend\": \"STORAGE_BACKEND_LOCAL\",
      \"location\": \"ro/$ROOTFS_ID\",
      \"size_bytes\": $ROOTFS_SIZE,
      \"created_by\": \"setup-script\",
      \"labels\": {
        \"os\": \"ubuntu\",
        \"version\": \"18.04\",
        \"arch\": \"x86_64\"
      }
    }
  }" && log_success "Rootfs registered" || log_error "Failed to register rootfs"

# List registered assets
log_info "Listing registered assets..."
echo ""
curl -X POST "$ASSETMANAGER_ENDPOINT/asset.v1.AssetManagerService/ListAssets" \
  -H "Content-Type: application/json" \
  -d '{}' | jq -r '.assets[] | "[\(.type)] \(.id): \(.name) (\(.size_bytes | tonumber / 1048576 | floor)MB)"' || log_error "Failed to list assets"

echo ""
log_success "Asset registration complete!"
echo ""
log_info "Next steps:"
echo "  1. Update the example client to use the registered asset IDs:"
echo "     - Kernel: $KERNEL_ID"
echo "     - Rootfs: $ROOTFS_ID"
echo "  2. Or set environment variables:"
echo "     export VM_KERNEL_ASSET_ID=$KERNEL_ID"
echo "     export VM_ROOTFS_ASSET_ID=$ROOTFS_ID"