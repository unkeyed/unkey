package quotacheck

import (
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/healthcheck"
	"github.com/unkeyed/unkey/pkg/otel/logging"
)

// Service implements the QuotaCheckService Restate virtual object.
type Service struct {
	hydrav1.UnimplementedQuotaCheckServiceServer
	db              db.Database
	clickhouse      clickhouse.ClickHouse
	logger          logging.Logger
	heartbeat       healthcheck.Heartbeat
	slackWebhookURL string
}

var _ hydrav1.QuotaCheckServiceServer = (*Service)(nil)

// Config holds the configuration for the quota check service.
type Config struct {
	DB         db.Database
	Clickhouse clickhouse.ClickHouse
	Logger     logging.Logger
	// Heartbeat sends health signals after successful quota check runs.
	// Must not be nil - use healthcheck.NewNoop() if monitoring is not needed.
	Heartbeat healthcheck.Heartbeat
	// SlackWebhookURL is the webhook URL for sending quota exceeded notifications.
	// If empty, no Slack notifications are sent.
	SlackWebhookURL string
}

// New creates a new quota check service.
func New(cfg Config) (*Service, error) {
	if err := assert.NotNil(cfg.Heartbeat, "Heartbeat must not be nil; use healthcheck.NewNoop() if not needed"); err != nil {
		return nil, err
	}

	return &Service{
		UnimplementedQuotaCheckServiceServer: hydrav1.UnimplementedQuotaCheckServiceServer{},
		db:                                   cfg.DB,
		clickhouse:                           cfg.Clickhouse,
		logger:                               cfg.Logger,
		heartbeat:                            cfg.Heartbeat,
		slackWebhookURL:                      cfg.SlackWebhookURL,
	}, nil
}
