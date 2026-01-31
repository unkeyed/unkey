package billingjob

import (
	"context"
	"fmt"
	"time"

	"github.com/unkeyed/unkey/pkg/billing"
	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/encryption"
	"github.com/unkeyed/unkey/pkg/otel/logging"
)

// Cmd is the billing-job command that generates invoices for customer billing.
// It runs as a scheduled job (typically monthly) to process billing for all
// workspaces or a specific workspace.
var Cmd = &cli.Command{
	Version:  "",
	Commands: []*cli.Command{},
	Aliases:  []string{},
	Name:     "billing-job",
	Usage:    "Generate billing invoices for customer end users",
	Description: `Generate billing invoices for customer end users based on API usage.

This command aggregates usage data from ClickHouse and generates Stripe invoices
for all end users in the specified workspace(s). It should be run monthly,
typically on the 1st of each month to bill for the previous month's usage.

CONFIGURATION:
The command requires database, ClickHouse, and Stripe credentials to function.
It can process a single workspace or all workspaces with billing enabled.

EXAMPLES:
# Generate invoices for a specific workspace for the previous month
unkey billing-job --workspace-id ws_123 --database-dsn ... --clickhouse-url ... --stripe-key ...

# Generate invoices for a specific billing period
unkey billing-job --workspace-id ws_123 --period-start 2025-01-01 --period-end 2025-02-01 ...

# Using environment variables
WORKSPACE_ID=ws_123 DATABASE_DSN=... CLICKHOUSE_URL=... STRIPE_KEY=... unkey billing-job`,
	Flags: []cli.Flag{
		// Workspace configuration
		cli.String("workspace-id", "Workspace ID to generate invoices for. If not provided, processes all billing-enabled workspaces.",
			cli.EnvVar("WORKSPACE_ID")),

		// Billing period configuration
		cli.String("period-start", "Start of billing period (YYYY-MM-DD). Defaults to first day of previous month.",
			cli.EnvVar("BILLING_PERIOD_START")),
		cli.String("period-end", "End of billing period (YYYY-MM-DD). Defaults to first day of current month.",
			cli.EnvVar("BILLING_PERIOD_END")),

		// Database configuration
		cli.String("database-dsn", "DSN for the primary database",
			cli.EnvVar("DATABASE_DSN"), cli.Required()),

		// ClickHouse configuration
		cli.String("clickhouse-url", "URL for the ClickHouse database",
			cli.EnvVar("CLICKHOUSE_URL"), cli.Required()),

		// Stripe configuration
		cli.String("stripe-key", "Stripe API secret key",
			cli.EnvVar("STRIPE_KEY"), cli.Required()),
		cli.String("stripe-webhook-secret", "Stripe webhook signing secret",
			cli.EnvVar("STRIPE_WEBHOOK_SECRET")),
		cli.String("stripe-client-id", "Stripe Connect client ID for OAuth",
			cli.EnvVar("STRIPE_CLIENT_ID")),

		// Encryption configuration
		cli.String("master-key", "Master encryption key for Stripe tokens (base64 encoded, at least 32 bytes)",
			cli.EnvVar("MASTER_KEY"), cli.Required()),

		// Execution options
		cli.Bool("dry-run", "Preview invoice generation without creating actual invoices",
			cli.Default(false), cli.EnvVar("DRY_RUN")),
		cli.Bool("verbose", "Enable verbose logging",
			cli.Default(false), cli.EnvVar("VERBOSE")),
	},
	Action: run,
}

