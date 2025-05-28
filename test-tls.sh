#!/bin/bash
set -e

# Directory for temporary files
TEMP_DIR=$(mktemp -d)
echo "Created temporary directory: $TEMP_DIR"

# Cleanup function to remove temporary files
cleanup() {
  echo "Cleaning up..."
  rm -rf "$TEMP_DIR"
}

# Register the cleanup function to be called on script exit
trap cleanup EXIT

# Generate self-signed certificate and key
echo "Generating self-signed certificate and key..."
openssl req -x509 -newkey rsa:4096 -keyout "$TEMP_DIR/key.pem" -out "$TEMP_DIR/cert.pem" \
  -days 365 -nodes -subj "/CN=localhost" -addext "subjectAltName=DNS:localhost"

# Set proper permissions
chmod 600 "$TEMP_DIR/key.pem"
chmod 644 "$TEMP_DIR/cert.pem"

echo "Certificate and key generated successfully!"
echo "Certificate: $TEMP_DIR/cert.pem"
echo "Key: $TEMP_DIR/key.pem"

# Start the Unkey server with TLS in the background
echo "Starting Unkey server with TLS..."
echo "Press Ctrl+C to stop the server when testing is complete"

# Build the command to run the server
# Note: Using --test-mode to bypass the database requirement for simple testing
# Modify this command based on your environment
cd go # Navigate to the go directory
go run ./main.go api \
  --tls-cert-file="$TEMP_DIR/cert.pem" \
  --tls-key-file="$TEMP_DIR/key.pem" \
  --test-mode \
  --database-primary="mysql://fake:fake@localhost/fake" \
  --http-port=7071 &
cd - > /dev/null # Return to original directory

# Save the server process ID
SERVER_PID=$!

# Wait for the server to start
echo "Waiting for server to start..."
sleep 3

# Make a curl request to test the HTTPS connection
echo "Testing HTTPS connection..."
echo "Note: Using -k flag to skip certificate verification since we're using a self-signed certificate"
echo "Waiting 2 more seconds for server to be fully ready..."
sleep 2
curl -k -v https://localhost:7071/v2/liveness 2>&1 | grep -i "SSL connection\|TLS\|certificate\|protocol"

# Wait for user to manually verify and press Ctrl+C
echo ""
echo "Server is running with HTTPS enabled on port 7071"
echo "You can manually verify it works by opening https://localhost:7071/v2/liveness in your browser"
echo "or using additional curl commands"
echo ""
echo "Press Ctrl+C to stop the server when testing is complete"

# Wait for the server process
wait $SERVER_PID || {
  echo "Server exited with error code $?"
  exit 1
}
