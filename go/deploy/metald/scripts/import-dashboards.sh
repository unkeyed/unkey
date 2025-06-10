#!/bin/bash
set -euo pipefail

# Import Grafana dashboards for metald monitoring
# Requires the LGTM stack to be running (use: make o11y)

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DASHBOARD_DIR="$(dirname "$SCRIPT_DIR")/grafana-dashboards"
GRAFANA_URL="${GRAFANA_URL:-http://localhost:3000}"
GRAFANA_USER="${GRAFANA_USER:-admin}"
GRAFANA_PASS="${GRAFANA_PASS:-admin}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if Grafana is accessible
check_grafana() {
    log_info "Checking Grafana accessibility at $GRAFANA_URL..."
    
    if ! curl -s -f -u "$GRAFANA_USER:$GRAFANA_PASS" "$GRAFANA_URL/api/health" >/dev/null; then
        log_error "Grafana is not accessible at $GRAFANA_URL"
        log_error "Make sure the LGTM stack is running: make o11y"
        exit 1
    fi
    
    log_info "Grafana is accessible"
}

# Remove metald dashboards by searching for them
remove_existing_dashboards() {
    log_info "Removing existing metald dashboards..."
    
    # Search for dashboards containing "metald" in title
    local search_response=$(curl -s -u "$GRAFANA_USER:$GRAFANA_PASS" \
        "$GRAFANA_URL/api/search?type=dash-db&query=metald")
    
    if [ "$search_response" = "null" ] || [ "$search_response" = "[]" ]; then
        log_info "No existing metald dashboards found"
        return 0
    fi
    
    # Parse and delete each dashboard
    local deleted=0
    while IFS= read -r uid; do
        if [ -n "$uid" ] && [ "$uid" != "null" ]; then
            log_info "Removing dashboard with UID: $uid"
            local delete_response=$(curl -s -X DELETE \
                -u "$GRAFANA_USER:$GRAFANA_PASS" \
                "$GRAFANA_URL/api/dashboards/uid/$uid")
            
            if echo "$delete_response" | jq -e '.message == "Dashboard deleted"' >/dev/null 2>&1; then
                ((deleted++))
                log_info "✅ Successfully removed dashboard: $uid"
            else
                log_warn "⚠️  Could not remove dashboard $uid: $delete_response"
            fi
        fi
    done < <(echo "$search_response" | jq -r '.[].uid // empty')
    
    if [ $deleted -gt 0 ]; then
        log_info "Removed $deleted existing dashboards"
    fi
}


# Create datasource if it doesn't exist
setup_prometheus_datasource() {
    log_info "Setting up Prometheus datasource..."
    
    # Check if datasource already exists
    local existing=$(curl -s -u "$GRAFANA_USER:$GRAFANA_PASS" \
        "$GRAFANA_URL/api/datasources/name/prometheus" 2>/dev/null || echo "null")
    
    if [ "$existing" != "null" ]; then
        log_info "Prometheus datasource already exists"
        return 0
    fi
    
    # Create the datasource
    local datasource_payload='{
        "name": "prometheus",
        "type": "prometheus",
        "url": "http://localhost:9090",
        "access": "proxy",
        "isDefault": true,
        "basicAuth": false
    }'
    
    local response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -u "$GRAFANA_USER:$GRAFANA_PASS" \
        -d "$datasource_payload" \
        "$GRAFANA_URL/api/datasources")
    
    if echo "$response" | jq -e '.id' >/dev/null 2>&1; then
        log_info "✅ Successfully created Prometheus datasource"
    else
        log_warn "⚠️  Could not create Prometheus datasource (may already exist)"
        log_warn "   Response: $response"
    fi
}

# Main function
main() {
    echo "🚀 Metald Dashboard Import Script"
    echo "================================"
    
    # Check dependencies
    if ! command -v curl >/dev/null 2>&1; then
        log_error "curl is required but not installed"
        exit 1
    fi
    
    if ! command -v jq >/dev/null 2>&1; then
        log_error "jq is required but not installed"
        exit 1
    fi
    
    # Check if dashboard directory exists
    if [ ! -d "$DASHBOARD_DIR" ]; then
        log_error "Dashboard directory not found: $DASHBOARD_DIR"
        exit 1
    fi
    
    # Check Grafana accessibility
    check_grafana
    
    # Setup Prometheus datasource
    setup_prometheus_datasource
    
    # Remove existing metald dashboards
    remove_existing_dashboards
    
    # Import all dashboards
    log_info "Importing dashboards from: $DASHBOARD_DIR"
    
    local imported=0
    local failed=0
    
    for dashboard_file in "$DASHBOARD_DIR"/*.json; do
        if [ -f "$dashboard_file" ]; then
            local dashboard_name=$(basename "$dashboard_file" .json)
            log_info "Importing dashboard: $dashboard_name"
            
            # Validate JSON first
            if ! jq . "$dashboard_file" >/dev/null 2>&1; then
                log_error "❌ Invalid JSON in $dashboard_file"
                ((failed++))
                continue
            fi
            
            # Create payload and import
            local payload=$(cat "$dashboard_file" | jq -c '{dashboard: ., overwrite: true, message: "Imported via script"}')
            local response=$(curl -s -X POST \
                -H "Content-Type: application/json" \
                -u "$GRAFANA_USER:$GRAFANA_PASS" \
                -d "$payload" \
                "$GRAFANA_URL/api/dashboards/db")
            
            # Check if import was successful
            if echo "$response" | jq -e '.status == "success"' >/dev/null 2>&1; then
                local dashboard_uid=$(echo "$response" | jq -r '.uid')
                local dashboard_url="$GRAFANA_URL/d/$dashboard_uid"
                log_info "✅ Successfully imported $dashboard_name"
                log_info "   Dashboard URL: $dashboard_url"
                ((imported++))
            else
                log_error "❌ Failed to import $dashboard_name"
                log_error "   Response: $response"
                ((failed++))
            fi
        fi
    done
    
    echo ""
    echo "📊 Import Summary"
    echo "================="
    log_info "Successfully imported: $imported dashboards"
    
    if [ $failed -gt 0 ]; then
        log_error "Failed to import: $failed dashboards"
        exit 1
    else
        log_info "🎉 All dashboards imported successfully!"
        echo ""
        echo "🔗 Access your dashboards at: $GRAFANA_URL"
        echo "   Username: $GRAFANA_USER"
        echo "   Password: $GRAFANA_PASS"
        echo ""
        echo "📋 Available dashboards:"
        echo "   • VM Operations: $GRAFANA_URL/d/metald-vm-ops"
        echo "   • Security Operations: $GRAFANA_URL/d/metald-security-ops"
        echo "   • Billing & Metrics: $GRAFANA_URL/d/metald-billing"
        echo "   • Multi-Tenant Billing: $GRAFANA_URL/d/metald-multi-tenant-billing"
        echo "   • System Health: $GRAFANA_URL/d/metald-system-health"
    fi
}

# Run main function
main "$@"