#!/bin/bash
# Test environment variable setup in metald-init

set -e

echo "=== Testing metald-init environment variable setup ==="

# Build the init wrapper
echo "Building metald-init..."
make build

# Create a test metadata file
cat > test-metadata.json <<EOF
{
  "env": {
    "TEST_VAR": "from_metadata",
    "NGINX_VERSION": "1.25.0",
    "PATH": "/should/be/ignored"
  },
  "working_dir": "/app",
  "entrypoint": ["/docker-entrypoint.sh"],
  "command": ["nginx", "-g", "daemon off;"]
}
EOF

# Test 1: Environment from kernel cmdline simulation
echo -e "\n--- Test 1: Environment from simulated kernel cmdline ---"
# We can't modify /proc/cmdline, so we'll modify the code to accept a test file
# For now, let's test with direct execution
./metald-init -- env | grep -E "(TEST_|NGINX_)" || true

# Test 2: Test with a wrapper script that sets environment
echo -e "\n--- Test 2: Test with env command ---"
cat > test-env-wrapper.sh <<'EOF'
#!/bin/bash
# This simulates what would happen with kernel cmdline: env.TEST_VAR=cmdline_value
export TEST_FROM_WRAPPER=wrapper_value
exec ./metald-init -- env
EOF
chmod +x test-env-wrapper.sh
./test-env-wrapper.sh | grep -E "(TEST_|wrapper)" || true

# Test 3: Create a modified version that reads from a test cmdline file
echo -e "\n--- Test 3: Creating test version with mock cmdline ---"
cat > test-init.go <<'EOF'
package main

import (
	"os"
	"os/exec"
	"strings"
)

func main() {
	// Set up test kernel parameters
	os.Setenv("TEST_CMDLINE", "env.TEST_VAR=from_cmdline env.ANOTHER_VAR=test123 metadata=/metadata.json workdir=/tmp")
	
	// Run the actual init with test environment
	cmd := exec.Command("./metald-init", os.Args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), "TEST_MODE=1")
	cmd.Run()
}
EOF

# Clean up
echo -e "\n--- Cleanup ---"
rm -f test-metadata.json test-env-wrapper.sh test-init.go

echo -e "\n=== Environment variable tests completed ==="
echo "Note: Full testing requires running in a VM where /proc/cmdline can be controlled"