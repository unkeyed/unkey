package engine

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	logdrainv1 "github.com/unkeyed/unkey/gen/proto/logdrain/v1"
	"github.com/unkeyed/unkey/gen/rpc/vault"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/logdrain/internal/db"
	"github.com/unkeyed/unkey/svc/logdrain/internal/metrics"
	"github.com/unkeyed/unkey/svc/logdrain/internal/source"
	"github.com/unkeyed/unkey/svc/logdrain/sink"
	"google.golang.org/protobuf/proto"
)

// deliveryTimeout bounds one delivery attempt to keep it within the lease refresh margin.
const deliveryTimeout = 30 * time.Second

// maxRetryHint caps destination-directed delays so one response cannot suppress
// delivery attempts for more than one day.
const maxRetryHint = 24 * time.Hour

// workQueueSize is the default bound for queued drains.
const workQueueSize = 1024

// errLeaseLost means a state mutation was rejected by the lease fence.
var errLeaseLost = errors.New("logdrain lease lost")

// Config provides the dependencies and scheduling limits required by [Engine].
type Config struct {
	// DB reads drain configuration and persists delivery state.
	DB db.Database
	// LeaseID selects leases assigned to this process. It must be unique among
	// running processes. Fencing tokens authorize state writes.
	LeaseID string
	// Source reads the event stream being exported.
	Source source.Source
	// Vault decrypts destination credentials for each attempt.
	Vault vault.VaultServiceClient
	// Deliveries accepts delivery telemetry and may be nil to disable it.
	Deliveries deliveryBuffer
	// Clock provides time for watermarks and telemetry; tests inject a mock. Nil defaults to the real clock.
	Clock clock.Clock
	// PollInterval controls how often the engine scans for due drains.
	PollInterval time.Duration
	// WatermarkLag protects against late ClickHouse inserts by delaying timestamp windows.
	WatermarkLag time.Duration
	// BatchSize caps the number of events shipped in one attempt.
	BatchSize int
	// PauseThreshold caps consecutive failures before pausing a drain.
	PauseThreshold int
	// MaxConcurrentDrains sizes the drain worker pool. Values below 1 are treated as 1.
	// Correctness does not depend on this limit. Leases serialize work across
	// processes, and the in-flight set serializes work within this process.
	MaxConcurrentDrains int
	// WorkQueueSize bounds queued drains. Zero uses the default of 1024.
	WorkQueueSize int

	// UnsafeAllowPrivateEndpoints disables the HTTP sink SSRF guard. Development only.
	UnsafeAllowPrivateEndpoints bool
}

// deliveryBuffer accepts delivery telemetry rows for asynchronous insertion into ClickHouse; implementations must not block.
type deliveryBuffer interface {
	Buffer(schema.LogdrainDeliveryV1)
}

// Engine coordinates source paging, destination delivery, and durable offsets.
type Engine struct {
	cfg     Config
	factory factory
	work    chan workItem

	inflightMu sync.Mutex
	inflight   map[string]struct{}
}

// workItem preserves the lease fence and poll watermark for queued work.
type workItem struct {
	id           string
	fencingToken string
	now          time.Time
}

type deliveryAttempt struct {
	completed time.Time
	duration  time.Duration
	result    sink.Result
}

// New wires an engine to its sink factory without starting background work.
func New(cfg Config) (*Engine, error) {
	if err := assert.NotEmpty(cfg.LeaseID, "lease ID must not be empty"); err != nil {
		return nil, err
	}
	if cfg.Clock == nil {
		cfg.Clock = clock.New()
	}
	queueSize := cfg.WorkQueueSize
	if queueSize == 0 {
		queueSize = workQueueSize
	}
	// Keyed by ciphertext, so entries self-invalidate when credentials
	// change. The TTL bounds how long plaintext credentials stay in memory.
	decryptCache, err := cache.New(cache.Config[cache.ScopedKey, string]{
		Fresh:    10 * time.Minute,
		Stale:    10 * time.Minute,
		MaxSize:  10_000,
		Resource: "logdrain_decrypted_credentials",
		Clock:    cfg.Clock,
	})
	if err != nil {
		return nil, fmt.Errorf("build decrypt cache: %w", err)
	}
	return &Engine{
		cfg:        cfg,
		factory:    factory{vault: cfg.Vault, decryptCache: decryptCache, unsafeAllowPrivateEndpoints: cfg.UnsafeAllowPrivateEndpoints},
		work:       make(chan workItem, queueSize),
		inflight:   make(map[string]struct{}),
		inflightMu: sync.Mutex{},
	}, nil
}

