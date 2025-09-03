#!/bin/bash
set -euo pipefail

# Enhanced Go file formatting and linting script
# Usage: ./scripts/format-go.sh <file_path>

file_path="${1:-}"

if [[ -z "${file_path}" ]]; then
    echo "Usage: $0 <go_file_path>"
    exit 1
fi

if [[ ! -f "${file_path}" ]]; then
    echo "Error: File '${file_path}' does not exist"
    exit 1
fi

if [[ "${file_path}" != *.go ]]; then
    echo "Error: File '${file_path}' is not a Go file"
    exit 1
fi

echo "Processing Go file: ${file_path}"

# Step 1: Format with gofmt
echo "Running gofmt..."
if gofumpt -w "${file_path}"; then
    echo "✓ gofmt completed"
else
    echo "✗ gofmt failed"
    exit 1
fi

# Step 2: Run goimports if available
if command -v goimports >/dev/null 2>&1; then
    echo "Running goimports..."
    if goimports -w "${file_path}"; then
        echo "✓ goimports completed"
    else
        echo "✗ goimports failed"
        exit 1
    fi
else
    echo "ℹ goimports not installed, skipping import formatting"
fi

# Step 3: Run go vet on the package
if command -v go >/dev/null 2>&1; then
    dir=$(dirname "${file_path}")
    echo "Running go vet on package in $dir..."
    if go vet "$dir" 2>/dev/null; then
        echo "✓ go vet passed"
    else
        echo "⚠ go vet found issues (not blocking)"
        # Don't exit on vet issues, just warn
    fi
else
    echo "ℹ go command not available, skipping vet"
fi

# Step 4: Optional: Run golangci-lint if available and configured
if command -v golangci-lint >/dev/null 2>&1 && [[ -f .golangci.yml || -f .golangci.yaml ]]; then
    echo "Running golangci-lint..."
    if golangci-lint run "${file_path}" 2>/dev/null; then
        echo "✓ golangci-lint passed"
    else
        echo "⚠ golangci-lint found issues (not blocking)"
    fi
fi

echo "✓ Go file processing completed: ${file_path}"