func run(ctx context.Context, cmd *cli.Command) error {
	logger := logging.New()
	verbose := cmd.Bool("verbose")

	logger.Info("=== Starting Billing Job ===")
	logger.Info("Initializing billing job",
		"dry_run", cmd.Bool("dry-run"),
		"verbose", verbose,
	)

	// Parse billing period
	periodStart, periodEnd, err := parseBillingPeriod(cmd.String("period-start"), cmd.String("period-end"))
	if err != nil {
		logger.Error("Failed to parse billing period", "error", err)
		return fmt.Errorf("failed to parse billing period: %w", err)
	}

	logger.Info("Billing period configuration",
		"period_start", periodStart.Format("2006-01-02"),
		"period_end", periodEnd.Format("2006-01-02"),
		"duration_days", int(periodEnd.Sub(periodStart).Hours()/24),
	)

	// Check for dry run mode
	if cmd.Bool("dry-run") {
		logger.Warn("*** DRY RUN MODE *** No invoices will be created")
	}

	// =========================================
	// Step 1: Connect to Database
	// =========================================
	logger.Info("[1/5] Connecting to database...")
	databaseDSN := cmd.String("database-dsn")
	logger.Debug("Database DSN configured",
		"dsn_length", len(databaseDSN),
	)

	database, err := db.New(db.Config{
		PrimaryDSN:  databaseDSN,
		ReadOnlyDSN: "",
		Logger:      logger,
	})
	if err != nil {
		logger.Error("FAILED to connect to database", "error", err)
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	logger.Info("[1/5] Successfully connected to database")

	// =========================================
	// Step 2: Connect to ClickHouse
	// =========================================
	logger.Info("[2/5] Connecting to ClickHouse...")
	clickhouseURL := cmd.String("clickhouse-url")
	logger.Debug("ClickHouse URL configured",
		"url_length", len(clickhouseURL),
	)

	ch, err := clickhouse.New(clickhouse.Config{
		URL:    clickhouseURL,
		Logger: logger,
	})
	if err != nil {
		logger.Error("FAILED to connect to ClickHouse", "error", err)
		return fmt.Errorf("failed to connect to ClickHouse: %w", err)
	}
	logger.Info("[2/5] Successfully connected to ClickHouse")

	// =========================================
	// Step 3: Initialize Services
	// =========================================
	logger.Info("[3/5] Initializing billing services...")

	// Initialize encryption service
	masterKeyStr := cmd.String("master-key")
	logger.Debug("Master key configured",
		"key_length", len(masterKeyStr),
	)

	masterKey := []byte(masterKeyStr)
	workspaceEncryption, err := encryption.NewWorkspaceEncryption(masterKey)
	if err != nil {
		logger.Error("Failed to initialize encryption", "error", err)
		return fmt.Errorf("failed to initialize encryption: %w", err)
	}
	logger.Info("Encryption service initialized")

	// Initialize billing services
	stripeKey := cmd.String("stripe-key")
	stripeClientID := cmd.String("stripe-client-id")
	webhookSecret := cmd.String("stripe-webhook-secret")

	logger.Debug("Stripe configuration",
		"stripe_key_length", len(stripeKey),
		"client_id_set", stripeClientID != "",
		"webhook_secret_set", webhookSecret != "",
	)

	connectService := billing.NewStripeConnectService(database, workspaceEncryption, stripeClientID)
	pricingService := billing.NewPricingModelService(database)
	endUserService := billing.NewEndUserService(database, ch, stripeKey, connectService)
	usageAggregator := billing.NewUsageAggregator(ch)
	billingService := billing.NewBillingService(
		database,
		usageAggregator,
		pricingService,
		endUserService,
		connectService,
		stripeKey,
		webhookSecret,
	)
	logger.Info("[3/5] All billing services initialized")

	// =========================================
	// Step 4: List Workspaces
	// =========================================
	logger.Info("[4/5] Finding workspaces to process...")

	// Get workspace ID if specified
	workspaceID := cmd.String("workspace-id")
	var workspaces []string

	if workspaceID != "" {
		// Process single workspace
		logger.Info("Processing single workspace", "workspace_id", workspaceID)
		workspaces = []string{workspaceID}
	} else {
		// Process all workspaces with billing enabled
		logger.Info("Looking for all billing-enabled workspaces...")
		workspaces, err = listBillingEnabledWorkspaces(ctx, database)
		if err != nil {
			logger.Error("FAILED to list billing-enabled workspaces", "error", err)
			return fmt.Errorf("failed to list billing-enabled workspaces: %w", err)
		}
		logger.Info("Found billing-enabled workspaces", "count", len(workspaces))
	}

	// =========================================
	// Step 5: Generate Invoices
	// =========================================
	logger.Info("[5/5] Generating invoices...")

	if cmd.Bool("dry-run") {
		logger.Warn("DRY RUN: Would process the following workspaces:")
		for _, wsID := range workspaces {
			logger.Info("  - Would process workspace", "workspace_id", wsID)
		}
		logger.Warn("DRY RUN: No invoices were created")
		logger.Info("=== Billing Job Completed (Dry Run) ===")
		return nil
	}

	// Process each workspace
	var (
		totalWorkspaces      = len(workspaces)
		processedWorkspaces  = 0
		failedWorkspaces     = 0
		totalEndUsers       = 0
		totalVerifications  int64
		totalRateLimits     int64
		totalInvoicesCreated = 0
	)

	for i, wsID := range workspaces {
		logger.Info("-----------------------------------")
		logger.Info(fmt.Sprintf "[%d/%d] Processing workspace", i+1, totalWorkspaces), "workspace_id", wsID)

		// Count end users in workspace
		endUsers, err := endUserService.ListEndUsers(ctx, wsID)
		if err != nil {
			logger.Error("Failed to list end users for workspace",
				"workspace_id", wsID,
				"error", err,
			)
			failedWorkspaces++
			continue
		}
		logger.Info("Found end users in workspace", "workspace_id", wsID, "count", len(endUsers))
		totalEndUsers += len(endUsers)

		// Generate invoices
		err = billingService.GenerateInvoices(ctx, wsID, periodStart, periodEnd)
		if err != nil {
			logger.Error("FAILED to generate invoices for workspace",
				"workspace_id", wsID,
				"error", err,
			)
			failedWorkspaces++
			// Continue processing other workspaces
			continue
		}

		processedWorkspaces++
		totalInvoicesCreated++ // Simplified - actual count would come from billing service

		logger.Info("Successfully generated invoices for workspace",
			"workspace_id", wsID,
		)
	}

	// =========================================
	// Summary
	// =========================================
	logger.Info("===================================")
	logger.Info("=== Billing Job Completed ===")
	logger.Info("Summary:",
		"total_workspaces", totalWorkspaces,
		"processed_workspaces", processedWorkspaces,
		"failed_workspaces", failedWorkspaces,
		"total_end_users", totalEndUsers,
		"invoices_created", totalInvoicesCreated,
	)

	if failedWorkspaces > 0 {
		logger.Error("Some workspaces failed processing",
			"failed_count", failedWorkspaces,
		)
		return fmt.Errorf("failed to generate invoices for %d workspaces", failedWorkspaces)
	}

	logger.Info("All workspaces processed successfully!")
	return nil
}

