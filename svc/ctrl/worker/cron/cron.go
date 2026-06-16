// Package cron implements the unified Restate CronService.
//
// All scheduled background tasks are handlers on a single
// hydra.v1.CronService virtual object. Each task lives in its own
// subpackage (auditlogexport, keyrefill, quotacheck, ...) that defines
// its own state keys, constants, and helpers. The Service struct in
// this file is a thin shim that holds one Handler instance per task
// and delegates each proto-generated RunX method to the corresponding
// Handler's Handle method.
//
// Adding a new cron task is "make a subpackage with a Handle(ctx, req)
// method, add a field on Service, wire it in New, add a one-line
// delegating method" — the namespace stays clean as the set grows.
package cron

import (
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/healthcheck"
	"github.com/unkeyed/unkey/svc/ctrl/internal/billingmeter"
	"github.com/unkeyed/unkey/svc/ctrl/worker/cron/auditlogexport"
	"github.com/unkeyed/unkey/svc/ctrl/worker/cron/deploybilling"
	"github.com/unkeyed/unkey/svc/ctrl/worker/cron/keylastusedsync"
	"github.com/unkeyed/unkey/svc/ctrl/worker/cron/keyrefill"
	"github.com/unkeyed/unkey/svc/ctrl/worker/cron/quotacheck"
	"github.com/unkeyed/unkey/svc/ctrl/worker/cron/ratelimitcleanup"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	rldb "github.com/unkeyed/unkey/internal/services/ratelimit/db"
)

// Service implements [hydrav1.CronServiceServer]. It holds one Handler
// instance per task; each RunX method is a thin delegate to the
// corresponding Handler's Handle method.
type Service struct {
	hydrav1.UnimplementedCronServiceServer

	auditLogExport   *auditlogexport.Handler
	deployBilling    *deploybilling.Handler
	keyLastUsedSync  *keylastusedsync.Handler
	keyRefill        *keyrefill.Handler
	quotaCheck       *quotacheck.Handler
	ratelimitCleanup *ratelimitcleanup.Handler
}

var _ hydrav1.CronServiceServer = (*Service)(nil)

// Heartbeats groups the per-task healthcheck pingers. Every field must
// be non-nil — use healthcheck.NewNoop() for tasks where monitoring is
// not configured. This keeps each handler's heartbeat call unconditional
// (no nil checks scattered through the codebase).
type Heartbeats struct {
	QuotaCheck        healthcheck.Heartbeat
	KeyRefill         healthcheck.Heartbeat
	KeyLastUsedSync   healthcheck.Heartbeat
	AuditLogExport    healthcheck.Heartbeat
	DeployBillingPush healthcheck.Heartbeat
}

// Config holds Service dependencies. All fields except
// SlackQuotaCheckWebhookURL are required.
type Config struct {
	// DB is the primary application database. Must not be nil.
	DB db.Database
	// Clickhouse is the analytics database. Must not be nil — pass
	// clickhouse.NewNoop() if unavailable.
	Clickhouse clickhouse.ClickHouse
	// Clock provides timestamps for cutoffs. Optional; defaults to a real clock.
	Clock clock.Clock
	// RatelimitDB wraps the ratelimit database. Must not be nil.
	RatelimitDB *rldb.Database

	// SlackQuotaCheckWebhookURL is the Slack webhook for quota-exceeded
	// notifications. Empty disables Slack notifications.
	SlackQuotaCheckWebhookURL string

	// BillingUsageReader reads month-to-date Deploy usage for the billing
	// push. Pass the concrete *clickhouse.Client (the meter query is not on
	// the ClickHouse interface). Nil disables the push.
	BillingUsageReader deploybilling.UsageReader
	// StripeSecretKey authenticates the Deploy billing push to Stripe. Empty
	// disables the push.
	StripeSecretKey string

	// Heartbeats is the per-task healthcheck wiring. Every field is required.
	Heartbeats Heartbeats
}

