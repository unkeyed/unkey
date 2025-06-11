#!/bin/bash

# Simple test script to import all dashboards
set -euo pipefail

DASHBOARD_DIR="/home/imeyer/code/github.com/unkeyed/unkey/go/deploy/metald/grafana-dashboards"
GRAFANA_URL="http://localhost:3000"
GRAFANA_USER="admin"
GRAFANA_PASS="admin"

echo "🧪 Testing dashboard import..."

imported=0
failed=0

for dashboard_file in "$DASHBOARD_DIR"/*.json; do
    if [ -f "$dashboard_file" ]; then
        echo "Processing: $(basename "$dashboard_file")"
        
        # Create payload
        payload=$(cat "$dashboard_file" | jq -c '{dashboard: ., overwrite: true, message: "Test import"}')
        
        # Import
        response=$(curl -s -X POST \
            -H "Content-Type: application/json" \
            -u "$GRAFANA_USER:$GRAFANA_PASS" \
            -d "$payload" \
            "$GRAFANA_URL/api/dashboards/db")
        
        if echo "$response" | jq -e '.status == "success"' >/dev/null 2>&1; then
            uid=$(echo "$response" | jq -r '.uid')
            echo "✅ Imported: $uid"
            ((imported++))
        else
            echo "❌ Failed: $response"
            ((failed++))
        fi
    fi
done

echo ""
echo "📊 Results: $imported imported, $failed failed"