// Run polls until ctx is cancelled; a failed poll is logged and retried on the next tick.
func (e *Engine) Run(ctx context.Context) error {
	defer e.factory.decryptCache.Close()
	var workers sync.WaitGroup
	for range max(e.cfg.MaxConcurrentDrains, 1) {
		workers.Go(func() {
			e.worker(ctx)
		})
	}
	defer workers.Wait()
	if err := e.poll(ctx); err != nil {
		logger.Error("logdrain poll failed", "error", err)
	}
	ticker := e.cfg.Clock.NewTicker(e.cfg.PollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C():
			if err := e.poll(ctx); err != nil {
				logger.Error("logdrain poll failed", "error", err)
			}
		}
	}
}

// poll enqueues every drain due at one shared timestamp, blocking when the
// queue is full until workers make room or the context is cancelled.
// The shared enqueue timestamp gives queued work a consistent, conservative
// watermark. The in-flight set prevents duplicate local work. The lease ID
// routes leases, while the fencing token rejects stale workers.
func (e *Engine) poll(ctx context.Context) error {
	now := e.cfg.Clock.Now()
	drains, err := e.cfg.DB.ListDueLogdrains(ctx, e.cfg.LeaseID)
	if err != nil {
		metrics.PollsTotal.WithLabelValues("error").Inc()
		return fmt.Errorf("list due logdrains: %w", err)
	}
	metrics.PollsTotal.WithLabelValues("success").Inc()
	type drainGroup struct {
		status db.LogdrainsStatus
		stream db.LogdrainsStream
	}
	counts := map[drainGroup]int64{}
	for _, status := range []db.LogdrainsStatus{
		db.LogdrainsStatusRunning,
		db.LogdrainsStatusPausedByUser,
		db.LogdrainsStatusPausedByFailure,
	} {
		for _, stream := range []db.LogdrainsStream{db.LogdrainsStreamAuditLogs} {
			counts[drainGroup{status: status, stream: stream}] = 0
		}
	}
	groups, countErr := e.cfg.DB.CountLogdrainsByStatus(ctx)
	if countErr != nil {
		logger.Error("count logdrains by status failed", "error", countErr)
	} else {
		for _, group := range groups {
			counts[drainGroup{status: group.Status, stream: group.Stream}] = group.Drains
		}
		for group, count := range counts {
			metrics.Drains.WithLabelValues(string(group.status), string(group.stream)).Set(float64(count))
		}
	}
	for _, drain := range drains {
		e.tryEnqueue(ctx, workItem{id: drain.LogdrainID, fencingToken: drain.FencingToken, now: now})
	}
	e.inflightMu.Lock()
	inflight := len(e.inflight)
	e.inflightMu.Unlock()
	metrics.WorkQueueDepth.Set(float64(len(e.work)))
	metrics.WorkQueueCapacity.Set(float64(cap(e.work)))
	metrics.InflightDrains.Set(float64(inflight))
	return nil
}

// tryEnqueue blocks after claiming an in-flight slot so due drains are never dropped.
func (e *Engine) tryEnqueue(ctx context.Context, item workItem) {
	e.inflightMu.Lock()
	if _, ok := e.inflight[item.id]; ok {
		e.inflightMu.Unlock()
		return
	}
	e.inflight[item.id] = struct{}{}
	e.inflightMu.Unlock()
	select {
	case e.work <- item:
	case <-ctx.Done():
		e.removeInflight(item.id)
		return
	}
}

// worker processes queued drains until the engine context is cancelled.
func (e *Engine) worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case item := <-e.work:
			e.process(ctx, item)
		}
	}
}

// removeInflight allows a later poll to enqueue the drain again.
func (e *Engine) removeInflight(id string) {
	e.inflightMu.Lock()
	delete(e.inflight, id)
	e.inflightMu.Unlock()
}

