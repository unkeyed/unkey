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
	"time"

	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/email"
	"github.com/unkeyed/unkey/pkg/healthcheck"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/ctrl/internal/billingmeter"
	"github.com/unkeyed/unkey/svc/ctrl/internal/invoicecloser"
	"github.com/unkeyed/unkey/svc/ctrl/internal/workos"
	"github.com/unkeyed/unkey/svc/ctrl/worker/cron/auditlogcleanup"
	"github.com/unkeyed/unkey/svc/ctrl/worker/cron/auditlogexport"
	"github.com/unkeyed/unkey/svc/ctrl/worker/cron/deploybilling"
	"github.com/unkeyed/unkey/svc/ctrl/worker/cron/deployspendcheck"
	"github.com/unkeyed/unkey/svc/ctrl/worker/cron/idlepreview"
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

	auditLogCleanup      *auditlogcleanup.Handler
	auditLogExport       *auditlogexport.Handler
	deployBilling        *deploybilling.Handler
	deployBillingPush    *deploybilling.PushHandler
	deploySpendCheck     *deployspendcheck.Handler
	deploySpendCheckWork *deployspendcheck.CheckHandler
	idlePreview          *idlepreview.Handler
	keyLastUsedSync      *keylastusedsync.Handler
	keyRefill            *keyrefill.Handler
	quotaCheck           *quotacheck.Handler
	ratelimitCleanup     *ratelimitcleanup.Handler
}

var _ hydrav1.CronServiceServer = (*Service)(nil)

// DeployBillingPushServer returns the DeployBillingPushService implementation,
// fanned out to by the deploy billing orchestrator. Bound as its own restate
// service alongside the CronService.
func (s *Service) DeployBillingPushServer() hydrav1.DeployBillingPushServiceServer {
	return s.deployBillingPush
}

// DeploySpendCheckServer returns the DeploySpendCheckService implementation,
// fanned out to by the spend-check orchestrator. Bound as its own restate
// service alongside the CronService.
func (s *Service) DeploySpendCheckServer() hydrav1.DeploySpendCheckServiceServer {
	return s.deploySpendCheckWork
}

