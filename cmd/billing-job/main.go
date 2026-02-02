package billingjob

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/unkeyed/unkey/pkg/billing"
	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/encryption"
	"github.com/unkeyed/unkey/pkg/otel/logging"
)

func main() {
	if err := Cmd.Run(context.Background(), os.Args[1:]); err != nil {
		os.Exit(1)
	}
}

// No-op logger for db.New() which requires non-nil Logger
type noopLogger struct{}

func (n *noopLogger) With(args ...any) logging.Logger                  { return n }
func (n *noopLogger) WithAttrs(attrs ...slog.Attr) logging.Logger      { return n }
func (n *noopLogger) WithCallDepth(depth int) logging.Logger           { return n }
func (n *noopLogger) Debug(msg string, args ...any)                    {}
func (n *noopLogger) Info(msg string, args ...any)                     {}
func (n *noopLogger) Warn(msg string, args ...any)                     {}
func (n *noopLogger) Error(msg string, args ...any)                    {}

// Simple logger that writes to stdout for CLI output
type simpleLogger struct {
	verbose bool
}

func (l *simpleLogger) Infof(format string, args ...interface{}) {
	logOutput("[INFO] "+format, args...)
}

func (l *simpleLogger) Warnf(format string, args ...interface{}) {
	logOutput("[WARN] "+format, args...)
}

func (l *simpleLogger) Errorf(format string, args ...interface{}) {
	logOutput("[ERROR] "+format, args...)
}

func (l *simpleLogger) Debugf(format string, args ...interface{}) {
	if l.verbose {
		logOutput("[DEBUG] "+format, args...)
	}
}

// logOutput writes to stdout, ignoring errors (un actionable in CLI context)
func logOutput(format string, args ...interface{}) {
	fmt.Printf(format+"\n", args...)
}

// Cmd is the billing-job command that generates invoices for customer billing.
var Cmd = &cli.Command{
	Version:  "",
	Commands: []*cli.Command{},
	Aliases:  []string{},
	Name:     "billing-job",
	Usage:    "Generate billing invoices for customer end users",
	Description: `Generate billing invoices for customer end users based on API usage.`,
	Flags: []cli.Flag{
		cli.String("workspace-id", "Workspace ID to generate invoices for. If not provided, processes all billing-enabled workspaces.",
			cli.EnvVar("WORKSPACE_ID")),
		cli.String("period-start", "Start of billing period (YYYY-MM-DD). Defaults to first day of previous month.",
			cli.EnvVar("BILLING_PERIOD_START")),
		cli.String("period-end", "End of billing period (YYYY-MM-DD). Defaults to first day of current month.",
			cli.EnvVar("BILLING_PERIOD_END")),
		cli.String("database-dsn", "DSN for the primary database",
			cli.EnvVar("DATABASE_DSN"), cli.Required()),
		cli.String("clickhouse-url", "URL for the ClickHouse database",
			cli.EnvVar("CLICKHOUSE_URL"), cli.Required()),
		cli.String("stripe-key", "Stripe API secret key",
			cli.EnvVar("STRIPE_KEY"), cli.Required()),
		cli.String("stripe-webhook-secret", "Stripe webhook signing secret",
			cli.EnvVar("STRIPE_WEBHOOK_SECRET")),
		cli.String("stripe-client-id", "Stripe Connect client ID for OAuth",
			cli.EnvVar("STRIPE_CLIENT_ID")),
		cli.String("master-key", "Master encryption key for Stripe tokens",
			cli.EnvVar("MASTER_KEY"), cli.Required()),
		cli.Bool("dry-run", "Preview invoice generation without creating actual invoices",
			cli.Default(false), cli.EnvVar("DRY_RUN")),
		cli.Bool("verbose", "Enable verbose logging",
			cli.Default(false), cli.EnvVar("VERBOSE")),
	},
	Action: run,
}

