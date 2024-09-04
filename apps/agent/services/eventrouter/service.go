package eventrouter

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

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

func (s *Service) CreateHandler() (string, http.HandlerFunc) {
	return "POST /v0/events", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		s.logger.Info().Msg("Received events")

		ctx, span := tracing.Start(r.Context(), tracing.NewSpanName("eventrouter", "v0/events"))
		defer span.End()

		err := auth.Authorize(ctx, s.authToken, r.Header.Get("Authorization"))
		if err != nil {
			s.logger.Warn().Err(err).Msg("failed to authorize request")
			w.WriteHeader(403)
			w.Write([]byte("Unauthorized"))
			return
		}

		datasource := r.URL.Query().Get("name")
		if datasource == "" {
			w.WriteHeader(400)
			w.Write([]byte("missing ?name= parameter"))
			return
		}

		dec := json.NewDecoder(r.Body)

		rows := []any{}

		for {
			var v any
			err := dec.Decode(&v)
			if err != nil {
				if err == io.EOF {
					break
				}
				s.logger.Err(err).Msg("Error decoding row")
				w.WriteHeader(400)
				w.Write([]byte("Error decoding row"))
				return

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

		b, err := json.Marshal(response)
		if err != nil {
			s.logger.Err(err).Msg("Error marshalling response")
			w.WriteHeader(500)
			w.Write([]byte("Error marshalling response"))
			return
		}
		w.Header().Add("Content-Type", "application/json")
		w.Write(b)

	}
}
