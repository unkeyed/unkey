// Package keyrefill implements the CronService.RunKeyRefill handler.
// The handler processes all keys that need their usage limits refilled
// today and writes one clickhouse_outbox row per refilled key.
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
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/healthcheck"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/restate/restateutil"
	"github.com/unkeyed/unkey/pkg/uid"
)

// stateKeyProcessedKeys tracks key IDs already processed within the
// current VO key (date), so a crash mid-loop resumes cleanly.
const stateKeyProcessedKeys = "processed_keys"

// batchSize is the number of keys to fetch and process in a single batch.
const batchSize = 100

// Config holds the handler's dependencies.
type Config struct {
	// DB is the primary application database. Must not be nil.
	DB db.Database
	// Heartbeat is pinged on successful completion. Must not be nil; use
	// healthcheck.NewNoop() if monitoring is not configured.
	Heartbeat healthcheck.Heartbeat
}

// Handler executes RunKeyRefill.
type Handler struct {
	db        db.Database
	heartbeat healthcheck.Heartbeat
}

// New constructs a Handler.
func New(cfg Config) (*Handler, error) {
	if err := assert.All(
		assert.NotNil(cfg.DB, "DB must not be nil"),
		assert.NotNil(cfg.Heartbeat, "Heartbeat must not be nil; use healthcheck.NewNoop()"),
	); err != nil {
		return nil, err
	}
	return &Handler{db: cfg.DB, heartbeat: cfg.Heartbeat}, nil
}

// Handle processes all keys that need their usage limits refilled
// today. Keyed by date (YYYY-MM-DD); state tracks processed key IDs.
func (h *Handler) Handle(
	ctx restate.ObjectContext,
	_ *hydrav1.RunKeyRefillRequest,
) (*hydrav1.RunKeyRefillResponse, error) {
	dateKey := restate.Key(ctx)
	logger.Info("running key refill", "date", dateKey)

	todayDay, isLastDay, err := parseDateKey(dateKey)
	if err != nil {
		return nil, fmt.Errorf("invalid date key %q: %w", dateKey, err)
	}

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

	for {
		keys, fetchErr := restate.Run(ctx, func(rc restate.RunContext) ([]db.ListKeysForRefillRow, error) {
			return db.Query.ListKeysForRefill(rc, h.db.RO(), db.ListKeysForRefillParams{
				TodayDay:         sql.NullInt16{Int16: int16(todayDay), Valid: true},
				IsLastDayOfMonth: isLastDayInt,
				AfterPk:          cursor,
				Limit:            batchSize,
			})
		}, restate.WithName(fmt.Sprintf("fetch keys batch %d", batchNum)))
		if fetchErr != nil {
			return nil, fmt.Errorf("fetch keys: %w", fetchErr)
		}

		if len(keys) == 0 {
			break
		}

		cursor = keys[len(keys)-1].Pk

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

		nowTime, nowErr := restateutil.Now(ctx)
		if nowErr != nil {
			return nil, fmt.Errorf("get now: %w", nowErr)
		}
		now := nowTime.UnixMilli()

		keyIDs := make([]string, len(keysToProcess))
		for i, key := range keysToProcess {
			keyIDs[i] = key.ID
		}

		_, updateErr := restate.Run(ctx, func(rc restate.RunContext) (restate.Void, error) {
			return restate.Void{}, db.Query.RefillKeysByIDs(rc, h.db.RW(), db.RefillKeysByIDsParams{
				Now: sql.NullInt64{Int64: now, Valid: true},
				Ids: keyIDs,
			})
		}, restate.WithName(fmt.Sprintf("update keys batch %d", batchNum)))
		if updateErr != nil {
			return nil, fmt.Errorf("update keys: %w", updateErr)
		}

		outboxRows, buildErr := buildOutboxRows(keysToProcess, now)
		if buildErr != nil {
			return nil, fmt.Errorf("build outbox rows: %w", buildErr)
		}
		_, auditErr := restate.Run(ctx, func(rc restate.RunContext) (restate.Void, error) {
			if err := db.BulkQuery.InsertClickhouseOutboxes(rc, h.db.RW(), outboxRows); err != nil {
				return restate.Void{}, fmt.Errorf("insert clickhouse outbox rows: %w", err)
			}
			return restate.Void{}, nil
		}, restate.WithName(fmt.Sprintf("insert audit logs batch %d", batchNum)))
		if auditErr != nil {
			return nil, fmt.Errorf("insert audit logs: %w", auditErr)
		}

		for _, key := range keysToProcess {
			processedKeys[key.ID] = true
		}

		restate.Set(ctx, stateKeyProcessedKeys, processedKeys)
		totalKeysRefilled += len(keysToProcess)
		batchNum++

		if totalKeysRefilled%1000 == 0 {
			logger.Info("refill progress", "cursor", cursor, "refilled", totalKeysRefilled)
		}
	}

	logger.Info("key refill complete",
		"date", dateKey,
		"keys_refilled", totalKeysRefilled,
	)

	if _, err := restate.Run(ctx, func(rc restate.RunContext) (restate.Void, error) {
		return restate.Void{}, h.heartbeat.Ping(rc)
	}, restate.WithName("send heartbeat")); err != nil {
		return nil, fmt.Errorf("send heartbeat: %w", err)
	}

	return &hydrav1.RunKeyRefillResponse{
		KeysRefilled: int32(totalKeysRefilled),
	}, nil
}

// parseDateKey parses a "YYYY-MM-DD" prefix and returns the day of
// month and whether it's the last day of the month. Extra suffix
// segments (e.g. "YYYY-MM-DD-test-abc") are ignored.
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

	t := time.Date(year, time.Month(month)+1, 0, 0, 0, 0, 0, time.UTC)
	lastDay := t.Day()

	return day, day == lastDay, nil
}

// buildOutboxRows creates clickhouse_outbox rows for the refilled keys.
// Each row carries one auditlog.Event envelope; the AuditLogExport
// drainer ships them into ClickHouse audit_logs_raw_v1.
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
			Description:   fmt.Sprintf("Refilled key %s", displayName(key)),
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
					Name: displayName(key),
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

// displayName returns a display name for the key, using the name if
// available, falling back to the ID.
func displayName(key db.ListKeysForRefillRow) string {
	if key.Name.Valid && key.Name.String != "" {
		return key.Name.String
	}
	return key.ID
}
