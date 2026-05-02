#!/bin/bash
set -e

# Logdrain Performance Testing Script
# This script runs end-to-end performance testing of the logdrain service
# using real ClickHouse data and monitoring actual Prometheus metrics.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Configuration
CLICKHOUSE_URL="clickhouse://default:password@localhost:9000?secure=false"
MYSQL_DSN="unkey:password@tcp(localhost:3306)/unkey?parseTime=true&interpolateParams=true"
METRICS_URL="http://localhost:9402/metrics"
TEST_DURATION="5m"
SAMPLE_INTERVAL="5s"

# Per-(drain, environment) record counts. Tenants come from MySQL log_drains —
# generating logs for random workspaces is meaningless because the coordinator
# only forwards (workspace, project, environment, source) tuples that have a
# real drain attached.
LOGS_PER_DRAIN=2000
REQUESTS_PER_DRAIN=1000
DEPLOYMENTS_PER_DRAIN=3
BATCH_SIZE=5000

echo "🚀 Starting Logdrain Performance Testing"
echo "========================================"
echo "📊 Per-drain/env volume:"
echo "   • Runtime logs per drain/env: $LOGS_PER_DRAIN"
echo "   • Request logs per drain/env: $REQUESTS_PER_DRAIN"
echo "   • Deployments per drain/env:  $DEPLOYMENTS_PER_DRAIN"
echo "   • Batch flush size:           $BATCH_SIZE"
echo "   • Test duration:              $TEST_DURATION"
echo ""

# Check prerequisites
echo "🔍 Checking prerequisites..."

# Check if kubectl is available and connected
if ! kubectl get namespace unkey >/dev/null 2>&1; then
    echo "❌ kubectl not connected or unkey namespace not found"
    echo "💡 Make sure you're connected to the dev k8s cluster"
    exit 1
fi
echo "✅ kubectl connected to k8s cluster"

# Check if ClickHouse is accessible
if ! nc -z localhost 9000 >/dev/null 2>&1; then
    echo "❌ ClickHouse not accessible on localhost:9000"
    echo "💡 Make sure Tilt is running (make dev) — it forwards 9000 automatically"
    exit 1
fi
echo "✅ ClickHouse accessible"

# Check if MySQL is accessible (data generator queries log_drains for tenants)
if ! nc -z localhost 3306 >/dev/null 2>&1; then
    echo "❌ MySQL not accessible on localhost:3306"
    echo "💡 Make sure Tilt is running (make dev) — it forwards 3306 automatically"
    exit 1
fi
echo "✅ MySQL accessible"

# Check if logdrain service exists
if ! kubectl get deployment logdrain -n unkey >/dev/null 2>&1; then
    echo "❌ logdrain deployment not found"
    echo "💡 Deploy logdrain with: kubectl apply -f dev/k8s/manifests/logdrain.yaml"
    exit 1
fi
echo "✅ logdrain deployment found"

# Tilt forwards both ClickHouse (9000) and logdrain metrics (9402) automatically.
if ! nc -z localhost 9402 >/dev/null 2>&1; then
    echo "❌ logdrain metrics not reachable on localhost:9402"
    echo "💡 Make sure Tilt is running (make dev)"
    exit 1
fi
echo "✅ logdrain metrics reachable on localhost:9402"

cleanup() {
    echo ""
    echo "🧹 Cleaning up..."
    kill $MONITOR_PID 2>/dev/null || true
    echo "✅ Cleanup completed"
}
trap cleanup EXIT

echo ""
echo "🏗️  Phase 1: Data Generation"
echo "=========================="

# Build and run data generator
echo "📝 Generating test data in ClickHouse..."
cd "$SCRIPT_DIR"

go run "$PROJECT_ROOT/benchmarks/logdrain/cmd/data-generator" \
    -clickhouse="$CLICKHOUSE_URL" \
    -mysql="$MYSQL_DSN" \
    -logs-per-drain=$LOGS_PER_DRAIN \
    -requests-per-drain=$REQUESTS_PER_DRAIN \
    -deployments=$DEPLOYMENTS_PER_DRAIN \
    -batch-size=$BATCH_SIZE

echo "✅ Test data generation completed!"
echo ""

echo "🔄 Phase 2: Logdrain Readiness"
echo "==============================="

# logdrain polls ClickHouse on its tick interval — no restart needed to pick
# up new rows. Just wait until the pod is healthy. Tilt owns the deployment
# lifecycle; running `kubectl rollout restart` here would replace the live-
# updated binary with the last fully baked image and crash.
echo "⏳ Waiting for logdrain pod to be ready..."
if ! kubectl wait --for=condition=available deployment/logdrain -n unkey --timeout=60s; then
    echo "❌ logdrain isn't healthy. Check 'kubectl logs -l app=logdrain -n unkey' and Tilt status"
    exit 1
fi
echo "✅ logdrain ready"
echo ""

echo "📊 Phase 3: Performance Monitoring"  
echo "=================================="

# Start metrics monitoring in background
echo "📈 Starting metrics monitoring for $TEST_DURATION..."
go run cmd/metrics-monitor/main.go \
    -metrics-url="$METRICS_URL" \
    -duration="$TEST_DURATION" \
    -interval="$SAMPLE_INTERVAL" &
MONITOR_PID=$!

# Wait for monitoring to complete
wait $MONITOR_PID
MONITOR_EXIT_CODE=$?

echo ""
if [ $MONITOR_EXIT_CODE -eq 0 ]; then
    echo "✅ Performance testing completed successfully!"
else
    echo "❌ Performance monitoring failed with exit code $MONITOR_EXIT_CODE"
fi

echo ""
echo "📋 Phase 4: Final Status Check"
echo "=============================="

# Check final logdrain status
echo "🔍 Final logdrain status:"
kubectl get pods -l app=logdrain -n unkey
echo ""

# Check if any drains were created (this would be manual via UI)
echo "💡 To test with real drains:"
echo "   1. Open dashboard: kubectl port-forward svc/dashboard 3000:3000 -n unkey"
echo "   2. Navigate to Settings → Log Drains"  
echo "   3. Create Axiom drain"
echo "   4. Re-run this script to see delivery metrics"
echo ""

echo "🎯 Performance test completed!"
echo "Check the metrics report above for throughput analysis."