// Heartbeats groups the per-task healthcheck pingers. Every field must
// be non-nil — use healthcheck.NewNoop() for tasks where monitoring is
// not configured. This keeps each handler's heartbeat call unconditional
// (no nil checks scattered through the codebase).
type Heartbeats struct {
	QuotaCheck         healthcheck.Heartbeat
	KeyRefill          healthcheck.Heartbeat
	KeyLastUsedSync    healthcheck.Heartbeat
	AuditLogExport     healthcheck.Heartbeat
	AuditLogCleanup    healthcheck.Heartbeat
	DeployBillingPush  healthcheck.Heartbeat
	DeployBillingClose healthcheck.Heartbeat
	DeploySpendCheck   healthcheck.Heartbeat
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

	// BillingPusher overrides the Deploy billing meter sink. Optional: when nil
	// it is derived from StripeSecretKey (Stripe when set, no-op otherwise).
	// Integration tests inject a fake to assert what would be pushed without a
	// real Stripe call.
	BillingPusher billingmeter.Pusher
	// BillingCloser overrides the Deploy invoice closer used by the month-end
	// close. Optional: when nil it is derived from StripeSecretKey. Integration
	// tests inject a fake to record finalize calls.
	BillingCloser invoicecloser.Closer

	// WorkOSAPIKey authenticates the spend-cap check's lookup of org admin
	// emails (the budget-alert recipients). Empty resolves no recipients, so
	// the check logs crossings but sends no email.
	WorkOSAPIKey string
	// ResendAPIKey authenticates the budget-alert email send. Empty uses a noop
	// sender that logs instead of sending.
	ResendAPIKey string
	// BillingBaseURL is the dashboard origin used to build the alert's billing
	// link, e.g. "https://app.unkey.com".
	BillingBaseURL string
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
		assert.NotNil(cfg.Heartbeats.AuditLogCleanup, "Heartbeats.AuditLogCleanup must not be nil; use healthcheck.NewNoop()"),
		assert.NotNil(cfg.Heartbeats.DeployBillingPush, "Heartbeats.DeployBillingPush must not be nil; use healthcheck.NewNoop()"),
		assert.NotNil(cfg.Heartbeats.DeployBillingClose, "Heartbeats.DeployBillingClose must not be nil; use healthcheck.NewNoop()"),
		assert.NotNil(cfg.Heartbeats.DeploySpendCheck, "Heartbeats.DeploySpendCheck must not be nil; use healthcheck.NewNoop()"),
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
	auditLogCleanupH, err := auditlogcleanup.New(auditlogcleanup.Config{
		DB:        cfg.DB,
		Heartbeat: cfg.Heartbeats.AuditLogCleanup,
	})
	if err != nil {
		return nil, err
	}

	// The push is enabled only when ClickHouse (usage source) and Stripe
	// (sink) are both configured; otherwise it runs as a no-op so the cron
	// binding and schedule stay uniform across environments.
	var billingPusher billingmeter.Pusher = billingmeter.NewNoop()
	var billingCloser invoicecloser.Closer = invoicecloser.NewNoop()
	if cfg.StripeSecretKey != "" {
		billingPusher = billingmeter.NewStripe(cfg.StripeSecretKey)
		billingCloser = invoicecloser.NewStripe(cfg.StripeSecretKey)
	} else {
		// Deliberately loud: in an environment that is supposed to bill, a
		// missing Stripe key means usage is silently never pushed and
		// invoices never close. Error level so it pages via log alerting
		// instead of hiding in Info noise.
		logger.Error("deploy billing pusher and invoice closer are DISABLED: no stripe secret key configured")
	}
	// Explicit overrides win over the Stripe-key derivation, so integration
	// tests can drive the push and close against fakes.
	if cfg.BillingPusher != nil {
		billingPusher = cfg.BillingPusher
	}
	if cfg.BillingCloser != nil {
		billingCloser = cfg.BillingCloser
	}
	// The aggregation wait only applies when the close talks to real Stripe;
	// fake closers in tests and the noop have nothing to wait for.
	var billingFinalizeDelay time.Duration
	if cfg.StripeSecretKey != "" {
		billingFinalizeDelay = deploybilling.DefaultFinalizeDelay
	}
	deployBillingH, err := deploybilling.New(deploybilling.Config{
		UsageReader:    cfg.BillingUsageReader,
		DB:             cfg.DB,
		Heartbeat:      cfg.Heartbeats.DeployBillingPush,
		Closer:         billingCloser,
		CloseHeartbeat: cfg.Heartbeats.DeployBillingClose,
		FinalizeDelay:  billingFinalizeDelay,
	})
	if err != nil {
		return nil, err
	}
	idlePreviewH, err := idlepreview.New(idlepreview.Config{
		DB:         cfg.DB,
		Clickhouse: cfg.Clickhouse,
	})
	if err != nil {
		return nil, err
	}

	deployBillingPushH, err := deploybilling.NewPushHandler(billingPusher)
	if err != nil {
		return nil, err
	}

	// Spend check reuses the billing usage reader (same ClickHouse meter query);
	// a nil reader makes the per-workspace check a no-op, matching the push.
	deploySpendCheckH, err := deployspendcheck.New(deployspendcheck.Config{
		DB:        cfg.DB,
		Heartbeat: cfg.Heartbeats.DeploySpendCheck,
	})
	if err != nil {
		return nil, err
	}
	// The alert email uses a real Resend sender only when a key is configured;
	// otherwise it logs. Likewise WorkOS resolves recipients only with a key.
	// Budget alerts use the published template's own sender and subject, so the
	// send leaves From empty; no default From to pass.
	var alertSender email.Sender = email.NewNoop()
	if cfg.ResendAPIKey != "" {
		alertSender = email.NewResend(cfg.ResendAPIKey, "")
	} else {
		// Deliberately loud, same as the billing pusher above: without a
		// Resend key every budget alert and suspension notice is silently
		// dropped, so customers hit their spend cap with no warning.
		logger.Error("deploy spend-cap alert emails are DISABLED: no resend api key configured")
	}
	if cfg.WorkOSAPIKey == "" {
		logger.Error("deploy spend-cap alert recipients are DISABLED: no workos api key configured; alerts resolve no admins")
	}
	deploySpendCheckWorkH, err := deployspendcheck.NewCheckHandler(deployspendcheck.CheckConfig{
		Usage:          cfg.BillingUsageReader,
		Admins:         workos.New(cfg.WorkOSAPIKey),
		Email:          alertSender,
		BillingBaseURL: cfg.BillingBaseURL,
	})
	if err != nil {
		return nil, err
	}

	return &Service{
		UnimplementedCronServiceServer: hydrav1.UnimplementedCronServiceServer{},
		auditLogCleanup:                auditLogCleanupH,
		auditLogExport:                 auditLogExportH,
		deployBilling:                  deployBillingH,
		deployBillingPush:              deployBillingPushH,
		deploySpendCheck:               deploySpendCheckH,
		deploySpendCheckWork:           deploySpendCheckWorkH,
		idlePreview:                    idlePreviewH,
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

func (s *Service) RunAuditLogOutboxCleanup(
	ctx restate.ObjectContext,
	req *hydrav1.RunAuditLogOutboxCleanupRequest,
) (*hydrav1.RunAuditLogOutboxCleanupResponse, error) {
	return s.auditLogCleanup.Handle(ctx, req)
}

func (s *Service) RunDeployBillingPush(
	ctx restate.ObjectContext,
	req *hydrav1.RunDeployBillingPushRequest,
) (*hydrav1.RunDeployBillingPushResponse, error) {
	return s.deployBilling.Handle(ctx, req)
}

func (s *Service) RunScaleDownIdlePreviewDeployments(
	ctx restate.ObjectContext,
	req *hydrav1.RunScaleDownIdlePreviewDeploymentsRequest,
) (*hydrav1.RunScaleDownIdlePreviewDeploymentsResponse, error) {
	return s.idlePreview.Handle(ctx, req)
}

func (s *Service) RunDeployBillingClose(
	ctx restate.ObjectContext,
	req *hydrav1.RunDeployBillingCloseRequest,
) (*hydrav1.RunDeployBillingCloseResponse, error) {
	return s.deployBilling.HandleClose(ctx, req)
}

func (s *Service) RunDeploySpendCheck(
	ctx restate.ObjectContext,
	req *hydrav1.RunDeploySpendCheckRequest,
) (*hydrav1.RunDeploySpendCheckResponse, error) {
	return s.deploySpendCheck.Handle(ctx, req)
}
