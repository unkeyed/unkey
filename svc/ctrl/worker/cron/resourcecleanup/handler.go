package resourcecleanup

import (
	"context"
	"errors"
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/healthcheck"
	"github.com/unkeyed/unkey/pkg/restate/restateutil"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/pkg/metrics"
)

const (
	batchSize     = 500
	batchesPerRun = 10
	queryTimeout  = 10 * time.Second
	objectKey     = "resource-cleanup"
)

type Config struct {
	DB        db.Database
	Heartbeat healthcheck.Heartbeat
}

type Handler struct {
	db        db.Database
	heartbeat healthcheck.Heartbeat
}

func New(cfg Config) (*Handler, error) {
	if err := assert.All(assert.NotNil(cfg.DB, "DB is required"), assert.NotNil(cfg.Heartbeat, "Heartbeat is required")); err != nil {
		return nil, err
	}
	return &Handler{db: cfg.DB, heartbeat: cfg.Heartbeat}, nil
}

func RetryPolicy() restate.HandlerOption {
	return restate.WithInvocationRetryPolicy(
		restate.WithInitialRetryInterval(100*time.Millisecond),
		restate.WithRetryIntervalFactor(2),
		restate.WithMaxRetryInterval(5*time.Second),
		restate.WithMaxRetryAttempts(5),
		restate.KillOnMaxAttempts(),
	)
}

func Schedule(ctx restate.Context) error {
	now, err := restateutil.Now(ctx)
	if err != nil {
		return err
	}
	hydrav1.NewCronServiceClient(ctx, objectKey).RunResourceCleanup().Send(&hydrav1.RunResourceCleanupRequest{},
		restate.WithIdempotencyKey(objectKey+"-"+now.UTC().Format("2006-01-02T15:04")))
	return nil
}

func (h *Handler) Handle(ctx restate.ObjectContext, _ *hydrav1.RunResourceCleanupRequest) (*hydrav1.RunResourceCleanupResponse, error) {
	if err := assert.Equal(restate.Key(ctx), objectKey, "resource cleanup must use its singleton key"); err != nil {
		return nil, restate.ToTerminalError(err)
	}
	var failures error
	for _, scan := range []struct {
		name string
		read func(context.Context, uint64) (repairPage, error)
	}{
		{name: "environments", read: h.scanEnvironments},
		{name: "deployments", read: h.scanDeployments},
	} {
		failures = errors.Join(failures, h.repair(ctx, scan.name, scan.read))
	}
	var total int64
	for _, rule := range cleanupRules {
		deleted, err := h.clean(ctx, rule)
		total += deleted
		failures = errors.Join(failures, err)
	}
	if failures != nil {
		return nil, failures
	}
	if err := restate.RunVoid(ctx, func(rc restate.RunContext) error {
		return h.heartbeat.Ping(rc)
	}, restate.WithName("cleanup heartbeat")); err != nil {
		return nil, err
	}
	return &hydrav1.RunResourceCleanupResponse{RowsDeleted: total}, nil
}

type cleanupPage struct {
	AfterPK uint64
	Scanned int
	Deleted int64
}

func (h *Handler) clean(ctx restate.ObjectContext, rule cleanupRule) (int64, error) {
	if err := reportScan(ctx, rule.name, false); err != nil {
		return 0, err
	}
	cursor, err := restate.Get[uint64](ctx, rule.name)
	if err != nil {
		return 0, err
	}
	var total int64
	for range batchesPerRun {
		page, err := restate.Run(ctx, func(rc restate.RunContext) (cleanupPage, error) {
			return h.cleanPage(rc, rule, cursor)
		}, restate.WithName("cleanup "+rule.name), restate.WithMaxRetryAttempts(3))
		if err != nil {
			return total, fmt.Errorf("cleanup %s: %w", rule.name, err)
		}
		total += page.Deleted
		metrics.ResourceCleanupRowsDeleted.WithLabelValues(rule.name).Add(float64(page.Deleted))
		cursor = page.AfterPK
		if page.Scanned < batchSize {
			cursor = 0
		}
		restate.Set(ctx, rule.name, cursor)
		if cursor == 0 {
			return total, reportScan(ctx, rule.name, true)
		}
	}
	return total, nil
}

func (h *Handler) cleanPage(ctx context.Context, rule cleanupRule, cursor uint64) (cleanupPage, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()
	rows, err := h.db.RW().QueryContext(ctx, rule.scan, cursor, batchSize)
	if err != nil {
		return cleanupPage{}, err
	}
	defer rows.Close()
	page := cleanupPage{AfterPK: cursor, Scanned: 0, Deleted: 0}
	for rows.Next() {
		if err := rows.Scan(&page.AfterPK); err != nil {
			return cleanupPage{}, err
		}
		page.Scanned++
	}
	if err := rows.Err(); err != nil {
		return cleanupPage{}, err
	}
	if err := rows.Close(); err != nil {
		return cleanupPage{}, err
	}
	if page.Scanned == 0 {
		return page, nil
	}
	result, err := h.db.RW().ExecContext(ctx, rule.remove, cursor, page.AfterPK)
	if err != nil {
		return cleanupPage{}, err
	}
	page.Deleted, err = result.RowsAffected()
	return page, err
}

func reportScan(ctx restate.ObjectContext, name string, completed bool) error {
	key := "last-scan:" + name
	lastScan, err := restate.Get[int64](ctx, key)
	if err != nil {
		return err
	}
	if completed || lastScan == 0 {
		now, err := restateutil.Now(ctx)
		if err != nil {
			return err
		}
		lastScan = now.Unix()
		restate.Set(ctx, key, lastScan)
	}
	metrics.ResourceCleanupLastScanTimestamp.WithLabelValues(name).Set(float64(lastScan))
	return nil
}
