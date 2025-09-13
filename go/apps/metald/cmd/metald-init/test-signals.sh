#!/bin/bash
# Test signal handling in metald-init

set -e

echo "=== Testing metald-init signal handling ==="

# Build the init wrapper
echo "Building metald-init..."
make build

# Test 1: Test with sleep and SIGTERM
echo -e "\n--- Test 1: SIGTERM forwarding ---"
echo "Starting sleep process through metald-init..."
./metald-init -- sleep 30 &
INIT_PID=$!
sleep 1

# Get the sleep process PID
SLEEP_PID=$(pgrep -P $INIT_PID sleep || echo "not found")
echo "Init PID: $INIT_PID, Sleep PID: $SLEEP_PID"

if [ "$SLEEP_PID" != "not found" ]; then
    echo "Sending SIGTERM to init process..."
    kill -TERM $INIT_PID
    
    # Wait a bit and check if process terminated
    sleep 2
    if ! kill -0 $INIT_PID 2>/dev/null; then
        echo "✓ Init process terminated after SIGTERM"
    else
        echo "✗ Init process still running, forcing kill"
        kill -9 $INIT_PID 2>/dev/null || true
    fi
else
    echo "✗ Could not find sleep process"
    kill -9 $INIT_PID 2>/dev/null || true
fi

# Test 2: Test with a script that handles signals
echo -e "\n--- Test 2: Signal handling with trap ---"
cat > signal-test.sh <<'EOF'
#!/bin/bash
echo "Signal test script started with PID $$"
trap 'echo "Received SIGTERM, cleaning up..."; exit 0' TERM
trap 'echo "Received SIGINT, cleaning up..."; exit 0' INT

echo "Waiting for signals..."
while true; do
    sleep 1
done
EOF
chmod +x signal-test.sh

echo "Starting signal test script..."
./metald-init -- ./signal-test.sh &
INIT_PID=$!
sleep 1

echo "Sending SIGTERM to init..."
kill -TERM $INIT_PID
sleep 2

if ! kill -0 $INIT_PID 2>/dev/null; then
    echo "✓ Init and child process terminated gracefully"
else
    echo "✗ Process still running, forcing kill"
    kill -9 $INIT_PID 2>/dev/null || true
fi

# Test 3: Test zombie reaping
echo -e "\n--- Test 3: Zombie process reaping ---"
cat > zombie-test.sh <<'EOF'
#!/bin/bash
echo "Creating zombie processes..."
# Create a process that exits immediately (becomes zombie)
(sleep 0.1) &
(sleep 0.1) &
(sleep 0.1) &
echo "Created 3 potential zombie processes"
# Keep running so we can check
sleep 5
echo "Zombie test complete"
EOF
chmod +x zombie-test.sh

echo "Starting zombie test..."
./metald-init -- ./zombie-test.sh &
INIT_PID=$!
sleep 2

# Check for zombie processes
ZOMBIES=$(ps aux | grep -E "Z.*defunct" | grep -v grep | wc -l)
echo "Number of zombie processes: $ZOMBIES"

if [ $ZOMBIES -eq 0 ]; then
    echo "✓ No zombie processes found - reaping works!"
else
    echo "✗ Found $ZOMBIES zombie processes"
fi

# Clean up
kill -TERM $INIT_PID 2>/dev/null || true
wait $INIT_PID 2>/dev/null || true

# Cleanup
rm -f signal-test.sh zombie-test.sh

echo -e "\n=== Signal handling tests completed ==="