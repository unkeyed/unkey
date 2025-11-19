package seed

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/unkeyed/unkey/go/pkg/cli"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

var ingressCmd = &cli.Command{
	Name:  "ingress",
	Usage: "Seed database with deployment, gateway, instance, and ingress route for testing ingress/gateway",
	Flags: []cli.Flag{
		cli.String("database-primary", "MySQL database DSN", cli.Default("unkey:password@tcp(127.0.0.1:3306)/unkey?parseTime=true&interpolateParams=true"), cli.EnvVar("UNKEY_DATABASE_PRIMARY")),
		cli.String("slug", "Slug to match local seed (e.g., 'local' uses ws_local, proj_local, etc.)", cli.Default("local")),
		cli.String("hostname", "Hostname for ingress route", cli.Default("unkey.local")),
		cli.String("region", "Region for gateway and instance", cli.Default("local")),
		cli.String("address", "Address for instance (IP or hostname)", cli.Default("127.0.0.1:8787")),
	},
	Action: seedIngress,
}

func seedIngress(ctx context.Context, cmd *cli.Command) error {
	logger := logging.New()

	// Connect to MySQL
	database, err := db.New(db.Config{
		PrimaryDSN:  cmd.RequireString("database-primary"),
		ReadOnlyDSN: "",
		Logger:      logger,
	})
	if err != nil {
		return fmt.Errorf("failed to connect to MySQL: %w", err)
	}

	slug := cmd.String("slug")
	hostname := cmd.String("hostname")
	region := cmd.String("region")
	address := cmd.String("address")
	now := time.Now().UnixMilli()

	// IDs based on slug (must match local seed)
	workspaceID := fmt.Sprintf("ws_%s", slug)
	projectID := fmt.Sprintf("proj_%s", slug)
	envID := fmt.Sprintf("env_%s", slug)

	// New IDs for ingress entities
	deploymentID := uid.New(uid.DeploymentPrefix)
	gatewayID := uid.New(uid.GatewayPrefix)
	instanceID := uid.New(uid.InstancePrefix)
	ingressRouteID := uid.New(uid.IngressRoutePrefix)

	// Run everything in a single transaction
	err = db.Tx(ctx, database.RW(), func(ctx context.Context, tx db.DBTX) error {
		// 1. Create deployment
		logger.Info("creating deployment", "id", deploymentID)
		err := db.Query.InsertDeployment(ctx, tx, db.InsertDeploymentParams{
			ID:                       deploymentID,
			WorkspaceID:              workspaceID,
			ProjectID:                projectID,
			EnvironmentID:            envID,
			GitCommitSha:             sql.NullString{String: "abc123", Valid: true},
			GitBranch:                sql.NullString{String: "main", Valid: true},
			RuntimeConfig:            json.RawMessage(`{}`),
			GatewayConfig:            []byte("{}"),
			GitCommitMessage:         sql.NullString{String: "Local dev seed", Valid: true},
			GitCommitAuthorHandle:    sql.NullString{String: "local", Valid: true},
			GitCommitAuthorAvatarUrl: sql.NullString{},
			GitCommitTimestamp:       sql.NullInt64{Int64: now, Valid: true},
			OpenapiSpec:              sql.NullString{},
			Status:                   db.DeploymentsStatusReady,
			CreatedAt:                now,
			UpdatedAt:                sql.NullInt64{},
		})
		if err != nil && !db.IsDuplicateKeyError(err) {
			return fmt.Errorf("failed to create deployment: %w", err)
		}

		// 2. Create gateway
		logger.Info("creating gateway", "id", gatewayID)
		err = db.Query.InsertGateway(ctx, tx, db.InsertGatewayParams{
			ID:             gatewayID,
			WorkspaceID:    workspaceID,
			EnvironmentID:  envID,
			K8sServiceName: fmt.Sprintf("gateway-%s", slug),
			Region:         region,
			Image:          "unkey/gateway:local",
			Health:         db.NullGatewaysHealth{GatewaysHealth: db.GatewaysHealthHealthy, Valid: true},
			Replicas:       1,
		})
		if err != nil && !db.IsDuplicateKeyError(err) {
			return fmt.Errorf("failed to create gateway: %w", err)
		}

		// 3. Create instance
		logger.Info("creating instance", "id", instanceID, "address", address)
		err = db.Query.UpsertInstance(ctx, tx, db.UpsertInstanceParams{
			ID:            instanceID,
			DeploymentID:  deploymentID,
			WorkspaceID:   workspaceID,
			ProjectID:     projectID,
			Region:        region,
			Address:       address,
			CpuMillicores: 1000,
			MemoryMb:      512,
			Status:        db.InstancesStatusRunning,
		})
		if err != nil {
			return fmt.Errorf("failed to create instance: %w", err)
		}

		// 4. Create ingress route
		logger.Info("creating ingress route", "id", ingressRouteID, "hostname", hostname)
		err = db.Query.InsertIngressRoute(ctx, tx, db.InsertIngressRouteParams{
			ID:            ingressRouteID,
			ProjectID:     projectID,
			DeploymentID:  deploymentID,
			EnvironmentID: envID,
			Hostname:      hostname,
			Sticky:        db.IngressRoutesStickyLive,
			CreatedAt:     now,
			UpdatedAt:     sql.NullInt64{},
		})
		if err != nil && !db.IsDuplicateKeyError(err) {
			return fmt.Errorf("failed to create ingress route: %w", err)
		}

		return nil
	})
	if err != nil {
		return err
	}

	// Print summary
	logger.Info("ingress seed completed successfully")
	logger.Info("deployment", "id", deploymentID)
	logger.Info("gateway", "id", gatewayID)
	logger.Info("instance", "id", instanceID, "address", address)
	logger.Info("ingressRoute", "id", ingressRouteID, "hostname", hostname)

	return nil
}
