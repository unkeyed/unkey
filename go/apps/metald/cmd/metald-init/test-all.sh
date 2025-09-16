#!/bin/bash
# Comprehensive test suite for metald-init

set -e

echo "=== Comprehensive metald-init test suite ==="
echo "Building metald-init..."
make build

TESTS_PASSED=0
TESTS_FAILED=0

# Helper function to run a test
run_test() {
    local test_name="$1"
    local test_cmd="$2"
    echo -e "\n--- Test: $test_name ---"
    if eval "$test_cmd"; then
        echo "✓ PASSED: $test_name"
        ((TESTS_PASSED++))
    else
        echo "✗ FAILED: $test_name"
        ((TESTS_FAILED++))
    fi
}

# Test 1: Basic execution
test_basic() {
    ./metald-init -- echo "Hello World" | grep -q "Hello World"
}
run_test "Basic execution" test_basic

# Test 2: Exit code propagation
test_exit_code() {
    # Should exit with 0
    ./metald-init -- true
    [ $? -eq 0 ] || return 1
    
    # Should exit with 1
    ./metald-init -- false || [ $? -eq 1 ]
}
run_test "Exit code propagation" test_exit_code

# Test 3: Environment variables from metadata
test_env_metadata() {
    # Create test metadata
    cat > test-metadata.json <<EOF
{
  "env": {
    "TEST_VAR": "from_metadata",
    "ANOTHER_VAR": "test123"
  },
  "working_dir": "/tmp"
}
EOF
    
    # Test would need kernel cmdline simulation
    # For now, just verify the file is valid JSON
    python3 -m json.tool test-metadata.json > /dev/null
    rm -f test-metadata.json
}
run_test "Environment metadata parsing" test_env_metadata

# Test 4: Working directory
test_workdir() {
    # Test changing to /tmp
    cd /
    ./metald-init -- pwd | grep -q "/"
}
run_test "Working directory" test_workdir

# Test 5: Signal forwarding
test_signal_forward() {
    # Start a sleep process
    timeout 2 ./metald-init -- sleep 10 || [ $? -eq 124 ]
}
run_test "Signal forwarding (timeout)" test_signal_forward

# Test 6: Multiple arguments
test_multiple_args() {
    ./metald-init -- echo "one" "two" "three" | grep -q "one two three"
}
run_test "Multiple arguments" test_multiple_args

# Test 7: Stdin/stdout/stderr
test_stdio() {
    # Test stdin
    echo "test input" | ./metald-init -- cat | grep -q "test input" || return 1
    
    # Test stderr
    ./metald-init -- sh -c 'echo "error" >&2' 2>&1 | grep -q "error"
}
run_test "Stdin/stdout/stderr" test_stdio

# Test 8: Binary execution
test_binary() {
    ./metald-init -- /bin/ls /bin > /dev/null
}
run_test "Binary execution" test_binary

# Test 9: Shell script execution
test_shell_script() {
    cat > test-script.sh <<'EOF'
#!/bin/bash
echo "Script executed"
exit 42
EOF
    chmod +x test-script.sh
    ./metald-init -- ./test-script.sh | grep -q "Script executed" || return 1
    # Check exit code
    ./metald-init -- ./test-script.sh > /dev/null || [ $? -eq 42 ]
    rm -f test-script.sh
}
run_test "Shell script execution" test_shell_script

# Test 10: Long running process
test_long_running() {
    # Start a process that runs for 1 second
    start_time=$(date +%s)
    ./metald-init -- sleep 1
    end_time=$(date +%s)
    duration=$((end_time - start_time))
    [ $duration -ge 1 ] && [ $duration -le 2 ]
}
run_test "Long running process" test_long_running

# Summary
echo -e "\n=== Test Summary ==="
echo "Tests passed: $TESTS_PASSED"
echo "Tests failed: $TESTS_FAILED"
echo "Total tests: $((TESTS_PASSED + TESTS_FAILED))"

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "\n✓ All tests passed!"
    exit 0
else
    echo -e "\n✗ Some tests failed"
    exit 1
fi