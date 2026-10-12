// Package workspaceusage seeds the data that workspace.getLimits and
// workspace.getUsage read, so the Limits and Usage settings pages show meters,
// breaches, and the edge cases of the usage rows. Run `unkey dev seed local`
// first. Every row carries a seed marker, so a second run replaces the rows of
// the first run
package workspaceusage

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"time"

	driver "github.com/ClickHouse/clickhouse-go/v2"
	keysdb "github.com/unkeyed/unkey/internal/services/keys/db"
	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/db"
	dbtype "github.com/unkeyed/unkey/pkg/db/types"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/uid"
)

const (
	marker          = "seed-workspace-usage"
	gib             = 1024 * 1024 * 1024
	unlimitedDomain = 1_000_000

	// No krane serves this region, so the seeded topologies reserve capacity
	// but never schedule pods
	seedRegionID = "rgn_seed_workspace_usage"

	// The production environment gets this share of the reserved capacity,
	// the preview environment gets the rest
	productionShare = 0.75
)

var Cmd = &cli.Command{
	Name:  "workspace-usage",
	Usage: "Seed limits usage and monthly usage for the Limits and Usage settings pages",
	Description: `Uses the default app and its production and preview environments from
'unkey dev seed local'. It seeds:

  MySQL       running deployments that reserve CPU, memory, and disk; custom
              domains; log drains; and optionally new custom domain and log
              drain limits
  ClickHouse  billable API operations, active gateway keys, and hourly compute
              usage for the current and previous month, plus rows of a deleted
              app and rows without an app id

--fill sets the used share of each limit. Use a value above 1 to see the
over-limit state.`,
	Flags: []cli.Flag{
		cli.String("workspace", "Workspace ID to seed", cli.Default("ws_local")),
		cli.String("project", "Project ID or slug that holds the default app", cli.Default("local-api")),
		cli.Float("fill", "Used share of each metered limit, for example 0.8 or 1.2", cli.Default(0.8)),
		cli.Int("custom-domains-max", "Custom domain limit to set before seeding, -1 keeps the current limit", cli.Default(5)),
		cli.Int("logdrains-max", "Log drain limit to set before seeding, -1 keeps the current limit", cli.Default(3)),
		cli.String("clickhouse-url", "ClickHouse URL", cli.Default("clickhouse://default:password@127.0.0.1:9000")),
		cli.String("database-primary", "MySQL database DSN", cli.Default("unkey:password@tcp(127.0.0.1:3306)/unkey?parseTime=true&interpolateParams=true"), cli.EnvVar("UNKEY_DATABASE_PRIMARY")),
	},
	Action: seed,
}

// target holds the ids of the resources that receive the seeded rows
type target struct {
	workspaceID  string
	projectID    string
	appID        string
	productionID string
	previewID    string
}

// usageRow is one hour of compute usage for one container
type usageRow struct {
	time           time.Time
	projectID      string
	appID          string
	environmentID  string
	resourceID     string
	cpuSeconds     float64
	memoryGiBHours float64
	diskGiBHours   float64
	egressBytes    int64
}

