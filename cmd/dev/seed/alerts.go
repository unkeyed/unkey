package seed

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"slices"
	"time"

	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/db"
	dbtype "github.com/unkeyed/unkey/pkg/db/types"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/worker/cron/deployanomaly"
)

const (
	alertBucketSize = 5 * time.Minute
	alertBaseline   = 24 * time.Hour
	alertHistory    = 7 * 24 * time.Hour
)

type seedAlertMetric string

type seedAlertStatus string

const (
	seedMetricError5xx          seedAlertMetric = "error_5xx"
	seedMetricError4xx          seedAlertMetric = "error_4xx"
	seedMetricRequests          seedAlertMetric = "requests"
	seedMetricRequestsDrop      seedAlertMetric = "requests_drop"
	seedMetricEgressBytes       seedAlertMetric = "egress_bytes"
	seedMetricCPUSeconds        seedAlertMetric = "cpu_seconds"
	seedMetricMemoryUtilization seedAlertMetric = "memory_utilization"
	seedMetricOOMKilled         seedAlertMetric = "oom_killed"
	seedMetricCrashLoop         seedAlertMetric = "crash_loop"
	seedAlertStatusOpen         seedAlertStatus = "open"
	seedAlertStatusResolved     seedAlertStatus = "resolved"
)

type alertSeedTarget struct {
	workspaceID   string
	projectID     string
	appID         string
	environmentID string
	deploymentID  string
}

type alertSeedDefinition struct {
	metric            seedAlertMetric
	windowStart       time.Time
	status            seedAlertStatus
	resolvedAfter     time.Duration
	resolutionMessage string
}

type alertSeedRow struct {
	id                string
	metric            seedAlertMetric
	status            seedAlertStatus
	firedAt           int64
	lastSeenAt        int64
	resolvedAt        sql.NullInt64
	resolutionMessage sql.NullString
	observedValue     float64
	baselineMean      float64
	baselineStddev    float64
	thresholdSigma    float64
	windowStart       int64
	windowEnd         int64
}

type alertSeriesPoint struct {
	time   int64
	value  float64
	weight float64
}

type alertSeedDataset struct {
	alerts      []alertSeedRow
	requests    []schema.FrontlineRequest
	checkpoints []schema.InstanceCheckpoint
	events      []schema.InstanceEventV1
}

var alertsCmd = &cli.Command{
	Name:        "alerts",
	Usage:       "Seed Deploy anomaly alerts and their chart data",
	Description: "Creates one alert for every supported metric with matching seven-day ClickHouse chart data.",
	Flags: []cli.Flag{
		cli.String("workspace", "Workspace ID to seed alerts for", cli.Default("ws_local")),
		cli.String("clickhouse-url", "ClickHouse URL", cli.Default("clickhouse://default:password@127.0.0.1:9000")),
		cli.String("database-primary", "MySQL database DSN", cli.Default("unkey:password@tcp(127.0.0.1:3306)/unkey?parseTime=true&interpolateParams=true"), cli.EnvVar("UNKEY_DATABASE_PRIMARY")),
	},
	Action: seedAlerts,
}

func seedAlerts(ctx context.Context, cmd *cli.Command) error {
	database, err := db.New(db.Config{
		PrimaryDSN:  cmd.RequireString("database-primary"),
		ReadOnlyDSN: "",
		Tags:        sqlcomment.Disabled(),
	})
	if err != nil {
		return fault.Wrap(err, fault.Internal("failed to connect to MySQL"))
	}

	target, err := findAlertSeedTarget(ctx, database, cmd.RequireString("workspace"))
	if err != nil {
		return err
	}
	now := time.Now()
	if err := alignAlertSeedAppLifetime(ctx, database, target, alertSeedHistoryStart(now)); err != nil {
		return fault.Wrap(err, fault.Internal("failed to align the demo app lifetime"))
	}

	ch, err := clickhouse.New(clickhouse.Config{URL: cmd.RequireString("clickhouse-url")})
	if err != nil {
		return fault.Wrap(err, fault.Internal("failed to connect to ClickHouse"))
	}

	dataset := generateAlertSeedDataset(target, now)
	markerCount, err := insertAlertSeedDeployments(ctx, database, target, now)
	if err != nil {
		return fault.Wrap(err, fault.Internal("failed to insert deployment markers"))
	}
	if err := insertClickHouseRows(ctx, ch, dataset.requests); err != nil {
		return fault.Wrap(err, fault.Internal("failed to insert frontline requests"))
	}
	if err := insertClickHouseRows(ctx, ch, dataset.checkpoints); err != nil {
		return fault.Wrap(err, fault.Internal("failed to insert instance checkpoints"))
	}
	if err := insertClickHouseRows(ctx, ch, dataset.events); err != nil {
		return fault.Wrap(err, fault.Internal("failed to insert instance events"))
	}
	if err := insertAlertSeedRows(ctx, database, target, dataset.alerts); err != nil {
		return fault.Wrap(err, fault.Internal("failed to insert alert events"))
	}

	logger.Info("seeded Deploy anomaly alerts",
		"workspace_id", target.workspaceID,
		"app_id", target.appID,
		"environment_id", target.environmentID,
		"deployment_id", target.deploymentID,
		"deployment_markers", markerCount,
		"alerts", len(dataset.alerts),
		"frontline_requests", len(dataset.requests),
		"instance_checkpoints", len(dataset.checkpoints),
		"instance_events", len(dataset.events),
	)
	return nil
}

