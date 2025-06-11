#!/bin/bash

GRAFANA_URL="http://localhost:3000"
GRAFANA_USER="admin"
GRAFANA_PASS="admin"

for f in grafana-dashboards/*.json; do
    echo "=== Processing $f ==="
    
    echo "1. Creating payload..."
    if ! payload=$(cat "$f" | jq -c '{dashboard: ., overwrite: true, message: "Debug test"}'); then
        echo "❌ Failed to create payload"
        continue
    fi
    echo "✅ Payload created (${#payload} bytes)"
    
    echo "2. Sending to Grafana..."
    if ! response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -u "$GRAFANA_USER:$GRAFANA_PASS" \
        -d "$payload" \
        "$GRAFANA_URL/api/dashboards/db"); then
        echo "❌ Failed to send request"
        continue
    fi
    echo "✅ Response received (${#response} bytes)"
    
    echo "3. Checking response..."
    if echo "$response" | jq -e '.status == "success"' >/dev/null 2>&1; then
        uid=$(echo "$response" | jq -r '.uid')
        echo "✅ Success: $uid"
    else
        echo "❌ Import failed: $(echo "$response" | jq -r '.message // "unknown error"')"
    fi
    
    echo "4. Done with $f"
    echo ""
done

echo "Script completed"