// New constructs a Service. Returns an error if any required field is
// missing.
func New(cfg Config) (*Service, error) {
	if err := assert.All(
		assert.NotNil(cfg.DB, "DB must not be nil"),
		assert.NotNil(cfg.Clickhouse, "Clickhouse must not be nil; use clickhouse.NewNoop() if unavailable"),
		assert.NotNil(cfg.RatelimitDB, "RatelimitDB must not be nil"),
		assert.NotNil(cfg.Heartbeats.QuotaCheck, "Heartbeats.QuotaCheck must not be nil; use healthcheck.NewNoop()"),
		assert.NotNil(cfg.Heartbeats.KeyRefill, "Heartbeats.KeyRefill must not be nil; use healthcheck.NewNoop()"),
		assert.NotNil(cfg.Heartbeats.KeyLastUsedSync, "Heartbeats.KeyLastUsedSync must not be nil; use healthcheck.NewNoop()"),
		assert.NotNil(cfg.Heartbeats.AuditLogExport, "Heartbeats.AuditLogExport must not be nil; use healthcheck.NewNoop()"),
		assert.NotNil(cfg.Heartbeats.DeployBillingPush, "Heartbeats.DeployBillingPush must not be nil; use healthcheck.NewNoop()"),
	); err != nil {
		return nil, err
	}
	if cfg.Clock == nil {
		cfg.Clock = clock.New()
	}

	auditLogExportH, err := auditlogexport.New(auditlogexport.Config{
		DB:         cfg.DB,
		Clickhouse: cfg.Clickhouse,
		Heartbeat:  cfg.Heartbeats.AuditLogExport,
	})
	if err != nil {
		return nil, err
	}
	keyLastUsedSyncH, err := keylastusedsync.New(keylastusedsync.Config{
		Heartbeat: cfg.Heartbeats.KeyLastUsedSync,
	})
	if err != nil {
		return nil, err
	}
	keyRefillH, err := keyrefill.New(keyrefill.Config{
		DB:        cfg.DB,
		Heartbeat: cfg.Heartbeats.KeyRefill,
	})
	if err != nil {
		return nil, err
	}
	quotaCheckH, err := quotacheck.New(quotacheck.Config{
		DB:              cfg.DB,
		Clickhouse:      cfg.Clickhouse,
		Heartbeat:       cfg.Heartbeats.QuotaCheck,
		SlackWebhookURL: cfg.SlackQuotaCheckWebhookURL,
	})
	if err != nil {
		return nil, err
	}
	ratelimitCleanupH, err := ratelimitcleanup.New(ratelimitcleanup.Config{
		DB:    cfg.RatelimitDB,
		Clock: cfg.Clock,
	})
	if err != nil {
		return nil, err
	}

	// The push is enabled only when ClickHouse (usage source) and Stripe
	// (sink) are both configured; otherwise it runs as a no-op so the cron
	// binding and schedule stay uniform across environments.
	var billingPusher billingmeter.Pusher = billingmeter.NewNoop()
	if cfg.StripeSecretKey != "" {
		billingPusher = billingmeter.NewStripe(cfg.StripeSecretKey)
	}
	deployBillingH, err := deploybilling.New(deploybilling.Config{
		UsageReader: cfg.BillingUsageReader,
		Pusher:      billingPusher,
		DB:          cfg.DB,
		Heartbeat:   cfg.Heartbeats.DeployBillingPush,
	})
	if err != nil {
		return nil, err
	}

	return &Service{
		UnimplementedCronServiceServer: hydrav1.UnimplementedCronServiceServer{},
		auditLogExport:                 auditLogExportH,
		deployBilling:                  deployBillingH,
		keyLastUsedSync:                keyLastUsedSyncH,
		keyRefill:                      keyRefillH,
		quotaCheck:                     quotaCheckH,
		ratelimitCleanup:               ratelimitCleanupH,
	}, nil
}

func (s *Service) RunAuditLogExport(
	ctx restate.ObjectContext,
	req *hydrav1.RunAuditLogExportRequest,
) (*hydrav1.RunAuditLogExportResponse, error) {
	return s.auditLogExport.Handle(ctx, req)
}

func (s *Service) RunKeyLastUsedSync(
	ctx restate.ObjectContext,
	req *hydrav1.RunKeyLastUsedSyncRequest,
) (*hydrav1.RunKeyLastUsedSyncResponse, error) {
	return s.keyLastUsedSync.Handle(ctx, req)
}

func (s *Service) RunKeyRefill(
	ctx restate.ObjectContext,
	req *hydrav1.RunKeyRefillRequest,
) (*hydrav1.RunKeyRefillResponse, error) {
	return s.keyRefill.Handle(ctx, req)
}

func (s *Service) RunQuotaCheck(
	ctx restate.ObjectContext,
	req *hydrav1.RunQuotaCheckRequest,
) (*hydrav1.RunQuotaCheckResponse, error) {
	return s.quotaCheck.Handle(ctx, req)
}

func (s *Service) RunRatelimitGlobalCountersCleanup(
	ctx restate.ObjectContext,
	req *hydrav1.RunRatelimitGlobalCountersCleanupRequest,
) (*hydrav1.RunRatelimitGlobalCountersCleanupResponse, error) {
	return s.ratelimitCleanup.Handle(ctx, req)
}

func (s *Service) RunDeployBillingPush(
	ctx restate.ObjectContext,
	req *hydrav1.RunDeployBillingPushRequest,
) (*hydrav1.RunDeployBillingPushResponse, error) {
	return s.deployBilling.Handle(ctx, req)
}
