#!/bin/bash
# Test firecracker without jailer to confirm the tap device issue

echo "Stopping metald service..."
sudo systemctl stop metald

echo "Starting metald without jailer..."
sudo UNKEY_METALD_JAILER_ENABLED=false /usr/local/bin/metald &
METALD_PID=$!

echo "Waiting for metald to start..."
sleep 3

echo "Metald started with PID: $METALD_PID"
echo "Run the test client to create a VM"
echo ""
echo "To stop metald: sudo kill $METALD_PID"