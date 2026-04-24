package auditlogarchive

import (
	"fmt"

	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/healthcheck"
)

// Service exports expired audit log rows from ClickHouse to S3 as Parquet,
// then deletes them. Singleton VO keyed "default". The S3 bucket is the
// durable compliance record; once a row leaves CH it lives only as a
// Parquet file in this bucket.
type Service struct {
	hydrav1.UnimplementedAuditLogArchiveServiceServer
	clickhouse clickhouse.ClickHouse
	heartbeat  healthcheck.Heartbeat
	cfg        S3Config
	disabled   bool
}

var _ hydrav1.AuditLogArchiveServiceServer = (*Service)(nil)

// S3Config addresses the destination bucket. We embed credentials in the
// CH `s3()` table function call rather than relying on instance IAM; CH
// Cloud nodes don't share the customer's IAM identity.
type S3Config struct {
	// Endpoint is the full S3 base URL, e.g.
	// "https://my-bucket.s3.us-east-1.amazonaws.com" for AWS or
	// "https://s3.eu-central-1.amazonaws.com/my-bucket" for path-style.
	// MinIO and other compatible stores work too. Must NOT end in slash.
	Endpoint string
	// Prefix is the key prefix under the bucket where Parquet files land.
	// Files are written as
	// "{prefix}/cutoff={cutoff_iso}/run_{cutoff_millis}.parquet". Empty
	// prefix is allowed; objects land at the bucket root.
	Prefix string
	// AccessKey + SecretKey authenticate the CH worker against S3. CH
	// passes these in the HTTP request to the bucket.
	AccessKey string
	SecretKey string
}

// Config holds the service dependencies.
type Config struct {
	Clickhouse clickhouse.ClickHouse
	// Heartbeat pings an external monitor after each successful archive
	// pass. Must not be nil. Use healthcheck.NewNoop() if not needed.
	Heartbeat healthcheck.Heartbeat
	// S3 addresses the destination bucket. Required even when Disabled is
	// true so a misconfigured deploy fails fast at boot rather than at
	// the next cron tick.
	S3 S3Config
	// Disabled is the kill switch. When true, RunArchive returns
	// immediately without touching CH or S3. Used to pause retention
	// deletion during incidents.
	Disabled bool
}

// New constructs the service.
func New(cfg Config) (*Service, error) {
	if err := assert.All(
		assert.NotNil(cfg.Clickhouse, "Clickhouse must not be nil"),
		assert.NotNil(cfg.Heartbeat, "Heartbeat must not be nil; use healthcheck.NewNoop() if not needed"),
		assert.NotEmpty(cfg.S3.Endpoint, "S3.Endpoint must not be empty"),
		assert.NotEmpty(cfg.S3.AccessKey, "S3.AccessKey must not be empty"),
		assert.NotEmpty(cfg.S3.SecretKey, "S3.SecretKey must not be empty"),
	); err != nil {
		return nil, err
	}
	if err := validateEndpoint(cfg.S3.Endpoint); err != nil {
		return nil, fmt.Errorf("S3 endpoint: %w", err)
	}
	if err := validatePrefix(cfg.S3.Prefix); err != nil {
		return nil, fmt.Errorf("S3 prefix: %w", err)
	}

	return &Service{
		UnimplementedAuditLogArchiveServiceServer: hydrav1.UnimplementedAuditLogArchiveServiceServer{},
		clickhouse: cfg.Clickhouse,
		heartbeat:  cfg.Heartbeat,
		cfg:        cfg.S3,
		disabled:   cfg.Disabled,
	}, nil
}
