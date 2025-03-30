package clickhouse

import (
	"context"
	"time"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fmsg"
	"github.com/unkeyed/unkey/apps/agent/pkg/batch"
	"github.com/unkeyed/unkey/apps/agent/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/util"
)

type Clickhouse struct {
	conn   ch.Conn
	logger logging.Logger

	requests         *batch.BatchProcessor[schema.ApiRequestV1]
	keyVerifications *batch.BatchProcessor[schema.KeyVerificationRequestV1]
}

type Config struct {
	URL    string
	Logger logging.Logger
}

func New(config Config) (*Clickhouse, error) {

	opts, err := ch.ParseDSN(config.URL)
	if err != nil {
		return nil, fault.Wrap(err, fmsg.With("parsing clickhouse DSN failed"))
	}

	// opts.TLS = &tls.Config{}
	opts.Debug = true
	opts.Debugf = func(format string, v ...any) {
		config.Logger.Debug().Msgf(format, v...)
	}
	conn, err := ch.Open(opts)
	if err != nil {
		return nil, fault.Wrap(err, fmsg.With("opening clickhouse failed"))
	}

	err = util.Retry(func() error {
		return conn.Ping(context.Background())
	}, 10, func(n int) time.Duration {
		return time.Duration(n) * time.Second
	})
	if err != nil {
		return nil, fault.Wrap(err, fmsg.With("pinging clickhouse failed"))
	}
	c := &Clickhouse{
		conn:   conn,
		logger: config.Logger,

		requests: batch.New[schema.ApiRequestV1](batch.Config[schema.ApiRequestV1]{
			BatchSize:     1000,
			BufferSize:    100000,
			FlushInterval: time.Second,
			Consumers:     4,
			Flush: func(ctx context.Context, rows []schema.ApiRequestV1) {
				table := "raw_api_requests_v1"
				err := flush(ctx, conn, table, rows)
				if err != nil {
					config.Logger.Error().Err(err).Str("table", table).Msg("failed to flush batch")
				}
			},
		}),
		keyVerifications: batch.New[schema.KeyVerificationRequestV1](batch.Config[schema.KeyVerificationRequestV1]{
			BatchSize:     1000,
			BufferSize:    100000,
			FlushInterval: time.Second,
			Consumers:     4,
			Flush: func(ctx context.Context, rows []schema.KeyVerificationRequestV1) {
				table := "raw_key_verifications_v1"
				err := flush(ctx, conn, table, rows)
				if err != nil {
					config.Logger.Error().Err(err).Str("table", table).Msg("failed to flush batch")
				}
			},
		}),
	}

	// err = c.conn.Ping(context.Background())
	// if err != nil {
	// 	return nil, fault.Wrap(err, fmsg.With("pinging clickhouse failed"))
	// }
	return c, nil
}

func (c *Clickhouse) Shutdown(ctx context.Context) error {
	c.requests.Close()
	return c.conn.Close()
}

func (c *Clickhouse) BufferApiRequest(req schema.ApiRequestV1) {
	c.requests.Buffer(req)
}

func (c *Clickhouse) BufferKeyVerification(req schema.KeyVerificationRequestV1) {
	c.keyVerifications.Buffer(req)
}
