#!/bin/bash

# Quick test script to validate dashboard structure
# This doesn't require Grafana to be running

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DASHBOARD_DIR="$(dirname "$SCRIPT_DIR")/grafana-dashboards"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

echo "🧪 Testing Dashboard Structure"
echo "============================="

# Check dependencies
if ! command -v jq >/dev/null 2>&1; then
    log_error "jq is required but not installed"
    exit 1
fi

# Test each dashboard
total=0
passed=0

for dashboard_file in "$DASHBOARD_DIR"/*.json; do
    if [ -f "$dashboard_file" ]; then
        dashboard_name=$(basename "$dashboard_file" .json)
        total=$((total + 1))
        
        echo ""
        log_info "Testing $dashboard_name..."
        
        # Test 1: Valid JSON
        if ! jq . "$dashboard_file" >/dev/null 2>&1; then
            log_error "❌ Invalid JSON structure"
            continue
        fi
        
        # Test 2: Has required fields
        if ! jq -e '.title' "$dashboard_file" >/dev/null; then
            log_error "❌ Missing title field"
            continue
        fi
        
        if ! jq -e '.panels' "$dashboard_file" >/dev/null; then
            log_error "❌ Missing panels array"
            continue
        fi
        
        # Test 3: Check panel structure
        panel_count=$(jq '.panels | length' "$dashboard_file")
        if [ "$panel_count" -eq 0 ]; then
            log_error "❌ No panels defined"
            continue
        fi
        
        # Test 4: Check datasource references
        datasource_refs=$(jq '[.panels[].datasource.type] | unique' "$dashboard_file")
        if echo "$datasource_refs" | grep -q "prometheus"; then
            log_info "✅ Uses Prometheus datasource"
        else
            log_error "❌ Missing Prometheus datasource references"
            continue
        fi
        
        # Test 5: Check for metrics queries
        query_count=$(jq '[.panels[].targets[]?.expr] | length' "$dashboard_file")
        if [ "$query_count" -gt 0 ]; then
            log_info "✅ Has $query_count PromQL queries"
        else
            log_error "❌ No PromQL queries found"
            continue
        fi
        
        # Test 6: Dashboard import payload structure
        test_payload=$(cat "$dashboard_file" | jq '{
            dashboard: .,
            overwrite: true,
            message: "Test payload"
        }' 2>/dev/null)
        
        if [ $? -eq 0 ]; then
            log_info "✅ Valid import payload structure"
        else
            log_error "❌ Cannot create valid import payload"
            continue
        fi
        
        passed=$((passed + 1))
        log_info "✅ All tests passed for $dashboard_name"
    fi
done

echo ""
echo "📊 Test Summary"
echo "==============="
log_info "Tested: $total dashboards"
log_info "Passed: $passed dashboards"

if [ $passed -eq $total ]; then
    log_info "🎉 All dashboards are valid and ready for import!"
    echo ""
    echo "Next steps:"
    echo "1. Start LGTM stack: make o11y"
    echo "2. Import dashboards: ./scripts/import-dashboards.sh"
    echo "3. Start metald with telemetry: UNKEY_METALD_OTEL_ENABLED=true ./build/metald"
else
    failed=$((total - passed))
    log_error "❌ $failed dashboard(s) failed validation"
    exit 1
fi