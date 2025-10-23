package actions

import (
	"context"
	"database/sql"
	"time"

	"github.com/unkeyed/unkey/go/pkg/cli"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

var CreditsCmd = &cli.Command{
	Name:  "credits",
	Usage: "Migrate key credits to separate credits table",
	Description: `Migrate credit data from the keys table to the separate credits table.

This migration will:
1. Find all keys with remaining_requests set that don't have credits
2. Create corresponding entries in the credits table in batches
3. Preserve refill settings (refill_day, refill_amount, last_refill_at)

The migration is idempotent and will skip keys that already have credits entries.

EXAMPLES:
unkey migrate credits                    # Run credits migration
unkey migrate credits --dry-run          # Preview migration without applying changes
unkey migrate credits --batch-size 1000  # Run migration with custom batch size`,
	Flags: []cli.Flag{
		cli.Bool("dry-run", "Preview migration without making changes"),
		cli.Int("batch-size", "Number of keys to process in each batch", cli.Default(1000)),
		cli.String("primary-dsn", "Primary database DSN", cli.Required()),
		cli.String("readonly-dsn", "Read-only database DSN (optional)"),
	},
	Action: migrateCredits,
}

func migrateCredits(ctx context.Context, cmd *cli.Command) error {
	logger := logging.New()

	// Parse flags
	dryRun := cmd.Bool("dry-run")
	batchSize := cmd.Int("batch-size")
	primaryDSN := cmd.String("primary-dsn")
	readonlyDSN := cmd.String("readonly-dsn")

	if dryRun {
		logger.Info("Running in dry-run mode - no changes will be made")
	}

	// Initialize database
	database, err := db.New(db.Config{
		PrimaryDSN:  primaryDSN,
		ReadOnlyDSN: readonlyDSN,
		Logger:      logger,
	})
	if err != nil {
		return fault.Wrap(err, fault.Internal("Failed to initialize database"))
	}
	defer database.Close()

	// Start migration
	logger.Info("Starting credits migration",
		"batch_size", batchSize,
		"dry_run", dryRun,
	)

	var totalMigrated int
	offset := 0

	for {
		// Use the generated sqlc query to find keys without credits
		keys, err := db.Query.FindKeysWithoutCredits(ctx, database.RO(), db.FindKeysWithoutCreditsParams{
			Limit:  int32(batchSize),
			Offset: int32(offset),
		})
		if err != nil {
			return fault.Wrap(err, fault.Internal("Failed to fetch keys"))
		}

		if len(keys) == 0 {
			break // No more keys to process
		}

		logger.Info("Processing batch",
			"batch_size", len(keys),
			"offset", offset,
		)

		if !dryRun {
			// Prepare batch insert parameters
			creditParams := make([]db.InsertCreditParams, 0, len(keys))

			for _, key := range keys {
				creditID := uid.New("credit")
				now := time.Now().UnixMilli()

				// Convert nullable values
				remaining := int32(0)
				if key.RemainingRequests.Valid {
					remaining = key.RemainingRequests.Int32
				}

				// Handle refilled_at - convert from interface{} (can be nil or int64)
				var refilledAt sql.NullInt64
				if key.LastRefillAtUnix != nil {
					if unixTime, ok := key.LastRefillAtUnix.(int64); ok {
						refilledAt = sql.NullInt64{
							Int64: unixTime,
							Valid: true,
						}
					}
				}

				creditParams = append(creditParams, db.InsertCreditParams{
					ID:           creditID,
					WorkspaceID:  key.WorkspaceID,
					KeyID:        sql.NullString{String: key.ID, Valid: true},
					IdentityID:   sql.NullString{String: "", Valid: false}, // null for key credits
					Remaining:    remaining,
					RefillDay:    key.RefillDay,
					RefillAmount: key.RefillAmount,
					CreatedAt:    key.CreatedAtM,
					UpdatedAt:    sql.NullInt64{Int64: now, Valid: true},
					RefilledAt:   refilledAt,
				})
			}

			err = db.BulkQuery.InsertCredits(ctx, database.RW(), creditParams)
			if err != nil {
				logger.Error("Failed to bulk insert credits",
					"batch_size", len(creditParams),
					"error", err.Error(),
				)
			} else {
				totalMigrated += len(creditParams)
			}
		} else {
			// Dry run - just count
			totalMigrated += len(keys)
		}

		if totalMigrated%10000 == 0 && totalMigrated > 0 {
			logger.Info("Migration progress",
				"total_migrated", totalMigrated,
			)
		}

		offset += batchSize
	}

	logger.Info("Migration completed",
		"total_migrated", totalMigrated,
		"dry_run", dryRun,
	)

	return nil
}
