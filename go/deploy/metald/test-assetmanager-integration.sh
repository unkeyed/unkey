#!/bin/bash
set -euo pipefail

echo "=== Testing AssetManager Integration with Metald ==="
echo

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test configuration
METALD_PORT=8080
ASSETMANAGERD_PORT=8082

# Function to check if a service is running
check_service() {
    local service=$1
    local port=$2
    if curl -s -f http://localhost:$port/health >/dev/null 2>&1; then
        echo -e "${GREEN}✓ $service is running on port $port${NC}"
        return 0
    else
        echo -e "${RED}✗ $service is not running on port $port${NC}"
        return 1
    fi
}

# Function to start a service in the background
start_service() {
    local service=$1
    local binary=$2
    local port=$3
    local log_file="/tmp/${service}.log"
    
    echo "Starting $service..."
    $binary > $log_file 2>&1 &
    local pid=$!
    
    # Wait for service to start
    local count=0
    while ! curl -s -f http://localhost:$port/health >/dev/null 2>&1; do
        sleep 1
        count=$((count + 1))
        if [ $count -gt 30 ]; then
            echo -e "${RED}Failed to start $service after 30 seconds${NC}"
            cat $log_file
            kill $pid 2>/dev/null || true
            return 1
        fi
    done
    
    echo -e "${GREEN}Started $service (PID: $pid)${NC}"
    return 0
}

# Test 1: Check if services are already running
echo "=== Test 1: Checking existing services ==="
ASSETMANAGERD_RUNNING=false
METALD_RUNNING=false

if check_service "assetmanagerd" $ASSETMANAGERD_PORT; then
    ASSETMANAGERD_RUNNING=true
fi

if check_service "metald" $METALD_PORT; then
    METALD_RUNNING=true
    echo -e "${YELLOW}Stopping existing metald to test with new configuration...${NC}"
    pkill -f "metald" || true
    sleep 2
fi

# Test 2: Start assetmanagerd if not running
echo
echo "=== Test 2: Starting assetmanagerd (if needed) ==="
ASSETMANAGERD_PID=""
if [ "$ASSETMANAGERD_RUNNING" = false ]; then
    # Build assetmanagerd first
    echo "Building assetmanagerd..."
    (cd ../assetmanagerd && make build)
    
    if [ -f "../assetmanagerd/bin/assetmanagerd" ]; then
        start_service "assetmanagerd" "../assetmanagerd/bin/assetmanagerd" $ASSETMANAGERD_PORT
        ASSETMANAGERD_PID=$!
    else
        echo -e "${RED}assetmanagerd binary not found!${NC}"
        echo "Please build assetmanagerd first: cd ../assetmanagerd && make build"
        exit 1
    fi
fi

# Test 3: Test metald with assetmanager enabled
echo
echo "=== Test 3: Testing metald with AssetManager enabled ==="
export UNKEY_METALD_ASSETMANAGER_ENABLED=true
export UNKEY_METALD_ASSETMANAGER_ENDPOINT=http://localhost:$ASSETMANAGERD_PORT
export UNKEY_METALD_BACKEND=firecracker
export UNKEY_METALD_JAILER_ENABLED=false
export UNKEY_METALD_BILLING_ENABLED=false

echo "Starting metald with AssetManager integration enabled..."
./build/metald > /tmp/metald-with-assetmanager.log 2>&1 &
METALD_PID=$!

# Wait for metald to start
sleep 5

# Check if metald started successfully
if check_service "metald" $METALD_PORT; then
    echo -e "${GREEN}✓ Metald started successfully with AssetManager integration${NC}"
    
    # Check logs for assetmanager client initialization
    if grep -q "initialized asset manager client" /tmp/metald-with-assetmanager.log; then
        echo -e "${GREEN}✓ AssetManager client initialized successfully${NC}"
    else
        echo -e "${YELLOW}⚠ AssetManager client initialization not found in logs${NC}"
    fi
    
    # Show relevant log entries
    echo
    echo "Relevant log entries:"
    grep -E "(asset|AssetManager)" /tmp/metald-with-assetmanager.log | tail -10 || true
else
    echo -e "${RED}✗ Metald failed to start with AssetManager integration${NC}"
    echo "Log output:"
    tail -20 /tmp/metald-with-assetmanager.log
fi

# Stop metald
kill $METALD_PID 2>/dev/null || true
sleep 2

# Test 4: Test metald with assetmanager disabled (fallback test)
echo
echo "=== Test 4: Testing metald with AssetManager disabled (fallback) ==="
export UNKEY_METALD_ASSETMANAGER_ENABLED=false

echo "Starting metald with AssetManager integration disabled..."
./build/metald > /tmp/metald-without-assetmanager.log 2>&1 &
METALD_PID=$!

# Wait for metald to start
sleep 5

# Check if metald started successfully
if check_service "metald" $METALD_PORT; then
    echo -e "${GREEN}✓ Metald started successfully without AssetManager (fallback mode)${NC}"
    
    # Check logs for fallback behavior
    if grep -q "AssetManager not configured" /tmp/metald-without-assetmanager.log || grep -q "enabled\": false" /tmp/metald-without-assetmanager.log; then
        echo -e "${GREEN}✓ Fallback to hardcoded assets confirmed${NC}"
    fi
else
    echo -e "${RED}✗ Metald failed to start in fallback mode${NC}"
    echo "Log output:"
    tail -20 /tmp/metald-without-assetmanager.log
fi

# Cleanup
echo
echo "=== Cleanup ==="
kill $METALD_PID 2>/dev/null || true
if [ -n "$ASSETMANAGERD_PID" ]; then
    kill $ASSETMANAGERD_PID 2>/dev/null || true
fi

echo
echo "=== Test Summary ==="
echo "Test logs saved to:"
echo "  - /tmp/assetmanagerd.log"
echo "  - /tmp/metald-with-assetmanager.log"
echo "  - /tmp/metald-without-assetmanager.log"
echo
echo "To manually test the integration:"
echo "  1. Start assetmanagerd: cd ../assetmanagerd && ./bin/assetmanagerd"
echo "  2. Start metald: UNKEY_METALD_ASSETMANAGER_ENABLED=true ./build/metald"
echo "  3. Create a VM and check logs for asset preparation"