// parseBillingPeriod parses the billing period from command line arguments.
// If not provided, defaults to the previous month.
func parseBillingPeriod(startStr, endStr string) (time.Time, time.Time, error) {
	now := time.Now().UTC()

	var periodStart, periodEnd time.Time
	var err error

	if startStr != "" {
		periodStart, err = time.Parse("2006-01-02", startStr)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid period-start format (expected YYYY-MM-DD): %w", err)
		}
	} else {
		// Default to first day of previous month
		firstOfCurrentMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		periodStart = firstOfCurrentMonth.AddDate(0, -1, 0)
	}

	if endStr != "" {
		periodEnd, err = time.Parse("2006-01-02", endStr)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid period-end format (expected YYYY-MM-DD): %w", err)
		}
	} else {
		// Default to first day of current month
		periodEnd = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	}

	// Validate period
	if !periodEnd.After(periodStart) {
		return time.Time{}, time.Time{}, fmt.Errorf("period-end must be after period-start")
	}

	return periodStart, periodEnd, nil
}

// listBillingEnabledWorkspaces returns workspace IDs that have billing enabled
// (i.e., have a connected Stripe account).
func listBillingEnabledWorkspaces(ctx context.Context, database db.Database) ([]string, error) {
	// Query workspaces with connected Stripe accounts
	accounts, err := db.Query.StripeConnectedAccountListActive(ctx, database.RO())
	if err != nil {
		return nil, fmt.Errorf("failed to list connected accounts: %w", err)
	}

	workspaceIDs := make([]string, 0, len(accounts))
	for _, account := range accounts {
		workspaceIDs = append(workspaceIDs, account.WorkspaceID)
	}

	return workspaceIDs, nil
}