func seed(ctx context.Context, cmd *cli.Command) error {
	fill := cmd.Float("fill")
	if fill < 0 {
		return fmt.Errorf("--fill must not be negative, got %v", fill)
	}

	database, err := db.New(db.Config{
		PrimaryDSN:  cmd.RequireString("database-primary"),
		ReadOnlyDSN: "",
		Tags:        sqlcomment.Disabled(),
	})
	if err != nil {
		return fmt.Errorf("failed to connect to MySQL: %w", err)
	}
	defer func() { _ = database.Close() }()

	workspaceID := cmd.RequireString("workspace")
	t, err := findTarget(ctx, database, workspaceID, cmd.RequireString("project"))
	if err != nil {
		return err
	}

	if err := setLimits(ctx, database, workspaceID, cmd.Int("custom-domains-max"), cmd.Int("logdrains-max")); err != nil {
		return err
	}
	limits, err := keysdb.Query.FindLimitsByWorkspaceID(ctx, database.RO(), workspaceID)
	if err != nil {
		return fmt.Errorf("failed to read the limits of %s: %w", workspaceID, err)
	}

	now := time.Now().UTC()
	reserved := reservation{
		cpuMillicores: int64(fill * float64(limits.CpuCoresMax) * 1000),
		memoryMib:     int64(fill * float64(limits.MemoryMibMax)),
		storageMib:    int64(fill * float64(limits.StorageMibMax)),
	}
	customDomains := countFor(fill, limits.CustomDomainsMax)
	logdrains := countFor(fill, limits.LogdrainsMax)

	err = db.TxRetry(ctx, database.RW(), func(ctx context.Context, tx db.DBTX) error {
		if err := replaceDeployments(ctx, tx, t, reserved, now.UnixMilli()); err != nil {
			return err
		}
		if err := replaceCustomDomains(ctx, tx, t, customDomains, now.UnixMilli()); err != nil {
			return err
		}
		return replaceLogdrains(ctx, tx, workspaceID, logdrains, now.UnixMilli())
	})
	if err != nil {
		return err
	}

	ch, err := clickhouse.New(clickhouse.Config{URL: cmd.RequireString("clickhouse-url")})
	if err != nil {
		return fmt.Errorf("failed to connect to ClickHouse: %w", err)
	}
	defer func() { _ = ch.Close() }()

	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	lastMonth := monthStart.AddDate(0, -1, 0)
	billable := int64(fill * float64(limits.ApiBillableOperationsCountMaxPerMonth))
	if err := replaceBillable(ctx, ch, workspaceID, map[time.Time]int64{
		monthStart: billable,
		lastMonth:  billable * 6 / 10,
	}); err != nil {
		return err
	}
	if err := replaceActiveKeys(ctx, ch, t, monthStart, lastMonth); err != nil {
		return err
	}
	if err := replaceComputeUsage(ctx, ch, t, reserved, lastMonth, now); err != nil {
		return err
	}

	logger.Info("seeded workspace usage",
		"workspace_id", workspaceID,
		"fill", fill,
		"reserved_vcpus", float64(reserved.cpuMillicores)/1000,
		"reserved_memory_mib", reserved.memoryMib,
		"reserved_storage_mib", reserved.storageMib,
		"custom_domains", customDomains,
		"logdrains", logdrains,
		"billable_operations_this_month", billable,
	)
	return nil
}

// reservation is the capacity the running seed deployments reserve together
type reservation struct {
	cpuMillicores int64
	memoryMib     int64
	storageMib    int64
}

func findTarget(ctx context.Context, database db.Database, workspaceID, project string) (target, error) {
	p, err := db.Query.FindProjectByIdOrSlug(ctx, database.RO(), db.FindProjectByIdOrSlugParams{
		WorkspaceID: workspaceID,
		Project:     project,
	})
	if err != nil {
		return target{}, fmt.Errorf("failed to find project %s, run 'unkey dev seed local' first: %w", project, err)
	}
	app, err := db.Query.FindAppByProjectAndSlug(ctx, database.RO(), db.FindAppByProjectAndSlugParams{
		ProjectID: p.ID,
		Slug:      "default",
	})
	if err != nil {
		return target{}, fmt.Errorf("failed to find the default app of %s: %w", project, err)
	}
	ids := map[string]string{}
	for _, slug := range []string{"production", "preview"} {
		env, err := db.Query.FindEnvironmentByAppIdAndSlug(ctx, database.RO(), db.FindEnvironmentByAppIdAndSlugParams{
			AppID: app.ID,
			Slug:  slug,
		})
		if err != nil {
			return target{}, fmt.Errorf("failed to find the %s environment: %w", slug, err)
		}
		ids[slug] = env.ID
	}
	return target{
		workspaceID:  workspaceID,
		projectID:    p.ID,
		appID:        app.ID,
		productionID: ids["production"],
		previewID:    ids["preview"],
	}, nil
}

func setLimits(ctx context.Context, database db.Database, workspaceID string, customDomainsMax, logdrainsMax int) error {
	if customDomainsMax >= 0 {
		if _, err := database.RW().ExecContext(ctx, "UPDATE limits SET custom_domains_max = ? WHERE workspace_id = ?", customDomainsMax, workspaceID); err != nil {
			return fmt.Errorf("failed to set the custom domain limit: %w", err)
		}
	}
	if logdrainsMax >= 0 {
		if _, err := database.RW().ExecContext(ctx, "UPDATE limits SET logdrains_max = ? WHERE workspace_id = ?", logdrainsMax, workspaceID); err != nil {
			return fmt.Errorf("failed to set the log drain limit: %w", err)
		}
	}
	return nil
}

