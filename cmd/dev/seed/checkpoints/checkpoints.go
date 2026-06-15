// Package checkpoints seeds heimdall instance_checkpoints data so the full
// Deploy billing pipeline (ClickHouse aggregation -> Stripe meter push, and the
// dashboard usage charts) can be tested without running real workloads.
//
// Heimdall bills from cumulative counters integrated over consecutive
// checkpoint pairs, dropping any pair more than ~2 minutes apart. A realistic
// month of usage is therefore a dense, contiguous counter series, not two rows
// a month apart, so this seeder emits one checkpoint per tick across the
// requested window. See pkg/clickhouse/instance_meter.go for the billing math.
package checkpoints

import (
	"context"
	"fmt"
	"time"

	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
)

// maxBillableGap mirrors maxSampleGapMillis in pkg/clickhouse: pairs spaced
// further apart than this are dropped by the billing query, so a tick at or
// above it would produce zero billable usage.
const maxBillableGap = 2 * time.Minute

// Cmd seeds a dense, billable instance_checkpoints series for Deploy billing tests.
var Cmd = &cli.Command{
	Name:        "checkpoints",
	Usage:       "Seed heimdall instance checkpoints for Deploy billing tests",
	Description: "Generates a dense, billable instance_checkpoints series for an existing workspace/project/app/environment so monthly Deploy billing can be exercised end to end.",
	Flags: []cli.Flag{
		cli.String("workspace", "Workspace ID to seed usage for (required; must already exist)", cli.Required()),
		cli.String("project", "Project ID or slug. If omitted, pick interactively from the workspace."),
		cli.String("app", "App ID or slug. If omitted, pick interactively from the project."),
		cli.String("environment", "Environment ID or slug. If omitted, pick interactively from the app."),
		cli.String("deployment", "Deployment ID to attribute usage to (resource_id). Defaults to the latest deployment in the environment, or a synthetic id if none exist."),

		cli.Float("vcpu", "Average vCPU cores used while running", cli.Default(0.5)),
		cli.String("memory", "Average working-set memory used while running (e.g. 512Mi, 1Gi)", cli.Default("512Mi")),
		cli.String("disk", "Allocated disk size (e.g. 1Gi). Empty means no disk usage.", cli.Default("")),
		cli.String("egress-per-day", "Public network egress per active day (e.g. 2Gi). Empty means none.", cli.Default("")),
		cli.Int("replicas", "Number of concurrent instances (separate container_uids)", cli.Default(1)),

		cli.Int("days", "Number of days of usage to generate, ending now", cli.Default(30)),
		cli.Float("hours-per-day", "Hours the instances run each day (24 = 24/7)", cli.Default(24.0)),
		cli.Duration("tick", "Interval between checkpoints. Must be < 2m to stay billable.", cli.Default(60*time.Second)),

		cli.String("region", "Region label written on each checkpoint", cli.Default("local")),
		cli.String("platform", "Platform label written on each checkpoint", cli.Default("dev")),

		cli.Int("batch-size", "Rows per ClickHouse insert", cli.Default(50_000)),
		cli.String("clickhouse-url", "ClickHouse URL", cli.Default("clickhouse://default:password@127.0.0.1:9000")),
		cli.String("database-primary", "MySQL database DSN", cli.Default("unkey:password@tcp(127.0.0.1:3306)/unkey?parseTime=true&interpolateParams=true"), cli.EnvVar("UNKEY_DATABASE_PRIMARY")),
	},
	Action: seed,
}