func alertSeedHistoryStart(now time.Time) time.Time {
	return now.Truncate(alertBucketSize).Add(-alertBucketSize).Add(-alertHistory)
}

func alignAlertSeedAppLifetime(
	ctx context.Context,
	database db.Database,
	target alertSeedTarget,
	createdAt time.Time,
) error {
	_, err := database.RW().ExecContext(ctx, `
		UPDATE apps
		SET created_at = LEAST(created_at, ?)
		WHERE workspace_id = ? AND id = ?`, createdAt.UnixMilli(), target.workspaceID, target.appID)
	return err
}

func findAlertSeedTarget(ctx context.Context, database db.Database, workspaceID string) (alertSeedTarget, error) {
	target := alertSeedTarget{
		workspaceID:   workspaceID,
		projectID:     "",
		appID:         "",
		environmentID: "",
		deploymentID:  "",
	}
	err := database.RO().QueryRowContext(ctx, `
		SELECT a.project_id, a.id, e.id
		FROM apps AS a
		JOIN environments AS e ON e.app_id = a.id
		WHERE a.workspace_id = ? AND e.kind = 'production'
		ORDER BY a.created_at ASC, e.created_at ASC
		LIMIT 1`, workspaceID).Scan(&target.projectID, &target.appID, &target.environmentID)
	if errors.Is(err, sql.ErrNoRows) {
		return alertSeedTarget{}, fault.New(fmt.Sprintf("workspace %s has no app with a production environment; run 'unkey dev seed local' first", workspaceID))
	}
	if err != nil {
		return alertSeedTarget{}, fault.Wrap(err, fault.Internal("failed to find a production app"))
	}

	err = database.RO().QueryRowContext(ctx, `
		SELECT id
		FROM deployments
		WHERE workspace_id = ? AND app_id = ? AND environment_id = ?
		ORDER BY created_at DESC
		LIMIT 1`, target.workspaceID, target.appID, target.environmentID).Scan(&target.deploymentID)
	if err == nil {
		return target, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return alertSeedTarget{}, fault.Wrap(err, fault.Internal("failed to find a deployment"))
	}

	target.deploymentID = uid.New(uid.DeploymentPrefix)
	createdAt := time.Now().Add(-10 * time.Hour)
	err = insertAlertSeedDeployment(
		ctx,
		database,
		target,
		target.deploymentID,
		createdAt,
		"7ac42be",
		"Improve request processing",
	)
	if err != nil {
		return alertSeedTarget{}, fault.Wrap(err, fault.Internal("failed to create a demo deployment"))
	}

	logger.Info("created demo deployment because the production environment had none", "deployment_id", target.deploymentID)
	return target, nil
}

func insertAlertSeedDeployments(
	ctx context.Context,
	database db.Database,
	target alertSeedTarget,
	now time.Time,
) (int, error) {
	markers := []struct {
		age     time.Duration
		gitSHA  string
		message string
	}{
		{age: 5*24*time.Hour + 3*time.Hour, gitSHA: "12ca09f", message: "Tune request concurrency"},
		{age: 2*24*time.Hour + 6*time.Hour, gitSHA: "b18fd72", message: "Reduce image startup time"},
	}
	for _, marker := range markers {
		if err := insertAlertSeedDeployment(
			ctx,
			database,
			target,
			uid.New(uid.DeploymentPrefix),
			now.Add(-marker.age),
			marker.gitSHA,
			marker.message,
		); err != nil {
			return 0, err
		}
	}
	return len(markers), nil
}

