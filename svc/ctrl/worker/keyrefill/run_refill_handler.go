package keyrefill

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/uid"
)

const stateKeyProcessedKeys = "processed_keys"

// batchSize is the number of keys to fetch and process in a single batch.
// This balances between minimizing round trips and keeping queries efficient.
const batchSize = 100

// RunRefill processes all keys that need their usage limits refilled.
// This handler is intended to be called on a schedule via GitHub Actions.
func (s *Service) RunRefill(
	ctx restate.ObjectContext,
	_ *hydrav1.RunRefillRequest,
) (*hydrav1.RunRefillResponse, error) {
	dateKey := restate.Key(ctx)
	logger.Info("running key refill", "date", dateKey)

	// Parse date key to get day of month and check if last day
	todayDay, isLastDay, err := parseDateKey(dateKey)
	if err != nil {
		return nil, fmt.Errorf("invalid date key %q: %w", dateKey, err)
	}

	// Load processed keys state for resumability
	processedKeys, err := restate.Get[map[string]bool](ctx, stateKeyProcessedKeys)
	if err != nil {
		return nil, fmt.Errorf("get processed keys state: %w", err)
	}
	if processedKeys == nil {
		processedKeys = make(map[string]bool)
	}

	var totalKeysRefilled int
	var cursor uint64
	var batchNum int

	isLastDayInt := 0
	if isLastDay {
		isLastDayInt = 1
	}

	// Process keys in batches using cursor-based pagination on pk
	for {
		// Fetch batch of keys needing refill via UNION ALL deferred join on pk
		keys, fetchErr := restate.Run(ctx, func(rc restate.RunContext) ([]db.ListKeysForRefillRow, error) {
			return db.Query.ListKeysForRefill(rc, s.db.RO(), db.ListKeysForRefillParams{
				AfterPk:          cursor,
				Limit:            batchSize,
				TodayDay:         sql.NullInt16{Int16: int16(todayDay), Valid: true},
				Limit_2:          batchSize,
				IsLastDayOfMonth: isLastDayInt,
				Limit_3:          batchSize,
				Limit_4:          batchSize,
			})
		}, restate.WithName(fmt.Sprintf("fetch keys batch %d", batchNum)))
		if fetchErr != nil {
			return nil, fmt.Errorf("fetch keys: %w", fetchErr)
		}

		if len(keys) == 0 {
			break
		}

		// Advance cursor to the last pk in this batch
		cursor = keys[len(keys)-1].Pk

		// Filter out already processed keys
		var keysToProcess []db.ListKeysForRefillRow
		for _, key := range keys {
			if !processedKeys[key.ID] {
				keysToProcess = append(keysToProcess, key)
			}
		}

		if len(keysToProcess) == 0 {
			batchNum++
			continue
		}

		// Get current timestamp for updates
		now, nowErr := restate.Run(ctx, func(restate.RunContext) (int64, error) {
			return time.Now().UnixMilli(), nil
		}, restate.WithName("get now timestamp"))
		if nowErr != nil {
			return nil, fmt.Errorf("get now: %w", nowErr)
		}

		keyIDs := make([]string, len(keysToProcess))
		for i, key := range keysToProcess {
			keyIDs[i] = key.ID
		}

		// Batch update keys
		_, updateErr := restate.Run(ctx, func(rc restate.RunContext) (restate.Void, error) {
			return restate.Void{}, db.Query.RefillKeysByIDs(rc, s.db.RW(), db.RefillKeysByIDsParams{
				Now: sql.NullInt64{Int64: now, Valid: true},
				Ids: keyIDs,
			})
		}, restate.WithName(fmt.Sprintf("update keys batch %d", batchNum)))
		if updateErr != nil {
			return nil, fmt.Errorf("update keys: %w", updateErr)
		}

		// Enqueue audit log envelopes into clickhouse_outbox in bulk.
		outboxRows, buildErr := buildOutboxRows(keysToProcess, now)
		if buildErr != nil {
			return nil, fmt.Errorf("build outbox rows: %w", buildErr)
		}
		_, auditErr := restate.Run(ctx, func(rc restate.RunContext) (restate.Void, error) {
			if err := db.BulkQuery.InsertClickhouseOutboxes(rc, s.db.RW(), outboxRows); err != nil {
				return restate.Void{}, fmt.Errorf("insert clickhouse outbox rows: %w", err)
			}
			return restate.Void{}, nil
		}, restate.WithName(fmt.Sprintf("insert audit logs batch %d", batchNum)))
		if auditErr != nil {
			return nil, fmt.Errorf("insert audit logs: %w", auditErr)
		}

		// Mark keys as processed
		for _, key := range keysToProcess {
			processedKeys[key.ID] = true
		}

		restate.Set(ctx, stateKeyProcessedKeys, processedKeys)
		totalKeysRefilled += len(keysToProcess)
		batchNum++

		// Log progress periodically
		if totalKeysRefilled%1000 == 0 {
			logger.Info("refill progress", "cursor", cursor, "refilled", totalKeysRefilled)
		}
	}

	logger.Info("key refill complete",
		"date", dateKey,
		"keys_refilled", totalKeysRefilled,
	)

	// Send heartbeat to indicate successful completion
	_, err = restate.Run(ctx, func(rc restate.RunContext) (restate.Void, error) {
		return restate.Void{}, s.heartbeat.Ping(rc)
	}, restate.WithName("send heartbeat"))
	if err != nil {
		return nil, fmt.Errorf("send heartbeat: %w", err)
	}

	return &hydrav1.RunRefillResponse{
		KeysRefilled: int32(totalKeysRefilled),
	}, nil
}

