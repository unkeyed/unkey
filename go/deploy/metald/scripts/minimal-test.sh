#!/bin/bash

for f in grafana-dashboards/*.json; do
    echo "=== Processing $f ==="
    
    # Just validate JSON, don't import
    if jq . "$f" >/dev/null; then
        echo "✅ Valid JSON"
    else
        echo "❌ Invalid JSON"
    fi
    
    echo "Done with $f"
done

echo "Script completed"