func insertAlertSeedDeployment(
	ctx context.Context,
	database db.Database,
	target alertSeedTarget,
	deploymentID string,
	createdAt time.Time,
	gitSHA string,
	commitMessage string,
) error {
	createdAtMs := createdAt.UnixMilli()
	return db.Query.InsertDeployment(ctx, database.RW(), db.InsertDeploymentParams{
		ID:                            deploymentID,
		K8sName:                       uid.DNS1035(12),
		WorkspaceID:                   target.workspaceID,
		ProjectID:                     target.projectID,
		AppID:                         target.appID,
		EnvironmentID:                 target.environmentID,
		GitCommitSha:                  sql.NullString{String: gitSHA, Valid: true},
		GitBranch:                     sql.NullString{String: "main", Valid: true},
		SentinelConfig:                []byte("{}"),
		GitCommitMessage:              sql.NullString{String: commitMessage, Valid: true},
		GitCommitAuthorHandle:         sql.NullString{String: "local", Valid: true},
		GitCommitAuthorAvatarUrl:      sql.NullString{},
		GitCommitTimestamp:            sql.NullInt64{Int64: createdAtMs, Valid: true},
		EncryptedEnvironmentVariables: []byte("{}"),
		Command:                       dbtype.StringSlice{},
		Status:                        mysqltype.DeploymentsStatusReady,
		CpuMillicores:                 500,
		MemoryMib:                     1024,
		StorageMib:                    0,
		Port:                          8080,
		ShutdownSignal:                db.DeploymentsShutdownSignalSIGTERM,
		UpstreamProtocol:              db.DeploymentsUpstreamProtocolHttp1,
		Healthcheck:                   dbtype.NullHealthcheck{Healthcheck: nil, Valid: false},
		PrNumber:                      sql.NullInt64{},
		ForkRepositoryFullName:        sql.NullString{},
		DeploymentTrigger:             db.DeploymentsTriggerUnkey,
		TriggeredBy:                   sql.NullString{String: "system", Valid: true},
		TriggerReason:                 sql.NullString{String: "Create local anomaly alert demo data", Valid: true},
		CreatedAt:                     createdAtMs,
		UpdatedAt:                     sql.NullInt64{Int64: createdAtMs, Valid: true},
	})
}

