#!/bin/bash

# Run ConnectRPC Client Example
# This script shows what VM creation data looks like without actually creating VMs

set -euo pipefail

echo "========================================"
echo "  ConnectRPC VM Creation Data Inspector"
echo "========================================"
echo

# Check if we're in the right directory
if [[ ! -f "go.mod" ]]; then
    echo "Error: Please run this script from the project root directory"
    echo "Usage: ./examples/run-client-example.sh"
    exit 1
fi

# Build and run the example
echo "Building client example..."
go build -o examples/client-example examples/client/main.go

echo "Running client example..."
echo
./examples/client-example

# Cleanup
rm -f examples/client-example

echo
echo "========================================"
echo "Example completed!"
echo
echo "The above output shows you exactly what data"
echo "would be sent to create VMs using ConnectRPC."
echo
echo "To actually create VMs:"
echo "1. Start Firecracker: sudo firecracker --api-sock /tmp/firecracker.sock"
echo "2. Set environment: export UNKEY_METALD_BACKEND=firecracker"
echo "3. Set endpoint: export UNKEY_METALD_FC_ENDPOINT=unix:///tmp/firecracker.sock"
echo "4. Start VMM server: ./api"
echo "5. Uncomment the actual creation code in the example"
echo "========================================"
