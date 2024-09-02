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
)

type Clickhouse struct {
	conn   ch.Conn
	logger logging.Logger

	requests *batch.BatchProcessor[schema.ApiRequestV1]
}

type Config struct {
	Addr     string
	Username string
	Password string
	Logger   logging.Logger
}

func New(config Config) (*Clickhouse, error) {

	conn, err := ch.Open(&ch.Options{
		Addr: []string{config.Addr},
		Auth: ch.Auth{
			Database: "default",
			Username: config.Username,
			Password: config.Password,
		},
		Debug: true,
		Debugf: func(format string, v ...any) {
			config.Logger.Debug().Msgf(format, v...)
		},
	})
	if err != nil {
		return nil, fault.Wrap(err, fmsg.With("opening clickhouse failed"))
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
				table := "api_requests__v1"
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