func generateAlertSeedDataset(target alertSeedTarget, now time.Time) alertSeedDataset {
	endBucket := now.Truncate(alertBucketSize).Add(-alertBucketSize)
	definitions := []alertSeedDefinition{
		{
			metric:            seedMetricError5xx,
			windowStart:       endBucket.Add(-15 * time.Minute),
			status:            seedAlertStatusOpen,
			resolvedAfter:     0,
			resolutionMessage: "",
		},
		{
			metric:            seedMetricEgressBytes,
			windowStart:       endBucket.Add(-30 * time.Minute),
			status:            seedAlertStatusOpen,
			resolvedAfter:     0,
			resolutionMessage: "",
		},
		{
			metric:            seedMetricMemoryUtilization,
			windowStart:       endBucket.Add(-45 * time.Minute),
			status:            seedAlertStatusOpen,
			resolvedAfter:     0,
			resolutionMessage: "",
		},
		{
			metric:            seedMetricOOMKilled,
			windowStart:       endBucket.Add(-60 * time.Minute),
			status:            seedAlertStatusOpen,
			resolvedAfter:     0,
			resolutionMessage: "",
		},
		{
			metric:            seedMetricRequestsDrop,
			windowStart:       endBucket.Add(-10 * time.Hour),
			status:            seedAlertStatusOpen,
			resolvedAfter:     0,
			resolutionMessage: "",
		},
		{
			metric:            seedMetricError4xx,
			windowStart:       endBucket.Add(-3 * time.Hour),
			status:            seedAlertStatusResolved,
			resolvedAfter:     20 * time.Minute,
			resolutionMessage: "Baseline adapted after 24 hours",
		},
		{
			metric:            seedMetricRequests,
			windowStart:       endBucket.Add(-5 * time.Hour),
			status:            seedAlertStatusResolved,
			resolvedAfter:     25 * time.Minute,
			resolutionMessage: "Metric returned to baseline for 3 consecutive windows",
		},
		{
			metric:            seedMetricCPUSeconds,
			windowStart:       endBucket.Add(-7 * time.Hour),
			status:            seedAlertStatusResolved,
			resolvedAfter:     30 * time.Minute,
			resolutionMessage: "Deployment stopped",
		},
		{
			metric:            seedMetricCrashLoop,
			windowStart:       endBucket.Add(-9 * time.Hour),
			status:            seedAlertStatusResolved,
			resolvedAfter:     15 * time.Minute,
			resolutionMessage: "Metric returned to baseline for 3 consecutive windows",
		},
	}

	series := make(map[seedAlertMetric][]alertSeriesPoint, len(definitions))
	anomalyWindows := make(map[seedAlertMetric]int64, len(definitions))
	for _, definition := range definitions {
		anomalyWindows[definition.metric] = definition.windowStart.UnixMilli()
	}
	historyStart := alertSeedHistoryStart(now)
	historyBuckets := int(endBucket.Sub(historyStart)/alertBucketSize) + 1

	dataset := alertSeedDataset{
		alerts:      make([]alertSeedRow, 0, len(definitions)),
		requests:    make([]schema.FrontlineRequest, 0, historyBuckets*14),
		checkpoints: make([]schema.InstanceCheckpoint, 0, historyBuckets*5),
		events:      make([]schema.InstanceEventV1, 0, 8),
	}
	containerUID := uid.New(uid.InstancePrefix) + "/0"
	podUID := uid.New("pod")
	var cpuCounter, egressCounter int64

	for bucketIndex, bucket := 0, historyStart; !bucket.After(endBucket); bucketIndex, bucket = bucketIndex+1, bucket.Add(alertBucketSize) {
		bucketMs := bucket.UnixMilli()
		count2xx := int64(12)
		var count4xx, count5xx int64
		if bucketIndex%10 == 0 {
			count4xx = 1
		}
		if bucketIndex%20 == 0 {
			count5xx = 1
		}
		requestDropWindow := anomalyWindows[seedMetricRequestsDrop]
		if bucketMs >= requestDropWindow-int64(12*alertBucketSize/time.Millisecond) && bucketMs < requestDropWindow {
			count2xx = 240
		}
		if anomalyWindows[seedMetricRequests] == bucketMs {
			count2xx, count4xx, count5xx = 600, 1, 1
		}
		if anomalyWindows[seedMetricError4xx] == bucketMs {
			count2xx, count4xx, count5xx = 114, 25, 0
		}
		if anomalyWindows[seedMetricError5xx] == bucketMs {
			count2xx, count4xx, count5xx = 91, 0, 20
		}
		if anomalyWindows[seedMetricRequestsDrop] == bucketMs {
			count2xx, count4xx, count5xx = 1, 0, 0
		}
		dataset.requests = appendFrontlineSeedRequests(dataset.requests, target, bucket, 200, count2xx)
		dataset.requests = appendFrontlineSeedRequests(dataset.requests, target, bucket, 404, count4xx)
		dataset.requests = appendFrontlineSeedRequests(dataset.requests, target, bucket, 500, count5xx)
		requestCount := count2xx + count4xx + count5xx
		series[seedMetricError5xx] = append(series[seedMetricError5xx], alertSeriesPoint{time: bucketMs, value: ratio(count5xx, requestCount), weight: float64(requestCount)})
		series[seedMetricError4xx] = append(series[seedMetricError4xx], alertSeriesPoint{time: bucketMs, value: ratio(count4xx, requestCount), weight: float64(requestCount)})
		series[seedMetricRequests] = append(series[seedMetricRequests], alertSeriesPoint{time: bucketMs, value: float64(requestCount), weight: 0})
		series[seedMetricRequestsDrop] = append(series[seedMetricRequestsDrop], alertSeriesPoint{time: bucketMs, value: float64(requestCount), weight: 0})

		egressBytes := int64(400_000 + (bucketIndex%5)*25_000)
		cpuUsec := int64(12_000_000 + (bucketIndex%5)*500_000)
		memoryRatio := 0.35 + float64(bucketIndex%5)*0.02
		if anomalyWindows[seedMetricEgressBytes] == bucketMs {
			egressBytes = 8 * 1024 * 1024
		}
		if anomalyWindows[seedMetricCPUSeconds] == bucketMs {
			cpuUsec = 65_000_000
		}
		if anomalyWindows[seedMetricMemoryUtilization] == bucketMs {
			memoryRatio = 0.94
		}
		const memoryAllocatedBytes = int64(1024 * 1024 * 1024)
		memoryBytes := int64(math.Round(memoryRatio * float64(memoryAllocatedBytes)))
		memoryRatio = float64(memoryBytes) / float64(memoryAllocatedBytes)
		series[seedMetricEgressBytes] = append(series[seedMetricEgressBytes], alertSeriesPoint{time: bucketMs, value: float64(egressBytes), weight: 0})
		series[seedMetricCPUSeconds] = append(series[seedMetricCPUSeconds], alertSeriesPoint{time: bucketMs, value: float64(cpuUsec) / 1_000_000, weight: 0})
		series[seedMetricMemoryUtilization] = append(series[seedMetricMemoryUtilization], alertSeriesPoint{time: bucketMs, value: memoryRatio, weight: 0})

		for sample := 0; sample < 5; sample++ {
			if sample == 1 {
				egressCounter += egressBytes
				cpuCounter += cpuUsec
			}
			dataset.checkpoints = append(dataset.checkpoints, schema.InstanceCheckpoint{
				NodeID:                     "seed-alerts-node",
				WorkspaceID:                target.workspaceID,
				ProjectID:                  target.projectID,
				AppID:                      target.appID,
				EnvironmentID:              target.environmentID,
				ResourceType:               "deployment",
				ResourceID:                 target.deploymentID,
				PodUID:                     podUID,
				InstanceID:                 "alerts-demo-instance",
				ContainerUID:               containerUID,
				RestartCount:               0,
				Ts:                         bucket.Add(time.Duration(sample) * time.Minute).UnixMilli(),
				EventKind:                  "checkpoint",
				CPUUsageUsec:               cpuCounter,
				MemoryBytes:                memoryBytes,
				CPUAllocatedMillicores:     500,
				MemoryAllocatedBytes:       memoryAllocatedBytes,
				DiskAllocatedBytes:         0,
				DiskUsedBytes:              0,
				NetworkEgressPublicBytes:   egressCounter,
				NetworkEgressPrivateBytes:  0,
				NetworkIngressPublicBytes:  0,
				NetworkIngressPrivateBytes: 0,
				Region:                     "local",
				Platform:                   "dev",
				Attributes: schema.InstanceCheckpointAttributes{
					Image:              "",
					ImageID:            "",
					QOSClass:           "",
					EBPFProgramVersion: "",
					EBPFPinDir:         "",
					NetworkAttached:    true,
					Collectors:         nil,
				}.Marshal(),
			})
		}

		oomCount := 0
		if anomalyWindows[seedMetricOOMKilled] == bucketMs {
			oomCount = 3
			dataset.events = appendInstanceSeedEvents(dataset.events, target, bucket, "terminated", "OOMKilled", oomCount)
		}
		crashCount := 0
		if anomalyWindows[seedMetricCrashLoop] == bucketMs {
			crashCount = 4
			dataset.events = appendInstanceSeedEvents(dataset.events, target, bucket, "waiting", "CrashLoopBackOff", crashCount)
		}
		series[seedMetricOOMKilled] = append(series[seedMetricOOMKilled], alertSeriesPoint{time: bucketMs, value: float64(oomCount), weight: 0})
		series[seedMetricCrashLoop] = append(series[seedMetricCrashLoop], alertSeriesPoint{time: bucketMs, value: float64(crashCount), weight: 0})
	}

	config := deployanomaly.DefaultConfig(deployanomaly.SensitivityNormal)
	for _, definition := range definitions {
		observed, mean, stddev := alertStatistics(series[definition.metric], definition.windowStart.UnixMilli())
		thresholdSigma := config.SigmaK
		switch definition.metric {
		case seedMetricMemoryUtilization, seedMetricOOMKilled, seedMetricCrashLoop:
			mean = 0
			stddev = 0
			thresholdSigma = 0
		case seedMetricRequestsDrop:
			mean = alertRecentMedian(series[definition.metric], definition.windowStart.UnixMilli(), 12)
			stddev = 0
			thresholdSigma = 0
		case seedMetricError5xx:
			mean = alertWeightedMean(series[definition.metric], definition.windowStart.UnixMilli())
			stddev = max(stddev, config.MinimumStddevRatio*mean, config.StddevFloors.Error5xxRatio)
		case seedMetricError4xx:
			mean = alertWeightedMean(series[definition.metric], definition.windowStart.UnixMilli())
			stddev = max(stddev, config.MinimumStddevRatio*mean, config.StddevFloors.Error4xxRatio)
		case seedMetricRequests:
			stddev = max(stddev, config.MinimumStddevRatio*mean, config.StddevFloors.Requests)
		case seedMetricEgressBytes:
			stddev = max(stddev, config.MinimumStddevRatio*mean, config.StddevFloors.EgressBytes)
		case seedMetricCPUSeconds:
			stddev = max(stddev, config.MinimumStddevRatio*mean, config.StddevFloors.CPUSeconds)
		}
		windowEnd := definition.windowStart.Add(alertBucketSize)
		firedAt := windowEnd.Add(time.Minute)
		row := alertSeedRow{
			id:                uid.New(uid.AlertPrefix),
			metric:            definition.metric,
			status:            definition.status,
			firedAt:           firedAt.UnixMilli(),
			lastSeenAt:        windowEnd.UnixMilli(),
			resolvedAt:        sql.NullInt64{},
			resolutionMessage: sql.NullString{},
			observedValue:     observed,
			baselineMean:      mean,
			baselineStddev:    stddev,
			thresholdSigma:    thresholdSigma,
			windowStart:       definition.windowStart.UnixMilli(),
			windowEnd:         windowEnd.UnixMilli(),
		}
		if definition.status == seedAlertStatusResolved {
			row.resolvedAt = sql.NullInt64{Int64: firedAt.Add(definition.resolvedAfter).UnixMilli(), Valid: true}
			row.resolutionMessage = sql.NullString{String: definition.resolutionMessage, Valid: true}
		}
		dataset.alerts = append(dataset.alerts, row)
	}

	return dataset
}

