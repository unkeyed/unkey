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
	offset := 0

	isLastDayInt := 0
	if isLastDay {
		isLastDayInt = 1
	}

	// Process keys in batches
	for {
		// Fetch batch of keys needing refill
		keys, fetchErr := restate.Run(ctx, func(rc restate.RunContext) ([]db.ListKeysForRefillRow, error) {
			return db.Query.ListKeysForRefill(rc, s.db.RO(), db.ListKeysForRefillParams{
				TodayDay:         sql.NullInt16{Int16: int16(todayDay), Valid: true},
				IsLastDayOfMonth: isLastDayInt,
				Limit:            batchSize,
				Offset:           int32(offset),
			})
		}, restate.WithName(fmt.Sprintf("fetch keys batch %d", offset/batchSize)))
		if fetchErr != nil {
			return nil, fmt.Errorf("fetch keys: %w", fetchErr)
		}

		if len(keys) == 0 {
			break
		}

		// Filter out already processed keys
		var keysToProcess []db.ListKeysForRefillRow
		for _, key := range keys {
			if !processedKeys[key.ID] {
				keysToProcess = append(keysToProcess, key)
			}
		}

		if len(keysToProcess) == 0 {
			break
		}

		// Get current timestamp for updates
		now, nowErr := restate.Run(ctx, func(restate.RunContext) (int64, error) {
			return time.Now().UnixMilli(), nil
		}, restate.WithName("get now timestamp"))
		if nowErr != nil {
			return nil, fmt.Errorf("get now: %w", nowErr)
		}

		// Collect key IDs for batch update
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
		}, restate.WithName(fmt.Sprintf("update keys batch %d", offset/batchSize)))
		if updateErr != nil {
			return nil, fmt.Errorf("update keys: %w", updateErr)
		}

		// Create audit logs in bulk
		auditLogs, auditTargets := buildAuditLogs(keysToProcess, now)
		_, auditErr := restate.Run(ctx, func(rc restate.RunContext) (restate.Void, error) {
			if err := db.BulkQuery.InsertAuditLogs(rc, s.db.RW(), auditLogs); err != nil {
				return restate.Void{}, fmt.Errorf("insert audit logs: %w", err)
			}

			if err := db.BulkQuery.InsertAuditLogTargets(rc, s.db.RW(), auditTargets); err != nil {
				return restate.Void{}, fmt.Errorf("insert audit log targets: %w", err)
			}

			return restate.Void{}, nil
		}, restate.WithName(fmt.Sprintf("insert audit logs batch %d", offset/batchSize)))
		if auditErr != nil {
			return nil, fmt.Errorf("insert audit logs: %w", auditErr)
		}

		// Mark keys as processed
		for _, key := range keysToProcess {
			processedKeys[key.ID] = true
		}

		restate.Set(ctx, stateKeyProcessedKeys, processedKeys)
		totalKeysRefilled += len(keysToProcess)
		offset += len(keys)

		// Log progress periodically
		if offset%1000 == 0 {
			logger.Info("refill progress", "offset", offset, "refilled", totalKeysRefilled)
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

// parseDateKey parses a date key in "YYYY-MM-DD" format and returns the day of month
// and whether it's the last day of the month.
func parseDateKey(dateKey string) (day int, isLastDay bool, err error) {
	parts := strings.Split(dateKey, "-")
	if len(parts) != 3 {
		return 0, false, fmt.Errorf("expected YYYY-MM-DD format")
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

// buildAuditLogs creates audit log entries and targets for refilled keys.
func buildAuditLogs(keys []db.ListKeysForRefillRow, now int64) ([]db.InsertAuditLogParams, []db.InsertAuditLogTargetParams) {
	auditLogs := make([]db.InsertAuditLogParams, 0, len(keys))
	// Each refill creates 2 targets: key target and workspace target
	auditTargets := make([]db.InsertAuditLogTargetParams, 0, len(keys)*2)

	for _, key := range keys {
		auditLogID := uid.New(uid.AuditLogPrefix, 24)
		bucketID := key.WorkspaceID // Default bucket is workspace ID
		bucket := "unkey_mutations"

		// Create audit log entry
		auditLogs = append(auditLogs, db.InsertAuditLogParams{
			ID:          auditLogID,
			WorkspaceID: key.WorkspaceID,
			BucketID:    bucketID,
			Bucket:      bucket,
			Event:       string(auditlog.KeyUpdateEvent),
			Time:        now,
			Display:     fmt.Sprintf("Refilled key %s", keyDisplayName(key)),
			RemoteIp:    sql.NullString{},
			UserAgent:   sql.NullString{},
			ActorType:   "system",
			ActorID:     "keyrefill",
			ActorName:   sql.NullString{String: "Key Refill Service", Valid: true},
			ActorMeta:   json.RawMessage("{}"),
			CreatedAt:   now,
		})

		// Create key target
		keyMeta := map[string]any{
			"refill_amount":      key.RefillAmount.Int32,
			"previous_remaining": key.RemainingRequests.Int32,
			"new_remaining":      key.RefillAmount.Int32,
		}
		keyMetaJSON, _ := json.Marshal(keyMeta)

		auditTargets = append(auditTargets, db.InsertAuditLogTargetParams{
			WorkspaceID: key.WorkspaceID,
			BucketID:    bucketID,
			Bucket:      bucket,
			AuditLogID:  auditLogID,
			DisplayName: keyDisplayName(key),
			Type:        "key",
			ID:          key.ID,
			Name:        key.Name,
			Meta:        keyMetaJSON,
			CreatedAt:   now,
		})

		// Create workspace target
		auditTargets = append(auditTargets, db.InsertAuditLogTargetParams{
			WorkspaceID: key.WorkspaceID,
			BucketID:    bucketID,
			Bucket:      bucket,
			AuditLogID:  auditLogID,
			DisplayName: key.WorkspaceID,
			Type:        "workspace",
			ID:          key.WorkspaceID,
			Name:        sql.NullString{},
			Meta:        json.RawMessage("{}"),
			CreatedAt:   now,
		})
	}

	return auditLogs, auditTargets
}

// keyDisplayName returns a display name for the key, using the name if available.
func keyDisplayName(key db.ListKeysForRefillRow) string {
	if key.Name.Valid && key.Name.String != "" {
		return key.Name.String
	}
	return key.ID
}