// process reads and delivers batches without holding a MySQL transaction.
// Every state write uses the work item's fencing token and valid lease.
func (e *Engine) process(ctx context.Context, item workItem) {
	defer e.removeInflight(item.id)
	delivered := 0
	current := source.Cursor{Time: 0, EventID: ""}
	stream := db.LogdrainsStream("")
	windowEnd := item.now.Add(-e.cfg.WatermarkLag).UnixMilli()
	for {
		drain, err := e.cfg.DB.GetLeasedAndDueLogdrain(ctx, db.GetLeasedAndDueLogdrainParams{
			LogdrainID:   item.id,
			FencingToken: item.fencingToken,
		})
		if errors.Is(err, sql.ErrNoRows) {
			return
		}
		if err != nil {
			logger.Error("read leased logdrain failed", "error", err, "drain_id", item.id)
			return
		}
		stream = drain.Stream
		if stream != db.LogdrainsStreamAuditLogs {
			cause := fmt.Errorf("unsupported stream %q", stream)
			logger.Error("unsupported logdrain stream", "error", cause, "drain_id", item.id)
			if failErr := e.recordFailure(ctx, drain, 0); failErr != nil {
				logger.Error("record logdrain failure state failed", "error", failErr, "drain_id", item.id)
			}
			return
		}
		current = source.Cursor{Time: drain.CommittedOffsetInsertedAt, EventID: drain.CommittedOffsetEventID}
		if windowEnd <= current.Time {
			break
		}
		events, advance, err := e.cfg.Source.Read(ctx, drain.WorkspaceID, current, windowEnd, e.cfg.BatchSize)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			logger.Error("read logdrain source failed", "error", err, "drain_id", item.id)
			if failErr := e.recordFailure(ctx, drain, 0); failErr != nil {
				logger.Error("record logdrain failure state failed", "error", failErr, "drain_id", item.id)
			}
			return
		}
		caughtUp := len(events) < e.cfg.BatchSize
		nextAttemptDelay := time.Duration(0)
		if caughtUp {
			// An empty event ID leaves events exactly at the next window boundary unread.
			advance = source.Cursor{Time: windowEnd, EventID: ""}
			nextAttemptDelay = e.cfg.PollInterval
		}
		var delivery deliveryAttempt
		if len(events) > 0 {
			delivery, err = e.deliverEvents(ctx, drain, events)
			if err != nil {
				logger.Error("deliver logdrain events failed", "error", err, "drain_id", item.id)
				return
			}
			if !delivery.result.Acknowledged {
				return
			}
		}
		rowsAffected, err := e.cfg.DB.RecordLogdrainSuccess(ctx, db.RecordLogdrainSuccessParams{
			CommittedOffsetInsertedAt: advance.Time,
			CommittedOffsetEventID:    advance.EventID,
			NextAttemptDelayMillis:    nextAttemptDelay.Milliseconds(),
			LogdrainID:                drain.ID,
			FencingToken:              drain.FencingToken,
		})
		if err != nil {
			if len(events) > 0 && !delivery.completed.IsZero() {
				e.recordDelivery(drain, stream, delivery.completed, "error", len(events), delivery.duration, delivery.result, err)
			}
			logger.Error("record logdrain success failed", "error", err, "drain_id", item.id)
			return
		}
		if rowsAffected == 0 {
			cause := fmt.Errorf("%w before cursor advance to (%d, %q)", errLeaseLost, advance.Time, advance.EventID)
			if len(events) > 0 && !delivery.completed.IsZero() {
				e.recordDelivery(drain, stream, delivery.completed, "error", len(events), delivery.duration, delivery.result, cause)
			}
			logger.Error("record logdrain success rejected", "error", cause, "drain_id", item.id)
			return
		}
		if len(events) > 0 && !delivery.completed.IsZero() {
			// The delivery succeeded and its new offset is now committed.
			e.recordDelivery(drain, stream, delivery.completed, "success", len(events), delivery.duration, delivery.result, nil)
		}
		delivered += len(events)
		current = advance
		if caughtUp {
			break
		}
	}
	logger.Info("logdrain cycle complete", "drain_id", item.id, "stream", stream, "events", delivered, "offset", current.Time)
}