func appendFrontlineSeedRequests(rows []schema.FrontlineRequest, target alertSeedTarget, bucket time.Time, status int32, count int64) []schema.FrontlineRequest {
	for index := int64(0); index < count; index++ {
		rows = append(rows, schema.FrontlineRequest{
			RequestID:       fmt.Sprintf("req_alert_%d_%d_%d", bucket.UnixMilli(), status, index),
			Time:            bucket.Add(time.Duration(index%5) * time.Second).UnixMilli(),
			WorkspaceID:     target.workspaceID,
			ProjectID:       target.projectID,
			AppID:           target.appID,
			EnvironmentID:   target.environmentID,
			FrontlineID:     "fl_alerts_demo",
			DeploymentID:    target.deploymentID,
			InstanceID:      "alerts-demo-instance",
			InstanceAddress: "10.0.0.10",
			Region:          "local",
			Platform:        "dev",
			Method:          "GET",
			Host:            "alerts.local",
			Path:            "/api/orders",
			QueryString:     "",
			QueryParams:     map[string][]string{},
			RequestHeaders:  []string{},
			RequestBody:     "",
			ResponseStatus:  status,
			ResponseHeaders: []string{},
			ResponseBody:    "",
			UserAgent:       "unkey-alert-seeder",
			IPAddress:       "127.0.0.1",
			TotalLatency:    25,
			InstanceLatency: 20,
			GatewayLatency:  5,
		})
	}
	return rows
}