// countFor returns the number of rows that fill the share of max. An
// unlimited custom domain limit gets a few rows, so the page has rows to show
func countFor(fill float64, max uint32) int {
	if max >= unlimitedDomain {
		return 3
	}
	return int(math.Round(fill * float64(max)))
}

func replaceDeployments(ctx context.Context, tx db.DBTX, t target, reserved reservation, now int64) error {
	_, err := tx.ExecContext(ctx, `
		DELETE dt FROM deployment_topology dt
		JOIN deployments d ON d.id = dt.deployment_id
		WHERE d.workspace_id = ? AND d.k8s_name LIKE ?`, t.workspaceID, marker+"%")
	if err != nil {
		return fmt.Errorf("failed to clear seeded topologies: %w", err)
	}
	if _, err = tx.ExecContext(ctx, "DELETE FROM deployments WHERE workspace_id = ? AND k8s_name LIKE ?", t.workspaceID, marker+"%"); err != nil {
		return fmt.Errorf("failed to clear seeded deployments: %w", err)
	}

	for _, d := range []struct {
		environmentID string
		share         float64
	}{
		{environmentID: t.productionID, share: productionShare},
		{environmentID: t.previewID, share: 1 - productionShare},
	} {
		id := uid.New(uid.DeploymentPrefix)
		err = db.Query.InsertDeployment(ctx, tx, db.InsertDeploymentParams{
			ID:                            id,
			K8sName:                       marker + "-" + uid.DNS1035(),
			WorkspaceID:                   t.workspaceID,
			ProjectID:                     t.projectID,
			AppID:                         t.appID,
			EnvironmentID:                 d.environmentID,
			Source:                        db.DeploymentsSourceUnknown,
			ImageRequested:                sql.NullString{},
			GitCommitSha:                  sql.NullString{},
			GitBranch:                     sql.NullString{},
			SentinelConfig:                []byte("{}"),
			GitCommitMessage:              sql.NullString{},
			GitCommitAuthorHandle:         sql.NullString{},
			GitCommitAuthorAvatarUrl:      sql.NullString{},
			GitCommitTimestamp:            sql.NullInt64{},
			EncryptedEnvironmentVariables: []byte{},
			Command:                       dbtype.StringSlice{},
			Status:                        mysqltype.DeploymentsStatusReady,
			CpuMillicores:                 int32(float64(reserved.cpuMillicores) * d.share),
			MemoryMib:                     int32(float64(reserved.memoryMib) * d.share),
			StorageMib:                    uint32(float64(reserved.storageMib) * d.share),
			Port:                          8080,
			ShutdownSignal:                db.DeploymentsShutdownSignalSIGTERM,
			UpstreamProtocol:              db.DeploymentsUpstreamProtocolHttp1,
			Healthcheck:                   dbtype.NullHealthcheck{Healthcheck: nil, Valid: false},
			PrNumber:                      sql.NullInt64{},
			ForkRepositoryFullName:        sql.NullString{},
			DeploymentTrigger:             db.DeploymentsTriggerUnknown,
			TriggeredBy:                   sql.NullString{String: marker, Valid: true},
			TriggerReason:                 sql.NullString{},
			CreatedAt:                     now,
			UpdatedAt:                     sql.NullInt64{},
		})
		if err != nil {
			return fmt.Errorf("failed to create seeded deployment: %w", err)
		}
		err = db.Query.InsertDeploymentTopology(ctx, tx, db.InsertDeploymentTopologyParams{
			WorkspaceID:                t.workspaceID,
			DeploymentID:               id,
			RegionID:                   seedRegionID,
			AutoscalingReplicasMin:     1,
			AutoscalingReplicasMax:     1,
			AutoscalingThresholdCpu:    sql.NullInt16{},
			AutoscalingThresholdMemory: sql.NullInt16{},
			DesiredStatus:              db.DeploymentTopologyDesiredStatusRunning,
			CreatedAt:                  now,
		})
		if err != nil {
			return fmt.Errorf("failed to create seeded topology: %w", err)
		}
	}
	return nil
}