// deliverEvents ships one batch. An unacknowledged result means the destination
// rejected the delivery. An error means an unexpected delivery or state failure.
func (e *Engine) deliverEvents(ctx context.Context, drain db.GetLeasedAndDueLogdrainRow, events []sink.Event) (deliveryAttempt, error) {
	var attempt deliveryAttempt
	destination, err := e.factory.build(ctx, drain)
	if err != nil {
		if ctx.Err() != nil {
			return attempt, nil
		}
		return attempt, errors.Join(err, e.recordFailure(ctx, drain, 0))
	}
	deliveryCtx, cancel := context.WithTimeout(ctx, deliveryTimeout)
	defer cancel()
	started := e.cfg.Clock.Now()
	result, err := destination.Deliver(deliveryCtx, sink.Batch{SchemaVersion: "v1", DrainID: drain.ID, WorkspaceID: drain.WorkspaceID, Events: events})
	completed := e.cfg.Clock.Now()
	attempt = deliveryAttempt{
		completed: completed,
		duration:  completed.Sub(started),
		result:    result,
	}
	if ctx.Err() != nil {
		return attempt, nil
	}
	if err == nil && result.Acknowledged {
		return attempt, nil
	}
	e.recordDelivery(drain, drain.Stream, attempt.completed, "error", len(events), attempt.duration, result, err)
	if failErr := e.recordFailure(ctx, drain, result.RetryAfter); failErr != nil {
		return attempt, errors.Join(err, failErr)
	}
	return attempt, err
}

// recordDelivery emits asynchronous telemetry when enabled. A nil Deliveries
// buffer disables telemetry.
func (e *Engine) recordDelivery(drain db.GetLeasedAndDueLogdrainRow, stream db.LogdrainsStream, completed time.Time, outcome string, events int, duration time.Duration, result sink.Result, cause error) {
	kind := "unknown"
	cfg := &logdrainv1.Config{}
	if proto.Unmarshal(drain.Config, cfg) == nil {
		switch cfg.Destination.(type) {
		case *logdrainv1.Config_Http:
			kind = "http"
		case *logdrainv1.Config_Axiom:
			kind = "axiom"
		}
	}
	metrics.DeliveriesTotal.WithLabelValues(kind, string(stream), outcome).Inc()
	metrics.DeliveryDurationSeconds.WithLabelValues(kind, string(stream), outcome).Observe(duration.Seconds())
	if outcome == "success" {
		metrics.EventsDeliveredTotal.WithLabelValues(kind, string(stream)).Add(float64(events))
	}
	if e.cfg.Deliveries == nil {
		return
	}
	message := ""
	if cause != nil {
		message = cause.Error()
	}
	responseStatus := int32(0)
	responseBody := ""
	if !result.Acknowledged {
		responseStatus = int32(result.HTTPStatus)
		responseBody = result.ResponseBody
	}
	e.cfg.Deliveries.Buffer(schema.LogdrainDeliveryV1{
		WorkspaceID:       drain.WorkspaceID,
		DrainID:           drain.ID,
		Stream:            string(stream),
		Time:              completed.UnixMilli(),
		Outcome:           outcome,
		Events:            int64(events),
		WebhookDurationMs: duration.Milliseconds(),
		RequestBodyBytes:  result.RequestBodyBytes,
		ResponseStatus:    responseStatus,
		ResponseBody:      responseBody,
		Error:             message,
	})
}

// recordFailure atomically persists retry or pause state for the current fencing token.
func (e *Engine) recordFailure(ctx context.Context, drain db.GetLeasedAndDueLogdrainRow, retryHint time.Duration) error {
	metrics.DrainFailuresTotal.WithLabelValues(string(drain.Stream)).Inc()
	pause := int(drain.ConsecutiveFailures)+1 >= e.cfg.PauseThreshold
	status := db.LogdrainsStatusRunning
	if pause {
		status = db.LogdrainsStatusPausedByFailure
	}
	rowsAffected, err := e.cfg.DB.RecordLogdrainFailure(ctx, db.RecordLogdrainFailureParams{
		Status:           status,
		RetryAfterMillis: retryDelay(int(drain.ConsecutiveFailures), retryHint).Milliseconds(),
		LogdrainID:       drain.ID,
		FencingToken:     drain.FencingToken,
	})
	if err != nil {
		return fmt.Errorf("record logdrain failure: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("%w before failure state update", errLeaseLost)
	}
	if pause {
		metrics.DrainsPausedTotal.WithLabelValues(string(drain.Stream)).Inc()
	}
	return nil
}

// backoff doubles from 1 minute through 128 minutes, then retries every 4 hours.
// The 49 waits before a threshold of 50 span 7 days and 15 minutes.
func backoff(failures int) time.Duration {
	if failures >= 8 {
		return 4 * time.Hour
	}
	return time.Minute * time.Duration(1<<failures)
}

// retryDelay honors a destination hint when it is longer than local backoff.
// A one-day cap prevents a destination from suppressing retries indefinitely.
func retryDelay(failures int, retryHint time.Duration) time.Duration {
	if retryHint > maxRetryHint {
		retryHint = maxRetryHint
	}
	return max(backoff(failures), retryHint)
}
