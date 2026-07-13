// Package cron wires hydra.v1.CronService to per-task handlers in subpackages.
//
// Each RunX delegates. New task: handler subpackage, field on Service, wire in New.
package cron

import (
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/healthcheck"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/ctrl/internal/billingmeter"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/internal/invoicecloser"
	"github.com/unkeyed/unkey/svc/ctrl/worker/cron/auditlogcleanup"
	"github.com/unkeyed/unkey/svc/ctrl/worker/cron/auditlogexport"
	"github.com/unkeyed/unkey/svc/ctrl/worker/cron/deploybilling"
	"github.com/unkeyed/unkey/svc/ctrl/worker/cron/idlepreview"
	"github.com/unkeyed/unkey/svc/ctrl/worker/cron/keylastusedsync"
	"github.com/unkeyed/unkey/svc/ctrl/worker/cron/keyrefill"
	"github.com/unkeyed/unkey/svc/ctrl/worker/cron/quotacheck"
	"github.com/unkeyed/unkey/svc/ctrl/worker/cron/ratelimitcleanup"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	rldb "github.com/unkeyed/unkey/internal/services/ratelimit/db"
)

// Service implements hydrav1.CronServiceServer.
type Service struct {
	hydrav1.UnimplementedCronServiceServer

	auditLogCleanup  *auditlogcleanup.Handler
	auditLogExport   *auditlogexport.Handler
	deployBilling    *deploybilling.Handler
	idlePreview      *idlepreview.Handler
	keyLastUsedSync  *keylastusedsync.Handler
	keyRefill        *keyrefill.Handler
	quotaCheck       *quotacheck.Handler
	ratelimitCleanup *ratelimitcleanup.Handler
}

var _ hydrav1.CronServiceServer = (*Service)(nil)

// Heartbeats per task. Use healthcheck.NewNoop() when monitoring is off.
type Heartbeats struct {
	QuotaCheck         healthcheck.Heartbeat
	KeyRefill          healthcheck.Heartbeat
	KeyLastUsedSync    healthcheck.Heartbeat
	AuditLogExport     healthcheck.Heartbeat
	AuditLogCleanup    healthcheck.Heartbeat
	DeployBillingPush  healthcheck.Heartbeat
	DeployBillingClose healthcheck.Heartbeat
}

// Config wires Service dependencies. Only SlackQuotaCheckWebhookURL is optional.
type Config struct {
	// DB is the primary application database. Must not be nil.
	DB db.Database
	// Clickhouse analytics DB. Use clickhouse.NewNoop() if unavailable.
	Clickhouse clickhouse.ClickHouse
	// Clock for cutoffs. Defaults to real time.
	Clock clock.Clock
	// RatelimitDB wraps the ratelimit database. Must not be nil.
	RatelimitDB *rldb.Database

	// SlackQuotaCheckWebhookURL for quota alerts. Empty disables Slack.
	SlackQuotaCheckWebhookURL string

	// Deploy usage from ClickHouse (*clickhouse.Client). Nil disables push.
	BillingUsageReader deploybilling.UsageReader
	// Empty disables Deploy push and close.
	StripeSecretKey string

	// Test override for meter sink. Default from StripeSecretKey, else noop.
	BillingPusher billingmeter.Pusher
	// Test override for invoice closer. Default from StripeSecretKey, else noop.
	BillingCloser invoicecloser.Closer

	Heartbeats Heartbeats
}

// New builds Service from cfg.
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

	// No Stripe key: noop pusher and closer, same cron schedule either way.
	var billingPusher billingmeter.Pusher = billingmeter.NewNoop()
	var billingCloser invoicecloser.Closer = invoicecloser.NewNoop()
	if cfg.StripeSecretKey != "" {
		billingPusher = billingmeter.NewStripe(cfg.StripeSecretKey)
		billingCloser = invoicecloser.NewStripe(cfg.StripeSecretKey)
	} else {
		// Prod with no key bills nobody. Error so alerting catches it.
		logger.Error("deploy billing pusher and invoice closer are DISABLED: no stripe secret key configured")
	}
	if cfg.BillingPusher != nil {
		billingPusher = cfg.BillingPusher
	}
	if cfg.BillingCloser != nil {
		billingCloser = cfg.BillingCloser
	}
	deployBillingH, err := deploybilling.New(deploybilling.Config{
		UsageReader:    cfg.BillingUsageReader,
		Pusher:         billingPusher,
		DB:             cfg.DB,
		Heartbeat:      cfg.Heartbeats.DeployBillingPush,
		Closer:         billingCloser,
		CloseHeartbeat: cfg.Heartbeats.DeployBillingClose,
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

	return &Service{
		UnimplementedCronServiceServer: hydrav1.UnimplementedCronServiceServer{},
		auditLogCleanup:                auditLogCleanupH,
		auditLogExport:                 auditLogExportH,
		deployBilling:                  deployBillingH,
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

func (s *Service) CloseDeployBillingWorkspace(
	ctx restate.ObjectContext,
	req *hydrav1.CloseDeployBillingWorkspaceRequest,
) (*hydrav1.CloseDeployBillingWorkspaceResponse, error) {
	return s.deployBilling.HandleCloseWorkspace(ctx, req)
}
