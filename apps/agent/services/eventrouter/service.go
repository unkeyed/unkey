package eventrouter

import (
	"context"
	"encoding/json"
	"io"
	"time"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fmsg"
	"github.com/Southclaws/fault/ftag"
	"github.com/gofiber/fiber/v2"
	"github.com/unkeyed/unkey/apps/agent/gen/openapi"
	"github.com/unkeyed/unkey/apps/agent/pkg/auth"
	"github.com/unkeyed/unkey/apps/agent/pkg/batch"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/metrics"
	"github.com/unkeyed/unkey/apps/agent/pkg/prometheus"
	"github.com/unkeyed/unkey/apps/agent/pkg/tinybird"
	"github.com/unkeyed/unkey/apps/agent/pkg/tracing"
)

type event struct {
	datasource string
	row        any
}

type Config struct {
	BatchSize     int
	BufferSize    int
	FlushInterval time.Duration

	Tinybird  *tinybird.Client
	Logger    logging.Logger
	Metrics   metrics.Metrics
	AuthToken string
}

type Service struct {
	logger    logging.Logger
	metrics   metrics.Metrics
	batcher   batch.BatchProcessor[event]
	tb        *tinybird.Client
	authToken string
}

func New(config Config) (*Service, error) {

	flush := func(ctx context.Context, events []event) {
		if len(events) == 0 {
			return
		}
		// config.Metrics.RecordFlush()
		eventsByDatasource := map[string][]any{}
		for _, e := range events {
			if _, ok := eventsByDatasource[e.datasource]; !ok {
				eventsByDatasource[e.datasource] = []any{}
			}
			eventsByDatasource[e.datasource] = append(eventsByDatasource[e.datasource], e.row)
		}
		for datasource, rows := range eventsByDatasource {
			err := config.Tinybird.Ingest(datasource, rows)
			if err != nil {
				config.Logger.Err(err).Str("datasource", datasource).Int("rows", len(rows)).Msg("Error ingesting")
			}
			prometheus.EventRouterFlushedRows.With(map[string]string{
				"datasource": datasource,
			}).Add(float64(len(rows)))

		}
	}

	batcher := batch.New(batch.Config[event]{
		BatchSize:     config.BatchSize,
		BufferSize:    config.BufferSize,
		FlushInterval: config.FlushInterval,
		Flush:         flush,
	})
	return &Service{
		logger:    config.Logger,
		metrics:   config.Metrics,
		batcher:   *batcher,
		tb:        config.Tinybird,
		authToken: config.AuthToken,
	}, nil
}

func (s *Service) CreateHandler() (string, fiber.Handler) {
	return "/v0/events", func(c *fiber.Ctx) error {
		s.logger.Info().Msg("Received events")

		ctx, span := tracing.Start(c.UserContext(), tracing.NewSpanName("eventrouter", "v0/events"))
		defer span.End()

		err := auth.Authorize(ctx, s.authToken, c.Get("Authorization"))
		if err != nil {
			s.logger.Warn().Err(err).Msg("failed to authorize request")
			return fault.New("unauthorized", ftag.With(ftag.Unauthenticated))

		}

		datasource := c.Query("name")
		if datasource == "" {
			return fault.New("missing query parameter", fmsg.WithDesc("bad request", "missing ?name= parameters"), ftag.With(ftag.InvalidArgument))
		}

		defer c.Request().CloseBodyStream()
		dec := json.NewDecoder(c.Request().BodyStream())

		rows := []any{}

		for {
			var v any
			err := dec.Decode(&v)
			if err != nil {
				if err == io.EOF {
					break
				}
				s.logger.Err(err).Msg("Error decoding row")
				return fault.New("error decoding row", ftag.With(ftag.InvalidArgument))

			}
			rows = append(rows, v)
		}
		s.logger.Info().Int("rows", len(rows)).Msg("Received events")
		s.logger.Info().Int("rows", len(rows)).Msg("Received events")

		for _, row := range rows {
			s.batcher.Buffer(event{datasource, row})
		}

		response := openapi.V0EventsResponseBody{
			SuccessfulRows:  len(rows),
			QuarantinedRows: 0,
		}

		return c.JSON(response)

	}
}