func replaceCustomDomains(ctx context.Context, tx db.DBTX, t target, count int, now int64) error {
	suffix := "." + marker + ".test"
	if _, err := tx.ExecContext(ctx, "DELETE FROM custom_domains WHERE workspace_id = ? AND domain LIKE ?", t.workspaceID, "%"+suffix); err != nil {
		return fmt.Errorf("failed to clear seeded custom domains: %w", err)
	}
	for i := range count {
		// failed starts no verification workflow, so nothing touches DNS
		err := db.Query.InsertCustomDomain(ctx, tx, db.InsertCustomDomainParams{
			ID:                    uid.New(uid.DomainPrefix),
			WorkspaceID:           t.workspaceID,
			ProjectID:             t.projectID,
			AppID:                 t.appID,
			EnvironmentID:         t.productionID,
			Domain:                fmt.Sprintf("app-%d%s", i+1, suffix),
			ChallengeType:         db.CustomDomainsChallengeTypeHTTP01,
			VerificationStatus:    db.CustomDomainsVerificationStatusFailed,
			VerificationToken:     uid.Secure(24),
			OwnershipVerified:     false,
			CnameVerified:         false,
			TargetCname:           uid.DNS1035() + suffix,
			VerificationError:     sql.NullString{String: "Seeded by " + marker, Valid: true},
			DomainConnectProvider: sql.NullString{},
			DomainConnectUrl:      sql.NullString{},
			LastCheckedAt:         sql.NullInt64{},
			CreatedAt:             now,
		})
		if err != nil {
			return fmt.Errorf("failed to create seeded custom domain: %w", err)
		}
	}
	return nil
}

func replaceLogdrains(ctx context.Context, tx db.DBTX, workspaceID string, count int, now int64) error {
	if _, err := tx.ExecContext(ctx, "DELETE FROM logdrains WHERE workspace_id = ? AND name LIKE ?", workspaceID, marker+"%"); err != nil {
		return fmt.Errorf("failed to clear seeded log drains: %w", err)
	}
	for i := range count {
		// paused_by_user keeps the logdrain service from delivering to it
		_, err := tx.ExecContext(ctx,
			"INSERT INTO logdrains (id, workspace_id, name, stream, config, status, lease_id, fencing_token, created_at) VALUES (?, ?, ?, 'audit_logs', ?, 'paused_by_user', '', '', ?)",
			uid.New("ld"), workspaceID, fmt.Sprintf("%s-%d", marker, i+1), []byte("{}"), now,
		)
		if err != nil {
			return fmt.Errorf("failed to create seeded log drain: %w", err)
		}
	}
	return nil
}

// replaceBillable writes the billable operations of each month, 90 percent
// key verifications and 10 percent rate limit operations
func replaceBillable(ctx context.Context, ch *clickhouse.Client, workspaceID string, operations map[time.Time]int64) error {
	for _, table := range []string{"billable_verifications_per_month_v2", "billable_ratelimits_per_month_v2"} {
		for month := range operations {
			err := ch.Exec(ctx, "ALTER TABLE default."+table+` DELETE
				WHERE workspace_id = {workspace_id:String} AND year = {year:Int16} AND month = {month:Int8}
				SETTINGS mutations_sync = 1`,
				driver.Named("workspace_id", workspaceID),
				driver.Named("year", month.Year()),
				driver.Named("month", int(month.Month())),
			)
			if err != nil {
				return fmt.Errorf("failed to clear %s: %w", table, err)
			}
		}
	}
	for month, total := range operations {
		for table, count := range map[string]int64{
			"billable_verifications_per_month_v2": total * 9 / 10,
			"billable_ratelimits_per_month_v2":    total - total*9/10,
		} {
			err := ch.Exec(ctx, "INSERT INTO default."+table+" (year, month, workspace_id, count) VALUES (?, ?, ?, ?)",
				month.Year(), int(month.Month()), workspaceID, count)
			if err != nil {
				return fmt.Errorf("failed to insert %s: %w", table, err)
			}
		}
	}
	return nil
}

// replaceActiveKeys writes gateway verifications. This month has keys of the
// default app, keys of a deleted app, and keys without an app id, so the Usage
// page shows a named, a deleted, and an unattributed row
func replaceActiveKeys(ctx context.Context, ch *clickhouse.Client, t target, monthStart, lastMonth time.Time) error {
	err := ch.Exec(ctx, `ALTER TABLE default.key_verifications_per_month_v3 DELETE
		WHERE workspace_id = {workspace_id:String} AND key_space_id = {key_space_id:String}
		SETTINGS mutations_sync = 1`,
		driver.Named("workspace_id", t.workspaceID),
		driver.Named("key_space_id", marker),
	)
	if err != nil {
		return fmt.Errorf("failed to clear seeded key verifications: %w", err)
	}

	deletedAppID := uid.New(uid.AppPrefix)
	for _, group := range []struct {
		month time.Time
		appID string
		keys  int
	}{
		{month: monthStart, appID: t.appID, keys: 8},
		{month: monthStart, appID: deletedAppID, keys: 2},
		{month: monthStart, appID: "", keys: 2},
		{month: lastMonth, appID: t.appID, keys: 6},
	} {
		for range group.keys {
			err := ch.Exec(ctx, `INSERT INTO default.key_verifications_per_month_v3
				(time, workspace_id, key_space_id, identity_id, external_id, key_id, outcome, source, app_id, tags, count)
				VALUES (?, ?, ?, '', '', ?, 'VALID', 'gateway', ?, [], ?)`,
				group.month, t.workspaceID, marker, uid.New(uid.KeyPrefix), group.appID, 25,
			)
			if err != nil {
				return fmt.Errorf("failed to insert seeded key verifications: %w", err)
			}
		}
	}
	return nil
}