func run(ctx context.Context, cmd *cli.Command) error {
	logger := &simpleLogger{verbose: cmd.Bool("verbose")}

	logOutput("=== Starting Billing Job ===")
	logger.Infof("Initializing billing job (dry_run=%v, verbose=%v)", cmd.Bool("dry-run"), cmd.Bool("verbose"))

	// Parse billing period
	periodStart, periodEnd, err := parseBillingPeriod(cmd.String("period-start"), cmd.String("period-end"))
	if err != nil {
		logger.Errorf("Failed to parse billing period: %v", err)
		return fmt.Errorf("failed to parse billing period: %w", err)
	}

	logger.Infof("Billing period: %s to %s (%d days)",
		periodStart.Format("2006-01-02"),
		periodEnd.Format("2006-01-02"),
		int(periodEnd.Sub(periodStart).Hours()/24),
	)

	if cmd.Bool("dry-run") {
		logger.Warnf("*** DRY RUN MODE *** No invoices will be created")
	}

	// Step 1: Connect to Database
	logOutput("[1/5] Connecting to database...\n")
	logger.Debugf("Database DSN length: %d", len(cmd.String("database-dsn")))

	database, err := db.New(db.Config{
		PrimaryDSN:  cmd.String("database-dsn"),
		ReadOnlyDSN: "",
		Logger:      &noopLogger{},
	})
	if err != nil {
		logger.Errorf("FAILED to connect to database: %v", err)
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	logOutput("[1/5] Successfully connected to database\n")

	// Step 2: Connect to ClickHouse
	logOutput("[2/5] Connecting to ClickHouse...\n")
	logger.Debugf("ClickHouse URL length: %d", len(cmd.String("clickhouse-url")))

	ch, err := clickhouse.New(clickhouse.Config{
		URL:    cmd.String("clickhouse-url"),
		Logger: &noopLogger{},
	})
	if err != nil {
		logger.Errorf("FAILED to connect to ClickHouse: %v", err)
		return fmt.Errorf("failed to connect to ClickHouse: %w", err)
	}
	logOutput("[2/5] Successfully connected to ClickHouse\n")

	// Step 3: Initialize Services
	logOutput("[3/5] Initializing billing services...\n")

	// Decode master key from base64 (matching dashboard/vault behavior)
	masterKey, err := base64.StdEncoding.DecodeString(cmd.String("master-key"))
	if err != nil {
		logger.Errorf("Failed to decode master key: %v", err)
		return fmt.Errorf("failed to decode master key: %w", err)
	}
	workspaceEncryption, err := encryption.NewWorkspaceEncryption(masterKey)
	if err != nil {
		logger.Errorf("Failed to initialize encryption: %v", err)
		return fmt.Errorf("failed to initialize encryption: %w", err)
	}
	logOutput("Encryption service initialized\n")

	stripeKey := cmd.String("stripe-key")
	stripeClientID := cmd.String("stripe-client-id")
	webhookSecret := cmd.String("stripe-webhook-secret")

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
	logOutput("[3/5] All billing services initialized\n")

	// Step 4: Find Workspaces
	logOutput("[4/5] Finding workspaces to process...\n")

	workspaceID := cmd.String("workspace-id")
	var workspaces []string

	if workspaceID != "" {
		logger.Infof("Processing single workspace: %s", workspaceID)
		workspaces = []string{workspaceID}
	} else {
		logOutput("Looking for all billing-enabled workspaces...\n")
		workspaces, err = listBillingEnabledWorkspaces(ctx, database)
		if err != nil {
			logger.Errorf("FAILED to list billing-enabled workspaces: %v", err)
			return fmt.Errorf("failed to list billing-enabled workspaces: %w", err)
		}
		logger.Infof("Found %d billing-enabled workspaces\n", len(workspaces))
	}

	// Step 5: Generate Invoices
	logOutput("[5/5] Generating invoices...\n")

	if cmd.Bool("dry-run") {
		logger.Warnf("DRY RUN: Would process %d workspaces:\n", len(workspaces))
		for _, wsID := range workspaces {
			logOutput("  - %s\n", wsID)
		}
		logOutput("=== Billing Job Completed (Dry Run) ===\n")
		return nil
	}

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
		logOutput("-----------------------------------\n")
		logOutput("[%d/%d] Processing workspace: %s\n", i+1, totalWorkspaces, wsID)

		// Count end users in workspace
		endUsers, err := endUserService.ListEndUsers(ctx, wsID)
		if err != nil {
			logger.Errorf("Failed to list end users for workspace %s: %v", wsID, err)
			failedWorkspaces++
			continue
		}
		logOutput("Found %d end users in workspace %s\n", len(endUsers), wsID)
		totalEndUsers += len(endUsers)

		// Generate invoices
		err = billingService.GenerateInvoices(ctx, wsID, periodStart, periodEnd)
		if err != nil {
			logger.Errorf("FAILED to generate invoices for workspace %s: %v", wsID, err)
			failedWorkspaces++
			continue
		}

		processedWorkspaces++
		totalInvoicesCreated++
		logOutput("Successfully generated invoices for workspace %s\n", wsID)
	}

	// Summary
	logOutput("===================================\n")
	logOutput("=== Billing Job Completed ===\n")
	logOutput("Summary:\n")
	logOutput("  Total workspaces: %d\n", totalWorkspaces)
	logOutput("  Processed workspaces: %d\n", processedWorkspaces)
	logOutput("  Failed workspaces: %d\n", failedWorkspaces)
	logOutput("  Total end users: %d\n", totalEndUsers)
	logOutput("  Total verifications: %d\n", totalVerifications)
	logOutput("  Total rate limits: %d\n", totalRateLimits)
	logOutput("  Invoices created: %d\n", totalInvoicesCreated)
	logOutput("===================================\n")

	if failedWorkspaces > 0 {
		logger.Errorf("Some workspaces failed processing: %d/%d\n", failedWorkspaces, totalWorkspaces)
		return fmt.Errorf("failed to generate invoices for %d workspaces", failedWorkspaces)
	}

	logOutput("All workspaces processed successfully!\n")
	return nil
}

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
		firstOfCurrentMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		periodStart = firstOfCurrentMonth.AddDate(0, -1, 0)
	}

	if endStr != "" {
		periodEnd, err = time.Parse("2006-01-02", endStr)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid period-end format (expected YYYY-MM-DD): %w", err)
		}
	} else {
		periodEnd = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	}

	if !periodEnd.After(periodStart) {
		return time.Time{}, time.Time{}, fmt.Errorf("period-end must be after period-start")
	}

	return periodStart, periodEnd, nil
}

func listBillingEnabledWorkspaces(ctx context.Context, database db.Database) ([]string, error) {
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