func appendInstanceSeedEvents(rows []schema.InstanceEventV1, target alertSeedTarget, bucket time.Time, eventKind, reason string, count int) []schema.InstanceEventV1 {
	for index := range count {
		rows = append(rows, schema.InstanceEventV1{
			Time:             bucket.Add(time.Duration(index+1) * time.Minute).UnixMilli(),
			WorkspaceID:      target.workspaceID,
			ProjectID:        target.projectID,
			AppID:            target.appID,
			EnvironmentID:    target.environmentID,
			DeploymentID:     target.deploymentID,
			PodUID:           fmt.Sprintf("alerts-demo-pod-%d", index),
			PodName:          fmt.Sprintf("alerts-demo-%d", index),
			NodeName:         "seed-alerts-node",
			ContainerName:    "app",
			ContainerID:      fmt.Sprintf("containerd://alerts-demo-%d", index),
			RestartCount:     int32(index),
			EventKind:        eventKind,
			ExitCode:         137,
			Signal:           9,
			Reason:           reason,
			Message:          "Seeded anomaly event for the dashboard demo",
			Region:           "local",
			Platform:         "dev",
			EventFingerprint: fmt.Sprintf("alerts-%s-%d", reason, index),
			Attributes:       "{}",
		})
	}
	return rows
}

