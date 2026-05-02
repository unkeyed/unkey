// Package main generates real log data in ClickHouse for logdrain performance testing.
//
// Writes to runtime_logs_raw_v1 and sentinel_requests_raw_v1 using the shared
// pkg/clickhouse Buffer so we exercise the same insert path as the rest of the
// platform. Raw database/sql Prepare/Exec doesn't work against ClickHouse —
// it errors with "Unexpected packet Query received from client".
//
// Tenant identifiers (workspace_id, project_id, environment_id) are pulled
// from MySQL `log_drains` rows so the coordinator actually picks the data up.
// Generating logs for random workspaces produces nothing observable: the
// coordinator only queries (workspace, project, environment, source) tuples
// that have an enabled drain attached.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand/v2"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/unkeyed/unkey/pkg/batch"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
)

// Defaults are sized for "several million rows" end-to-end perf testing
// against the local k8s ClickHouse + logdrain stack. With 1 drain × 1 env
// × 2 deployments × 1_000_000 logs we land 2M runtime + 1M request rows
// (3M total), which is enough to (a) sustain logdrain past the warm-up
// window and (b) characterise per-group ceiling vs. per-replica
// concurrency without burning a full hour of generator time.
var (
	clickhouseURL = flag.String("clickhouse", "clickhouse://default:password@localhost:9000", "ClickHouse connection URL")
	mysqlDSN      = flag.String("mysql", "unkey:password@tcp(localhost:3306)/unkey?parseTime=true&interpolateParams=true", "MySQL DSN for log_drains lookup")
	logsPerDrain  = flag.Int("logs-per-drain", 1_000_000, "Runtime logs to generate per (drain, environment, deployment)")
	requestsPerDr = flag.Int("requests-per-drain", 500_000, "Request logs to generate per (drain, environment, deployment)")
	deployments   = flag.Int("deployments", 2, "Synthetic deployment IDs per drain (each gets its own log stream)")
	batchSize     = flag.Int("batch-size", 10_000, "Records per ClickHouse flush")
	consumers     = flag.Int("consumers", 4, "Concurrent flush workers per buffer (drives generator-side throughput)")
)

type drainTarget struct {
	drainID      string
	workspaceID  string
	projectID    string // empty for workspace-scoped drains
	environments []string
	sources      []string // "runtime", "request"
}

func main() {
	flag.Parse()

	mysqlDB, err := sql.Open("mysql", *mysqlDSN)
	if err != nil {
		log.Fatalf("open mysql: %v", err)
	}
	defer mysqlDB.Close()

	targets, err := loadDrainTargets(context.Background(), mysqlDB)
	if err != nil {
		log.Fatalf("load drain targets: %v", err)
	}
	if len(targets) == 0 {
		log.Fatal("no enabled log drains in MySQL — create a drain in the dashboard first, otherwise the coordinator has nothing to forward")
	}

	log.Printf("🚀 logdrain data generation starting")
	log.Printf("   ClickHouse:       %s", *clickhouseURL)
	log.Printf("   Drains:           %d", len(targets))
	log.Printf("   Logs/drain/env:   runtime=%d  request=%d", *logsPerDrain, *requestsPerDr)
	log.Printf("   Deployments/drain: %d", *deployments)

	ch, err := clickhouse.New(clickhouse.Config{URL: *clickhouseURL})
	if err != nil {
		log.Fatalf("connect ClickHouse: %v", err)
	}

	// BufferSize is sized as `BatchSize * 4 * Consumers` so each consumer
	// has roughly one batch's worth of slack on its in-channel — anything
	// smaller and the producer goroutines (one per drainTarget) block on
	// every send once the consumers fall behind on a network burp, which
	// distorts the steady-state generator throughput numbers.
	//nolint:exhaustruct // Drop, OnFlushError use defaults; we want blocking sends and the package-level error logger.
	runtimeBuf := clickhouse.NewBuffer[schema.RuntimeLog](
		ch,
		"default.runtime_logs_raw_v1",
		clickhouse.BufferConfig{
			Name:          "perf-runtime-logs",
			BatchSize:     *batchSize,
			BufferSize:    *batchSize * 4 * *consumers,
			FlushInterval: 5 * time.Second,
			Consumers:     *consumers,
		},
	)
	//nolint:exhaustruct // see above
	requestBuf := clickhouse.NewBuffer[schema.SentinelRequest](
		ch,
		"default.sentinel_requests_raw_v1",
		clickhouse.BufferConfig{
			Name:          "perf-sentinel-requests",
			BatchSize:     *batchSize,
			BufferSize:    *batchSize * 4 * *consumers,
			FlushInterval: 5 * time.Second,
			Consumers:     *consumers,
		},
	)

	start := time.Now()
	totalRuntime, totalRequest := 0, 0

	for _, t := range targets {
		if hasSource(t.sources, "runtime") {
			totalRuntime += generateRuntime(runtimeBuf, t)
		}
		if hasSource(t.sources, "request") {
			totalRequest += generateRequest(requestBuf, t)
		}
	}

	log.Printf("📤 flushing buffers...")
	runtimeBuf.Close()
	requestBuf.Close()

	if err := ch.Close(); err != nil {
		log.Printf("⚠️  clickhouse close: %v", err)
	}

	dur := time.Since(start)
	total := totalRuntime + totalRequest
	log.Printf("✅ done: %d records (%d runtime, %d request) in %s (%.0f rec/s)",
		total, totalRuntime, totalRequest, dur, float64(total)/dur.Seconds())
}