// parseDateKey parses a date key with a "YYYY-MM-DD" prefix and returns the day of month
// and whether it's the last day of the month. Extra suffix segments (e.g. "YYYY-MM-DD-test-abc")
// are ignored.
func parseDateKey(dateKey string) (day int, isLastDay bool, err error) {
	parts := strings.Split(dateKey, "-")
	if len(parts) < 3 {
		return 0, false, fmt.Errorf("expected at least YYYY-MM-DD format")
	}

	year, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, false, fmt.Errorf("invalid year: %w", err)
	}

	month, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, false, fmt.Errorf("invalid month: %w", err)
	}
	if month < 1 || month > 12 {
		return 0, false, fmt.Errorf("month must be 1-12")
	}

	day, err = strconv.Atoi(parts[2])
	if err != nil {
		return 0, false, fmt.Errorf("invalid day: %w", err)
	}

	// Calculate last day of month
	t := time.Date(year, time.Month(month)+1, 0, 0, 0, 0, 0, time.UTC)
	lastDay := t.Day()

	return day, day == lastDay, nil
}

// buildOutboxRows creates clickhouse_outbox rows for the refilled keys.
// Each row carries one auditlog.Event envelope (key target only;
// and the AuditLogExportService drainer ships them into ClickHouse
// audit_logs_raw_v1 on the next tick.
func buildOutboxRows(keys []db.ListKeysForRefillRow, now int64) ([]db.InsertClickhouseOutboxParams, error) {
	rows := make([]db.InsertClickhouseOutboxParams, 0, len(keys))

	for _, key := range keys {
		envelope := auditlog.Event{
			EventID:       uid.New(uid.AuditLogPrefix, 24),
			Time:          now,
			WorkspaceID:   key.WorkspaceID,
			Bucket:        "unkey_mutations",
			Source:        auditlog.EventSourcePlatform,
			Event:         string(auditlog.KeyUpdateEvent),
			Description:   fmt.Sprintf("Refilled key %s", keyDisplayName(key)),
			RemoteIP:      "",
			UserAgent:     "",
			Meta:          nil,
			CorrelationID: "",
			Actor: auditlog.EventActor{
				Type: "system",
				ID:   "keyrefill",
				Name: "Key Refill Service",
				Meta: nil,
			},
			Targets: []auditlog.EventTarget{
				{
					Type: "key",
					ID:   key.ID,
					Name: keyDisplayName(key),
					Meta: map[string]any{
						"refill_amount":      key.RefillAmount.Int64,
						"previous_remaining": key.RemainingRequests.Int64,
						"new_remaining":      key.RefillAmount.Int64,
					},
				},
			},
		}
		payload, err := json.Marshal(envelope)
		if err != nil {
			return nil, fmt.Errorf("marshal audit envelope for key %s: %w", key.ID, err)
		}

		rows = append(rows, db.InsertClickhouseOutboxParams{
			Version:     auditlog.OutboxVersionV1,
			WorkspaceID: key.WorkspaceID,
			EventID:     envelope.EventID,
			Payload:     payload,
			CreatedAt:   now,
		})
	}

	return rows, nil
}

// keyDisplayName returns a display name for the key, using the name if available.
func keyDisplayName(key db.ListKeysForRefillRow) string {
	if key.Name.Valid && key.Name.String != "" {
		return key.Name.String
	}
	return key.ID
}