func alertStatistics(points []alertSeriesPoint, anomalyWindow int64) (observed, mean, stddev float64) {
	baselineStart := anomalyWindow - alertBaseline.Milliseconds()
	baseline := make([]float64, 0, int(alertBaseline/alertBucketSize))
	for _, point := range points {
		switch {
		case point.time == anomalyWindow:
			observed = point.value
		case point.time >= baselineStart && point.time < anomalyWindow:
			baseline = append(baseline, point.value)
		}
	}
	for _, value := range baseline {
		mean += value
	}
	mean /= float64(len(baseline))
	for _, value := range baseline {
		stddev += math.Pow(value-mean, 2)
	}
	stddev = math.Sqrt(stddev / float64(len(baseline)))
	return observed, mean, stddev
}

func alertWeightedMean(points []alertSeriesPoint, anomalyWindow int64) float64 {
	baselineStart := anomalyWindow - alertBaseline.Milliseconds()
	var weightedTotal, totalWeight float64
	for _, point := range points {
		if point.time < baselineStart || point.time >= anomalyWindow {
			continue
		}
		weightedTotal += point.value * point.weight
		totalWeight += point.weight
	}
	if totalWeight == 0 {
		return 0
	}
	return weightedTotal / totalWeight
}

func alertRecentMedian(points []alertSeriesPoint, anomalyWindow int64, bucketCount int) float64 {
	values := make([]float64, 0, bucketCount)
	for index := len(points) - 1; index >= 0 && len(values) < bucketCount; index-- {
		if points[index].time < anomalyWindow {
			values = append(values, points[index].value)
		}
	}
	if len(values) == 0 {
		return 0
	}
	slices.Sort(values)
	middle := len(values) / 2
	if len(values)%2 == 0 {
		return (values[middle-1] + values[middle]) / 2
	}
	return values[middle]
}

func ratio(numerator, denominator int64) float64 {
	if denominator == 0 {
		return 0
	}
	return float64(numerator) / float64(denominator)
}

func insertClickHouseRows[T schema.Row](ctx context.Context, ch *clickhouse.Client, rows []T) error {
	if len(rows) == 0 {
		return nil
	}
	batch, err := ch.Conn().PrepareBatch(ctx, clickhouse.InsertQuery[T]())
	if err != nil {
		return err
	}
	for index := range rows {
		if err := batch.AppendStruct(&rows[index]); err != nil {
			return err
		}
	}
	return batch.Send()
}

func insertAlertSeedRows(ctx context.Context, database db.Database, target alertSeedTarget, rows []alertSeedRow) error {
	return db.TxRetry(ctx, database.RW(), func(ctx context.Context, tx db.DBTX) error {
		for _, row := range rows {
			_, err := tx.ExecContext(ctx, `
				INSERT INTO alert_events (
					id, workspace_id, project_id, app_id, environment_id, deployment_id,
					metric, status, fired_at, last_seen_at, resolved_at,
					resolution_message, observed_value, baseline_mean, baseline_stddev,
					threshold_sigma, window_start, window_end, created_at, updated_at
				) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				row.id,
				target.workspaceID,
				target.projectID,
				target.appID,
				target.environmentID,
				target.deploymentID,
				row.metric,
				row.status,
				row.firedAt,
				row.lastSeenAt,
				row.resolvedAt,
				row.resolutionMessage,
				row.observedValue,
				row.baselineMean,
				row.baselineStddev,
				row.thresholdSigma,
				row.windowStart,
				row.windowEnd,
				row.firedAt,
				row.resolvedAt,
			)
			if err != nil {
				return err
			}
		}
		return nil
	})
}