func loadDrainTargets(ctx context.Context, db *sql.DB) ([]drainTarget, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, workspace_id, COALESCE(project_id, ''), environments, sources
		FROM log_drains
		WHERE deleted_at IS NULL AND enabled = true
	`)
	if err != nil {
		return nil, fmt.Errorf("query log_drains: %w", err)
	}
	defer rows.Close()

	var targets []drainTarget
	for rows.Next() {
		var (
			t            drainTarget
			environments []byte
			sources      []byte
		)
		if err := rows.Scan(&t.drainID, &t.workspaceID, &t.projectID, &environments, &sources); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		if err := json.Unmarshal(environments, &t.environments); err != nil {
			return nil, fmt.Errorf("decode environments for drain %s: %w", t.drainID, err)
		}
		if err := json.Unmarshal(sources, &t.sources); err != nil {
			return nil, fmt.Errorf("decode sources for drain %s: %w", t.drainID, err)
		}
		targets = append(targets, t)
	}
	return targets, rows.Err()
}

func hasSource(sources []string, want string) bool {
	for _, s := range sources {
		if s == want {
			return true
		}
	}
	return false
}

func projectFor(t drainTarget) string {
	if t.projectID != "" {
		return t.projectID
	}
	// Workspace-scoped drain: synthesize a stable project ID so the
	// coordinator's group key still works.
	return "proj_" + strings.TrimPrefix(t.drainID, "ld_")
}

func generateRuntime(buf *batch.BatchProcessor[schema.RuntimeLog], t drainTarget) int {
	now := time.Now().UnixMilli()
	expiresAt := time.Now().Add(90 * 24 * time.Hour)
	count := 0
	for _, env := range t.environments {
		for d := 0; d < *deployments; d++ {
			deploymentID := fmt.Sprintf("dep_%s_%s_%d", t.drainID, env, d)
			for l := 0; l < *logsPerDrain; l++ {
				buf.Buffer(schema.RuntimeLog{
					Time:       now + int64(count*100),
					InsertedAt: now + int64(count*100) + int64(rand.IntN(1000)),
					// Mint a log_id in Vector's exact shape ("log_" + 16
					// lowercase hex chars from 8 random bytes) so the
					// coordinator's cursor tiebreaker has the same
					// uniqueness profile as production.
					LogID:         fmt.Sprintf("log_%016x", rand.Uint64()),
					Severity:      randomSeverity(),
					Message:       fmt.Sprintf("Runtime log %d from %s", l, deploymentID),
					WorkspaceID:   t.workspaceID,
					ProjectID:     projectFor(t),
					EnvironmentID: env,
					AppID:         "app_" + t.drainID,
					DeploymentID:  deploymentID,
					K8sPodName:    fmt.Sprintf("pod-%s-%d", deploymentID, rand.IntN(3)),
					Region:        "us-east-1",
					Platform:      "kubernetes",
					Attributes:    randomAttributes(count),
					ExpiresAt:     expiresAt,
				})
				count++
			}
		}
	}
	log.Printf("   runtime  drain=%s ws=%s envs=%v: %d records", t.drainID, t.workspaceID, t.environments, count)
	return count
}

func generateRequest(buf *batch.BatchProcessor[schema.SentinelRequest], t drainTarget) int {
	now := time.Now().UnixMilli()
	count := 0
	for _, env := range t.environments {
		for d := 0; d < *deployments; d++ {
			deploymentID := fmt.Sprintf("dep_%s_%s_%d", t.drainID, env, d)
			for l := 0; l < *requestsPerDr; l++ {
				buf.Buffer(schema.SentinelRequest{
					RequestID:       fmt.Sprintf("req_%s_%d_%d", t.drainID, count, time.Now().UnixNano()),
					Time:            now + int64(count*200),
					WorkspaceID:     t.workspaceID,
					ProjectID:       projectFor(t),
					EnvironmentID:   env,
					SentinelID:      fmt.Sprintf("sentinel_%d", rand.IntN(5)),
					DeploymentID:    deploymentID,
					InstanceID:      fmt.Sprintf("instance_%d", rand.IntN(10)),
					InstanceAddress: fmt.Sprintf("10.0.%d.%d", rand.IntN(255), rand.IntN(255)),
					Region:          "us-east-1",
					Platform:        "kubernetes",
					Method:          randomHTTPMethod(),
					Host:            "api.test.unkey.dev",
					Path:            randomAPIPath(),
					QueryString:     "limit=100&offset=0",
					QueryParams:     map[string][]string{"limit": {"100"}, "offset": {"0"}},
					RequestHeaders:  []string{"Content-Type: application/json", "Authorization: Bearer token"},
					RequestBody:     `{"test": "data"}`,
					ResponseStatus:  randomHTTPStatus(),
					ResponseHeaders: []string{"Content-Type: application/json"},
					ResponseBody:    `{"status": "ok"}`,
					UserAgent:       "unkey-test-client/1.0",
					IPAddress:       fmt.Sprintf("%d.%d.%d.%d", rand.IntN(255), rand.IntN(255), rand.IntN(255), rand.IntN(255)),
					TotalLatency:    int64(rand.IntN(1000)),
					InstanceLatency: int64(rand.IntN(500)),
					SentinelLatency: int64(rand.IntN(50)),
				})
				count++
			}
		}
	}
	log.Printf("   request  drain=%s ws=%s envs=%v: %d records", t.drainID, t.workspaceID, t.environments, count)
	return count
}

func randomSeverity() string {
	severities := []string{"debug", "info", "warn", "error"}
	return severities[rand.IntN(len(severities))]
}

func randomHTTPMethod() string {
	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}
	return methods[rand.IntN(len(methods))]
}

func randomAPIPath() string {
	paths := []string{"/v1/keys/verify", "/v1/apis", "/v1/ratelimits", "/v1/workspaces", "/health"}
	return paths[rand.IntN(len(paths))]
}

func randomHTTPStatus() int32 {
	statuses := []int32{200, 201, 400, 401, 403, 404, 429, 500}
	return statuses[rand.IntN(len(statuses))]
}

func randomAttributes(id int) string {
	attrs := map[string]any{
		"record_id":    id,
		"test_data":    true,
		"timestamp":    time.Now().Unix(),
		"random_value": rand.IntN(1000),
	}
	data, _ := json.Marshal(attrs)
	return string(data)
}