// replaceComputeUsage writes hourly usage for both environments from the start
// of the previous month until now, sized from the reserved capacity. It also
// writes one hour of a deleted app and one hour without an app id
func replaceComputeUsage(ctx context.Context, ch *clickhouse.Client, t target, reserved reservation, from, until time.Time) error {
	err := ch.Exec(ctx, `ALTER TABLE default.instance_usage_per_hour_v1 DELETE
		WHERE workspace_id = {workspace_id:String} AND startsWith(resource_id, {marker:String})
		SETTINGS mutations_sync = 1`,
		driver.Named("workspace_id", t.workspaceID),
		driver.Named("marker", marker),
	)
	if err != nil {
		return fmt.Errorf("failed to clear seeded compute usage: %w", err)
	}

	vcpus := float64(reserved.cpuMillicores) / 1000
	memoryGiB := float64(reserved.memoryMib) / 1024
	diskGiB := float64(reserved.storageMib) / 1024
	rows := []usageRow{}
	for hour := from; hour.Before(until); hour = hour.Add(time.Hour) {
		for _, env := range []struct {
			id    string
			share float64
		}{
			{id: t.productionID, share: productionShare},
			{id: t.previewID, share: 1 - productionShare},
		} {
			rows = append(rows, usageRow{
				time:           hour,
				projectID:      t.projectID,
				appID:          t.appID,
				environmentID:  env.id,
				resourceID:     marker + "/" + env.id,
				cpuSeconds:     vcpus * env.share * 3600 * 0.35,
				memoryGiBHours: memoryGiB * env.share * 0.6,
				diskGiBHours:   diskGiB * env.share,
				egressBytes:    int64(env.share * 0.05 * gib),
			})
		}
	}
	monthStart := time.Date(until.Year(), until.Month(), 1, 0, 0, 0, 0, time.UTC)
	rows = append(rows,
		usageRow{
			time:           monthStart,
			projectID:      t.projectID,
			appID:          uid.New(uid.AppPrefix),
			environmentID:  uid.New(uid.EnvironmentPrefix),
			resourceID:     marker + "/deleted",
			cpuSeconds:     900,
			memoryGiBHours: 0.5,
			diskGiBHours:   0,
			egressBytes:    0,
		},
		usageRow{
			time:           monthStart,
			projectID:      t.projectID,
			appID:          "",
			environmentID:  t.previewID,
			resourceID:     marker + "/no-app-id",
			cpuSeconds:     120,
			memoryGiBHours: 0.1,
			diskGiBHours:   0,
			egressBytes:    0,
		},
	)

	batch, err := ch.Conn().PrepareBatch(ctx, `
		INSERT INTO default.instance_usage_per_hour_v1 (
			time, workspace_id, project_id, app_id, environment_id,
			resource_type, resource_id, container_uid, instance_id,
			cpu_seconds, memory_gib_hours, disk_gib_hours,
			network_egress_public_bytes, sample_pairs
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare compute usage batch: %w", err)
	}
	defer func() { _ = batch.Close() }()
	for _, row := range rows {
		err = batch.Append(
			row.time, t.workspaceID, row.projectID, row.appID, row.environmentID,
			"deployment", row.resourceID, row.resourceID, row.resourceID,
			row.cpuSeconds, row.memoryGiBHours, row.diskGiBHours,
			row.egressBytes, int64(240),
		)
		if err != nil {
			return fmt.Errorf("failed to append compute usage: %w", err)
		}
	}
	if err := batch.Send(); err != nil {
		return fmt.Errorf("failed to insert compute usage: %w", err)
	}
	return nil
}