func seed(ctx context.Context, cmd *cli.Command) error {
	gen, err := parseFlags(cmd)
	if err != nil {
		return err
	}

	database, err := db.New(db.Config{PrimaryDSN: cmd.RequireString("database-primary"), ReadOnlyDSN: ""})
	if err != nil {
		return fmt.Errorf("failed to connect to MySQL: %w", err)
	}

	gen.target, err = resolveTarget(ctx, database, cmd)
	if err != nil {
		return err
	}
	// Allocated columns are observational (utilization%), not billed. Use the
	// deployment's declared allocation when known, else round usage up.
	gen.cpuAllocMilli = gen.target.cpuAllocMilli(gen.vcpu)
	gen.memAllocBytes = gen.target.memAllocBytes(gen.memoryBytes)

	ch, err := clickhouse.New(clickhouse.Config{URL: cmd.String("clickhouse-url")})
	if err != nil {
		return fmt.Errorf("failed to connect to ClickHouse: %w", err)
	}

	buf := clickhouse.NewBuffer[schema.InstanceCheckpoint](ch, "default.instance_checkpoints_v1", clickhouse.BufferConfig{
		Name:          "seed-instance-checkpoints",
		BatchSize:     cmd.RequireInt("batch-size"),
		BufferSize:    cmd.RequireInt("batch-size") * 2,
		FlushInterval: 5 * time.Second,
		Consumers:     4,
		Drop:          false,
		OnFlushError:  nil,
	})

	rows := gen.generate(buf)
	buf.Close()

	exp := gen.expected()
	logger.Info("seeded checkpoints",
		"rows", rows,
		"workspace_id", gen.target.workspaceID,
		"project_id", gen.target.projectID,
		"environment_id", gen.target.environmentID,
		"resource_id", gen.target.resourceID,
	)
	logger.Info("expected billable totals",
		"days", gen.days,
		"replicas", gen.replicas,
		"cpu_seconds", fmt.Sprintf("%.2f", exp.cpuSeconds),
		"memory_gib_seconds", fmt.Sprintf("%.2f", exp.memGiBSeconds),
		"disk_gib_seconds", fmt.Sprintf("%.2f", exp.diskGiBSeconds),
		"egress_public_gib", fmt.Sprintf("%.4f", exp.egressGiB),
	)

	return nil
}

// parseFlags validates the usage-shape flags and returns a generator with
// everything but the target populated.
func parseFlags(cmd *cli.Command) (generator, error) {
	tick := cmd.Duration("tick")
	if tick <= 0 {
		return generator{}, fmt.Errorf("--tick must be positive")
	}
	if tick >= maxBillableGap {
		return generator{}, fmt.Errorf("--tick (%s) must be below the %s billing gap or no usage is billable", tick, maxBillableGap)
	}

	hoursPerDay := cmd.Float("hours-per-day")
	if hoursPerDay <= 0 || hoursPerDay > 24 {
		return generator{}, fmt.Errorf("--hours-per-day must be in (0, 24], got %v", hoursPerDay)
	}
	days := cmd.Int("days")
	if days <= 0 {
		return generator{}, fmt.Errorf("--days must be positive")
	}
	replicas := cmd.Int("replicas")
	if replicas <= 0 {
		return generator{}, fmt.Errorf("--replicas must be positive")
	}
	vcpu := cmd.Float("vcpu")
	if vcpu < 0 {
		return generator{}, fmt.Errorf("--vcpu must be >= 0")
	}

	memoryBytes, err := parseBytes(cmd.String("memory"))
	if err != nil {
		return generator{}, fmt.Errorf("invalid --memory: %w", err)
	}
	diskBytes, err := parseBytes(cmd.String("disk"))
	if err != nil {
		return generator{}, fmt.Errorf("invalid --disk: %w", err)
	}
	egressPerDay, err := parseBytes(cmd.String("egress-per-day"))
	if err != nil {
		return generator{}, fmt.Errorf("invalid --egress-per-day: %w", err)
	}

	return generator{ //nolint:exhaustruct // target and allocation fields are filled by the caller.
		vcpu:         vcpu,
		memoryBytes:  memoryBytes,
		diskBytes:    diskBytes,
		egressPerDay: egressPerDay,
		replicas:     replicas,
		days:         days,
		hoursPerDay:  hoursPerDay,
		tick:         tick,
		end:          time.Now(),
		region:       cmd.String("region"),
		platform:     cmd.String("platform"),
	}, nil
}
