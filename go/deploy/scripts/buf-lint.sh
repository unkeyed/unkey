#!/bin/bash
set -euo pipefail

# Enhanced Proto file linting and formatting script
# Usage: ./scripts/lint-proto.sh <file_path>

file_path="${1:-}"

if [[ -z "${file_path}" ]]; then
    echo "Usage: $0 <proto_file_path>"
    exit 1
fi

if [[ ! -f "${file_path}" ]]; then
    echo "Error: File '${file_path}' does not exist"
    exit 1
fi

if [[ "${file_path}" != *.proto ]]; then
    echo "Error: File '${file_path}' is not a proto file"
    exit 1
fi

echo "Processing proto file: ${file_path}"

# Check if buf is available
if ! command -v buf >/dev/null 2>&1; then
    echo "Error: buf command not found. Please install buf."
    echo "Install with: go install github.com/bufbuild/buf/cmd/buf@latest"
    exit 1
fi

# Step 1: Run buf format
echo "Running buf format..."
if buf format --write "${file_path}" 2>/dev/null; then
    echo "✓ buf format completed"
else
    echo "ℹ buf format not available or failed (continuing)"
fi

# Step 2: Run buf lint (this is the critical check)
echo "Running buf lint..."
if buf lint "${file_path}"; then
    echo "✓ buf lint passed"
else
    echo "✗ buf lint failed"
    exit 2  # Exit code 2 for linting failures
fi

# Step 3: Run buf breaking change detection
echo "Running buf breaking change detection..."

# Check if we're in a git repository
if ! git rev-parse --git-dir >/dev/null 2>&1; then
    echo "ℹ not in a git repository, skipping breaking change detection"
elif ! git rev-parse HEAD >/dev/null 2>&1; then
    echo "ℹ no commits found, skipping breaking change detection"
elif ! git rev-parse HEAD~1 >/dev/null 2>&1; then
    echo "ℹ only one commit found, skipping breaking change detection"
else
    # Try different breaking change detection strategies
    if buf breaking --against .git#branch=HEAD~1 "${file_path}" 2>/dev/null; then
        echo "✓ no breaking changes detected against HEAD~1"
    elif buf breaking --against .git#branch=main "${file_path}" 2>/dev/null; then
        echo "✓ no breaking changes detected against main branch"
    elif buf breaking --against .git#branch=master "${file_path}" 2>/dev/null; then
        echo "✓ no breaking changes detected against master branch"
    else
        echo "⚠ potential breaking changes detected (not blocking)"
        echo "ℹ run 'buf breaking --against .git#branch=main ${file_path}' manually for details"
    fi
fi

echo "Running buf breaking change detection..."

# Check if we're in a git repository
if ! git rev-parse --git-dir >/dev/null 2>&1; then
    echo "ℹ not in a git repository, skipping breaking change detection"
elif ! git rev-parse HEAD >/dev/null 2>&1; then
    echo "ℹ no commits found, skipping breaking change detection"
elif ! git rev-parse HEAD~1 >/dev/null 2>&1; then
    echo "ℹ only one commit found, skipping breaking change detection"
else
    # Try different breaking change detection strategies
    if buf breaking --against .git#branch=HEAD~1 "${file_path}" 2>/dev/null; then
        echo "✓ no breaking changes detected against HEAD~1"
    elif buf breaking --against .git#branch=main "${file_path}" 2>/dev/null; then
        echo "✓ no breaking changes detected against main branch"
    elif buf breaking --against .git#branch=master "${file_path}" 2>/dev/null; then
        echo "✓ no breaking changes detected against master branch"
    else
        echo "⚠ potential breaking changes detected (not blocking)"
        echo "ℹ run 'buf breaking --against .git#branch=main ${file_path}' manually for details"
    fi
fi

echo "✓ Proto file processing completed: ${file_path}"
